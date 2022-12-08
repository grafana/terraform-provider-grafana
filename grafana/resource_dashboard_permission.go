package grafana

import (
	"context"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	gapi "github.com/grafana/grafana-api-golang-client"
)

func ResourceDashboardPermission() *schema.Resource {
	return &schema.Resource{

		Description: `
* [Official documentation](https://grafana.com/docs/grafana/latest/permissions/dashboard_folder_permissions/)
* [HTTP API](https://grafana.com/docs/grafana/latest/http_api/dashboard_permissions/)
`,

		CreateContext: UpdateDashboardPermissions,
		ReadContext:   ReadDashboardPermissions,
		UpdateContext: UpdateDashboardPermissions,
		DeleteContext: DeleteDashboardPermissions,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
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

func UpdateDashboardPermissions(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi

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

	d.SetId(id)

	return ReadDashboardPermissions(ctx, d, meta)
}

func ReadDashboardPermissions(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi
	var (
		dashboardPermissions []*gapi.DashboardPermission
		err                  error
	)

	id := d.Id()
	if idInt, _ := strconv.Atoi(id); idInt == 0 {
		// id is not an int, so it must be a uid
		dashboardPermissions, err = client.DashboardPermissionsByUID(id)
	} else {
		dashboardPermissions, err = client.DashboardPermissions(int64(idInt))
	}

	if err != nil {
		if strings.HasPrefix(err.Error(), "status: 404") {
			log.Printf("[WARN] removing dashboard permissions %s from state because it no longer exists in grafana", id)
			d.SetId("")
			return nil
		}

		return diag.FromErr(err)
	}

	permissionItems := make([]interface{}, len(dashboardPermissions))
	count := 0
	for _, permission := range dashboardPermissions {
		if permission.DashboardID != -1 {
			permissionItem := make(map[string]interface{})
			permissionItem["role"] = permission.Role
			permissionItem["team_id"] = permission.TeamID
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
	client := meta.(*client).gapi

	emptyPermissions := gapi.PermissionItems{}

	var err error
	if uid, ok := d.GetOk("dashboard_uid"); ok {
		err = client.UpdateDashboardPermissionsByUID(uid.(string), &emptyPermissions)
	} else {
		err = client.UpdateDashboardPermissions(int64(d.Get("dashboard_id").(int)), &emptyPermissions)
	}
	if err != nil {
		if strings.HasPrefix(err.Error(), "status: 404") {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	return nil
}
