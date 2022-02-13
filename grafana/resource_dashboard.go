package grafana

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
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
				StateFunc:    normalizeDashboardConfigJSON,
				ValidateFunc: validateDashboardConfigJSON,
				Description:  "The complete dashboard model JSON.",
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
				Type:     schema.TypeString,
				Computed: true,
			},
			"dashboard_id": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"folder": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},
			"config_json": {
				Type:         schema.TypeString,
				Required:     true,
				StateFunc:    normalizeDashboardConfigJSON,
				ValidateFunc: validateDashboardConfigJSON,
			},
			"overwrite": {
				Type:     schema.TypeBool,
				Optional: true,
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
	query := url.Values{
		"type":         {"dash-db"},
		"dashboardIds": {strconv.FormatInt(dashboardID, 10)},
	}
	resp, err := client.FolderDashboardSearch(query)
	if err != nil {
		return nil, fmt.Errorf("error attempting to migrate state. Grafana returned an error while searching for dashboard with ID %s: %s", query.Get("dashboardIds"), err)
	}
	switch {
	case len(resp) > 1:
		// Search endpoint returned multiple dashboards. This is not likely.
		return nil, fmt.Errorf("error attempting to migrate state. Many dashboards returned by Grafana while searching for dashboard with ID, %s", query.Get("dashboardIds"))
	case len(resp) == 0:
		// Dashboard does not exist. Let Terraform recreate it.
		return rawState, nil
	}
	uid := resp[0].UID
	rawState["id"] = uid
	rawState["uid"] = uid
	dashboard, err := client.DashboardByUID(uid)
	// Set version if we can.
	// In the unlikely event that we don't get a dashboard back, we don't return
	// an error because Terraform will be able to reconcile this field without
	// much trouble.
	if err == nil && dashboard != nil {
		rawState["version"] = int64(dashboard.Model["version"].(float64))
	}
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
	d.Set("uid", resp.UID)
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

	d.SetId(dashboard.Model["uid"].(string))
	d.Set("uid", dashboard.Model["uid"].(string))
	d.Set("slug", dashboard.Meta.Slug)
	d.Set("folder", dashboard.Folder)
	d.Set("dashboard_id", int64(dashboard.Model["id"].(float64)))
	d.Set("version", int64(dashboard.Model["version"].(float64)))

	configJSONBytes, err := json.Marshal(dashboard.Model)
	if err != nil {
		return diag.FromErr(err)
	}
	remoteDashJSON, err := unmarshalDashboardConfigJSON(string(configJSONBytes))
	if err != nil {
		return diag.FromErr(err)
	}

	// If `uid` is not set in configuration, we need to delete it from the
	// dashboard JSON we just read from the Grafana API. This is so it does not
	// create a diff. We can assume the uid was randomly generated by Grafana or
	// it was removed after dashboard creation. In any case, the user doesn't
	// care to manage it.
	if configJSON := d.Get("config_json").(string); configJSON != "" {
		configuredDashJSON, err := unmarshalDashboardConfigJSON(configJSON)
		if err != nil {
			return diag.FromErr(err)
		}
		if _, ok := configuredDashJSON["uid"].(string); !ok {
			delete(remoteDashJSON, "uid")
		}
	}

	configJSON := normalizeDashboardConfigJSON(remoteDashJSON)
	d.Set("config_json", configJSON)

	return diags
}

func UpdateDashboard(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi
	dashboard := makeDashboard(d)
	dashboard.Model["id"] = d.Get("dashboard_id").(int)
	dashboard.Overwrite = true
	resp, err := client.NewDashboard(dashboard)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(resp.UID)
	d.Set("uid", resp.UID)
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
		Message:   d.Get("message").(string),
	}
	configJSON := d.Get("config_json").(string)
	dashboardJSON, err := unmarshalDashboardConfigJSON(configJSON)
	if err != nil {
		return dashboard
	}
	delete(dashboardJSON, "id")
	delete(dashboardJSON, "version")
	dashboard.Model = dashboardJSON
	return dashboard
}

// unmarshalDashboardConfigJSON is a convenience func for unmarshalling
// `config_json` field.
func unmarshalDashboardConfigJSON(configJSON string) (map[string]interface{}, error) {
	dashboardJSON := map[string]interface{}{}
	err := json.Unmarshal([]byte(configJSON), &dashboardJSON)
	if err != nil {
		return nil, err
	}
	return dashboardJSON, nil
}

// validateDashboardConfigJSON is the ValidateFunc for `config_json`. It
// ensures its value is valid JSON.
func validateDashboardConfigJSON(config interface{}, k string) ([]string, []error) {
	configJSON := config.(string)
	configMap := map[string]interface{}{}
	err := json.Unmarshal([]byte(configJSON), &configMap)
	if err != nil {
		return nil, []error{err}
	}
	return nil, nil
}

// normalizeDashboardConfigJSON is the StateFunc for the `config_json` field.
//
// It removes the following fields:
//
// * `id`:      an auto-incrementing ID Grafana assigns to dashboards upon
//              creation. We cannot know this before creation and therefore it cannot
//              be managed in code.
// * `version`: is incremented by Grafana each time a dashboard changes.
func normalizeDashboardConfigJSON(config interface{}) string {
	var dashboardJSON map[string]interface{}
	switch c := config.(type) {
	case map[string]interface{}:
		dashboardJSON = c
	case string:
		var err error
		dashboardJSON, err = unmarshalDashboardConfigJSON(c)
		if err != nil {
			return c
		}
	}

	delete(dashboardJSON, "id")
	delete(dashboardJSON, "version")

	// similarly to uid removal above, remove any attributes panels[].libraryPanel.*
	// from the dashboard JSON other than "name" or "uid".
	// Grafana will populate all other libraryPanel attributes, so delete them to avoid diff.
	panels, hasPanels := dashboardJSON["panels"]
	if hasPanels {
		for _, panel := range panels.([]interface{}) {
			panelMap := panel.(map[string]interface{})
			delete(panelMap, "id")
			if libraryPanel, ok := panelMap["libraryPanel"].(map[string]interface{}); ok {
				for k := range libraryPanel {
					if k != "name" && k != "uid" {
						delete(libraryPanel, k)
					}
				}
			}
		}
	}

	j, _ := json.Marshal(dashboardJSON)
	return string(j)
}
