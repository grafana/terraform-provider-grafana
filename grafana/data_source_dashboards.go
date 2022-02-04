package grafana

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DatasourceDashboards() *schema.Resource {
	return &schema.Resource{
		Description: `
Datasource for retrieving all dashboards. Specify list of folder IDs to search in for dashboards.

* [Official documentation](https://grafana.com/docs/grafana/latest/dashboards/)
* [Folder/Dashboard Search HTTP API](https://grafana.com/docs/grafana/latest/http_api/folder_dashboard_search/)
* [Dashboard HTTP API](https://grafana.com/docs/grafana/latest/http_api/dashboard/)
`,
		ReadContext: dataSourceReadDashboards,
		Schema: map[string]*schema.Schema{
			"folder_ids": {
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				Description: "Numerical IDs of Grafana folders to search in for dashboards.",
				Elem:        &schema.Schema{Type: schema.TypeInt},
			},
			"dashboards": {
				Description: "Map of dashboards with UIDs as keys and IDs as values.",
				Type:        schema.TypeMap,
				Computed:    true,
				Elem: &schema.Schema{
					Type: schema.TypeInt,
				},
			},
		},
	}
}

func dataSourceReadDashboards(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	dashboards := make(map[string]int64)
	client := meta.(*client).gapi
	params := map[string]string{
		"limit": "5000",
		"type":  "dash-db",
	}

	// search for dashboards in specified folders
	folderIds := d.Get("folder_ids").([]int)
	if len(folderIds) > 0 {
		folderIdsJSON, err := json.Marshal(folderIds)
		if err != nil {
			return diag.FromErr(err)
		}
		params["folderIds"] = string(folderIdsJSON)
	}

	results, err := client.FolderDashboardSearch(params)
	if err != nil {
		return diag.FromErr(err)
	}

	// map int dashboard IDs to string UIDs
	for _, thisResult := range results {
		thisDashboard, err := client.DashboardByUID(thisResult.UID)
		if err != nil {
			return diag.FromErr(err)
		}
		thisID := int64(thisDashboard.Model["id"].(float64))
		thisUID := thisDashboard.Model["uid"].(string)
		dashboards[thisUID] = thisID
	}

	d.Set("dashboards", dashboards)

	return nil
}
