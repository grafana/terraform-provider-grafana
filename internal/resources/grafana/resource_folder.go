package grafana

import (
	"context"
	"encoding/json"
	"regexp"
	"strconv"
	"time"

	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/client/folders"
	"github.com/grafana/grafana-openapi-client-go/client/search"
	"github.com/grafana/grafana-openapi-client-go/models"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

var folderUIDValidation = validation.StringMatch(regexp.MustCompile(`^[a-zA-Z0-9\-\_]+$`), "folder UIDs can only be alphanumeric, dashes, or underscores")

func resourceFolder() *common.Resource {
	schema := &schema.Resource{

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
				Type:         schema.TypeString,
				Computed:     true,
				Optional:     true,
				ForceNew:     true,
				Description:  "Unique identifier.",
				ValidateFunc: folderUIDValidation,
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
				Description: "Prevent deletion of the folder if it is not empty (contains dashboards or alert rules). This feature requires Grafana 10.2 or later.",
			},
			"parent_folder_uid": {
				Type:     schema.TypeString,
				Optional: true,
				Description: "The uid of the parent folder. " +
					"If set, the folder will be nested. " +
					"If not set, the folder will be created in the root folder. " +
					"Note: This requires the nestedFolders feature flag to be enabled on your Grafana instance.",
			},
		},
	}

	return common.NewLegacySDKResource(
		common.CategoryGrafanaOSS,
		"grafana_folder",
		orgResourceIDString("uid"),
		schema,
	).WithLister(listerFunctionOrgResource(listFolders))
}

func listFolders(ctx context.Context, client *goapi.GrafanaHTTPAPI, orgID int64) ([]string, error) {
	return listDashboardOrFolder(client, orgID, "dash-folder")
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
		err := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
			parentFolder, err := GetFolderByIDorUID(client.Folders, parentUID.(string))
			if err != nil {
				return retry.RetryableError(err)
			}

			body.ParentUID = parentFolder.UID
			return nil
		})

		if err != nil {
			return diag.Errorf("failed to find parent folder '%s': %s", parentUID, err)
		}
	}

	resp, err := client.Folders.CreateFolder(&body)
	if err != nil {
		return diag.Errorf("failed to create folder: %s", err)
	}

	folder := resp.GetPayload()
	d.SetId(MakeOrgResourceID(orgID, folder.UID))

	return ReadFolder(ctx, d, meta)
}

func UpdateFolder(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, idStr := OAPIClientFromExistingOrgResource(meta, d.Id())

	folder, err := GetFolderByIDorUID(client.Folders, idStr)
	if err != nil {
		return diag.Errorf("failed to get folder %s: %s", idStr, err)
	}

	if d.HasChange("parent_folder_uid") {
		parentUID, ok := d.GetOk("parent_folder_uid")
		if !ok {
			// If the parent folder UID is not set, we can just clear it
			folder.ParentUID = ""
		} else {
			err := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
				parentFolder, err := GetFolderByIDorUID(client.Folders, parentUID.(string))
				if err != nil {
					return retry.RetryableError(err)
				}
				folder.ParentUID = parentFolder.UID
				return nil
			})

			if err != nil {
				return diag.Errorf("failed to find parent folder '%s': %s", parentUID, err)
			}
		}
		body := models.MoveFolderCommand{
			ParentUID: folder.ParentUID,
		}
		if _, err := client.Folders.MoveFolder(folder.UID, &body); err != nil {
			return diag.FromErr(err)
		}
	}

	body := models.UpdateFolderCommand{
		Overwrite: true,
		Title:     d.Get("title").(string),
	}

	if _, err := client.Folders.UpdateFolder(folder.UID, &body); err != nil {
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

	d.SetId(MakeOrgResourceID(orgID, folder.UID))
	d.Set("org_id", strconv.FormatInt(orgID, 10))
	d.Set("title", folder.Title)
	d.Set("uid", folder.UID)
	d.Set("url", metaClient.GrafanaSubpath(folder.URL))
	d.Set("parent_folder_uid", folder.ParentUID)

	return nil
}

func DeleteFolder(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, uid := OAPIClientFromExistingOrgResource(meta, d.Id())
	deleteParams := folders.NewDeleteFolderParams().WithFolderUID(uid)
	if d.Get("prevent_destroy_if_not_empty").(bool) {
		searchParams := search.NewSearchParams().WithFolderUIDs([]string{uid})
		searchResp, err := client.Search.Search(searchParams)
		if err != nil {
			return diag.Errorf("failed to search for dashboards in folder: %s", err)
		}
		if len(searchResp.GetPayload()) > 0 {
			var dashboardAndFolderNames []string
			for _, dashboard := range searchResp.GetPayload() {
				dashboardAndFolderNames = append(dashboardAndFolderNames, dashboard.Title)
			}
			return diag.Errorf("folder %s is not empty and prevent_destroy_if_not_empty is set. It contains the following dashboards and/or folders: %v", uid, dashboardAndFolderNames)
		}
	} else {
		// If we're not preventing destroys, then we can force delete folders that have alert rules
		force := true
		deleteParams.WithForceDeleteRules(&force)
	}

	_, err := client.Folders.DeleteFolder(deleteParams)
	diag, _ := common.CheckReadError("folder", d, err)
	return diag
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

func GetFolderByIDorUID(client folders.ClientService, id string) (*models.Folder, error) {
	// If the ID is a number, find the folder UID
	// Getting the folder by ID is broken in some versions, but getting by UID works in all versions
	// We need to use two API calls in the numerical ID case, because the "list" call doesn't have all the info
	if numericalID, err := strconv.ParseInt(id, 10, 64); err == nil {
		resp, err := client.GetFolderByID(numericalID)
		if err != nil && !common.IsNotFoundError(err) {
			return nil, err
		} else if err == nil {
			return resp.GetPayload(), nil
		}
	}

	resp, err := client.GetFolderByUID(id)
	if err != nil {
		return nil, err
	}
	return resp.GetPayload(), nil
}
