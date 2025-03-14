package grafana

import (
	"context"
	"strconv"
	"sync"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
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
	resourceRoleAssignmentItemName = "grafana_role_assignment_item"
	resourceRoleAssignmentItemID   = common.NewResourceID(common.OptionalIntIDField("orgID"), common.StringIDField("roleUID"), common.StringIDField("type (user, team or service_account)"), common.StringIDField("identifier"))
	resourceRoleAssignmentMutex    sync.RWMutex

	// Check interface
	_ resource.ResourceWithImportState = (*resourceRoleAssignmentItem)(nil)
)

func makeResourceRoleAssignmentItem() *common.Resource {
	return common.NewResource(
		common.CategoryGrafanaEnterprise,
		resourceRoleAssignmentItemName,
		resourceRoleAssignmentItemID,
		&resourceRoleAssignmentItem{},
	)
}

type resourceRoleAssignmentItemModel struct {
	ID               types.String `tfsdk:"id"`
	OrgID            types.String `tfsdk:"org_id"`
	RoleUID          types.String `tfsdk:"role_uid"`
	TeamID           types.String `tfsdk:"team_id"`
	UserID           types.String `tfsdk:"user_id"`
	ServiceAccountID types.String `tfsdk:"service_account_id"`
}

type resourceRoleAssignmentItem struct {
	basePluginFrameworkResource
}

func (r *resourceRoleAssignmentItem) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceRoleAssignmentItemName
}

func (r *resourceRoleAssignmentItem) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	targetOneOf := stringvalidator.ExactlyOneOf(
		path.MatchRoot("team_id"),
		path.MatchRoot("user_id"),
		path.MatchRoot("service_account_id"),
	)

	resp.Schema = schema.Schema{
		MarkdownDescription: `Manages a single assignment for a role. Conflicts with the "grafana_role_assignment" resource which manages the entire set of assignments for a role.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"org_id": pluginFrameworkOrgIDAttribute(),
			"role_uid": schema.StringAttribute{
				Required:    true,
				Description: "the role UID onto which to assign an actor",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"team_id": schema.StringAttribute{
				Optional:    true,
				Description: "the team onto which the role is to be assigned",
				Validators: []validator.String{
					targetOneOf,
				},
				PlanModifiers: []planmodifier.String{
					&orgScopedAttributePlanModifier{},
					stringplanmodifier.RequiresReplace(),
				},
			},
			"user_id": schema.StringAttribute{
				Optional:    true,
				Description: "the user onto which the role is to be assigned",
				Validators: []validator.String{
					targetOneOf,
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"service_account_id": schema.StringAttribute{
				Optional:    true,
				Description: "the service account onto which the role is to be assigned",
				Validators: []validator.String{
					targetOneOf,
				},
				PlanModifiers: []planmodifier.String{
					&orgScopedAttributePlanModifier{},
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *resourceRoleAssignmentItem) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resourceRoleAssignmentMutex.RLock()
	defer resourceRoleAssignmentMutex.RUnlock()
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

func (r *resourceRoleAssignmentItem) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Read Terraform plan data into the model
	var data resourceRoleAssignmentItemModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, orgID, err := r.clientFromNewOrgResource(data.OrgID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get client", err.Error())
		return
	}

	// Get existing role assignments
	resourceRoleAssignmentMutex.Lock()
	defer resourceRoleAssignmentMutex.Unlock()
	getResp, err := client.AccessControl.GetRoleAssignments(data.RoleUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get role assignments", err.Error())
		return
	}
	roleAssignments := getResp.Payload

	assignmentType := ""
	resourceID := ""
	switch {
	case !data.TeamID.IsNull():
		_, teamIDStr := SplitOrgResourceID(data.TeamID.ValueString())
		teamID, err := strconv.ParseInt(teamIDStr, 10, 64)
		if err != nil {
			resp.Diagnostics.AddError("Failed to parse team ID", err.Error())
			return
		}
		roleAssignments.Teams = append(roleAssignments.Teams, teamID)
		assignmentType = "team"
		resourceID = teamIDStr
	case !data.UserID.IsNull():
		userID, err := strconv.ParseInt(data.UserID.ValueString(), 10, 64)
		if err != nil {
			resp.Diagnostics.AddError("Failed to parse user ID", err.Error())
			return
		}
		roleAssignments.Users = append(roleAssignments.Users, userID)
		assignmentType = "user"
		resourceID = data.UserID.ValueString()
	case !data.ServiceAccountID.IsNull():
		_, serviceAccountIDStr := SplitServiceAccountID(data.ServiceAccountID.ValueString())
		serviceAccountID, err := strconv.ParseInt(serviceAccountIDStr, 10, 64)
		if err != nil {
			resp.Diagnostics.AddError("Failed to parse service account ID", err.Error())
			return
		}
		roleAssignments.ServiceAccounts = append(roleAssignments.ServiceAccounts, serviceAccountID)
		assignmentType = "service_account"
		resourceID = serviceAccountIDStr
	}

	_, err = client.AccessControl.SetRoleAssignments(data.RoleUID.ValueString(), &models.SetRoleAssignmentsCommand{
		Teams:           roleAssignments.Teams,
		Users:           roleAssignments.Users,
		ServiceAccounts: roleAssignments.ServiceAccounts,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to set role assignments", err.Error())
		return
	}

	// Save data into Terraform state
	data.ID = types.StringValue(resourceRoleAssignmentItemID.Make(orgID, data.RoleUID.ValueString(), assignmentType, resourceID))
	data.OrgID = types.StringValue(strconv.FormatInt(orgID, 10))
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *resourceRoleAssignmentItem) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Read Terraform state data into the model
	var data resourceRoleAssignmentItemModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	// Read from API
	resourceRoleAssignmentMutex.RLock()
	defer resourceRoleAssignmentMutex.RUnlock()
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

func (r *resourceRoleAssignmentItem) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Update shouldn't happen as all attributes require replacement
	resp.Diagnostics.AddError("Update not supported", "Update not supported")
}

func (r *resourceRoleAssignmentItem) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Read Terraform prior state data into the model
	var data resourceRoleAssignmentItemModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	client, _, idFields, err := r.clientFromExistingOrgResource(resourceRoleAssignmentItemID, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get client", err.Error())
		return
	}
	roleUID, assignmentType, identifier := idFields[0].(string), idFields[1].(string), idFields[2].(string)

	// Get existing role assignments
	resourceRoleAssignmentMutex.Lock()
	defer resourceRoleAssignmentMutex.Unlock()
	getResp, err := client.AccessControl.GetRoleAssignments(roleUID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get role assignments", err.Error())
		return
	}
	roleAssignments := getResp.Payload

	// Delete from API
	switch assignmentType {
	case "team":
		for i, team := range roleAssignments.Teams {
			if strconv.FormatInt(team, 10) == identifier {
				roleAssignments.Teams = append(roleAssignments.Teams[:i], roleAssignments.Teams[i+1:]...)
				break
			}
		}
	case "user":
		for i, user := range roleAssignments.Users {
			if strconv.FormatInt(user, 10) == identifier {
				roleAssignments.Users = append(roleAssignments.Users[:i], roleAssignments.Users[i+1:]...)
				break
			}
		}
	case "service_account":
		for i, serviceAccount := range roleAssignments.ServiceAccounts {
			if strconv.FormatInt(serviceAccount, 10) == identifier {
				roleAssignments.ServiceAccounts = append(roleAssignments.ServiceAccounts[:i], roleAssignments.ServiceAccounts[i+1:]...)
				break
			}
		}
	default:
		// Should never happen due to the schema validation, but include for completeness
		resp.Diagnostics.AddError("Invalid assignment type", assignmentType)
		return
	}

	_, err = client.AccessControl.SetRoleAssignments(roleUID, &models.SetRoleAssignmentsCommand{
		Teams:           roleAssignments.Teams,
		Users:           roleAssignments.Users,
		ServiceAccounts: roleAssignments.ServiceAccounts,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to set role assignments", err.Error())
		return
	}
}

func (r *resourceRoleAssignmentItem) read(id string) (*resourceRoleAssignmentItemModel, diag.Diagnostics) {
	client, orgID, idFields, err := r.clientFromExistingOrgResource(resourceRoleAssignmentItemID, id)
	if err != nil {
		return nil, diag.Diagnostics{diag.NewErrorDiagnostic("Failed to get client", err.Error())}
	}
	roleUID, assignmentType, identifier := idFields[0].(string), idFields[1].(string), idFields[2].(string)

	// Try to get the role
	_, err = client.AccessControl.GetRole(roleUID)
	if err != nil {
		if common.IsNotFoundError(err) {
			return nil, nil
		}
		return nil, diag.Diagnostics{diag.NewErrorDiagnostic("Failed to get role", err.Error())}
	}

	// Get existing role assignments
	getResp, err := client.AccessControl.GetRoleAssignments(roleUID)
	if err != nil {
		return nil, diag.Diagnostics{diag.NewErrorDiagnostic("Failed to get role assignments", err.Error())}
	}

	// Find the assignment
	data := &resourceRoleAssignmentItemModel{
		ID:      types.StringValue(id),
		OrgID:   types.StringValue(strconv.FormatInt(orgID, 10)),
		RoleUID: types.StringValue(roleUID),
	}
	switch assignmentType {
	case "team":
		for _, team := range getResp.Payload.Teams {
			if teamIDStr := strconv.FormatInt(team, 10); teamIDStr == identifier {
				data.TeamID = types.StringValue(teamIDStr)
				break
			}
		}
	case "user":
		for _, user := range getResp.Payload.Users {
			if userIDStr := strconv.FormatInt(user, 10); userIDStr == identifier {
				data.UserID = types.StringValue(userIDStr)
				break
			}
		}
	case "service_account":
		for _, serviceAccount := range getResp.Payload.ServiceAccounts {
			if saIDStr := strconv.FormatInt(serviceAccount, 10); saIDStr == identifier {
				data.ServiceAccountID = types.StringValue(saIDStr)
				break
			}
		}
	default:
		// Should never happen due to the schema validation, but include for completeness
		return nil, diag.Diagnostics{diag.NewErrorDiagnostic("Invalid assignment type", assignmentType)}
	}

	if data.TeamID.IsNull() && data.UserID.IsNull() && data.ServiceAccountID.IsNull() {
		return nil, nil
	}

	return data, nil
}
