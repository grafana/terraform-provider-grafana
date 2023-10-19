package grafana

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
)

func ResourceFolderPermission() *schema.Resource {
	return &schema.Resource{

		Description: `
* [Official documentation](https://grafana.com/docs/grafana/latest/administration/roles-and-permissions/access-control/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/folder_permissions/)
`,

		CreateContext: UpdateFolderPermissions,
		ReadContext:   ReadFolderPermissions,
		UpdateContext: UpdateFolderPermissions,
		DeleteContext: DeleteFolderPermissions,

		Schema: map[string]*schema.Schema{
			"org_id": orgIDAttribute(),
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
				// Ignore the org ID of the team/SA when hashing. It works with or without it.
				Set: func(i interface{}) int {
					m := i.(map[string]interface{})
					_, teamID := SplitOrgResourceID(m["team_id"].(string))
					_, userID := SplitOrgResourceID(m["user_id"].(string))
					return schema.HashString(m["role"].(string) + teamID + userID + m["permission"].(string))
				},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"role": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validation.StringInSlice([]string{"Viewer", "Editor"}, false),
							Description:  "Manage permissions for `Viewer` or `Editor` roles.",
						},
						"team_id": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "0",
							Description: "ID of the team to manage permissions for.",
						},
						"user_id": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "0",
							Description: "ID of the user or service account to manage permissions for.",
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
	client, orgID := ClientFromNewOrgResource(meta, d)

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
		_, teamIDStr := SplitOrgResourceID(permission["team_id"].(string))
		teamID, _ := strconv.ParseInt(teamIDStr, 10, 64)
		if teamID > 0 {
			permissionItem.TeamID = teamID
		}
		_, userIDStr := SplitOrgResourceID(permission["user_id"].(string))
		userID, _ := strconv.ParseInt(userIDStr, 10, 64)
		if userID > 0 {
			permissionItem.UserID = userID
		}
		permissionItem.Permission = mapPermissionStringToInt64(permission["permission"].(string))
		permissionList.Items = append(permissionList.Items, &permissionItem)
	}

	folderUID := d.Get("folder_uid").(string)

	err := client.UpdateFolderPermissions(folderUID, &permissionList)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(MakeOrgResourceID(orgID, folderUID))

	return ReadFolderPermissions(ctx, d, meta)
}

func ReadFolderPermissions(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID, folderUID := ClientFromExistingOrgResource(meta, d.Id())

	folderPermissions, err := client.FolderPermissions(folderUID)
	if err, shouldReturn := common.CheckReadError("folder permissions", d, err); shouldReturn {
		return err
	}

	permissionItems := make([]interface{}, len(folderPermissions))
	count := 0
	for _, permission := range folderPermissions {
		if permission.FolderUID != "" {
			permissionItem := make(map[string]interface{})
			permissionItem["role"] = permission.Role
			permissionItem["team_id"] = strconv.FormatInt(permission.TeamID, 10)
			permissionItem["user_id"] = strconv.FormatInt(permission.UserID, 10)
			permissionItem["permission"] = mapPermissionInt64ToString(permission.Permission)

			permissionItems[count] = permissionItem
			count++
		}
	}

	d.SetId(MakeOrgResourceID(orgID, folderUID))
	d.Set("org_id", strconv.FormatInt(orgID, 10))
	d.Set("permissions", permissionItems)

	return nil
}

func DeleteFolderPermissions(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// since permissions are tied to folders, we can't really delete the permissions.
	// we will simply remove all permissions, leaving a folder that only an admin can access.
	// if for some reason the parent folder doesn't exist, we'll just ignore the error
	client, _, folderUID := ClientFromExistingOrgResource(meta, d.Id())
	emptyPermissions := gapi.PermissionItems{}
	err := client.UpdateFolderPermissions(folderUID, &emptyPermissions)
	diags, _ := common.CheckReadError("folder permissions", d, err)
	return diags
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
