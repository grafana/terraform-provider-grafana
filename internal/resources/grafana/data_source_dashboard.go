package grafana

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/grafana/grafana-openapi-client-go/client/search"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func datasourceDashboard() *schema.Resource {
	return &schema.Resource{
		Description: `
* [Official documentation](https://grafana.com/docs/grafana/latest/dashboards/)
* [Folder/Dashboard Search HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/folder_dashboard_search/)
* [Dashboard HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/dashboard/)
`,
		ReadContext: dataSourceDashboardRead,
		Schema: map[string]*schema.Schema{
			"org_id": orgIDAttribute(),
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
			"url": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The full URL of the dashboard.",
			},
		},
	}
}

func dataSourceDashboardRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	metaClient := meta.(*common.Client)
	client, orgID := OAPIClientFromNewOrgResource(meta, d)

	// get UID from ID if specified
	id := d.Get("dashboard_id").(int)
	uid := d.Get("uid").(string)
	if uid == "" {
		if id < 1 {
			return diag.FromErr(fmt.Errorf("must specify either `dashboard_id` or `uid`"))
		}

		searchType := "dash-db"
		params := search.NewSearchParams().WithType(&searchType).WithDashboardIds([]int64{int64(id)})
		resp, err := client.Search.Search(params)
		if err != nil {
			return diag.FromErr(err)
		}
		for _, d := range resp.GetPayload() {
			if d.ID == int64(id) {
				uid = d.UID
				break
			}
		}
		if uid == "" {
			return diag.FromErr(fmt.Errorf("no dashboard with id %d", id))
		}
	}

	resp, err := client.Dashboards.GetDashboardByUID(uid)
	if err != nil {
		return diag.FromErr(err)
	}
	dashboard := resp.GetPayload()
	model := dashboard.Dashboard.(map[string]interface{})

	d.SetId(MakeOrgResourceID(orgID, uid))
	d.Set("uid", model["uid"].(string))
	d.Set("dashboard_id", int64(model["id"].(float64)))
	configJSONBytes, err := json.Marshal(model)
	if err != nil {
		return diag.FromErr(err)
	}
	d.Set("config_json", string(configJSONBytes))
	d.Set("version", int64(model["version"].(float64)))
	d.Set("title", model["title"].(string))
	d.Set("folder", dashboard.Meta.FolderID)
	d.Set("is_starred", dashboard.Meta.IsStarred)
	d.Set("slug", dashboard.Meta.Slug)
	d.Set("url", metaClient.GrafanaSubpath(dashboard.Meta.URL))

	return nil
}
