package grafana

import (
	"context"
	"encoding/json"
	"fmt"

	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/client/search"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func datasourceDashboard() *common.DataSource {
	schema := &schema.Resource{
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
			"folder_uid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The UID of the folder where the Grafana dashboard is found.",
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
	return common.NewLegacySDKDataSource(common.CategoryGrafanaOSS, "grafana_dashboard", schema)
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
		dashboard, err := getDashboardByID(client, int64(id))
		if err != nil {
			return diag.FromErr(err)
		}
		uid = dashboard.UID
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
	d.Set("folder_uid", dashboard.Meta.FolderUID)
	d.Set("is_starred", dashboard.Meta.IsStarred)
	d.Set("slug", dashboard.Meta.Slug)
	d.Set("url", metaClient.GrafanaSubpath(dashboard.Meta.URL))

	return nil
}

func getDashboardByID(client *goapi.GrafanaHTTPAPI, id int64) (*models.Hit, error) {
	searchType := "dash-db"
	params := search.NewSearchParams().WithType(&searchType).WithDashboardIds([]int64{id})
	resp, err := client.Search.Search(params)
	if err != nil {
		return nil, err
	}
	for _, d := range resp.GetPayload() {
		if d.ID == id {
			return d, nil
		}
	}
	return nil, fmt.Errorf("no dashboard with id %d", id)
}
