package grafana

import (
	"context"
	"fmt"

	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/client/search"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func datasourceFolder() *common.DataSource {
	schema := &schema.Resource{
		Description: `
* [Official documentation](https://grafana.com/docs/grafana/latest/dashboards/manage-dashboards/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/folder/)
`,
		ReadContext: dataSourceFolderRead,
		Schema: common.CloneResourceSchemaForDatasource(resourceFolder().Schema, map[string]*schema.Schema{
			"org_id": orgIDAttribute(),
			"title": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The title of the folder.",
			},
			"prevent_destroy_if_not_empty": nil,
		}),
	}
	return common.NewLegacySDKDataSource(common.CategoryGrafanaOSS, "grafana_folder", schema)
}

func findFolderWithTitle(client *goapi.GrafanaHTTPAPI, title string) (string, error) {
	var page int64 = 1

	for {
		params := search.NewSearchParams().WithType(common.Ref("dash-folder")).WithPage(&page)
		resp, err := client.Search.Search(params)
		if err != nil {
			return "", err
		}

		if len(resp.Payload) == 0 {
			return "", fmt.Errorf("folder with title %s not found", title)
		}

		for _, folder := range resp.Payload {
			if folder.Title == title {
				return folder.UID, nil
			}
		}

		page++
	}
}

func dataSourceFolderRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID := OAPIClientFromNewOrgResource(meta, d)
	uid, err := findFolderWithTitle(client, d.Get("title").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(MakeOrgResourceID(orgID, uid))
	return ReadFolder(ctx, d, meta)
}
