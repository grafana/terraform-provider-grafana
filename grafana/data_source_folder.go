package grafana

import (
	"context"
	"fmt"
	"strconv"

	gapi "github.com/albeego/grafana-api-golang-client"
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
			return &f, nil
		}
	}

	return nil, fmt.Errorf("no folder with title %q", title)
}

func dataSourceFolderRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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

	return nil
}
