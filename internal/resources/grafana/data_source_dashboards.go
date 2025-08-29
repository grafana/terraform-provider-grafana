package grafana

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/grafana/grafana-openapi-client-go/client/search"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func datasourceDashboards() *common.DataSource {
	schema := &schema.Resource{
		Description: `
Datasource for retrieving all dashboards. Specify list of folder IDs to search in for dashboards.

* [Official documentation](https://grafana.com/docs/grafana/latest/dashboards/)
* [Folder/Dashboard Search HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/folder_dashboard_search/)
* [Dashboard HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/dashboard/)
`,
		ReadContext: dataSourceReadDashboards,
		Schema: map[string]*schema.Schema{
			"org_id": orgIDAttribute(),
			"folder_uids": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "UIDs of Grafana folders containing dashboards. Specify to filter for dashboards by folder (eg. `[\"General\"]` for General folder), or leave blank to get all dashboards in all folders.",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"limit": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     5000,
				Description: "Maximum number of dashboard search results to return.",
			},
			"tags": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "List of string Grafana dashboard tags to search for, eg. `[\"prod\"]`. Used only as search input, i.e., attribute value will remain unchanged.",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"dashboards": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"title": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"uid": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"folder_title": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
	return common.NewLegacySDKDataSource(common.CategoryGrafanaOSS, "grafana_dashboards", schema)
}

func dataSourceReadDashboards(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client, orgID := OAPIClientFromNewOrgResource(meta, d)

	limit := int64(d.Get("limit").(int))
	searchType := "dash-db"
	params := search.NewSearchParams().WithLimit(&limit).WithType(&searchType)

	id := sha256.New()
	id.Write(fmt.Appendf(nil, "%d", limit))

	// add tags and folder UIDs from attributes to dashboard search parameters
	if list, ok := d.GetOk("folder_uids"); ok {
		params.FolderUIDs = common.ListToStringSlice(list.([]any))
		id.Write(fmt.Appendf(nil, "%v", params.FolderUIDs))
	}

	if list, ok := d.GetOk("tags"); ok {
		params.Tag = common.ListToStringSlice(list.([]any))
		id.Write(fmt.Appendf(nil, "%v", params.Tag))
	}

	d.SetId(MakeOrgResourceID(orgID, id))

	resp, err := client.Search.Search(params)
	if err != nil {
		return diag.FromErr(err)
	}

	dashboards := make([]map[string]any, len(resp.GetPayload()))
	for i, result := range resp.GetPayload() {
		dashboards[i] = map[string]any{
			"title":        result.Title,
			"uid":          result.UID,
			"folder_title": result.FolderTitle,
		}
	}

	if err := d.Set("dashboards", dashboards); err != nil {
		return diag.Errorf("error setting dashboards attribute: %s", err)
	}

	return nil
}
