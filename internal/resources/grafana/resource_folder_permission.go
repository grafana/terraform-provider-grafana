package grafana

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

func resourceFolderPermission() *common.Resource {
	crudHelper := &resourcePermissionsHelper{
		resourceType:  foldersPermissionsType,
		roleAttribute: "role",
		getResource:   resourceFolderPermissionGet,
	}

	schema := &schema.Resource{
		Description: `
Manages the entire set of permissions for a folder. Permissions that aren't specified when applying this resource will be removed.
* [Official documentation](https://grafana.com/docs/grafana/latest/administration/roles-and-permissions/access-control/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/folder_permissions/)
`,

		CreateContext: crudHelper.updatePermissions,
		ReadContext:   crudHelper.readPermissions,
		UpdateContext: crudHelper.updatePermissions,
		DeleteContext: crudHelper.deletePermissions,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"folder_uid": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The UID of the folder.",
				ValidateFunc: folderUIDValidation,
			},
		},
	}
	crudHelper.addCommonSchemaAttributes(schema.Schema)

	return common.NewLegacySDKResource(
		common.CategoryGrafanaOSS,
		"grafana_folder_permission",
		orgResourceIDString("folderUID"),
		schema,
	)
}

func resourceFolderPermissionGet(d *schema.ResourceData, meta any) (string, error) {
	client, _ := OAPIClientFromNewOrgResource(meta, d)
	uid := d.Get("folder_uid").(string)
	if d.Id() != "" {
		client, _, uid = OAPIClientFromExistingOrgResource(meta, d.Id())
	}
	resp, err := client.Folders.GetFolderByUID(uid)
	if err != nil {
		return "", err
	}
	folder := resp.Payload
	d.Set("folder_uid", folder.UID)
	return folder.UID, nil
}
