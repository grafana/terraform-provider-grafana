// Warning: The following are still in SDK2 format. They will eventually be converted to Plugin Framework format.

package grafana

import (
	"context"
	"strconv"

	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/client/access_control"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

type resourcePermissionsHelper struct {
	resourceType  string
	roleAttribute string // Not all resources have the same name for this attribute

	// Given the resource data, check the resource exists and return the correct ID for permissions.
	// Ex: We support ID and UID for dashboards but the permissions are managed by UID.
	getResource func(d *schema.ResourceData, meta interface{}) (string, error)
}

func (h *resourcePermissionsHelper) addCommonSchemaAttributes(s map[string]*schema.Schema) {
	permissionSchema := map[string]*schema.Schema{
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
	}
	if h.resourceType == datasourcesPermissionsType {
		permissionSchema["permission"] = &schema.Schema{
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringInSlice([]string{"Query", "Edit", "Admin"}, false),
			Description:  "Permission to associate with item. Options: `Query`, `Edit` or `Admin` (`Admin` can only be used with Grafana v10.3.0+).",
		}
	}
	if h.roleAttribute != "" {
		permissionSchema[h.roleAttribute] = &schema.Schema{
			Type:         schema.TypeString,
			Optional:     true,
			ValidateFunc: validation.StringInSlice([]string{"Viewer", "Editor", "Admin"}, false),
			Description:  "Name of the basic role to manage permissions for. Options: `Viewer`, `Editor` or `Admin`.",
		}
	}

	commonSchema := map[string]*schema.Schema{
		"org_id": orgIDAttribute(),
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
				role := ""
				if h.roleAttribute != "" {
					role = m[h.roleAttribute].(string)
				}
				return schema.HashString(role + teamID + userID + m["permission"].(string))
			},
			Elem: &schema.Resource{
				Schema: permissionSchema,
			},
		},
	}

	for k, v := range commonSchema {
		s[k] = v
	}
}

func (h *resourcePermissionsHelper) updatePermissions(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID := OAPIClientFromNewOrgResource(meta, d)

	resourceID, err := h.getResource(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	var list []interface{}
	if v, ok := d.GetOk("permissions"); ok {
		list = v.(*schema.Set).List()
	}
	var permissionList []*models.SetResourcePermissionCommand
	for _, permission := range list {
		permission := permission.(map[string]interface{})
		permissionItem := models.SetResourcePermissionCommand{}
		if h.roleAttribute != "" && permission[h.roleAttribute].(string) != "" {
			permissionItem.BuiltInRole = permission[h.roleAttribute].(string)
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
		permissionItem.Permission = permission["permission"].(string)
		permissionList = append(permissionList, &permissionItem)
	}

	if err := h.updateResourcePermissions(client, resourceID, permissionList); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(MakeOrgResourceID(orgID, resourceID))

	return h.readPermissions(ctx, d, meta)
}

func (h *resourcePermissionsHelper) readPermissions(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID, resourceID := OAPIClientFromExistingOrgResource(meta, d.Id())

	// Check if the resource still exists
	_, err := h.getResource(d, meta)
	if err, shouldReturn := common.CheckReadError("resource", d, err); shouldReturn {
		return err
	}

	resp, err := client.AccessControl.GetResourcePermissions(resourceID, h.resourceType)
	if err, shouldReturn := common.CheckReadError("permissions", d, err); shouldReturn {
		return err
	}

	resourcePermissions := resp.Payload
	var permissionItems []interface{}
	for _, permission := range resourcePermissions {
		// Only managed permissions can be provisioned through this resource, so we disregard the permissions obtained through custom and fixed roles here
		if !permission.IsManaged || permission.IsInherited {
			continue
		}
		permissionItem := make(map[string]interface{})
		if h.roleAttribute != "" {
			permissionItem[h.roleAttribute] = permission.BuiltInRole
		}
		permissionItem["team_id"] = strconv.FormatInt(permission.TeamID, 10)
		permissionItem["user_id"] = strconv.FormatInt(permission.UserID, 10)
		permissionItem["permission"] = permission.Permission

		permissionItems = append(permissionItems, permissionItem)
	}

	d.SetId(MakeOrgResourceID(orgID, resourceID))
	d.Set("org_id", strconv.FormatInt(orgID, 10))
	d.Set("permissions", permissionItems)

	return nil
}

func (h *resourcePermissionsHelper) deletePermissions(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// since permissions are tied to the resource, we can't really delete the permissions.
	// we will simply remove all permissions, leaving a resource that only an admin can access.
	// if for some reason the resource doesn't exist, we'll just ignore the error
	client, _, resourceID := OAPIClientFromExistingOrgResource(meta, d.Id())
	err := h.updateResourcePermissions(client, resourceID, []*models.SetResourcePermissionCommand{})
	diags, _ := common.CheckReadError("permissions", d, err)
	return diags
}

func (h *resourcePermissionsHelper) updateResourcePermissions(client *goapi.GrafanaHTTPAPI, uid string, permissions []*models.SetResourcePermissionCommand) error {
	areEqual := func(a *models.ResourcePermissionDTO, b *models.SetResourcePermissionCommand) bool {
		return a.Permission == b.Permission && a.TeamID == b.TeamID && a.UserID == b.UserID && a.BuiltInRole == b.BuiltInRole
	}

	listResp, err := client.AccessControl.GetResourcePermissions(uid, h.resourceType)
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
			// Only managed and non-inherited permissions can be provisioned through this resource, so we disregard the permissions obtained through custom and fixed roles here
			if !current.IsManaged || current.IsInherited {
				continue
			}
			if areEqual(current, new) {
				continue addLoop
			}
		}

		permissionList = append(permissionList, new)
	}

	body := models.SetPermissionsCommand{Permissions: permissionList}
	params := access_control.NewSetResourcePermissionsParams().
		WithResource(h.resourceType).
		WithResourceID(uid).
		WithBody(&body)
	_, err = client.AccessControl.SetResourcePermissions(params)

	return err
}
