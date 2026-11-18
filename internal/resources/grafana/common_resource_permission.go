// This is common code for the folder/dashboards/datasources/service accounts permissions resources.
// They all use the same API for setting permissions, so the code is shared.

package grafana

import (
	"strconv"
	"strings"

	"github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/client/access_control"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	permissionTargetRole = "role"
	permissionTargetTeam = "team"
	permissionTargetUser = "user"

	dashboardsPermissionsType      = "dashboards"
	datasourcesPermissionsType     = "datasources"
	foldersPermissionsType         = "folders"
	serviceAccountsPermissionsType = "serviceaccounts"
)

// ---- _item resource helpers (single permission entry) ----

type resourcePermissionItemBaseModel struct {
	ID         types.String `tfsdk:"id"`
	OrgID      types.String `tfsdk:"org_id"`
	Role       types.String `tfsdk:"role"`
	Team       types.String `tfsdk:"team"`
	User       types.String `tfsdk:"user"`
	Permission types.String `tfsdk:"permission"`

	// Framework doesn't support embedding a base struct: https://github.com/hashicorp/terraform-plugin-framework/issues/242
	// So this is a generic ID to be written to FolderUID/DatasourceUID/etc
	ResourceID types.String `tfsdk:"-"`
}

type resourcePermissionBase struct {
	basePluginFrameworkResource
	resourceType string
}

func (r *resourcePermissionBase) addInSchemaAttributes(attributes map[string]schema.Attribute) map[string]schema.Attribute {
	targetOneOf := stringvalidator.ExactlyOneOf(
		path.MatchRoot(permissionTargetRole),
		path.MatchRoot(permissionTargetTeam),
		path.MatchRoot(permissionTargetUser),
	)

	attributes["id"] = schema.StringAttribute{
		Computed: true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	}
	attributes["org_id"] = pluginFrameworkOrgIDAttribute()
	attributes[permissionTargetRole] = schema.StringAttribute{
		Optional:    true,
		Description: "the role onto which the permission is to be assigned",
		Validators: []validator.String{
			stringvalidator.OneOf("Editor", "Viewer"),
			targetOneOf,
		},
	}
	attributes[permissionTargetTeam] = schema.StringAttribute{
		Optional:    true,
		Description: "the team onto which the permission is to be assigned",
		Validators: []validator.String{
			targetOneOf,
		},
		PlanModifiers: []planmodifier.String{
			&orgScopedAttributePlanModifier{},
		},
	}
	attributes[permissionTargetUser] = schema.StringAttribute{
		Optional:    true,
		Description: "the user or service account onto which the permission is to be assigned",
		Validators: []validator.String{
			targetOneOf,
		},
		PlanModifiers: []planmodifier.String{
			&orgScopedAttributePlanModifier{},
		},
	}
	attributes["permission"] = schema.StringAttribute{
		Required:    true,
		Description: "the permission to be assigned",
		Validators: []validator.String{
			stringvalidator.OneOf("Query", "View", "Edit", "Admin"),
		},
	}
	return attributes
}

func (r *resourcePermissionBase) readItem(id string, checkExistsFunc func(client *client.GrafanaHTTPAPI, itemID string) error, getClientOpts []access_control.ClientOption) (*resourcePermissionItemBaseModel, diag.Diagnostics) {
	client, orgID, splitID, err := r.clientFromExistingOrgResource(resourceFolderPermissionItemID, id)
	if err != nil {
		return nil, diag.Diagnostics{diag.NewErrorDiagnostic("Unable to parse resource ID", err.Error())}
	}
	itemID := splitID[0].(string)
	permissionTargetType := splitID[1].(string)
	permissionTargetID := splitID[2].(string)

	// Check that the resource exists. This depends on the resource type, so a generic function is passed in.
	if err := checkExistsFunc(client, itemID); err != nil {
		if common.IsNotFoundError(err) {
			return nil, nil
		}
		return nil, diag.Diagnostics{diag.NewErrorDiagnostic("Resource does not exist", err.Error())}
	}

	// GET
	permissionsResp, err := client.AccessControl.GetResourcePermissions(itemID, r.resourceType, getClientOpts...)
	if err != nil {
		if common.IsNotFoundError(err) {
			return nil, nil
		}
		return nil, diag.Diagnostics{diag.NewErrorDiagnostic("Failed to read permissions", err.Error())}
	}

	for _, permission := range permissionsResp.Payload {
		data := &resourcePermissionItemBaseModel{
			ResourceID: types.StringValue(itemID),
			ID:         types.StringValue(id),
			OrgID:      types.StringValue(strconv.FormatInt(orgID, 10)),
			Permission: types.StringValue(permission.Permission),
		}
		switch permissionTargetType {
		case permissionTargetTeam:
			if v := strconv.FormatInt(permission.TeamID, 10); v == permissionTargetID {
				data.Team = types.StringValue(v)
			} else {
				continue
			}
		case permissionTargetUser:
			if v := strconv.FormatInt(permission.UserID, 10); v == permissionTargetID {
				data.User = types.StringValue(v)
			} else {
				continue
			}
		case permissionTargetRole:
			if permission.BuiltInRole == permissionTargetID {
				data.Role = types.StringValue(permissionTargetID)
			} else {
				continue
			}
		default:
			return nil, diag.Diagnostics{diag.NewErrorDiagnostic("Unknown permission target type", permissionTargetType)}
		}

		return data, nil
	}

	return nil, nil
}

func (r *resourcePermissionBase) writeItem(itemID string, data *resourcePermissionItemBaseModel, extraOpts ...access_control.ClientOption) diag.Diagnostics {
	client, orgID, err := r.clientFromNewOrgResource(data.OrgID.ValueString())
	if err != nil {
		return diag.Diagnostics{diag.NewErrorDiagnostic("Failed to get client", err.Error())}
	}

	data.OrgID = types.StringValue(strconv.FormatInt(orgID, 10))
	if r.resourceType == serviceAccountsPermissionsType {
		_, itemID = SplitServiceAccountID(itemID)
	} else {
		_, itemID = SplitOrgResourceID(itemID)
	}
	data.ResourceID = types.StringValue(itemID)

	switch {
	case !data.User.IsNull():
		_, userIDStr := SplitOrgResourceID(data.User.ValueString())
		userID, parseErr := strconv.ParseInt(userIDStr, 10, 64)
		if parseErr != nil {
			return diag.Diagnostics{diag.NewErrorDiagnostic("Failed to parse user ID", parseErr.Error())}
		}
		_, err = client.AccessControl.SetResourcePermissionsForUser(
			access_control.NewSetResourcePermissionsForUserParams().
				WithUserID(userID).
				WithBody(&models.SetPermissionCommand{
					Permission: data.Permission.ValueString(),
				}).
				WithResource(r.resourceType).
				WithResourceID(itemID),
			extraOpts...,
		)
		data.ID = types.StringValue(
			resourceFolderPermissionItemID.Make(orgID, itemID, permissionTargetUser, userIDStr),
		)
	case !data.Team.IsNull():
		_, teamIDStr := SplitOrgResourceID(data.Team.ValueString())
		teamID, parseErr := strconv.ParseInt(teamIDStr, 10, 64)
		if parseErr != nil {
			return diag.Diagnostics{diag.NewErrorDiagnostic("Failed to parse user ID", parseErr.Error())}
		}
		_, err = client.AccessControl.SetResourcePermissionsForTeam(
			access_control.NewSetResourcePermissionsForTeamParams().
				WithTeamID(teamID).
				WithBody(&models.SetPermissionCommand{
					Permission: data.Permission.ValueString(),
				}).
				WithResource(r.resourceType).
				WithResourceID(itemID),
			extraOpts...,
		)
		data.ID = types.StringValue(
			resourceFolderPermissionItemID.Make(orgID, itemID, permissionTargetTeam, teamIDStr),
		)
	case !data.Role.IsNull():
		_, err = client.AccessControl.SetResourcePermissionsForBuiltInRole(
			access_control.NewSetResourcePermissionsForBuiltInRoleParams().
				WithBuiltInRole(data.Role.ValueString()).
				WithBody(&models.SetPermissionCommand{
					Permission: data.Permission.ValueString(),
				}).
				WithResource(r.resourceType).
				WithResourceID(itemID),
			extraOpts...,
		)
		data.ID = types.StringValue(
			resourceFolderPermissionItemID.Make(orgID, itemID, permissionTargetRole, data.Role.ValueString()),
		)
	}
	if err != nil {
		return diag.Diagnostics{diag.NewErrorDiagnostic("Failed to write permissions", err.Error())}
	}
	return nil
}

// ---- Bulk permission resource helpers (full set of permissions) ----

// bulkPermissionItemModel represents a single permission entry in a bulk permissions set.
type bulkPermissionItemModel struct {
	Role       types.String `tfsdk:"role"`
	TeamID     types.String `tfsdk:"team_id"`
	UserID     types.String `tfsdk:"user_id"`
	Permission types.String `tfsdk:"permission"`
}

// resourcePermissionBulkBase is embedded by bulk permission resources (grafana_folder_permission, etc.)
type resourcePermissionBulkBase struct {
	basePluginFrameworkResource
	resourceType string
}

// bulkPermissionsSchemaAttribute returns the schema block for the permissions set.
// permissionValues are the valid values for the "permission" field (e.g. "View", "Edit", "Admin").
func bulkPermissionsSchemaAttribute(description string, permissionValues []string) schema.Block {
	return schema.SetNestedBlock{
		Description: description,
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"role": schema.StringAttribute{
					Optional:    true,
					Description: "Name of the basic role to manage permissions for. Options: `Viewer`, `Editor` or `Admin`.",
					Validators: []validator.String{
						stringvalidator.OneOf("Viewer", "Editor", "Admin"),
					},
				},
				"team_id": schema.StringAttribute{
					Optional:    true,
					Description: "ID of the team to manage permissions for.",
					PlanModifiers: []planmodifier.String{
						&stripOrgScopedIDPlanModifier{},
					},
				},
				"user_id": schema.StringAttribute{
					Optional:    true,
					Description: "ID of the user or service account to manage permissions for.",
					PlanModifiers: []planmodifier.String{
						&stripOrgScopedIDPlanModifier{},
					},
				},
				"permission": schema.StringAttribute{
					Required:    true,
					Description: "Permission to associate with item. Options: " + strings.Join(permissionValues, ", ") + ".",
					Validators: []validator.String{
						stringvalidator.OneOf(permissionValues...),
					},
				},
			},
		},
	}
}

// readBulkPermissions fetches the current permissions from the API.
// team_id and user_id are returned as plain local IDs (no org prefix); the
// stripOrgScopedIDPlanModifier ensures plan values are normalized to the same format.
func (r *resourcePermissionBulkBase) readBulkPermissions(client *client.GrafanaHTTPAPI, resourceUID string) ([]bulkPermissionItemModel, diag.Diagnostics) {
	resp, err := client.AccessControl.GetResourcePermissions(resourceUID, r.resourceType)
	if err != nil {
		return nil, diag.Diagnostics{diag.NewErrorDiagnostic("Failed to read permissions", err.Error())}
	}

	// Filter out the acting identity's auto-attached Admin grant that the
	// server adds when this identity creates a resource. On Grafana Cloud,
	// whichever user or service account runs `terraform apply` is
	// auto-granted Admin on any folder/dashboard/etc it creates. That entry
	// is server-managed, not user-declared, so if we don't filter it out
	// the readback contains one more entry than the plan declared,
	// producing "Provider produced inconsistent result after apply".
	actingUserID := getActingUserID(client)

	var items []bulkPermissionItemModel
	for _, perm := range resp.Payload {
		if !perm.IsManaged || perm.IsInherited {
			continue
		}
		// Skip the acting identity's auto-Admin grant. Users who genuinely
		// want to manage this SA/user via TF can use the _item resource
		// (grafana_folder_permission_item et al.), which does not go
		// through this bulk readback path.
		if actingUserID > 0 && perm.UserID == actingUserID {
			continue
		}
		item := bulkPermissionItemModel{
			Permission: types.StringValue(perm.Permission),
		}
		if perm.BuiltInRole != "" {
			item.Role = types.StringValue(perm.BuiltInRole)
		} else {
			item.Role = types.StringNull()
		}
		if perm.TeamID > 0 {
			item.TeamID = types.StringValue(strconv.FormatInt(perm.TeamID, 10))
		} else {
			item.TeamID = types.StringNull()
		}
		if perm.UserID > 0 {
			item.UserID = types.StringValue(strconv.FormatInt(perm.UserID, 10))
		} else {
			item.UserID = types.StringNull()
		}
		items = append(items, item)
	}

	return items, nil
}

// applyBulkPermissions converts the permissions slice to API commands and applies them.
func (r *resourcePermissionBulkBase) applyBulkPermissions(client *client.GrafanaHTTPAPI, resourceUID string, permissions []bulkPermissionItemModel) diag.Diagnostics {
	var permissionList []*models.SetResourcePermissionCommand
	for _, item := range permissions {
		cmd := &models.SetResourcePermissionCommand{
			Permission: item.Permission.ValueString(),
		}
		if !item.Role.IsNull() && item.Role.ValueString() != "" {
			cmd.BuiltInRole = item.Role.ValueString()
		}
		if !item.TeamID.IsNull() && item.TeamID.ValueString() != "" {
			_, teamIDStr := SplitOrgResourceID(item.TeamID.ValueString())
			teamID, _ := strconv.ParseInt(teamIDStr, 10, 64)
			if teamID > 0 {
				cmd.TeamID = teamID
			}
		}
		if !item.UserID.IsNull() && item.UserID.ValueString() != "" {
			_, userIDStr := SplitOrgResourceID(item.UserID.ValueString())
			userID, _ := strconv.ParseInt(userIDStr, 10, 64)
			if userID > 0 {
				cmd.UserID = userID
			}
		}
		permissionList = append(permissionList, cmd)
	}

	if err := setResourcePermissions(client, resourceUID, r.resourceType, permissionList, nil, nil); err != nil {
		return diag.Diagnostics{diag.NewErrorDiagnostic("Failed to update permissions", err.Error())}
	}
	return nil
}

// setResourcePermissions ensures the live permission set on a resource
// matches the desired set exactly. It uses a two-phase approach:
//
//  1. Bulk POST /api/access-control/{resourceType}/{uid} with the full
//     desired set. This upserts everything the user declared.
//  2. GET the current state, and for any managed permission on the server
//     that is NOT in the desired set (and is not the acting identity's
//     server-attached auto-Admin), issue a per-target DELETE via the
//     SetResourcePermissionsFor{User,Team,BuiltInRole} endpoints with
//     Permission: "".
//
// Why two phases?
//
// The bulk endpoint's behavior on Grafana Cloud with the App Platform
// ResourcePermission backend is neither "incremental" (as the previous
// implementation assumed) nor cleanly "full-replace":
//
//   - Posting a partial payload with Permission:"" delete-commands wipes
//     the entire managed set (observed empirically on a Grafana Cloud
//     stack with the App Platform ResourcePermission backend active).
//   - Posting a partial payload of upserts *keeps* items not in the payload
//     (they neither get replaced nor removed).
//
// So bulk POST alone cannot remove drift or user-initiated deletions. The
// per-target endpoints (SetResourcePermissionsForUser etc.) do have reliable
// single-item semantics on both backends, so we use them for the delete
// phase. This matches the resource's documented contract of managing the
// entire declared set.
//
// The getListOpts parameter is used for phase 2's GET; setOpts is used for
// both the bulk POST and per-target DELETEs.
func setResourcePermissions(client *client.GrafanaHTTPAPI, uid string, resourceType string, desired []*models.SetResourcePermissionCommand, getListOpts, setOpts []access_control.ClientOption) error {
	// Phase 1: bulk upsert the desired set.
	body := models.SetPermissionsCommand{Permissions: desired}
	params := access_control.NewSetResourcePermissionsParams().
		WithResource(resourceType).
		WithResourceID(uid).
		WithBody(&body)
	if _, err := client.AccessControl.SetResourcePermissions(params, setOpts...); err != nil {
		return err
	}

	// Phase 2: remove drift. Fetch current state, find managed items not in
	// desired, and issue per-target deletes for each.
	actingUserID := getActingUserID(client)

	listResp, err := client.AccessControl.GetResourcePermissions(uid, resourceType, getListOpts...)
	if err != nil {
		return err
	}

	matchesDesired := func(current *models.ResourcePermissionDTO) bool {
		for _, d := range desired {
			if current.Permission == d.Permission &&
				current.TeamID == d.TeamID &&
				current.UserID == d.UserID &&
				current.BuiltInRole == d.BuiltInRole {
				return true
			}
		}
		return false
	}

	for _, current := range listResp.Payload {
		if !current.IsManaged || current.IsInherited {
			continue
		}
		// Don't try to delete the acting identity's server-attached
		// auto-Admin grant. It is not user-declared, and deleting it can
		// itself trigger side-effects (the server may re-add it or reject).
		if actingUserID > 0 && current.UserID == actingUserID {
			continue
		}
		if matchesDesired(current) {
			continue
		}

		var perTargetErr error
		switch {
		case current.UserID > 0:
			_, perTargetErr = client.AccessControl.SetResourcePermissionsForUser(
				access_control.NewSetResourcePermissionsForUserParams().
					WithUserID(current.UserID).
					WithResource(resourceType).
					WithResourceID(uid).
					WithBody(&models.SetPermissionCommand{Permission: ""}),
				setOpts...,
			)
		case current.TeamID > 0:
			_, perTargetErr = client.AccessControl.SetResourcePermissionsForTeam(
				access_control.NewSetResourcePermissionsForTeamParams().
					WithTeamID(current.TeamID).
					WithResource(resourceType).
					WithResourceID(uid).
					WithBody(&models.SetPermissionCommand{Permission: ""}),
				setOpts...,
			)
		case current.BuiltInRole != "":
			_, perTargetErr = client.AccessControl.SetResourcePermissionsForBuiltInRole(
				access_control.NewSetResourcePermissionsForBuiltInRoleParams().
					WithBuiltInRole(current.BuiltInRole).
					WithResource(resourceType).
					WithResourceID(uid).
					WithBody(&models.SetPermissionCommand{Permission: ""}),
				setOpts...,
			)
		}
		if perTargetErr != nil {
			return perTargetErr
		}
	}

	return nil
}

// getActingUserID returns the numeric user ID of the identity making the
// request. See the comment inside readBulkPermissions for background on
// why we care (and on the different response shapes for user vs. SA tokens).
func getActingUserID(client *client.GrafanaHTTPAPI) int64 {
	me, err := client.SignedInUser.GetSignedInUser()
	if err != nil || me == nil || me.Payload == nil {
		return 0
	}
	if me.Payload.ID > 0 {
		return me.Payload.ID
	}
	if strings.HasPrefix(me.Payload.UID, "service-account:") {
		if id, parseErr := strconv.ParseInt(strings.TrimPrefix(me.Payload.UID, "service-account:"), 10, 64); parseErr == nil {
			return id
		}
	}
	return 0
}
