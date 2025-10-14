// This is common code for the folder/dashboards/datasources/service accounts permissions resources.
// They all use the same API for setting permissions, so the code is shared.

package grafana

import (
	"strconv"

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

func (r *resourcePermissionBase) readItem(id string, checkExistsFunc func(client *client.GrafanaHTTPAPI, itemID string) error) (*resourcePermissionItemBaseModel, diag.Diagnostics) {
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
	permissionsResp, err := client.AccessControl.GetResourcePermissions(itemID, r.resourceType)
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

func (r *resourcePermissionBase) writeItem(itemID string, data *resourcePermissionItemBaseModel) diag.Diagnostics {
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
