package grafana

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DatasourceDashboard() *schema.Resource {
	return &schema.Resource{
		Description: `
* [Official documentation](https://grafana.com/docs/grafana/latest/dashboards/)
* [Folder/Dashboard Search HTTP API](https://grafana.com/docs/grafana/latest/http_api/folder_dashboard_search/)
* [Dashboard HTTP API](https://grafana.com/docs/grafana/latest/http_api/dashboard/)
`,
		ReadContext: dataSourceDashboardRead,
		Schema: map[string]*schema.Schema{
			"dashboard_id": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      -1,
				ExactlyOneOf: []string{"dashboard_id", "uid"},
				Description:  "The numerical ID of the Grafana dashboard. Specify either this or `uid`.",
			},
			"uid": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "",
				ExactlyOneOf: []string{"dashboard_id", "uid"},
				Description:  "The uid of the Grafana dashboard. Specify either this or `dashboard_id`.",
			},
			"config_json": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The complete dashboard model JSON.",
			},
			"version": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The numerical version of the Grafana dashboard.",
			},
			"title": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The title of the Grafana dashboard.",
			},
			"folder": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The numerical ID of the folder where the Grafana dashboard is found.",
			},
			"is_starred": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether or not the Grafana dashboard is starred. Starred Dashboards will show up on your own Home Dashboard by default, and are a convenient way to mark Dashboards that you’re interested in.",
			},
			"slug": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "URL slug of the dashboard (deprecated).",
			},
		},
	}
}

// search dashboards by ID
func findDashboardWithID(client *gapi.Client, id int64) (*gapi.FolderDashboardSearchResponse, error) {
	params := url.Values{
		"type":         {"dash-db"},
		"dashboardIds": {strconv.FormatInt(id, 10)},
	}
	resp, err := client.FolderDashboardSearch(params)
	if err != nil {
		return nil, err
	}
	for _, d := range resp {
		if int64(d.ID) == id {
			return &d, nil
		}
	}

	return nil, fmt.Errorf("no dashboard with id %d", id)
}

func dataSourceDashboardRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var dashboard *gapi.Dashboard
	client := meta.(*client).gapi

	// get UID from ID if specified
	id := d.Get("dashboard_id").(int)
	uid := d.Get("uid").(string)
	if id > 0 {
		res, err := findDashboardWithID(client, int64(id))
		if err != nil {
			return diag.FromErr(err)
		}
		uid = res.UID
	}

	dashboard, err := client.DashboardByUID(uid)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(uid)
	d.Set("uid", dashboard.Model["uid"].(string))
	d.Set("dashboard_id", int64(dashboard.Model["id"].(float64)))
	configJSONBytes, err := json.Marshal(dashboard.Model)
	if err != nil {
		return diag.FromErr(err)
	}
	d.Set("config_json", string(configJSONBytes))
	d.Set("version", int64(dashboard.Model["version"].(float64)))
	d.Set("title", dashboard.Model["title"].(string))
	d.Set("folder", dashboard.Folder)
	d.Set("is_starred", dashboard.Meta.IsStarred)
	d.Set("slug", dashboard.Meta.Slug)

	return nil
}
