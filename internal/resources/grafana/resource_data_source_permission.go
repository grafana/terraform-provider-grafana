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

func ResourceDatasourcePermission() *schema.Resource {
	return &schema.Resource{

		Description: `
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/datasource_permissions/)
`,

		CreateContext: UpdateDatasourcePermissions,
		ReadContext:   ReadDatasourcePermissions,
		UpdateContext: UpdateDatasourcePermissions,
		DeleteContext: DeleteDatasourcePermissions,

		Schema: map[string]*schema.Schema{
			"org_id": orgIDAttribute(),
			"datasource_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "ID of the datasource to apply permissions to.",
			},
			"permissions": {
				Type:        schema.TypeSet,
				Required:    true,
				Description: "The permission items to add/update. Items that are omitted from the list will be removed.",
				// Ignore the org ID of the team/SA when hashing. It works with or without it.
				Set: func(i interface{}) int {
					m := i.(map[string]interface{})
					_, teamID := SplitOrgResourceID(m["team_id"].(string))
					_, userID := SplitOrgResourceID((m["user_id"].(string)))
					return schema.HashString(m["built_in_role"].(string) + teamID + userID + m["permission"].(string))
				},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
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
						"built_in_role": {
							Type:         schema.TypeString,
							Optional:     true,
							Default:      "",
							ValidateFunc: validation.StringInSlice([]string{"Viewer", "Editor", "Admin"}, false),
							Description:  "Name of the basic role to manage permissions for. Options: `Viewer`, `Editor` or `Admin`. Can only be set from Grafana v9.2.3+.",
						},
						"permission": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringInSlice([]string{"Query", "Edit", "Admin"}, false),
							Description:  "Permission to associate with item. Options: `Query`, `Edit` or `Admin` (`Admin` can only be used with Grafana v10.3.0+).",
						},
					},
				},
			},
		},
	}
}

func UpdateDatasourcePermissions(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID := ClientFromNewOrgResource(meta, d)

	v, ok := d.GetOk("permissions")
	if !ok {
		return nil
	}

	_, datasourceIDStr := SplitOrgResourceID(d.Get("datasource_id").(string))
	datasourceID, _ := strconv.ParseInt(datasourceIDStr, 10, 64)

	dataSource, err := client.DataSource(datasourceID)
	if err != nil {
		return diag.FromErr(err)
	}

	var configuredPermissions []gapi.SetResourcePermissionItem
	for _, permission := range v.(*schema.Set).List() {
		permission := permission.(map[string]interface{})
		var permissionItem gapi.SetResourcePermissionItem
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
		if permission["built_in_role"].(string) != "" {
			permissionItem.BuiltinRole = permission["built_in_role"].(string)
		}
		permissionItem.Permission = permission["permission"].(string)
		configuredPermissions = append(configuredPermissions, permissionItem)
	}

	if err := updateDatasourcePermissions(client, dataSource.UID, configuredPermissions); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(MakeOrgResourceID(orgID, datasourceID))

	return ReadDatasourcePermissions(ctx, d, meta)
}

func ReadDatasourcePermissions(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, idStr := ClientFromExistingOrgResource(meta, d.Id())

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	dataSource, err := client.DataSource(id)
	if err != nil {
		return diag.FromErr(err)
	}

	response, err := client.ListDatasourceResourcePermissions(dataSource.UID)
	if err, shouldReturn := common.CheckReadError("datasource permissions", d, err); shouldReturn {
		return err
	}

	var permissionItems []interface{}
	for _, permission := range response {
		// Only managed permissions can be provisioned through this resource, so we disregard the permissions obtained through custom and fixed roles here
		if !permission.IsManaged {
			continue
		}
		permissionItem := make(map[string]interface{})
		permissionItem["built_in_role"] = permission.BuiltInRole
		permissionItem["team_id"] = strconv.FormatInt(permission.TeamID, 10)
		permissionItem["user_id"] = strconv.FormatInt(permission.UserID, 10)
		permissionItem["permission"] = permission.Permission

		permissionItems = append(permissionItems, permissionItem)
	}

	d.Set("permissions", permissionItems)

	return nil
}

func DeleteDatasourcePermissions(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, idStr := ClientFromExistingOrgResource(meta, d.Id())

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	dataSource, err := client.DataSource(id)
	if err != nil {
		return diag.FromErr(err)
	}

	err = updateDatasourcePermissions(client, dataSource.UID, []gapi.SetResourcePermissionItem{})
	diags, _ := common.CheckReadError("datasource permissions", d, err)
	return diags
}

func updateDatasourcePermissions(client *gapi.Client, uid string, permissions []gapi.SetResourcePermissionItem) error {
	areEqual := func(a *gapi.ResourcePermission, b gapi.SetResourcePermissionItem) bool {
		return a.Permission == b.Permission && a.TeamID == b.TeamID && a.UserID == b.UserID && a.BuiltInRole == b.BuiltinRole
	}

	response, err := client.ListDatasourceResourcePermissions(uid)
	if err != nil {
		return err
	}

	var permissionList []gapi.SetResourcePermissionItem

deleteLoop:
	for _, current := range response {
		// Only managed permissions can be provisioned through this resource, so we disregard the permissions obtained through custom and fixed roles here
		if !current.IsManaged {
			continue
		}
		for _, new := range permissions {
			if areEqual(current, new) {
				continue deleteLoop
			}
		}

		permToRemove := gapi.SetResourcePermissionItem{
			TeamID:      current.TeamID,
			UserID:      current.UserID,
			BuiltinRole: current.BuiltInRole,
			Permission:  "",
		}

		permissionList = append(permissionList, permToRemove)
	}

addLoop:
	for _, new := range permissions {
		for _, current := range response {
			if areEqual(current, new) {
				continue addLoop
			}
		}

		permToAdd := gapi.SetResourcePermissionItem{
			TeamID:      new.TeamID,
			UserID:      new.UserID,
			BuiltinRole: new.BuiltinRole,
			Permission:  new.Permission,
		}

		permissionList = append(permissionList, permToAdd)
	}

	_, err = client.SetDatasourceResourcePermissions(uid, gapi.SetResourcePermissionsBody{Permissions: permissionList})

	return err
}
