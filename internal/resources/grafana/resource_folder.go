package grafana

import (
	"context"
	"encoding/json"
	"log"
	"net/url"
	"strconv"
	"strings"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceFolder() *schema.Resource {
	return &schema.Resource{

		Description: `
* [Official documentation](https://grafana.com/docs/grafana/latest/dashboards/manage-dashboards/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/folder/)
`,

		CreateContext: CreateFolder,
		DeleteContext: DeleteFolder,
		ReadContext:   ReadFolder,
		UpdateContext: UpdateFolder,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Unique internal identifier.",
			},
			"uid": {
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
				Description: "Unique identifier.",
			},
			"title": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The title of the folder.",
			},
			"url": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The full URL of the folder.",
			},
			"prevent_destroy_if_not_empty": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Prevent deletion of the folder if it is not empty (contains dashboards or alert rules).",
			},
		},
	}
}

func CreateFolder(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).GrafanaAPI

	var resp gapi.Folder
	var err error
	title := d.Get("title").(string)
	if uid, ok := d.GetOk("uid"); ok {
		resp, err = client.NewFolder(title, uid.(string))
	} else {
		resp, err = client.NewFolder(title)
	}
	if err != nil {
		return diag.FromErr(err)
	}

	id := strconv.FormatInt(resp.ID, 10)
	d.SetId(id)
	d.Set("id", id)
	d.Set("uid", resp.UID)
	d.Set("title", resp.Title)

	return ReadFolder(ctx, d, meta)
}

func UpdateFolder(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).GrafanaAPI

	oldUID, newUID := d.GetChange("uid")

	if err := client.UpdateFolder(oldUID.(string), d.Get("title").(string), newUID.(string)); err != nil {
		return diag.FromErr(err)
	}

	return ReadFolder(ctx, d, meta)
}

func ReadFolder(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	gapiURL := meta.(*common.Client).GrafanaAPIURL
	client := meta.(*common.Client).GrafanaAPI

	folder, err := GetFolderByIDorUID(client, d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "status: 404") {
			log.Printf("[WARN] removing folder %s from state because it no longer exists in grafana", d.Id())
			d.SetId("")
			return nil
		}

		return diag.FromErr(err)
	}

	d.SetId(strconv.FormatInt(folder.ID, 10))
	d.Set("title", folder.Title)
	d.Set("uid", folder.UID)
	d.Set("url", strings.TrimRight(gapiURL, "/")+folder.URL)

	return nil
}

func DeleteFolder(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).GrafanaAPI

	deleteParams := []url.Values{}
	if d.Get("prevent_destroy_if_not_empty").(bool) {
		// Search for dashboards and fail if any are found
		dashboards, err := client.FolderDashboardSearch(url.Values{
			"type":      []string{"dash-db"},
			"folderIds": []string{d.Id()},
		})
		if err != nil {
			return diag.FromErr(err)
		}
		if len(dashboards) > 0 {
			var dashboardNames []string
			for _, dashboard := range dashboards {
				dashboardNames = append(dashboardNames, dashboard.Title)
			}
			return diag.Errorf("folder %s is not empty and prevent_destroy_if_not_empty is set. It contains the following dashboards: %v", d.Get("uid").(string), dashboardNames)
		}
	} else {
		// If we're not preventing destroys, then we can force delete folders that have alert rules
		deleteParams = append(deleteParams, gapi.ForceDeleteFolderRules())
	}

	if err := client.DeleteFolder(d.Get("uid").(string), deleteParams...); err != nil {
		return diag.FromErr(err)
	}

	return diag.Diagnostics{}
}

func ValidateFolderConfigJSON(configI interface{}, k string) ([]string, []error) {
	configJSON := configI.(string)
	configMap := map[string]interface{}{}
	err := json.Unmarshal([]byte(configJSON), &configMap)
	if err != nil {
		return nil, []error{err}
	}
	return nil, nil
}

func NormalizeFolderConfigJSON(configI interface{}) string {
	configJSON := configI.(string)

	configMap := map[string]interface{}{}
	err := json.Unmarshal([]byte(configJSON), &configMap)
	if err != nil {
		// The validate function should've taken care of this.
		return ""
	}

	// Some properties are managed by this provider and are thus not
	// significant when included in the JSON.
	delete(configMap, "id")
	delete(configMap, "version")

	ret, err := json.Marshal(configMap)
	if err != nil {
		// Should never happen.
		return configJSON
	}

	return string(ret)
}

func GetFolderByIDorUID(client *gapi.Client, id string) (*gapi.Folder, error) {
	// If the ID is a number, find the folder UID
	// Getting the folder by ID is broken in some versions, but getting by UID works in all versions
	// We need to use two API calls in the numerical ID case, because the "list" call doesn't have all the info
	uid := id
	if numericalID, err := strconv.ParseInt(id, 10, 64); err == nil {
		folders, err := client.Folders()
		if err != nil {
			return nil, err
		}
		for _, folder := range folders {
			if folder.ID == numericalID {
				uid = folder.UID
				break
			}
		}
	}

	return client.FolderByUID(uid)
}
