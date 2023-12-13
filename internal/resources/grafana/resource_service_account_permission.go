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
Manages the entire set of permissions for a service account. Permissions that aren't specified when applying this resource will be removed.

**Note:** This resource is available from Grafana 9.2.4 onwards.

* [Official documentation](https://grafana.com/docs/grafana/latest/administration/service-accounts/#manage-users-and-teams-permissions-for-a-service-account-in-grafana)`,

		CreateContext: CreateServiceAccountPermissions,
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
				Type:     schema.TypeSet,
				Optional: true,
				DefaultFunc: func() (interface{}, error) {
					return []interface{}{}, nil
				},
				Description: "The permission items to add/update. Items that are omitted from the list will be removed.",
				// Ignore the org ID of the team when hashing. It works with or without it.
				Set: func(i interface{}) int {
					m := i.(map[string]interface{})
					_, teamID := SplitOrgResourceID(m["team_id"].(string))
					_, userID := SplitOrgResourceID((m["user_id"].(string)))
					return schema.HashString(teamID + userID + m["permission"].(string))
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
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "0",
							Description: "ID of the user or service account to manage permissions for. Specify either this or `team_id`.",
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
	saPerms, diags := getServiceAccountPermissions(ctx, d, meta)
	d.Set("permissions", saPerms)
	return diags
}

func CreateServiceAccountPermissions(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID := DeprecatedClientFromNewOrgResource(meta, d)
	_, idStr := SplitOrgResourceID(d.Get("service_account_id").(string))
	d.SetId(MakeOrgResourceID(orgID, idStr))

	// On creation, the service account permissions are unknown, we need to start by reading them.
	currentPerms, diags := getServiceAccountPermissions(ctx, d, meta)
	if diags.HasError() {
		return diags
	}
	err := updateServiceAccountPermissions(client, idStr, currentPerms, d.Get("permissions"))
	if err != nil {
		return diag.FromErr(err)
	}

	return ReadServiceAccountPermissions(ctx, d, meta)
}

func UpdateServiceAccountPermissions(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, idStr := DeprecatedClientFromExistingOrgResource(meta, d.Id())

	old, new := d.GetChange("permissions")
	err := updateServiceAccountPermissions(client, idStr, old, new)
	if err != nil {
		return diag.FromErr(err)
	}

	return ReadServiceAccountPermissions(ctx, d, meta)
}

func DeleteServiceAccountPermissions(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	oAPIClient, _, _ := OAPIClientFromExistingOrgResource(meta, d.Id())

	_, serviceAccountID := SplitOrgResourceID(d.Get("service_account_id").(string))
	id, err := strconv.ParseInt(serviceAccountID, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = oAPIClient.ServiceAccounts.RetrieveServiceAccount(id)
	if diags, shouldReturn := common.CheckReadError("service account permissions", d, err); shouldReturn {
		return diags
	}

	client, _, idStr := DeprecatedClientFromExistingOrgResource(meta, d.Id())
	return diag.FromErr(updateServiceAccountPermissions(client, idStr, d.Get("permissions"), nil))
}

func getServiceAccountPermissions(ctx context.Context, d *schema.ResourceData, meta interface{}) (interface{}, diag.Diagnostics) {
	client, _, idStr := DeprecatedClientFromExistingOrgResource(meta, d.Id())
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	saPermissions, err := client.ListServiceAccountResourcePermissions(id)
	if err, shouldReturn := common.CheckReadError("service account permissions", d, err); shouldReturn {
		return nil, err
	}

	saPerms := make([]interface{}, 0)
	for _, p := range saPermissions {
		// Only managed service account permissions can be provisioned through this resource.
		if !p.IsManaged {
			continue
		}
		permMap := map[string]interface{}{
			"team_id":    strconv.FormatInt(p.TeamID, 10),
			"user_id":    strconv.FormatInt(p.UserID, 10),
			"permission": p.Permission,
		}
		saPerms = append(saPerms, permMap)
	}

	return saPerms, nil
}

func updateServiceAccountPermissions(client *gapi.Client, idStr string, from, to interface{}) error {
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return err
	}

	oldTeamPerms := make(map[int64]string, 0)
	oldUserPerms := make(map[int64]string, 0)
	for _, p := range listOrSet(from) {
		perm := p.(map[string]interface{})
		_, teamIDStr := SplitOrgResourceID(perm["team_id"].(string))
		teamID, _ := strconv.ParseInt(teamIDStr, 10, 64)
		_, userIDStr := SplitOrgResourceID(perm["user_id"].(string))
		userID, _ := strconv.ParseInt(userIDStr, 10, 64)
		if teamID > 0 {
			oldTeamPerms[teamID] = perm["permission"].(string)
		}
		if userID > 0 {
			oldUserPerms[userID] = perm["permission"].(string)
		}
	}

	var permissionList []gapi.SetResourcePermissionItem

	// Iterate over permissions from the configuration (the desired permission setup)
	for _, p := range listOrSet(to) {
		permission := p.(map[string]interface{})
		permissionItem := gapi.SetResourcePermissionItem{}
		_, teamIDStr := SplitOrgResourceID(permission["team_id"].(string))
		teamID, _ := strconv.ParseInt(teamIDStr, 10, 64)
		_, userIDStr := SplitOrgResourceID(permission["user_id"].(string))
		userID, _ := strconv.ParseInt(userIDStr, 10, 64)
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
		permissionList = append(permissionList, permissionItem)
	}

	// Remove the permissions that are in the state but not in the config
	for teamID := range oldTeamPerms {
		permissionList = append(permissionList, gapi.SetResourcePermissionItem{
			TeamID:     teamID,
			Permission: "",
		})
	}
	for userID := range oldUserPerms {
		permissionList = append(permissionList, gapi.SetResourcePermissionItem{
			UserID:     userID,
			Permission: "",
		})
	}

	_, err = client.SetServiceAccountResourcePermissions(id, gapi.SetResourcePermissionsBody{Permissions: permissionList})
	return err
}

func listOrSet(v interface{}) []interface{} {
	if v == nil {
		return make([]interface{}, 0)
	}
	if v, ok := v.(*schema.Set); ok {
		return v.List()
	}
	return v.([]interface{})
}
