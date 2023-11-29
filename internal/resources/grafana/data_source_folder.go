package grafana

import (
	"context"
	"fmt"
	"strconv"

	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/client/folders"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DatasourceFolder() *schema.Resource {
	return &schema.Resource{
		Description: `
* [Official documentation](https://grafana.com/docs/grafana/latest/dashboards/manage-dashboards/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/folder/)
`,
		ReadContext: dataSourceFolderRead,
		Schema: map[string]*schema.Schema{
			"org_id": orgIDAttribute(),
			"title": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the Grafana folder.",
			},
			"id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The numerical ID of the Grafana folder.",
			},
			"uid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The uid of the Grafana folder.",
			},
			"url": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The full URL of the folder.",
			},
		},
	}
}

func findFolderWithTitle(client *goapi.GrafanaHTTPAPI, title string) (*models.Folder, error) {
	var page int64 = 1

	for {
		params := folders.NewGetFoldersParams().WithPage(&page)
		resp, err := client.Folders.GetFolders(params)
		if err != nil {
			return nil, err
		}

		if len(resp.Payload) == 0 {
			return nil, fmt.Errorf("folder with title %s not found", title)
		}

		for _, folder := range resp.Payload {
			if folder.Title == title {
				resp, err := client.Folders.GetFolderByUID(folder.UID)
				if err != nil {
					return nil, err
				}
				return resp.Payload, nil
			}
		}

		page++
	}
}

func dataSourceFolderRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	metaClient := meta.(*common.Client)
	client, orgID := OAPIClientFromNewOrgResource(meta, d)

	folder, err := findFolderWithTitle(client, d.Get("title").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.FormatInt(folder.ID, 10))
	d.Set("org_id", strconv.FormatInt(orgID, 10))
	d.Set("uid", folder.UID)
	d.Set("title", folder.Title)
	d.Set("url", metaClient.GrafanaSubpath(folder.URL))

	return nil
}
