package grafana

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"golang.org/x/mod/semver"

	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/client/dashboards"
	"github.com/grafana/grafana-openapi-client-go/client/search"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

var (
	StoreDashboardSHA256 bool
)

func resourceDashboard() *common.Resource {
	schema := &schema.Resource{

		Description: `
Manages Grafana dashboards.

* [Official documentation](https://grafana.com/docs/grafana/latest/dashboards/)
* [HTTP API (legacy API, recommended for Grafana 12 or earlier)](https://grafana.com/docs/grafana/v11.6/developers/http_api/dashboard/)
* [HTTP API (new Kubernetes-style API, recommended for Grafana 13 and later)](https://grafana.com/docs/grafana/latest/developers/http_api/dashboard/)
`,

		CreateContext: common.WithDashboardMutex[schema.CreateContextFunc](CreateDashboard),
		ReadContext:   ReadDashboard,
		UpdateContext: common.WithDashboardMutex[schema.UpdateContextFunc](UpdateDashboard),
		DeleteContext: common.WithDashboardMutex[schema.DeleteContextFunc](DeleteDashboard),
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		CustomizeDiff: func(ctx context.Context, d *schema.ResourceDiff, meta any) error {
			oldVal, newVal := d.GetChange("config_json")
			oldUID := extractUID(oldVal.(string))
			newUID := extractUID(newVal.(string))
			if oldUID != newUID {
				d.ForceNew("config_json")
			}
			return nil
		},

		Schema: map[string]*schema.Schema{
			"org_id": orgIDAttribute(),
			"uid": {
				Type:     schema.TypeString,
				Computed: true,
				Description: "The unique identifier of a dashboard. This is used to construct its URL. " +
					"It's automatically generated if not provided when creating a dashboard. " +
					"The uid allows having consistent URLs for accessing dashboards and when syncing dashboards between multiple Grafana installs. ",
			},
			"dashboard_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The numeric ID of the dashboard computed by Grafana.",
			},
			"url": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The full URL of the dashboard.",
			},
			"version": {
				Type:     schema.TypeInt,
				Computed: true,
				Description: "Whenever you save a version of your dashboard, a copy of that version is saved " +
					"so that previous versions of your dashboard are not lost.",
			},
			"folder": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The id or UID of the folder to save the dashboard in.",
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					_, old = SplitOrgResourceID(old)
					_, new = SplitOrgResourceID(new)
					return old == "0" && new == "" || old == "" && new == "0" || old == new
				},
			},
			"config_json": {
				Type:         schema.TypeString,
				Required:     true,
				StateFunc:    NormalizeDashboardConfigJSON,
				ValidateFunc: validateDashboardConfigJSON,
				Description: `The complete dashboard model JSON.

Starting with Grafana v13, use the resource corresponding to your dashboard's API version for Kubernetes-style dashboards.

If you decide to use this legacy resource with a Kubernetes-style dashboard definition:
- In Grafana v12, provide the "spec" field of the dashboard definition.
- In Grafana v13 and later, provide the full Kubernetes-style dashboard JSON (including "apiVersion", "kind", "metadata", and "spec").
`,
			},
			"overwrite": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Set to true if you want to overwrite existing dashboard with newer version, same dashboard title in folder or same dashboard uid.",
			},
			"message": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Set a commit message for the version history.",
			},
		},
		SchemaVersion: 1, // The state upgrader was removed in v2. To upgrade, users can first upgrade to the last v1 release, apply, then upgrade to v2.
	}

	return common.NewLegacySDKResource(
		common.CategoryGrafanaOSS,
		"grafana_dashboard",
		orgResourceIDString("uid"),
		schema,
	).WithLister(listerFunctionOrgResource(listDashboards))
}

func listDashboards(ctx context.Context, client *goapi.GrafanaHTTPAPI, orgID int64) ([]string, error) {
	return listDashboardOrFolder(client, orgID, "dash-db")
}

func listDashboardOrFolder(client *goapi.GrafanaHTTPAPI, orgID int64, searchType string) ([]string, error) {
	uids := []string{}
	resp, err := client.Search.Search(search.NewSearchParams().WithType(common.Ref(searchType)))
	if err != nil {
		return nil, err
	}

	for _, item := range resp.Payload {
		uids = append(uids, MakeOrgResourceID(orgID, item.UID))
	}

	return uids, nil
}

func CreateDashboard(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client, orgID := OAPIClientFromNewOrgResource(meta, d)

	dashboard, err := makeDashboard(d)
	if err != nil {
		return diag.FromErr(err)
	}

	if dashboardJSON, ok := dashboard.Dashboard.(map[string]any); ok && isKubernetesStyleDashboard(dashboardJSON) {
		health, err := client.Health.GetHealth(nil)
		if err != nil {
			return diag.FromErr(err)
		}

		v := health.Payload.Version
		if !strings.HasPrefix(v, "v") {
			v = "v" + v
		}

		// For versions v12.x.x, we only support the spec to avoid to receive "empty title error"
		if semver.Major(v) == "v12" {
			return diag.Errorf("Grafana version 12 doesn't accept k8s-style json. You have to send only the spec")
		}
	}

	resp, err := client.Dashboards.PostDashboard(&dashboard)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(MakeOrgResourceID(orgID, *resp.Payload.UID))
	return ReadDashboard(ctx, d, meta)
}

func ReadDashboard(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	metaClient := meta.(*common.Client)
	client, orgID, uid := OAPIClientFromExistingOrgResource(meta, d.Id())

	preferredAPIVersion := preferredDashboardAPIVersion(getDashboardReadConfigJSON(d))

	resp, err := readDashboardByUID(ctx, client, uid, preferredAPIVersion)
	if err, shouldReturn := common.CheckReadError("dashboard", d, err); shouldReturn {
		return err
	}
	dashboard := resp.Payload
	model := dashboard.Dashboard.(map[string]any)

	d.SetId(MakeOrgResourceID(orgID, uid))
	d.Set("org_id", strconv.FormatInt(orgID, 10))
	d.Set("uid", model["uid"].(string))
	d.Set("dashboard_id", int64(model["id"].(float64)))
	d.Set("version", int64(model["version"].(float64)))
	d.Set("url", metaClient.GrafanaSubpath(dashboard.Meta.URL))
	d.Set("folder", dashboard.Meta.FolderUID)

	configJSONBytes, err := json.Marshal(dashboard.Dashboard)
	if err != nil {
		return diag.FromErr(err)
	}
	remoteDashJSON, err := UnmarshalDashboardConfigJSON(string(configJSONBytes))
	if err != nil {
		return diag.FromErr(err)
	}

	configJSON := d.Get("config_json").(string)
	configJSON, err = normalizeDashboardConfigJSONForState(configJSON, remoteDashJSON)
	if err != nil {
		return diag.FromErr(err)
	}
	d.Set("config_json", configJSON)

	return nil
}

func UpdateDashboard(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client, orgID := OAPIClientFromNewOrgResource(meta, d)

	dashboard, err := makeDashboard(d)
	if err != nil {
		return diag.FromErr(err)
	}
	if dashboardJSON, ok := dashboard.Dashboard.(map[string]any); ok && !isKubernetesStyleDashboard(dashboardJSON) {
		dashboardJSON["id"] = d.Get("dashboard_id").(int)
	}
	dashboard.Overwrite = true
	resp, err := client.Dashboards.PostDashboard(&dashboard)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(MakeOrgResourceID(orgID, *resp.Payload.UID))
	return ReadDashboard(ctx, d, meta)
}

func DeleteDashboard(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client, _, uid := OAPIClientFromExistingOrgResource(meta, d.Id())
	_, deleteErr := client.Dashboards.DeleteDashboardByUID(uid)
	err, _ := common.CheckReadError("dashboard", d, deleteErr)
	return err
}

func makeDashboard(d *schema.ResourceData) (models.SaveDashboardCommand, error) {
	_, folderID := SplitOrgResourceID(d.Get("folder").(string))
	dashboard := models.SaveDashboardCommand{
		Overwrite: d.Get("overwrite").(bool),
		Message:   d.Get("message").(string),
		FolderUID: folderID,
	}

	configJSON := d.Get("config_json").(string)
	dashboardJSON, err := UnmarshalDashboardConfigJSON(configJSON)
	if err != nil {
		return dashboard, err
	}
	delete(dashboardJSON, "id")
	dashboard.Dashboard = dashboardJSON
	return dashboard, nil
}

func isKubernetesStyleDashboard(dashboardJSON map[string]any) bool {
	_, hasAPIVersion := dashboardJSON["apiVersion"].(string)
	_, hasKind := dashboardJSON["kind"].(string)
	_, hasSpec := dashboardJSON["spec"].(map[string]any)
	return hasAPIVersion && hasKind && hasSpec
}

func getDashboardReadConfigJSON(d *schema.ResourceData) string {
	rawConfig := d.GetRawConfig()
	if !rawConfig.IsNull() && rawConfig.IsKnown() && rawConfig.Type().IsObjectType() && rawConfig.Type().HasAttribute("config_json") {
		configJSON := rawConfig.GetAttr("config_json")
		if configJSON.IsKnown() && !configJSON.IsNull() && configJSON.Type() == cty.String {
			return configJSON.AsString()
		}
	}

	return d.Get("config_json").(string)
}

func preferredDashboardAPIVersion(configJSON string) string {
	if configJSON == "" || common.SHA256Regexp.MatchString(configJSON) {
		return ""
	}

	dashboardJSON, err := UnmarshalDashboardConfigJSON(configJSON)
	if err != nil || !isKubernetesStyleDashboard(dashboardJSON) {
		return ""
	}

	apiVersion, _ := dashboardJSON["apiVersion"].(string)
	return extractDashboardAPIVersion(apiVersion)
}

func extractDashboardAPIVersion(apiVersion string) string {
	if apiVersion == "" {
		return ""
	}

	if _, version, ok := strings.Cut(apiVersion, "/"); ok && version != "" {
		return version
	}

	if strings.HasPrefix(apiVersion, "v") {
		return apiVersion
	}

	return ""
}

func readDashboardByUID(ctx context.Context, client *goapi.GrafanaHTTPAPI, uid, preferredAPIVersion string) (*dashboards.GetDashboardByUIDOK, error) {
	return client.Dashboards.GetDashboardByUID(uid, func(op *runtime.ClientOperation) {
		op.Context = ctx
		if preferredAPIVersion != "" {
			op.Params = newReadDashboardByUIDParams(ctx, uid, preferredAPIVersion)
		}
	})
}

type readDashboardByUIDParams struct {
	*dashboards.GetDashboardByUIDParams
	apiVersion string
}

func newReadDashboardByUIDParams(ctx context.Context, uid, apiVersion string) *readDashboardByUIDParams {
	return &readDashboardByUIDParams{
		GetDashboardByUIDParams: dashboards.NewGetDashboardByUIDParams().WithContext(ctx).WithUID(uid),
		apiVersion:              apiVersion,
	}
}

func (p *readDashboardByUIDParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {
	if err := p.GetDashboardByUIDParams.WriteToRequest(r, reg); err != nil {
		return err
	}
	if p.apiVersion != "" {
		if err := r.SetQueryParam("apiVersion", p.apiVersion); err != nil {
			return err
		}
	}
	return nil
}
func normalizeDashboardConfigJSONForState(configJSON string, remoteDashJSON map[string]any) (string, error) {
	// Skip if configJSON string is a sha256 hash.
	if configJSON != "" && !common.SHA256Regexp.MatchString(configJSON) {
		configuredDashJSON, err := UnmarshalDashboardConfigJSON(configJSON)
		if err != nil {
			return "", err
		}
		if isKubernetesStyleDashboard(configuredDashJSON) {
			return normalizeKubernetesDashboardConfigJSONForState(configuredDashJSON, remoteDashJSON)
		}
		if _, ok := configuredDashJSON["uid"].(string); !ok {
			delete(remoteDashJSON, "uid")
		}
	}
	return NormalizeDashboardConfigJSON(remoteDashJSON), nil
}

func normalizeKubernetesDashboardConfigJSONForState(configuredDashJSON map[string]any, remoteDashJSON map[string]any) (string, error) {
	configuredSpec, ok := configuredDashJSON["spec"].(map[string]any)
	if !ok {
		return NormalizeDashboardConfigJSON(configuredDashJSON), nil
	}

	localSpecJSON, _, err := normalizeDashboardBodyJSON(configuredSpec)
	if err != nil {
		return "", err
	}
	remoteSpecJSON, remoteSpecMap, err := normalizeDashboardBodyJSON(remoteDashJSON)
	if err != nil {
		return "", err
	}
	if localSpecJSON == remoteSpecJSON {
		return NormalizeDashboardConfigJSON(configuredDashJSON), nil
	}

	stateDashJSON, err := cloneDashboardJSON(configuredDashJSON)
	if err != nil {
		return "", err
	}
	stateDashJSON["spec"] = remoteSpecMap
	return NormalizeDashboardConfigJSON(stateDashJSON), nil
}

func normalizeDashboardBodyJSON(dashboardJSON map[string]any) (string, map[string]any, error) {
	normalizedDashJSON, err := cloneDashboardJSON(dashboardJSON)
	if err != nil {
		return "", nil, err
	}
	delete(normalizedDashJSON, "uid")

	normalizedJSON := NormalizeDashboardConfigJSON(normalizedDashJSON)
	normalizedMap, err := UnmarshalDashboardConfigJSON(normalizedJSON)
	if err != nil {
		return "", nil, err
	}
	return normalizedJSON, normalizedMap, nil
}

func cloneDashboardJSON(dashboardJSON map[string]any) (map[string]any, error) {
	clonedJSONBytes, err := json.Marshal(dashboardJSON)
	if err != nil {
		return nil, err
	}
	clonedDashboardJSON, err := UnmarshalDashboardConfigJSON(string(clonedJSONBytes))
	if err != nil {
		return nil, err
	}
	return clonedDashboardJSON, nil
}

// UnmarshalDashboardConfigJSON is a convenience func for unmarshalling
// `config_json` field.
func UnmarshalDashboardConfigJSON(configJSON string) (map[string]any, error) {
	dashboardJSON := map[string]any{}
	err := json.Unmarshal([]byte(configJSON), &dashboardJSON)
	if err != nil {
		return nil, err
	}
	return dashboardJSON, nil
}

// validateDashboardConfigJSON is the ValidateFunc for `config_json`. It
// ensures its value is valid JSON.
func validateDashboardConfigJSON(config any, k string) ([]string, []error) {
	configJSON := config.(string)
	configMap := map[string]any{}
	err := json.Unmarshal([]byte(configJSON), &configMap)
	if err != nil {
		return nil, []error{err}
	}
	return nil, nil
}

// NormalizeDashboardConfigJSON is the StateFunc for the `config_json` field.
//
// It removes the following fields:
//
//   - `id`:      an auto-incrementing ID Grafana assigns to dashboards upon
//     creation. We cannot know this before creation and therefore it cannot
//     be managed in code.
//   - `version`: is incremented by Grafana each time a dashboard changes.
func NormalizeDashboardConfigJSON(config any) string {
	var dashboardJSON map[string]any
	switch c := config.(type) {
	case map[string]any:
		dashboardJSON = c
	case string:
		var err error
		dashboardJSON, err = UnmarshalDashboardConfigJSON(c)
		if err != nil {
			return c
		}
	}

	delete(dashboardJSON, "id")
	delete(dashboardJSON, "version")

	// similarly to uid removal above, remove any attributes panels[].libraryPanel.*
	// from the dashboard JSON other than "name" or "uid".
	// Grafana will populate all other libraryPanel attributes, so delete them to avoid diff.
	if panels, ok := dashboardJSON["panels"].([]any); ok {
		for _, panel := range panels {
			panelMap := panel.(map[string]any)
			delete(panelMap, "id")
			if libraryPanel, ok := panelMap["libraryPanel"].(map[string]any); ok {
				for k := range libraryPanel {
					if k != "name" && k != "uid" {
						delete(libraryPanel, k)
					}
				}
			}
		}
	}

	j, _ := json.Marshal(dashboardJSON)

	if StoreDashboardSHA256 {
		configHash := sha256.Sum256(j)
		return fmt.Sprintf("%x", configHash[:])
	} else {
		return string(j)
	}
}

func extractUID(jsonStr string) string {
	var parsed map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		return ""
	}
	if uid, ok := parsed["uid"].(string); ok {
		return uid
	}
	return ""
}
