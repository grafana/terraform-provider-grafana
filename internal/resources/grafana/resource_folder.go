package grafana

import (
	"context"
	"encoding/json"
	"strconv"

	goapi "github.com/grafana/grafana-openapi-client-go/client/folders"
	"github.com/grafana/grafana-openapi-client-go/client/search"
	"github.com/grafana/grafana-openapi-client-go/models"

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
			"org_id": orgIDAttribute(),
			"uid": {
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
				ForceNew:    true,
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
			"parent_folder_uid": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "The uid of the parent folder. If set, the folder will be nested. If not set, the folder will be created in the root folder.",
			},
		},
	}
}

func CreateFolder(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID := OAPIClientFromNewOrgResource(meta, d)

	var body models.CreateFolderCommand
	if title := d.Get("title").(string); title != "" {
		body.Title = title
	}

	if uid, ok := d.GetOk("uid"); ok {
		body.UID = uid.(string)
	}

	if parentUID, ok := d.GetOk("parent_folder_uid"); ok {
		body.ParentUID = parentUID.(string)
	}

	params := goapi.NewCreateFolderParams().WithBody(&body)
	resp, err := client.Folders.CreateFolder(params, nil)
	if err != nil {
		return diag.Errorf("failed to create folder: %s", err)
	}

	folder := resp.GetPayload()
	d.SetId(MakeOrgResourceID(orgID, folder.ID))

	return ReadFolder(ctx, d, meta)
}

func UpdateFolder(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, idStr := OAPIClientFromExistingOrgResource(meta, d.Id())

	folder, err := GetFolderByIDorUID(client.Folders, idStr)
	if err != nil {
		return diag.Errorf("failed to get folder %s: %s", idStr, err)
	}

	params := goapi.NewUpdateFolderParams().
		WithBody(&models.UpdateFolderCommand{
			Overwrite: true,
			Title:     d.Get("title").(string),
		}).
		WithFolderUID(folder.UID)

	if _, err := client.Folders.UpdateFolder(params, nil); err != nil {
		return diag.FromErr(err)
	}

	return ReadFolder(ctx, d, meta)
}

func ReadFolder(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	metaClient := meta.(*common.Client)
	client, orgID, idStr := OAPIClientFromExistingOrgResource(meta, d.Id())

	folder, err := GetFolderByIDorUID(client.Folders, idStr)
	if err, shouldReturn := common.CheckReadError("folder", d, err); shouldReturn {
		return err
	}

	d.SetId(MakeOrgResourceID(orgID, folder.ID))
	d.Set("org_id", strconv.FormatInt(orgID, 10))
	d.Set("title", folder.Title)
	d.Set("uid", folder.UID)
	d.Set("url", metaClient.GrafanaSubpath(folder.URL))
	d.Set("parent_folder_uid", folder.ParentUID)

	return nil
}

func DeleteFolder(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, idStr := OAPIClientFromExistingOrgResource(meta, d.Id())
	deleteParams := goapi.NewDeleteFolderParams().WithFolderUID(d.Get("uid").(string))
	if d.Get("prevent_destroy_if_not_empty").(bool) {
		// Search for dashboards and fail if any are found
		folderID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return diag.Errorf("failed to parse folder ID: %s", err)
		}
		searchType := "dash-db"
		searchParams := search.NewSearchParams().WithFolderIds([]int64{folderID}).WithType(&searchType)
		searchResp, err := client.Search.Search(searchParams, nil)
		if err != nil {
			return diag.Errorf("failed to search for dashboards in folder: %s", err)
		}
		if len(searchResp.GetPayload()) > 0 {
			var dashboardNames []string
			for _, dashboard := range searchResp.GetPayload() {
				dashboardNames = append(dashboardNames, dashboard.Title)
			}
			return diag.Errorf("folder %s is not empty and prevent_destroy_if_not_empty is set. It contains the following dashboards: %v", d.Get("uid").(string), dashboardNames)
		}
	} else {
		// If we're not preventing destroys, then we can force delete folders that have alert rules
		force := true
		deleteParams.WithForceDeleteRules(&force)
	}

	if _, err := client.Folders.DeleteFolder(deleteParams, nil); err != nil {
		return diag.Errorf("failed to delete folder: %s", err)
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

func GetFolderByIDorUID(client goapi.ClientService, id string) (*models.Folder, error) {
	// If the ID is a number, find the folder UID
	// Getting the folder by ID is broken in some versions, but getting by UID works in all versions
	// We need to use two API calls in the numerical ID case, because the "list" call doesn't have all the info
	if numericalID, err := strconv.ParseInt(id, 10, 64); err == nil {
		params := goapi.NewGetFolderByIDParams().WithFolderID(numericalID)
		resp, err := client.GetFolderByID(params, nil)
		if err != nil && !common.IsNotFoundError(err) {
			return nil, err
		} else if err == nil {
			return resp.GetPayload(), nil
		}
	}

	params := goapi.NewGetFolderByUIDParams().WithFolderUID(id)
	resp, err := client.GetFolderByUID(params, nil)
	if err != nil {
		return nil, err
	}
	return resp.GetPayload(), nil
}
