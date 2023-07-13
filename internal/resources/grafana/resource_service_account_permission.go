package grafana

import (
	"context"
	"strconv"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
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
			"org_id": orgIDAttribute(),
			"service_account_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The id of the service account.",
			},
			"permissions": {
				Type:        schema.TypeSet,
				Required:    true,
				Description: "The permission items to add/update. Items that are omitted from the list will be removed.",
				// Ignore the org ID of the team when hashing. It works with or without it.
				Set: func(i interface{}) int {
					m := i.(map[string]interface{})
					_, teamID := SplitOrgResourceID(m["team_id"].(string))
					return schema.HashString(teamID + strconv.Itoa(m["user_id"].(int)) + m["permission"].(string))
				},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"team_id": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "0",
							Description: "ID of the team to manage permissions for. Specify either this or `user_id`.",
						},
						"user_id": {
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     0,
							Description: "ID of the user to manage permissions for. Specify either this or `team_id`.",
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
	orgID, serviceAccountIDStr := SplitOrgResourceID(d.Id())
	client := meta.(*common.Client).GrafanaAPI.WithOrgID(orgID)
	id, err := strconv.ParseInt(serviceAccountIDStr, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	saPermissions, err := client.GetServiceAccountPermissions(id)
	if err, shouldReturn := common.CheckReadError("service account permissions", d, err); shouldReturn {
		return err
	}

	saPerms := make([]interface{}, 0)
	for _, p := range saPermissions {
		// Only managed service account permissions can be provisioned through this resource.
		if !p.IsManaged {
			continue
		}
		permMap := map[string]interface{}{
			"team_id":    strconv.FormatInt(p.TeamID, 10),
			"user_id":    p.UserID,
			"permission": p.Permission,
		}
		saPerms = append(saPerms, permMap)
	}
	d.Set("permissions", saPerms)

	return nil
}

func UpdateServiceAccountPermissions(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	orgID, serviceAccountIDStr := SplitOrgResourceID(d.Get("service_account_id").(string))
	client := meta.(*common.Client).GrafanaAPI.WithOrgID(orgID)
	id, err := strconv.ParseInt(serviceAccountIDStr, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	// Get a list of permissions from Grafana state (current permission setup)
	state, config := d.GetChange("permissions")
	oldTeamPerms := make(map[int64]string, 0)
	oldUserPerms := make(map[int64]string, 0)
	for _, p := range state.(*schema.Set).List() {
		perm := p.(map[string]interface{})
		_, teamIDStr := SplitOrgResourceID(perm["team_id"].(string))
		teamID, _ := strconv.ParseInt(teamIDStr, 10, 64)
		userID := int64(perm["user_id"].(int))
		if teamID > 0 {
			oldTeamPerms[teamID] = perm["permission"].(string)
		}
		if userID > 0 {
			oldUserPerms[userID] = perm["permission"].(string)
		}
	}

	permissionList := gapi.ServiceAccountPermissionItems{}

	// Iterate over permissions from the configuration (the desired permission setup)
	for _, p := range config.(*schema.Set).List() {
		permission := p.(map[string]interface{})
		permissionItem := gapi.ServiceAccountPermissionItem{}
		_, teamIDStr := SplitOrgResourceID(permission["team_id"].(string))
		teamID, _ := strconv.ParseInt(teamIDStr, 10, 64)
		userID := int64(permission["user_id"].(int))
		if teamID > 0 {
			perm, has := oldTeamPerms[teamID]
			if has {
				delete(oldTeamPerms, teamID)
				// Skip permissions that have not been changed
				if perm == permission["permission"].(string) {
					continue
				}
			}
			permissionItem.TeamID = teamID
		} else if userID > 0 {
			perm, has := oldUserPerms[userID]
			if has {
				delete(oldUserPerms, userID)
				if perm == permission["permission"].(string) {
					continue
				}
			}
			permissionItem.UserID = userID
		}
		permissionItem.Permission = permission["permission"].(string)
		permissionList.Permissions = append(permissionList.Permissions, &permissionItem)
	}

	// Remove the permissions that are in the state but not in the config
	for teamID := range oldTeamPerms {
		permissionList.Permissions = append(permissionList.Permissions, &gapi.ServiceAccountPermissionItem{
			TeamID:     teamID,
			Permission: "",
		})
	}
	for userID := range oldUserPerms {
		permissionList.Permissions = append(permissionList.Permissions, &gapi.ServiceAccountPermissionItem{
			UserID:     userID,
			Permission: "",
		})
	}

	err = client.UpdateServiceAccountPermissions(id, &permissionList)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(MakeOrgResourceID(orgID, id))

	return ReadServiceAccountPermissions(ctx, d, meta)
}

func DeleteServiceAccountPermissions(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	orgID, serviceAccountIDStr := SplitOrgResourceID(d.Get("service_account_id").(string))
	client := meta.(*common.Client).GrafanaAPI.WithOrgID(orgID)
	id, err := strconv.ParseInt(serviceAccountIDStr, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	state, _ := d.GetChange("permissions")
	permissionList := gapi.ServiceAccountPermissionItems{}
	for _, p := range state.(*schema.Set).List() {
		perm := p.(map[string]interface{})
		_, teamIDStr := SplitOrgResourceID(perm["team_id"].(string))
		teamID, _ := strconv.ParseInt(teamIDStr, 10, 64)
		userID := int64(perm["user_id"].(int))
		permissionItem := gapi.ServiceAccountPermissionItem{}

		if teamID > 0 {
			permissionItem.TeamID = teamID
		} else if userID > 0 {
			permissionItem.UserID = userID
		}
		permissionItem.Permission = ""
		permissionList.Permissions = append(permissionList.Permissions, &permissionItem)
	}

	return diag.FromErr(client.UpdateServiceAccountPermissions(id, &permissionList))
}
