package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/provider/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceFolder() *schema.Resource {
	return &schema.Resource{

		Description: `
* [Official documentation](https://grafana.com/docs/grafana/latest/dashboards/dashboard-folders/)
* [HTTP API](https://grafana.com/docs/grafana/latest/http_api/folder/)
`,

		CreateContext: CreateFolder,
		DeleteContext: DeleteFolder,
		ReadContext:   ReadFolder,
		UpdateContext: UpdateFolder,

		// Import either by ID or UID
		Importer: &schema.ResourceImporter{
			StateContext: func(c context.Context, rd *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				_, err := strconv.ParseInt(rd.Id(), 10, 64)
				if err != nil {
					// If the ID is not a number, then it may be a UID
					client := meta.(*common.Client).GrafanaAPI
					folder, err := client.FolderByUID(rd.Id())
					if err != nil {
						return nil, fmt.Errorf("failed to find folder by ID or UID '%s': %w", rd.Id(), err)
					}
					rd.SetId(strconv.FormatInt(folder.ID, 10))
				}
				return []*schema.ResourceData{rd}, nil
			},
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

	id, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	folder, err := getFolderByID(client, id)
	if err != nil {
		if strings.HasPrefix(err.Error(), "status: 404") {
			log.Printf("[WARN] removing folder %d from state because it no longer exists in grafana", id)
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

	if err := client.DeleteFolder(d.Get("uid").(string)); err != nil {
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

// Hackish way to get the folder by ID.
// TODO: Revert to using the specific folder ID GET endpoint once it's fixed
// Broken in 8.5.0
func getFolderByID(client *gapi.Client, id int64) (*gapi.Folder, error) {
	folders, err := client.Folders()
	if err != nil {
		return nil, err
	}

	for _, folder := range folders {
		if folder.ID == id {
			// Need to use another API call, because the "list" call doesn't have all the info
			return client.FolderByUID(folder.UID)
		}
	}

	// Replicating the error that would usually be returned by the API call on a missing folder
	return nil, errors.New(`status: 404, body: {"message":"folder not found","status":"not-found"}`)
}
