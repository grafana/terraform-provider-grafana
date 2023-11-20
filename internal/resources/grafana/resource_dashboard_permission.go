package grafana

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/client/dashboard_permissions"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/internal/common"
)

func ResourceDashboardPermission() *schema.Resource {
	return &schema.Resource{

		Description: `
* [Official documentation](https://grafana.com/docs/grafana/latest/administration/roles-and-permissions/access-control/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/dashboard_permissions/)
`,

		CreateContext: UpdateDashboardPermissions,
		ReadContext:   ReadDashboardPermissions,
		UpdateContext: UpdateDashboardPermissions,
		DeleteContext: DeleteDashboardPermissions,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"org_id": orgIDAttribute(),
			"dashboard_id": {
				Type:         schema.TypeInt,
				ForceNew:     true,
				Computed:     true,
				Optional:     true,
				ExactlyOneOf: []string{"dashboard_id", "dashboard_uid"},
				Deprecated:   "use `dashboard_uid` instead",
				Description:  "ID of the dashboard to apply permissions to. Deprecated: use `dashboard_uid` instead.",
			},
			"dashboard_uid": {
				Type:         schema.TypeString,
				ForceNew:     true,
				Computed:     true,
				Optional:     true,
				ExactlyOneOf: []string{"dashboard_id", "dashboard_uid"},
				Description:  "UID of the dashboard to apply permissions to.",
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

func UpdateDashboardPermissions(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID := OAPIClientFromNewOrgResource(meta, d)

	v, ok := d.GetOk("permissions")
	if !ok {
		return nil
	}
	permissionList := models.UpdateDashboardACLCommand{}
	for _, permission := range v.(*schema.Set).List() {
		permission := permission.(map[string]interface{})
		permissionItem := models.DashboardACLUpdateItem{}
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
		permissionItem.Permission = parsePermissionType(permission["permission"].(string))
		permissionList.Items = append(permissionList.Items, &permissionItem)
	}

	var id string
	if dashboardID, ok := d.GetOk("dashboard_id"); ok {
		id = strconv.FormatInt(int64(dashboardID.(int)), 10)
	} else {
		id = d.Get("dashboard_uid").(string)
	}

	err := updateDashboardPermissions(client, id, &permissionList)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(MakeOrgResourceID(orgID, id))

	return ReadDashboardPermissions(ctx, d, meta)
}

func ReadDashboardPermissions(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, idStr := OAPIClientFromExistingOrgResource(meta, d.Id())
	var resp interface {
		GetPayload() []*models.DashboardACLInfoDTO
	}
	var err error

	if idInt, _ := strconv.ParseInt(idStr, 10, 64); idInt == 0 {
		// id is not an int, so it must be a uid
		params := dashboard_permissions.NewGetDashboardPermissionsListByUIDParams().WithUID(idStr)
		resp, err = client.DashboardPermissions.GetDashboardPermissionsListByUID(params, nil)
	} else {
		params := dashboard_permissions.NewGetDashboardPermissionsListByIDParams().WithDashboardID(idInt)
		resp, err = client.DashboardPermissions.GetDashboardPermissionsListByID(params, nil)
	}
	if err, shouldReturn := common.CheckReadError("dashboard permissions", d, err); shouldReturn {
		return err
	}

	dashboardPermissions := resp.GetPayload()
	permissionItems := make([]interface{}, len(dashboardPermissions))
	count := 0
	for _, permission := range dashboardPermissions {
		if permission.DashboardID != -1 {
			permissionItem := make(map[string]interface{})
			permissionItem["role"] = permission.Role
			permissionItem["team_id"] = strconv.FormatInt(permission.TeamID, 10)
			permissionItem["user_id"] = strconv.FormatInt(permission.UserID, 10)
			permissionItem["permission"] = permission.PermissionName

			permissionItems[count] = permissionItem
			count++
			d.Set("dashboard_id", permission.DashboardID)
			d.Set("dashboard_uid", permission.UID)
		}
	}

	d.Set("permissions", permissionItems)

	return nil
}

func DeleteDashboardPermissions(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// since permissions are tied to dashboards, we can't really delete the permissions.
	// we will simply remove all permissions, leaving a dashboard that only an admin can access.
	// if for some reason the parent dashboard doesn't exist, we'll just ignore the error
	client, _, idStr := OAPIClientFromExistingOrgResource(meta, d.Id())
	err := updateDashboardPermissions(client, idStr, &models.UpdateDashboardACLCommand{})
	diags, _ := common.CheckReadError("dashboard permissions", d, err)
	return diags
}

func updateDashboardPermissions(client *goapi.GrafanaHTTPAPI, id string, permissions *models.UpdateDashboardACLCommand) error {
	var err error
	if idInt, _ := strconv.ParseInt(id, 10, 64); idInt == 0 {
		// id is not an int, so it must be a uid
		params := dashboard_permissions.NewUpdateDashboardPermissionsByUIDParams().WithUID(id).WithBody(permissions)
		_, err = client.DashboardPermissions.UpdateDashboardPermissionsByUID(params, nil)
	} else {
		params := dashboard_permissions.NewUpdateDashboardPermissionsByIDParams().WithDashboardID(idInt).WithBody(permissions)
		_, err = client.DashboardPermissions.UpdateDashboardPermissionsByID(params, nil)
	}
	return err
}
