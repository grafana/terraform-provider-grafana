package grafana

import (
	"context"
	"fmt"
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
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     -1,
				Description: "The numerical ID of the Grafana dashboard.",
			},
			"uid": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "The uid of the Grafana dashboard.",
			},
			"version": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     0,
				Description: "The numerical version of the Grafana dashboard. Set to 0 or omit to get the latest version",
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
			"config_json": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The complete dashboard model JSON.",
			},
			"slug": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The complete dashboard model JSON.",
			},
		},
	}
}

// search dashboards by ID
func findDashboardWithID(client *gapi.Client, id int64) (*gapi.FolderDashboardSearchResponse, error) {
	params := map[string]string{
		"type":         "dash-db",
		"dashboardIds": strconv.FormatInt(id, 10),
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
	var err error
	var dashboard *gapi.Dashboard
	client := meta.(*client).gapi

	// get UID from ID if specified
	id := d.Get("dashboard_id").(int)
	uid := d.Get("uid").(string)
	switch {
	case (id < 1 && uid == ""):
		return diag.FromErr(fmt.Errorf("must specify either dashboard id or uid"))
	case (id > 0 && uid != ""):
		return diag.FromErr(fmt.Errorf("must specify either dashboard id or uid, but not both"))
	case (id > 0):
		res, err := findDashboardWithID(client, int64(id))
		if err != nil {
			return diag.FromErr(err)
		}
		uid = res.UID
	default:
		break
	}

	// TODO implement dashboard versions
	if version := d.Get("version").(int); version > 0 {
		panic("dashboard version not implemented")
		// dashboard, err = client.DashboardGetByVersion(uid, version)
		// if err != nil {
		// 	return diag.FromErr(err)
		// }
	} else {
		dashboard, err = client.DashboardByUID(uid)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId(uid)
	ReadDashboard(ctx, d, meta)
	d.Set("title", dashboard.Model["title"].(string))
	d.Set("is_starred", dashboard.Meta.IsStarred)

	return nil
}
