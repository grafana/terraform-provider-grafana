package grafana

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	gapi "github.com/grafana/grafana-api-golang-client"
)

func ResourceDashboard() *schema.Resource {
	return &schema.Resource{

		Description: `
Manages Grafana dashboards.

* [Official documentation](https://grafana.com/docs/grafana/latest/dashboards/)
* [HTTP API](https://grafana.com/docs/grafana/latest/http_api/dashboard/)
`,

		CreateContext: CreateDashboard,
		ReadContext:   ReadDashboard,
		UpdateContext: UpdateDashboard,
		DeleteContext: DeleteDashboard,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"uid": {
				Type:     schema.TypeString,
				Computed: true,
				Description: "The unique identifier of a dashboard. This is used to construct its URL. " +
					"It’s automatically generated if not provided when creating a dashboard. " +
					"The uid allows having consistent URLs for accessing dashboards and when syncing dashboards between multiple Grafana installs. ",
			},
			"slug": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "URL friendly version of the dashboard title. This field is deprecated, please use `uid` instead.",
				Deprecated:  "Use `uid` instead.",
			},
			"dashboard_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The numeric ID of the dashboard computed by Grafana.",
			},
			"version": {
				Type:     schema.TypeInt,
				Computed: true,
				Description: "Whenever you save a version of your dashboard, a copy of that version is saved " +
					"so that previous versions of your dashboard are not lost.",
			},
			"folder": {
				Type:        schema.TypeInt,
				Optional:    true,
				ForceNew:    true,
				Description: "The id of the folder to save the dashboard in.",
			},
			"config_json": {
				Type:         schema.TypeString,
				Required:     true,
				StateFunc:    NormalizeDashboardConfigJSON,
				ValidateFunc: ValidateDashboardConfigJSON,
				Description:  "The complete dashboard model JSON.",
			},
			"overwrite": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Set to true if you want to overwrite existing dashboard with newer version, same dashboard title in folder or same dashboard uid.",
			},
		},
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{
			{
				Type:    resourceDashboardV0().CoreConfigSchema().ImpliedType(),
				Upgrade: resourceDashboardStateUpgradeV0,
				Version: 0,
			},
		},
	}
}

// resourceDashboardV0 is the original schema for this resource. For a long
// time we relied on the `slug` field as our ID - even long after it was
// deprecated in Grafana. In Grafana 8, slug endpoints were completely removed
// so we had to finally move away from it and start using UID.
func resourceDashboardV0() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"slug": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "URL friendly version of the dashboard title.",
			},
			"dashboard_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The numeric ID of the dashboard computed by Grafana.",
			},
			"folder": {
				Type:        schema.TypeInt,
				Optional:    true,
				ForceNew:    true,
				Description: "The id of the folder to save the dashboard in.",
			},
			"config_json": {
				Type:         schema.TypeString,
				Required:     true,
				StateFunc:    NormalizeDashboardConfigJSON,
				ValidateFunc: ValidateDashboardConfigJSON,
				Description:  "The complete dashboard model JSON.",
			},
			"overwrite": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Set to true if you want to overwrite existing dashboard with newer version, same dashboard title in folder or same dashboard uid.",
			},
		},
	}
}

// resourceDashboardStateUpgradeV0 migrates from version 0 of this resource's
// schema to version 1.
// * Use UID instead of slug. Slug was deprecated in Grafana 5 in favor of UID.
//   Slug API endpoints were removed in Grafana 8.
// * Version field added to schema.
func resourceDashboardStateUpgradeV0(ctx context.Context, rawState map[string]interface{}, meta interface{}) (map[string]interface{}, error) {
	client := meta.(*client).gapi
	dashboardID := int64(rawState["dashboard_id"].(float64))
	params := map[string]string{
		"type":         "dash-db",
		"dashboardIds": strconv.FormatInt(dashboardID, 10),
	}
	resp, err := client.FolderDashboardSearch(params)
	if err != nil {
		return nil, fmt.Errorf("Error attempting to migrate state. Grafana returned an error while searching for dashboard with ID %s: %s", params["dashboardIds"], err)
	}
	if len(resp) > 1 {
		return nil, fmt.Errorf("Error attempting to migrate state. Many dashboards returned by Grafana while searching for dashboard with ID, %s", params["dashboardIds"])
	}
	uid := resp[0].UID
	rawState["id"] = uid
	rawState["uid"] = uid
	dashboard, err := client.DashboardByUID(uid)
	if len(resp) > 1 {
		return nil, fmt.Errorf("Error attempting to migrate state. Grafana returned an error while searching for dashboard with UID %s: %s", uid, err)
	}
	rawState["version"] = int64(dashboard.Model["version"].(float64))
	return rawState, nil
}

func CreateDashboard(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi
	dashboard := makeDashboard(d)
	resp, err := client.NewDashboard(dashboard)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(resp.UID)
	return ReadDashboard(ctx, d, meta)
}

func ReadDashboard(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi
	uid := d.Id()
	dashboard, err := client.DashboardByUID(uid)
	var diags diag.Diagnostics
	if err != nil {
		if strings.HasPrefix(err.Error(), "status: 404") {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  fmt.Sprintf("Dashboard %q is in state, but no longer exists in grafana", uid),
				Detail:   fmt.Sprintf("%q will be recreated when you apply", uid),
			})
			d.SetId("")
			return diags
		} else {
			return diag.FromErr(err)
		}
	}

	configJSONBytes, err := json.Marshal(dashboard.Model)
	if err != nil {
		return diag.FromErr(err)
	}

	configJSON := NormalizeDashboardConfigJSON(string(configJSONBytes))

	d.SetId(dashboard.Model["uid"].(string))
	d.Set("uid", dashboard.Model["uid"].(string))
	d.Set("slug", dashboard.Meta.Slug)
	d.Set("config_json", configJSON)
	d.Set("folder", dashboard.Folder)
	d.Set("dashboard_id", int64(dashboard.Model["id"].(float64)))
	d.Set("version", int64(dashboard.Model["version"].(float64)))

	return diags
}

func UpdateDashboard(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi
	dashboard := makeDashboard(d)
	dashboard.Overwrite = true
	resp, err := client.NewDashboard(dashboard)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(resp.UID)
	return ReadDashboard(ctx, d, meta)
}

func DeleteDashboard(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi
	uid := d.Id()
	err := client.DeleteDashboardByUID(uid)
	var diags diag.Diagnostics
	if err != nil && !strings.HasPrefix(err.Error(), "status: 404") {
		return diag.FromErr(err)
	}
	return diags
}

func makeDashboard(d *schema.ResourceData) gapi.Dashboard {
	dashboard := gapi.Dashboard{
		Folder:    int64(d.Get("folder").(int)),
		Overwrite: d.Get("overwrite").(bool),
	}
	configJSON := d.Get("config_json").(string)
	dashboardJSON := unmarshallDashboardJSON(configJSON)
	delete(dashboardJSON, "id")
	delete(dashboardJSON, "version")
	dashboard.Model = dashboardJSON
	return dashboard
}

// unmarshallDashboardJSON is a convenience func for unmarshalling dashboard JSON.
func unmarshallDashboardJSON(configJSON string) map[string]interface{} {
	dashboardJSON := map[string]interface{}{}
	err := json.Unmarshal([]byte(configJSON), &dashboardJSON)
	if err != nil {
		// The validate function should've taken care of this.
		panic(fmt.Errorf("Invalid JSON got into prepare func"))
	}
	return dashboardJSON
}

func ValidateDashboardConfigJSON(configI interface{}, k string) ([]string, []error) {
	configJSON := configI.(string)
	configMap := map[string]interface{}{}
	err := json.Unmarshal([]byte(configJSON), &configMap)
	if err != nil {
		return nil, []error{err}
	}
	return nil, nil
}

func NormalizeDashboardConfigJSON(configI interface{}) string {
	dashboardJSON := unmarshallDashboardJSON(configI.(string))
	// Some properties are managed by this provider and are thus not
	// significant when included in the JSON.
	delete(dashboardJSON, "id")
	delete(dashboardJSON, "uid")
	delete(dashboardJSON, "version")
	j, _ := json.Marshal(dashboardJSON)
	return string(j)
}
