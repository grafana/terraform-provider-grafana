package grafana

import (
	"context"
	"errors"
	"fmt"

	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/client/search"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
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
				Optional:    true,
				Description: "The title of the folder. If not set, only the uid is used to find the folder.",
			},
			"uid": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true, // If not set by user, this will be populated by reading the folder.
				Description: "The uid of the folder. If not set, only the title of the folder is used to find the folder.",
			},
			"prevent_destroy_if_not_empty": nil,
		}),
	}
	return common.NewLegacySDKDataSource(common.CategoryGrafanaOSS, "grafana_folder", schema)
}

// The following consts are only exported for usage in tests
const (
	FolderTitleOrUIDMissing       = "either title or uid must be set"
	FolderWithTitleNotFound       = "folder with title %s not found"
	FolderWithUIDNotFound         = "folder with uid %s not found"
	FolderWithTitleAndUIDNotFound = "folder with title %s and uid %s not found"
)

func findFolderWithTitleAndUID(client *goapi.GrafanaHTTPAPI, title string, uid string) (string, error) {
	if title == "" && uid == "" {
		return "", errors.New(FolderTitleOrUIDMissing)
	}

	var page int64 = 1

	for {
		params := search.NewSearchParams().WithType(common.Ref("dash-folder")).WithPage(&page)
		resp, err := client.Search.Search(params)
		if err != nil {
			return "", err
		}

		if len(resp.Payload) == 0 {
			switch {
			case title != "" && uid == "":
				err = fmt.Errorf(FolderWithTitleNotFound, title)
			case title == "" && uid != "":
				err = fmt.Errorf(FolderWithUIDNotFound, uid)
			case title != "" && uid != "":
				err = fmt.Errorf(FolderWithTitleAndUIDNotFound, title, uid)
			}
			return "", err
		}

		for _, folder := range resp.Payload {
			if (title == "" || folder.Title == title) && (uid == "" || folder.UID == uid) {
				return folder.UID, nil
			}
		}

		page++
	}
}

func dataSourceFolderRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client, orgID := OAPIClientFromNewOrgResource(meta, d)
	uid, err := findFolderWithTitleAndUID(client, d.Get("title").(string), d.Get("uid").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(MakeOrgResourceID(orgID, uid))
	return ReadFolder(ctx, d, meta)
}
