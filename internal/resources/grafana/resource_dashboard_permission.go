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
				// Ignore the org ID of the team when hashing. It works with or without it.
				Set: func(i interface{}) int {
					m := i.(map[string]interface{})
					_, teamID := SplitOrgResourceID(m["team_id"].(string))
					return schema.HashString(m["role"].(string) + teamID + strconv.Itoa(m["user_id"].(int)) + m["permission"].(string))
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

func UpdateDashboardPermissions(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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
		if permission["user_id"].(int) != -1 {
			permissionItem.UserID = int64(permission["user_id"].(int))
		}
		permissionItem.Permission = mapPermissionStringToInt64(permission["permission"].(string))
		permissionList.Items = append(permissionList.Items, &permissionItem)
	}

	var (
		id  string
		err error
	)

	if uid, ok := d.GetOk("dashboard_uid"); ok {
		id = uid.(string)
		err = client.UpdateDashboardPermissionsByUID(id, &permissionList)
	} else {
		id = strconv.Itoa(d.Get("dashboard_id").(int))
		err = client.UpdateDashboardPermissions(int64(d.Get("dashboard_id").(int)), &permissionList)
	}
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(MakeOrgResourceID(orgID, id))

	return ReadDashboardPermissions(ctx, d, meta)
}

func ReadDashboardPermissions(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, idStr := ClientFromExistingOrgResource(meta, d.Id())
	var (
		dashboardPermissions []*gapi.DashboardPermission
		err                  error
	)

	if idInt, _ := strconv.ParseInt(idStr, 10, 64); idInt == 0 {
		// id is not an int, so it must be a uid
		dashboardPermissions, err = client.DashboardPermissionsByUID(idStr)
	} else {
		dashboardPermissions, err = client.DashboardPermissions(idInt)
	}
	if err, shouldReturn := common.CheckReadError("dashboard permissions", d, err); shouldReturn {
		return err
	}

	permissionItems := make([]interface{}, len(dashboardPermissions))
	count := 0
	for _, permission := range dashboardPermissions {
		if permission.DashboardID != -1 {
			permissionItem := make(map[string]interface{})
			permissionItem["role"] = permission.Role
			permissionItem["team_id"] = strconv.FormatInt(permission.TeamID, 10)
			permissionItem["user_id"] = permission.UserID
			permissionItem["permission"] = mapPermissionInt64ToString(permission.Permission)

			permissionItems[count] = permissionItem
			count++
			d.Set("dashboard_id", permission.DashboardID)
			d.Set("dashboard_uid", permission.DashboardUID)
		}
	}

	d.Set("permissions", permissionItems)

	return nil
}

func DeleteDashboardPermissions(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// since permissions are tied to dashboards, we can't really delete the permissions.
	// we will simply remove all permissions, leaving a dashboard that only an admin can access.
	// if for some reason the parent dashboard doesn't exist, we'll just ignore the error
	client, _, idStr := ClientFromExistingOrgResource(meta, d.Id())

	emptyPermissions := gapi.PermissionItems{}

	var err error
	if idInt, _ := strconv.ParseInt(idStr, 10, 64); idInt == 0 {
		err = client.UpdateDashboardPermissionsByUID(idStr, &emptyPermissions)
	} else {
		err = client.UpdateDashboardPermissions(int64(d.Get("dashboard_id").(int)), &emptyPermissions)
	}
	diags, _ := common.CheckReadError("dashboard permissions", d, err)
	return diags
}
