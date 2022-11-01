package grafana

import (
	"context"
	"log"
	"strconv"
	"strings"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func ResourceServiceAccountPermission() *schema.Resource {
	return &schema.Resource{
		Description: `
**Note:** This resource is available from Grafana 9.2.4 onwards.

* [Official documentation](https://grafana.com/docs/grafana/latest/administration/service-accounts/#manage-users-and-teams-permissions-for-a-service-account-in-grafana)`,

		CreateContext: UpdateServiceAccountPermissions,
		ReadContext:   ReadServiceAccountPermissions,
		UpdateContext: UpdateServiceAccountPermissions,
		DeleteContext: DeleteServiceAccountPermissions,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"service_account_id": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "The id of the service account.",
			},
			"permissions": {
				Type:        schema.TypeSet,
				Required:    true,
				Description: "The permission items to add/update. Items that are omitted from the list will be removed.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
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
							ValidateFunc: validation.StringInSlice([]string{"Edit", "Admin"}, false),
							Description:  "Permission to associate with item. Must be `Edit` or `Admin`.",
						},
					},
				},
			},
		},
	}
}

func ReadServiceAccountPermissions(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi
	id, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	saPermissions, err := client.GetServiceAccountPermissions(id)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			d.SetId("")
			log.Printf("[WARN] removing permissions for account with ID %d from state because the service account no longer exists in grafana", id)
			return nil
		}

		return diag.FromErr(err)
	}

	saPerms := make([]interface{}, 0)
	for _, p := range saPermissions {
		// Only managed service account permissions can be provisioned through this resource.
		if !p.IsManaged {
			continue
		}
		permMap := map[string]interface{}{
			"team_id":    p.TeamID,
			"user_id":    p.UserID,
			"permission": p.Permission,
		}
		saPerms = append(saPerms, permMap)
	}
	if err = d.Set("permissions", saPerms); err != nil {
		return diag.FromErr(err)
	}
	if err = d.Set("service_account_id", id); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(strconv.FormatInt(id, 10))

	return nil
}

func UpdateServiceAccountPermissions(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi

	// Get a list of permissions from Grafana state (current permission setup)
	state, config := d.GetChange("permissions")
	oldTeamPerms := make(map[int64]string, 0)
	oldUserPerms := make(map[int64]string, 0)
	for _, p := range state.(*schema.Set).List() {
		perm := p.(map[string]interface{})
		teamId := int64(perm["team_id"].(int))
		userId := int64(perm["user_id"].(int))
		if teamId > 0 {
			oldTeamPerms[teamId] = perm["permission"].(string)
		}
		if userId > 0 {
			oldUserPerms[userId] = perm["permission"].(string)
		}
	}

	permissionList := gapi.ServiceAccountPermissionItems{}

	// Iterate over permissions from the configuration (the desired permission setup)
	for _, p := range config.(*schema.Set).List() {
		permission := p.(map[string]interface{})
		permissionItem := gapi.ServiceAccountPermissionItem{}
		teamId := int64(permission["team_id"].(int))
		userId := int64(permission["user_id"].(int))
		if teamId > 0 {
			perm, has := oldTeamPerms[teamId]
			if has {
				delete(oldTeamPerms, teamId)
				// Skip permissions that have not been changed
				if perm == permission["permission"].(string) {
					continue
				}
			}
			permissionItem.TeamID = teamId
		} else if userId > 0 {
			perm, has := oldUserPerms[userId]
			if has {
				delete(oldUserPerms, userId)
				if perm == permission["permission"].(string) {
					continue
				}
			}
			permissionItem.UserID = userId
		}
		permissionItem.Permission = permission["permission"].(string)
		permissionList.Permissions = append(permissionList.Permissions, &permissionItem)
	}

	// Remove the permissions that are in the state but not in the config
	for teamId := range oldTeamPerms {
		permissionList.Permissions = append(permissionList.Permissions, &gapi.ServiceAccountPermissionItem{
			TeamID:     teamId,
			Permission: "",
		})
	}
	for userId := range oldUserPerms {
		permissionList.Permissions = append(permissionList.Permissions, &gapi.ServiceAccountPermissionItem{
			UserID:     userId,
			Permission: "",
		})
	}

	saID := int64(d.Get("service_account_id").(int))
	err := client.UpdateServiceAccountPermissions(saID, &permissionList)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.FormatInt(saID, 10))

	return ReadServiceAccountPermissions(ctx, d, meta)
}

func DeleteServiceAccountPermissions(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi

	state, _ := d.GetChange("permissions")
	permissionList := gapi.ServiceAccountPermissionItems{}
	for _, p := range state.(*schema.Set).List() {
		perm := p.(map[string]interface{})
		teamId := int64(perm["team_id"].(int))
		userId := int64(perm["user_id"].(int))
		permissionItem := gapi.ServiceAccountPermissionItem{}

		if teamId > 0 {
			permissionItem.TeamID = teamId
		} else if userId > 0 {
			permissionItem.UserID = userId
		}
		permissionItem.Permission = ""
		permissionList.Permissions = append(permissionList.Permissions, &permissionItem)
	}

	id := int64(d.Get("service_account_id").(int))
	err := client.UpdateServiceAccountPermissions(id, &permissionList)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}
