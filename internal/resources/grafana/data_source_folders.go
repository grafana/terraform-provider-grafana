package grafana

import (
	"context"

	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/client/search"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func datasourceFolders() *common.DataSource {
	schema := &schema.Resource{
		ReadContext: readFolders,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Description: `
* [Official documentation](https://grafana.com/docs/grafana/latest/dashboards/manage-dashboards/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developer-resources/api-reference/http-api/folder/)
`,

		Schema: map[string]*schema.Schema{
			"org_id": orgIDAttribute(),
			"folders": {
				Type:        schema.TypeSet,
				Computed:    true,
				Description: "The Grafana instance's folders.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"title": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The folder title.",
						},
						"id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The folder ID.",
						},
						"uid": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The folder's unique identifier.",
						},
						"url": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The folder's URL",
						},
					},
				},
			},
		},
	}
	return common.NewLegacySDKDataSource(common.CategoryGrafanaOSS, "grafana_folders", schema)
}

// listAllFolders pages through /api/search and returns every folder. Results
// are sorted by title so the paginated listing is ordered consistently across
// requests; without a sort the order is not stable and paging can skip or
// duplicate folders once there is more than one page of them.
func listAllFolders(client *goapi.GrafanaHTTPAPI) ([]*models.Hit, error) {
	var folders []*models.Hit
	var page int64 = 1
	searchType := "dash-folder"
	for {
		params := search.NewSearchParams().WithType(&searchType).WithSort(common.Ref("alpha-asc")).WithPage(&page)
		resp, err := client.Search.Search(params)
		if err != nil {
			return nil, err
		}
		if len(resp.Payload) == 0 {
			break
		}

		folders = append(folders, resp.Payload...)
		page++
	}
	return folders, nil
}

func readFolders(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	metaClient := meta.(*common.Client)
	client, orgID := OAPIClientFromNewOrgResource(meta, d)

	folders, err := listAllFolders(client)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(MakeOrgResourceID(orgID, "folders"))

	folderItems := make([]any, 0)
	for _, folder := range folders {
		f := map[string]any{
			"title": folder.Title,
			"id":    folder.ID,
			"uid":   folder.UID,
			"url":   metaClient.GrafanaSubpath(folder.URL),
		}
		folderItems = append(folderItems, f)
	}

	return diag.FromErr(d.Set("folders", folderItems))
}
