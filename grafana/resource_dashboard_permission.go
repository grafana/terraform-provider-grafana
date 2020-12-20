package grafana

import (
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"

	gapi "github.com/grafana/grafana-api-golang-client"
)

func ResourceDashboardPermission() *schema.Resource {
	return &schema.Resource{
		Create: UpdateDashboardPermissions,
		Read:   ReadDashboardPermissions,
		Update: UpdateDashboardPermissions,
		Delete: DeleteDashboardPermissions,

		Schema: map[string]*schema.Schema{
			"dashboard_id": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
			"permissions": {
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"role": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validation.StringInSlice([]string{"Viewer", "Editor"}, false),
						},
						"team_id": {
							Type:     schema.TypeInt,
							Optional: true,
							Default:  -1,
						},
						"user_id": {
							Type:     schema.TypeInt,
							Optional: true,
							Default:  -1,
						},
						"permission": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringInSlice([]string{"View", "Edit", "Admin"}, false),
						},
					},
				},
			},
		},
	}
}

func UpdateDashboardPermissions(d *schema.ResourceData, meta interface{}) error {
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

	dashboardId := int64(d.Get("dashboard_id").(int))

	err := client.UpdateDashboardPermissions(dashboardId, &permissionList)
	if err != nil {
		return err
	}

	d.SetId(strconv.FormatInt(dashboardId, 10))

	return ReadDashboardPermissions(d, meta)
}

func ReadDashboardPermissions(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)

	dashboardId := int64(d.Get("dashboard_id").(int))

	dashboardPermissions, err := client.DashboardPermissions(dashboardId)
	if err != nil {
		if strings.HasPrefix(err.Error(), "status: 404") {
			log.Printf("[WARN] removing dashboard permissions %d from state because it no longer exists in grafana", dashboardId)
			d.SetId("")
			return nil
		}

		return err
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

			permissionItems[count] = permission
			count++
		}
	}

	d.Set("permissions", permissionItems)

	return nil
}

func DeleteDashboardPermissions(d *schema.ResourceData, meta interface{}) error {
	//since permissions are tied to dashboards, we can't really delete the permissions.
	//we will simply remove all permissions, leaving a dashboard that only an admin can access.
	//if for some reason the parent dashboard doesn't exist, we'll just ignore the error
	client := meta.(*gapi.Client)

	dashboardId := int64(d.Get("dashboard_id").(int))
	emptyPermissions := gapi.PermissionItems{}

	err := client.UpdateDashboardPermissions(dashboardId, &emptyPermissions)
	if err != nil {
		if strings.HasPrefix(err.Error(), "status: 404") {
			d.SetId("")
			return nil
		}
		return err
	}

	return nil
}
