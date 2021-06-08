package grafana

import (
	"context"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	gapi "github.com/grafana/grafana-api-golang-client"
)

func ResourceFolderPermission() *schema.Resource {
	return &schema.Resource{

		Description: `
* [Official documentation](https://grafana.com/docs/grafana/latest/permissions/dashboard_folder_permissions/)
* [HTTP API](https://grafana.com/docs/grafana/latest/http_api/folder_permissions/)
`,

		CreateContext: UpdateFolderPermissions,
		ReadContext:   ReadFolderPermissions,
		UpdateContext: UpdateFolderPermissions,
		DeleteContext: DeleteFolderPermissions,

		Schema: map[string]*schema.Schema{
			"folder_uid": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The UID of the folder.",
			},
			"permissions": {
				Type:        schema.TypeSet,
				Required:    true,
				Description: "The permission items to add/update. Items that are omitted from the list will be removed.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"role": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validation.StringInSlice([]string{"Viewer", "Editor"}, false),
							Description:  "Manage permissions for `Viewer` or `Editor` roles.",
						},
						"team_id": {
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     0,
							Description: "ID of the team to manage permissions for.",
						},
						"user_id": {
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     0,
							Description: "ID of the user to manage permissions for.",
						},
						"permission": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringInSlice([]string{"View", "Edit", "Admin"}, false),
							Description:  "Permission to associate with item. Must be one of `View`, `Edit`, or `Admin`.",
						},
					},
				},
			},
		},
	}
}

func UpdateFolderPermissions(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gapi.Client)

	v, ok := d.GetOk("permissions")
	if !ok {
		return nil
	}
	permissionList := gapi.PermissionItems{}
	for _, permission := range v.(*schema.Set).List() {
		permission := permission.(map[string]interface{})
		permissionItem := gapi.PermissionItem{}
		if permission["role"].(string) != "" {
			permissionItem.Role = permission["role"].(string)
		}
		if permission["team_id"].(int) != -1 {
			permissionItem.TeamID = int64(permission["team_id"].(int))
		}
		if permission["user_id"].(int) != -1 {
			permissionItem.UserID = int64(permission["user_id"].(int))
		}
		permissionItem.Permission = mapPermissionStringToInt64(permission["permission"].(string))
		permissionList.Items = append(permissionList.Items, &permissionItem)
	}

	folderUID := d.Get("folder_uid").(string)

	err := client.UpdateFolderPermissions(folderUID, &permissionList)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(folderUID)

	return ReadFolderPermissions(ctx, d, meta)
}

func ReadFolderPermissions(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gapi.Client)

	folderUID := d.Get("folder_uid").(string)

	folderPermissions, err := client.FolderPermissions(folderUID)
	if err != nil {
		if strings.HasPrefix(err.Error(), "status: 404") {
			log.Printf("[WARN] removing folder %s from state because it no longer exists in grafana", folderUID)
			d.SetId("")
			return nil
		}

		return diag.FromErr(err)
	}

	permissionItems := make([]interface{}, len(folderPermissions))
	count := 0
	for _, permission := range folderPermissions {
		if permission.FolderUID != "" {
			permissionItem := make(map[string]interface{})
			permissionItem["role"] = permission.Role
			permissionItem["team_id"] = permission.TeamID
			permissionItem["user_id"] = permission.UserID
			permissionItem["permission"] = mapPermissionInt64ToString(permission.Permission)

			permissionItems[count] = permissionItem
			count++
		}
	}

	d.Set("permissions", permissionItems)

	return nil
}

func DeleteFolderPermissions(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	//since permissions are tied to folders, we can't really delete the permissions.
	//we will simply remove all permissions, leaving a folder that only an admin can access.
	//if for some reason the parent folder doesn't exist, we'll just ignore the error
	client := meta.(*gapi.Client)

	folderUID := d.Get("folder_uid").(string)
	emptyPermissions := gapi.PermissionItems{}

	err := client.UpdateFolderPermissions(folderUID, &emptyPermissions)
	if err != nil {
		if strings.HasPrefix(err.Error(), "status: 404") {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	return nil
}

func mapPermissionStringToInt64(permission string) int64 {
	permissionInt := int64(-1)
	switch permission {
	case "View":
		permissionInt = int64(1)
	case "Edit":
		permissionInt = int64(2)
	case "Admin":
		permissionInt = int64(4)
	}
	return permissionInt
}

func mapPermissionInt64ToString(permission int64) string {
	permissionString := "-1"
	switch permission {
	case 1:
		permissionString = "View"
	case 2:
		permissionString = "Edit"
	case 4:
		permissionString = "Admin"
	}
	return permissionString
}
