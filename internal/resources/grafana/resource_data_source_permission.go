package grafana

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/client/access_control"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v2/internal/common"
)

const datasourcesPermissionsType = "datasources"

func resourceDatasourcePermission() *common.Resource {
	schema := &schema.Resource{

		Description: `
Manages the entire set of permissions for a datasource. Permissions that aren't specified when applying this resource will be removed.
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
				Type:     schema.TypeSet,
				Optional: true,
				DefaultFunc: func() (interface{}, error) {
					return []interface{}{}, nil
				},
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

	return common.NewLegacySDKResource(
		"grafana_data_source_permission",
		orgResourceIDInt("datasourceID"),
		schema,
	)
}

func UpdateDatasourcePermissions(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID := OAPIClientFromNewOrgResource(meta, d)

	var list []interface{}
	if v, ok := d.GetOk("permissions"); ok {
		list = v.(*schema.Set).List()
	}

	// TODO: Switch to UID, but support both until next major release
	_, datasourceID := SplitOrgResourceID(d.Get("datasource_id").(string))
	resp, err := client.Datasources.GetDataSourceByID(datasourceID)
	if err != nil {
		return diag.FromErr(err)
	}
	datasource := resp.Payload

	var configuredPermissions []*models.SetResourcePermissionCommand
	for _, permission := range list {
		permission := permission.(map[string]interface{})
		var permissionItem models.SetResourcePermissionCommand
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
			permissionItem.BuiltInRole = permission["built_in_role"].(string)
		}
		permissionItem.Permission = permission["permission"].(string)
		configuredPermissions = append(configuredPermissions, &permissionItem)
	}

	if err := updateResourcePermissions(client, datasource.UID, datasourcesPermissionsType, configuredPermissions); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(MakeOrgResourceID(orgID, datasourceID))

	return ReadDatasourcePermissions(ctx, d, meta)
}

func ReadDatasourcePermissions(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, id := OAPIClientFromExistingOrgResource(meta, d.Id())

	resp, err := client.Datasources.GetDataSourceByID(id)
	if diag, shouldReturn := common.CheckReadError("data source permissions", d, err); shouldReturn {
		return diag
	}
	datasource := resp.Payload

	listResp, err := client.AccessControl.GetResourcePermissions(datasource.UID, datasourcesPermissionsType)
	if err, shouldReturn := common.CheckReadError("datasource permissions", d, err); shouldReturn {
		return err
	}

	var permissionItems []interface{}
	for _, permission := range listResp.Payload {
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
	client, _, id := OAPIClientFromExistingOrgResource(meta, d.Id())

	resp, err := client.Datasources.GetDataSourceByID(id)
	if diags, shouldReturn := common.CheckReadError("data source permissions", d, err); shouldReturn {
		return diags
	}
	datasource := resp.Payload

	err = updateResourcePermissions(client, datasource.UID, datasourcesPermissionsType, []*models.SetResourcePermissionCommand{})
	diags, _ := common.CheckReadError("datasource permissions", d, err)
	return diags
}

func updateResourcePermissions(client *goapi.GrafanaHTTPAPI, uid, resourceType string, permissions []*models.SetResourcePermissionCommand) error {
	areEqual := func(a *models.ResourcePermissionDTO, b *models.SetResourcePermissionCommand) bool {
		return a.Permission == b.Permission && a.TeamID == b.TeamID && a.UserID == b.UserID && a.BuiltInRole == b.BuiltInRole
	}

	listResp, err := client.AccessControl.GetResourcePermissions(uid, resourceType)
	if err != nil {
		return err
	}

	var permissionList []*models.SetResourcePermissionCommand
deleteLoop:
	for _, current := range listResp.Payload {
		// Only managed and non-inherited permissions can be provisioned through this resource, so we disregard the permissions obtained through custom and fixed roles here
		if !current.IsManaged || current.IsInherited {
			continue
		}
		for _, new := range permissions {
			if areEqual(current, new) {
				continue deleteLoop
			}
		}

		permToRemove := models.SetResourcePermissionCommand{
			TeamID:      current.TeamID,
			UserID:      current.UserID,
			BuiltInRole: current.BuiltInRole,
			Permission:  "",
		}

		permissionList = append(permissionList, &permToRemove)
	}

addLoop:
	for _, new := range permissions {
		for _, current := range listResp.Payload {
			if areEqual(current, new) {
				continue addLoop
			}
		}

		permissionList = append(permissionList, new)
	}

	body := models.SetPermissionsCommand{Permissions: permissionList}
	params := access_control.NewSetResourcePermissionsParams().
		WithResource(resourceType).
		WithResourceID(uid).
		WithBody(&body)
	_, err = client.AccessControl.SetResourcePermissions(params)

	return err
}
