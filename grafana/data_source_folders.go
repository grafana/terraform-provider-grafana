package grafana

import (
	"context"
	"crypto/md5"
	"fmt"
	"strings"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DatasourceFolders() *schema.Resource {
	return &schema.Resource{
		ReadContext: readFolders,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Description: `
* [Official documentation](https://grafana.com/docs/grafana/latest/dashboards/dashboard-folders/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/folder/)
`,

		Schema: map[string]*schema.Schema{
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
}

func readFolders(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi
	folders, err := client.Folders()
	if err != nil {
		return diag.FromErr(err)
	}

	uids := []string{}
	for _, folder := range folders {
		uids = append(uids, folder.UID)
	}
	md5 := md5.Sum([]byte(strings.Join(uids, "-")))

	d.SetId(fmt.Sprintf("%x", md5))

	if err := d.Set("folders", flattenFolders(folders)); err != nil {
		return diag.FromErr(fmt.Errorf("error setting item: %v", err))
	}

	return nil
}

func flattenFolders(items []gapi.Folder) []interface{} {
	folderItems := make([]interface{}, 0)
	for _, folder := range items {
		f := map[string]interface{}{
			"title": folder.Title,
			"id":    folder.ID,
			"uid":   folder.UID,
			"url":   folder.URL,
		}
		folderItems = append(folderItems, f)
	}

	return folderItems
}
