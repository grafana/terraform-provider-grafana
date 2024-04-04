package grafana

import (
	"context"
	"strconv"

	"github.com/grafana/grafana-openapi-client-go/client/access_control"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v2/internal/common"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	resourceFolderPermissionItemName = "grafana_folder_permission_item"
	resourceFolderPermissionItemID   = common.NewResourceID(common.OptionalIntIDField("orgID"), common.StringIDField("folderUID"), common.StringIDField("type (role, team, or user)"), common.StringIDField("identifier"))

	// Check interface
	_ resource.ResourceWithImportState = (*resourceFolderPermissionItem)(nil)
)

const (
	permissionTargetRole = "role"
	permissionTargetTeam = "team"
	permissionTargetUser = "user"
)

func makeResourceFolderPermisisonItem() *common.Resource {
	return common.NewResource(resourceFolderPermissionItemName, resourceFolderPermissionItemID, &resourceFolderPermissionItem{})
}

type resourceFolderPermissionItemModel struct {
	ID         types.String `tfsdk:"id"`
	OrgID      types.String `tfsdk:"org_id"`
	FolderUID  types.String `tfsdk:"folder_uid"`
	Role       types.String `tfsdk:"role"`
	Team       types.String `tfsdk:"team"`
	User       types.String `tfsdk:"user"`
	Permission types.String `tfsdk:"permission"`
}

type resourceFolderPermissionItem struct {
	basePluginFrameworkResource
}

func (r *resourceFolderPermissionItem) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceFolderPermissionItemName
}

func (r *resourceFolderPermissionItem) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	targetOneOf := stringvalidator.ExactlyOneOf(
		path.MatchRoot(permissionTargetRole),
		path.MatchRoot(permissionTargetTeam),
		path.MatchRoot(permissionTargetUser),
	)

	resp.Schema = schema.Schema{
		MarkdownDescription: `Manages a single permission item for a folder. Conflicts with the "grafana_folder_permission" resource which manages the entire set of permissions for a folder.
		* [Official documentation](https://grafana.com/docs/grafana/latest/administration/roles-and-permissions/access-control/)
		* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/folder_permissions/)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"org_id": pluginFrameworkOrgIDAttribute(),
			"folder_uid": schema.StringAttribute{
				Required:    true,
				Description: "The UID of the folder.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			permissionTargetRole: schema.StringAttribute{
				Optional:    true,
				Description: "the role onto which the permission is to be assigned",
				Validators: []validator.String{
					stringvalidator.OneOf("Editor", "Viewer"),
					targetOneOf,
				},
			},
			permissionTargetTeam: schema.StringAttribute{
				Optional:    true,
				Description: "the team onto which the permission is to be assigned",
				Validators: []validator.String{
					targetOneOf,
				},
				PlanModifiers: []planmodifier.String{
					&orgScopedAttributePlanModifier{},
				},
			},
			permissionTargetUser: schema.StringAttribute{
				Optional:    true,
				Description: "the user onto which the permission is to be assigned",
				Validators: []validator.String{
					targetOneOf,
				},
				PlanModifiers: []planmodifier.String{
					&orgScopedAttributePlanModifier{},
				},
			},
			"permission": schema.StringAttribute{
				Required:    true,
				Description: "the permission to be assigned",
				Validators: []validator.String{
					stringvalidator.OneOf("View", "Edit", "Admin"),
				},
			},
		},
	}
}

func (r *resourceFolderPermissionItem) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	data, diags := r.read(req.ID)
	if diags != nil {
		resp.Diagnostics = diags
		return
	}
	if data == nil {
		resp.Diagnostics.AddError("Resource not found", "Resource not found")
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *resourceFolderPermissionItem) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Read Terraform plan data into the model
	var data resourceFolderPermissionItemModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if diags := r.write(&data); diags != nil {
		resp.Diagnostics = diags
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *resourceFolderPermissionItem) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Read Terraform state data into the model
	var data resourceFolderPermissionItemModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	// Read from API
	readData, diags := r.read(data.ID.ValueString())
	if diags != nil {
		resp.Diagnostics = diags
		return
	}
	if readData == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *resourceFolderPermissionItem) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Read Terraform plan data into the model
	var data resourceFolderPermissionItemModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if diags := r.write(&data); diags != nil {
		resp.Diagnostics = diags
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *resourceFolderPermissionItem) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Read Terraform prior state data into the model
	var data resourceFolderPermissionItemModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	data.Permission = types.StringValue("")

	if diags := r.write(&data); diags != nil {
		resp.Diagnostics = diags
	}
}

func (r *resourceFolderPermissionItem) read(id string) (*resourceFolderPermissionItemModel, diag.Diagnostics) {
	client, orgID, splitID, err := r.clientFromExistingOrgResource(resourceFolderPermissionItemID, id)
	if err != nil {
		return nil, diag.Diagnostics{diag.NewErrorDiagnostic("Unable to parse resource ID", err.Error())}
	}
	folderUID := splitID[0].(string)
	permissionTargetType := splitID[1].(string)
	permissionTargetID := splitID[2].(string)

	// Check that the folder exists
	_, err = client.Folders.GetFolderByUID(folderUID)
	if err != nil {
		if common.IsNotFoundError(err) {
			return nil, nil
		}
		return nil, diag.Diagnostics{diag.NewErrorDiagnostic("Failed to read folder", err.Error())}
	}

	// GET
	permissionsResp, err := client.AccessControl.GetResourcePermissions(folderUID, foldersPermissionsType)
	if err != nil {
		if common.IsNotFoundError(err) {
			return nil, nil
		}
		return nil, diag.Diagnostics{diag.NewErrorDiagnostic("Failed to read permissions", err.Error())}
	}

	for _, permission := range permissionsResp.Payload {
		data := &resourceFolderPermissionItemModel{
			ID:         types.StringValue(id),
			OrgID:      types.StringValue(strconv.FormatInt(orgID, 10)),
			FolderUID:  types.StringValue(folderUID),
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

func (r *resourceFolderPermissionItem) write(data *resourceFolderPermissionItemModel) diag.Diagnostics {
	client, orgID := r.clientFromNewOrgResource(data.OrgID.ValueString())
	folderUID := data.FolderUID.ValueString()
	data.OrgID = types.StringValue(strconv.FormatInt(orgID, 10))

	var err error
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
				WithResource(foldersPermissionsType).
				WithResourceID(folderUID),
		)
		data.ID = types.StringValue(
			resourceFolderPermissionItemID.Make(orgID, data.FolderUID.ValueString(), permissionTargetUser, userIDStr),
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
				WithResource(foldersPermissionsType).
				WithResourceID(folderUID),
		)
		data.ID = types.StringValue(
			resourceFolderPermissionItemID.Make(orgID, data.FolderUID.ValueString(), permissionTargetTeam, teamIDStr),
		)
	case !data.Role.IsNull():
		_, err = client.AccessControl.SetResourcePermissionsForBuiltInRole(
			access_control.NewSetResourcePermissionsForBuiltInRoleParams().
				WithBuiltInRole(data.Role.ValueString()).
				WithBody(&models.SetPermissionCommand{
					Permission: data.Permission.ValueString(),
				}).
				WithResource(foldersPermissionsType).
				WithResourceID(folderUID),
		)
		data.ID = types.StringValue(
			resourceFolderPermissionItemID.Make(orgID, data.FolderUID.ValueString(), permissionTargetRole, data.Role.ValueString()),
		)
	}
	if err != nil {
		return diag.Diagnostics{diag.NewErrorDiagnostic("Failed to write permissions", err.Error())}
	}
	return nil
}
