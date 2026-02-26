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
				Description: "The complete dashboard model JSON. When this is a K8s-style resource (has apiVersion and spec with v1beta1), " +
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
		tflog.Debug(ctx, "grafana_dashboard: config_json is K8s v1beta1, using App Platform API (not legacy /api/dashboards/db)")
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
	uid, folderUID, spec, err := appplatform.ParseK8sDashboardConfig(configJSON)
	if err != nil {
		return diag.FromErr(err)
	}
	if f := d.Get("folder").(string); f != "" {
		_, folderUID = SplitOrgResourceID(f)
	}
	overwrite := d.Get("overwrite").(bool)
	outJSON, id, diags := appplatform.CreateDashboardFromK8s(ctx, metaClient, orgID, uid, folderUID, spec, overwrite)
	if diags.HasError() {
		return diags
	}
	d.SetId(id)
	normalized := NormalizeDashboardConfigJSON(outJSON)
	d.Set("config_json", normalized)
	return readDashboardK8s(ctx, d, meta)
}

func ReadDashboard(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	if _, _, ok := appplatform.ParseK8sDashboardID(d.Id()); ok {
		tflog.Debug(ctx, "grafana_dashboard: state id is v1beta1, using App Platform API for read")
		return readDashboardK8s(ctx, d, meta)
	}

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

	configJSON := d.Get("config_json").(string)

	// Skip if configJSON string is a sha256 hash
	// If `uid` is not set in configuration, we need to delete it from the
	// dashboard JSON we just read from the Grafana API. This is so it does not
	// create a diff. We can assume the uid was randomly generated by Grafana or
	// it was removed after dashboard creation. In any case, the user doesn't
	// care to manage it.
	if configJSON != "" && !common.SHA256Regexp.MatchString(configJSON) {
		configuredDashJSON, err := UnmarshalDashboardConfigJSON(configJSON)
		if err != nil {
			return diag.FromErr(err)
		}
		if _, ok := configuredDashJSON["uid"].(string); !ok {
			delete(remoteDashJSON, "uid")
		}
	}
	configJSON = NormalizeDashboardConfigJSON(remoteDashJSON)
	d.Set("config_json", configJSON)

	return nil
}

func readDashboardK8s(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	metaClient := meta.(*common.Client)
	configJSON, uid, folderUID, version, url, diags := appplatform.ReadDashboardFromK8s(ctx, metaClient, d.Id())
	if diags.HasError() {
		return diags
	}
	normalized := NormalizeDashboardConfigJSON(configJSON)
	d.Set("config_json", normalized)
	d.Set("uid", uid)
	d.Set("folder", folderUID)
	d.Set("version", 0) // App Platform uses resource version string; schema expects int
	d.Set("url", metaClient.GrafanaSubpath(url))
	d.Set("dashboard_id", 0) // App Platform dashboards do not have numeric id
	_ = version // reserved for future use if schema supports string version
	orgID := parseOrgID(d)
	if orgID == 0 {
		orgID = metaClient.GrafanaOrgID
	}
	d.Set("org_id", strconv.FormatInt(orgID, 10))
	return nil
}

func UpdateDashboard(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	if _, uid, ok := appplatform.ParseK8sDashboardID(d.Id()); ok {
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
	configJSON := d.Get("config_json").(string)
	_, folderUID, spec, err := appplatform.ParseK8sDashboardConfig(configJSON)
	if err != nil {
		return diag.FromErr(err)
	}
	if f := d.Get("folder").(string); f != "" {
		_, folderUID = SplitOrgResourceID(f)
	}
	overwrite := d.Get("overwrite").(bool)
	outJSON, diags := appplatform.UpdateDashboardFromK8s(ctx, metaClient, d.Id(), uid, folderUID, spec, overwrite)
	if diags.HasError() {
		return diags
	}
	d.Set("config_json", NormalizeDashboardConfigJSON(outJSON))
	return readDashboardK8s(ctx, d, meta)
}

func DeleteDashboard(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	if _, _, ok := appplatform.ParseK8sDashboardID(d.Id()); ok {
		metaClient := meta.(*common.Client)
		return appplatform.DeleteDashboardFromK8s(ctx, metaClient, d.Id())
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
