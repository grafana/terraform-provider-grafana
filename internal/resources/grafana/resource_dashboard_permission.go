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

func ResourceDashboardPermission() *schema.Resource {
	return &schema.Resource{

		Description: `
Manages the entire set of permissions for a dashboard. Permissions that aren't specified when applying this resource will be removed.
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
	client, orgID, _ := ClientFromExistingOrgResource(meta, d.Id())

	var list []interface{}
	if v, ok := d.GetOk("permissions"); ok {
		list = v.(*schema.Set).List()
	}

	permissionList := make([]gapi.SetResourcePermissionItem, 0)
	for _, permission := range list {
		permission := permission.(map[string]interface{})
		permissionItem := gapi.SetResourcePermissionItem{}
		if permission["role"].(string) != "" {
			permissionItem.BuiltinRole = permission["role"].(string)
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
		permissionList = append(permissionList, permissionItem)
	}

	var id = d.Get("dashboard_uid").(string)

	if _, err := client.SetFolderResourcePermissions(id, gapi.SetResourcePermissionsBody{
		Permissions: permissionList,
	}); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(MakeOrgResourceID(orgID, id))

	return ReadDashboardPermissions(ctx, d, meta)
}

func ReadDashboardPermissions(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, idStr := ClientFromExistingOrgResource(meta, d.Id())
	dashboardPermissions, err := client.ListDashboardResourcePermissions(idStr)

	// if idInt, _ := strconv.ParseInt(idStr, 10, 64); idInt == 0 {
	// 	// id is not an int, so it must be a uid
	// 	resp, err = client.DashboardPermissions.GetDashboardPermissionsListByUID(idStr)
	// } else {
	// 	resp, err = client.DashboardPermissions.GetDashboardPermissionsListByID(idInt)
	// }
	if err, shouldReturn := common.CheckReadError("dashboard permissions", d, err); shouldReturn {
		return err
	}

	permissionItems := make([]interface{}, len(dashboardPermissions))
	for _, permission := range dashboardPermissions {
		permissionItem := make(map[string]interface{})
		permissionItem["role"] = permission.RoleName
		permissionItem["team_id"] = strconv.FormatInt(permission.TeamID, 10)
		permissionItem["user_id"] = strconv.FormatInt(permission.UserID, 10)
		permissionItem["permission"] = permission.Permission
		permissionItems = append(permissionItems, permissionItem)
		// d.Set("dashboard_id", permission.DashboardID)
		// d.Set("dashboard_uid", permission.UID)
	}

	d.Set("permissions", permissionItems)

	return nil
}

func DeleteDashboardPermissions(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// since permissions are tied to dashboards, we can't really delete the permissions.
	// we will simply remove all permissions, leaving a dashboard that only an admin can access.
	// if for some reason the parent dashboard doesn't exist, we'll just ignore the error
	client, _, idStr := ClientFromExistingOrgResource(meta, d.Id())
	_, err := client.SetFolderResourcePermissions(idStr, gapi.SetResourcePermissionsBody{
		Permissions: []gapi.SetResourcePermissionItem{},
	})
	diags, _ := common.CheckReadError("dashboard permissions", d, err)
	return diags
}
