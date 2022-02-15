package grafana

import (
	"context"
	"fmt"
	"strconv"

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
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ExactlyOneOf: []string{"title", "id", "uid"},
				Description:  "The name of the Grafana folder.",
			},
			"id": {
				Type:         schema.TypeInt,
				Optional:     true,
				ExactlyOneOf: []string{"title", "id", "uid"},
				Default:      -1,
				Description:  "The numerical ID of the Grafana folder.",
			},
			"uid": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ExactlyOneOf: []string{"title", "id", "uid"},
				Description:  "The unique identifier (uid) of the Grafana folder.",
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
	var folder *gapi.Folder
	var err error

	if title := d.Get("title").(string); title != "" {
		folder, err = findFolderWithTitle(client, title)
	}

	if uid := d.Get("uid").(string); uid != "" {
		folder, err = client.FolderByUID(uid)
	}

	if id := int64(d.Get("id").(int)); id != -1 {
		folder, err = client.Folder(id)
	}

	if err != nil {
		return diag.FromErr(err)
	}

	id := strconv.FormatInt(folder.ID, 10)
	d.SetId(id)
	d.Set("uid", folder.UID)
	d.Set("title", folder.Title)

	return nil
}
