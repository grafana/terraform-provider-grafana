package grafana

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/client/search"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/resources/appplatform"
)

var (
	StoreDashboardSHA256 bool
)

func resourceDashboard() *common.Resource {
	schema := &schema.Resource{

		Description: `
Manages Grafana dashboards.

* [Official documentation](https://grafana.com/docs/grafana/latest/dashboards/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/dashboard/)
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
				Description: "The complete dashboard model JSON. When this is a K8s-style resource (has apiVersion and spec under dashboard.grafana.app), " +
					"the provider uses the Grafana App Platform dashboard API instead of the legacy API; namespace is derived from org_id or the provider's Grafana Cloud stack.",
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
	configJSON := d.Get("config_json").(string)
	if appplatform.IsK8sDashboardConfig(configJSON) {
		tflog.Debug(ctx, "grafana_dashboard: config_json is K8s App Platform format, using dynamic K8s API")
		return createDashboardK8s(ctx, d, meta, configJSON)
	}

	client, orgID := OAPIClientFromNewOrgResource(meta, d)
	dashboard, err := makeDashboard(d)
	if err != nil {
		return diag.FromErr(err)
	}
	resp, err := client.Dashboards.PostDashboard(&dashboard)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(MakeOrgResourceID(orgID, *resp.Payload.UID))
	return ReadDashboard(ctx, d, meta)
}

func createDashboardK8s(ctx context.Context, d *schema.ResourceData, meta any, configJSON string) diag.Diagnostics {
	metaClient := meta.(*common.Client)
	_, orgID := OAPIClientFromNewOrgResource(meta, d)
	apiVersion, uid, folderUID, spec, err := appplatform.ParseK8sDashboardConfig(configJSON)
	if err != nil {
		return diag.FromErr(err)
	}
	if f := d.Get("folder").(string); f != "" {
		_, folderUID = SplitOrgResourceID(f)
	}
	overwrite := d.Get("overwrite").(bool)
	outJSON, resultUID, diags := appplatform.CreateDashboardFromK8s(ctx, metaClient, orgID, apiVersion, uid, folderUID, spec, overwrite)
	if diags.HasError() {
		return diags
	}
	d.SetId(MakeOrgResourceID(orgID, resultUID))
	d.Set("config_json", NormalizeDashboardConfigJSON(outJSON))
	return ReadDashboard(ctx, d, meta)
}

func ReadDashboard(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	metaClient := meta.(*common.Client)
	client, orgID, uid := OAPIClientFromExistingOrgResource(meta, d.Id())

	resp, err := client.Dashboards.GetDashboardByUID(uid)
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

	existingConfig := d.Get("config_json").(string)

	// If prior state was K8s format, reconstruct K8s envelope from the legacy
	// API response so that the state matches the user's config format.
	if appplatform.IsK8sDashboardConfig(existingConfig) {
		k8sJSON := appplatform.ReconstructK8sConfigJSON(existingConfig, remoteDashJSON, dashboard.Meta.FolderUID)
		d.Set("config_json", NormalizeDashboardConfigJSON(k8sJSON))
		return nil
	}

	// Legacy format: if `uid` is not set in configuration, delete it from the
	// dashboard JSON we just read from the Grafana API to avoid a spurious diff.
	if existingConfig != "" && !common.SHA256Regexp.MatchString(existingConfig) {
		configuredDashJSON, err := UnmarshalDashboardConfigJSON(existingConfig)
		if err != nil {
			return diag.FromErr(err)
		}
		if _, ok := configuredDashJSON["uid"].(string); !ok {
			delete(remoteDashJSON, "uid")
		}
	}
	d.Set("config_json", NormalizeDashboardConfigJSON(remoteDashJSON))

	return nil
}

func UpdateDashboard(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	configJSON := d.Get("config_json").(string)
	if appplatform.IsK8sDashboardConfig(configJSON) {
		_, _, uid := OAPIClientFromExistingOrgResource(meta, d.Id())
		return updateDashboardK8s(ctx, d, meta, uid)
	}

	client, orgID := OAPIClientFromNewOrgResource(meta, d)
	dashboard, err := makeDashboard(d)
	if err != nil {
		return diag.FromErr(err)
	}
	dashboard.Dashboard.(map[string]any)["id"] = d.Get("dashboard_id").(int)
	dashboard.Overwrite = true
	resp, err := client.Dashboards.PostDashboard(&dashboard)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(MakeOrgResourceID(orgID, *resp.Payload.UID))
	return ReadDashboard(ctx, d, meta)
}

func updateDashboardK8s(ctx context.Context, d *schema.ResourceData, meta any, uid string) diag.Diagnostics {
	metaClient := meta.(*common.Client)
	_, orgID := OAPIClientFromNewOrgResource(meta, d)
	configJSON := d.Get("config_json").(string)
	apiVersion, _, folderUID, spec, err := appplatform.ParseK8sDashboardConfig(configJSON)
	if err != nil {
		return diag.FromErr(err)
	}
	if f := d.Get("folder").(string); f != "" {
		_, folderUID = SplitOrgResourceID(f)
	}
	overwrite := d.Get("overwrite").(bool)
	outJSON, diags := appplatform.UpdateDashboardFromK8s(ctx, metaClient, orgID, apiVersion, uid, folderUID, spec, overwrite)
	if diags.HasError() {
		return diags
	}
	d.Set("config_json", NormalizeDashboardConfigJSON(outJSON))
	return ReadDashboard(ctx, d, meta)
}

func DeleteDashboard(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	configJSON := d.Get("config_json").(string)
	if appplatform.IsK8sDashboardConfig(configJSON) {
		metaClient := meta.(*common.Client)
		apiVersion, _, _, _, err := appplatform.ParseK8sDashboardConfig(configJSON)
		if err != nil {
			return diag.FromErr(err)
		}
		orgID := parseOrgID(d)
		if orgID == 0 {
			orgID = metaClient.GrafanaOrgID
		}
		_, _, uid := OAPIClientFromExistingOrgResource(meta, d.Id())
		return appplatform.DeleteDashboardFromK8s(ctx, metaClient, orgID, apiVersion, uid)
	}
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
// For legacy dashboard JSON it removes `id`, `version`, and panel-level noise.
// For K8s-style configs (apiVersion + spec) it strips computed metadata fields
// and normalizes the spec body the same way.
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

	if _, hasAPIVersion := dashboardJSON["apiVersion"]; hasAPIVersion {
		// K8s-style config: normalize metadata and spec separately.
		if meta, ok := dashboardJSON["metadata"].(map[string]any); ok {
			delete(meta, "uid")
			delete(meta, "folder_uid")
		}
		if spec, ok := dashboardJSON["spec"].(map[string]any); ok {
			normalizeDashboardBody(spec)
		}
	} else {
		normalizeDashboardBody(dashboardJSON)
	}

	j, _ := json.Marshal(dashboardJSON)

	if StoreDashboardSHA256 {
		configHash := sha256.Sum256(j)
		return fmt.Sprintf("%x", configHash[:])
	} else {
		return string(j)
	}
}

// normalizeDashboardBody strips server-computed fields from a flat dashboard
// model (the legacy format or the spec portion of a K8s config).
func normalizeDashboardBody(body map[string]any) {
	delete(body, "id")
	delete(body, "version")

	if panels, ok := body["panels"].([]any); ok {
		for _, panel := range panels {
			panelMap, ok := panel.(map[string]any)
			if !ok {
				continue
			}
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
}

func extractUID(jsonStr string) string {
	var parsed map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		return ""
	}
	if uid, ok := parsed["uid"].(string); ok && uid != "" {
		return uid
	}
	// K8s-style config: uid or name in metadata
	if meta, _ := parsed["metadata"].(map[string]any); meta != nil {
		if u, ok := meta["uid"].(string); ok && u != "" {
			return u
		}
		if n, ok := meta["name"].(string); ok {
			return n
		}
	}
	return ""
}
