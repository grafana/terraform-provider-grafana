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
	resourceDatasourcePermissionItemName = "grafana_data_source_permission_item"
	resourceDatasourcePermissionItemID   = common.NewResourceID(common.OptionalIntIDField("orgID"), common.StringIDField("datasourceUID"), common.StringIDField("type (role, team, or user)"), common.StringIDField("identifier"))

	// Check interface
	_ resource.ResourceWithImportState = (*resourceDatasourcePermissionItem)(nil)
)

func makeResourceDatasourcePermissionItem() *common.Resource {
	return common.NewResource(resourceDatasourcePermissionItemName, resourceDatasourcePermissionItemID, &resourceDatasourcePermissionItem{})
}

type resourceDatasourcePermissionItemModel struct {
	ID            types.String `tfsdk:"id"`
	OrgID         types.String `tfsdk:"org_id"`
	DatasourceUID types.String `tfsdk:"datasource_uid"`
	Role          types.String `tfsdk:"role"`
	Team          types.String `tfsdk:"team"`
	User          types.String `tfsdk:"user"`
	Permission    types.String `tfsdk:"permission"`
}

type resourceDatasourcePermissionItem struct {
	basePluginFrameworkResource
}

func (r *resourceDatasourcePermissionItem) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceDatasourcePermissionItemName
}

func (r *resourceDatasourcePermissionItem) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	targetOneOf := stringvalidator.ExactlyOneOf(
		path.MatchRoot(permissionTargetRole),
		path.MatchRoot(permissionTargetTeam),
		path.MatchRoot(permissionTargetUser),
	)

	resp.Schema = schema.Schema{
		MarkdownDescription: `Manages a single permission item for a datasource. Conflicts with the "grafana_data_source_permission" resource which manages the entire set of permissions for a datasource.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"org_id": pluginFrameworkOrgIDAttribute(),
			"datasource_uid": schema.StringAttribute{
				Required:    true,
				Description: "The UID of the datasource.",
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
				Description: "the user or service account onto which the permission is to be assigned",
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
					stringvalidator.OneOf("Query", "Edit", "Admin"),
				},
			},
		},
	}
}

func (r *resourceDatasourcePermissionItem) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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

func (r *resourceDatasourcePermissionItem) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Read Terraform plan data into the model
	var data resourceDatasourcePermissionItemModel
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

func (r *resourceDatasourcePermissionItem) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Read Terraform state data into the model
	var data resourceDatasourcePermissionItemModel
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

func (r *resourceDatasourcePermissionItem) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Read Terraform plan data into the model
	var data resourceDatasourcePermissionItemModel
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

func (r *resourceDatasourcePermissionItem) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Read Terraform prior state data into the model
	var data resourceDatasourcePermissionItemModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	data.Permission = types.StringValue("")

	if diags := r.write(&data); diags != nil {
		resp.Diagnostics = diags
	}
}

func (r *resourceDatasourcePermissionItem) read(id string) (*resourceDatasourcePermissionItemModel, diag.Diagnostics) {
	client, orgID, splitID, err := r.clientFromExistingOrgResource(resourceDatasourcePermissionItemID, id)
	if err != nil {
		return nil, diag.Diagnostics{diag.NewErrorDiagnostic("Unable to parse resource ID", err.Error())}
	}
	datasourceUID := splitID[0].(string)
	permissionTargetType := splitID[1].(string)
	permissionTargetID := splitID[2].(string)

	// Check that the datasource exists
	_, err = client.Datasources.GetDataSourceByUID(datasourceUID)
	if err != nil {
		if common.IsNotFoundError(err) {
			return nil, nil
		}
		return nil, diag.Diagnostics{diag.NewErrorDiagnostic("Failed to read datasource", err.Error())}
	}

	// GET
	permissionsResp, err := client.AccessControl.GetResourcePermissions(datasourceUID, datasourcesPermissionsType)
	if err != nil {
		if common.IsNotFoundError(err) {
			return nil, nil
		}
		return nil, diag.Diagnostics{diag.NewErrorDiagnostic("Failed to read permissions", err.Error())}
	}

	for _, permission := range permissionsResp.Payload {
		data := &resourceDatasourcePermissionItemModel{
			ID:            types.StringValue(id),
			OrgID:         types.StringValue(strconv.FormatInt(orgID, 10)),
			DatasourceUID: types.StringValue(datasourceUID),
			Permission:    types.StringValue(permission.Permission),
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

func (r *resourceDatasourcePermissionItem) write(data *resourceDatasourcePermissionItemModel) diag.Diagnostics {
	client, orgID := r.clientFromNewOrgResource(data.OrgID.ValueString())
	datasourceUID := data.DatasourceUID.ValueString()
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
				WithResource(datasourcesPermissionsType).
				WithResourceID(datasourceUID),
		)
		data.ID = types.StringValue(
			resourceDatasourcePermissionItemID.Make(orgID, data.DatasourceUID.ValueString(), permissionTargetUser, userIDStr),
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
				WithResource(datasourcesPermissionsType).
				WithResourceID(datasourceUID),
		)
		data.ID = types.StringValue(
			resourceDatasourcePermissionItemID.Make(orgID, data.DatasourceUID.ValueString(), permissionTargetTeam, teamIDStr),
		)
	case !data.Role.IsNull():
		_, err = client.AccessControl.SetResourcePermissionsForBuiltInRole(
			access_control.NewSetResourcePermissionsForBuiltInRoleParams().
				WithBuiltInRole(data.Role.ValueString()).
				WithBody(&models.SetPermissionCommand{
					Permission: data.Permission.ValueString(),
				}).
				WithResource(datasourcesPermissionsType).
				WithResourceID(datasourceUID),
		)
		data.ID = types.StringValue(
			resourceDatasourcePermissionItemID.Make(orgID, data.DatasourceUID.ValueString(), permissionTargetRole, data.Role.ValueString()),
		)
	}
	if err != nil {
		return diag.Diagnostics{diag.NewErrorDiagnostic("Failed to write permissions", err.Error())}
	}
	return nil
}
