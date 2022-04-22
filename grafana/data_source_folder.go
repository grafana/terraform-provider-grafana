package grafana

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DatasourceFolder() *schema.Resource {
	return &schema.Resource{
		Description: `
* [Official documentation](https://grafana.com/docs/grafana/latest/dashboards/dashboard_folders/)
* [HTTP API](https://grafana.com/docs/grafana/latest/http_api/folder/)
`,
		ReadContext: dataSourceFolderRead,
		Schema: map[string]*schema.Schema{
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

func findFolderWithTitle(client *gapi.Client, title string) (*gapi.Folder, error) {
	folders, err := client.Folders()
	if err != nil {
		return nil, err
	}

	for _, f := range folders {
		if f.Title == title {
			// Query the folder by UID, that API has additional information
			return client.FolderByUID(f.UID)
		}
	}

	return nil, fmt.Errorf("no folder with title %q", title)
}

func dataSourceFolderRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	gapiURL := meta.(*client).gapiURL
	client := meta.(*client).gapi
	title := d.Get("title").(string)
	folder, err := findFolderWithTitle(client, title)

	if err != nil {
		return diag.FromErr(err)
	}

	id := strconv.FormatInt(folder.ID, 10)
	d.SetId(id)
	d.Set("uid", folder.UID)
	d.Set("title", folder.Title)
	d.Set("url", strings.TrimRight(gapiURL, "/")+folder.URL)

	return nil
}
