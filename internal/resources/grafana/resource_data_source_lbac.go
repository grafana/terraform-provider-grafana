package grafana

import (
	"context"
	"fmt"
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
	resourceLBACRuleItemName = "grafana_data_source_lbac_rule"
	resourceLBACRuleID       = common.NewResourceID(common.OptionalIntIDField("orgID"), common.StringIDField("datasourceUID"), common.StringIDField("type (team)"), common.StringIDField("identifier"))
	resourceLBACRuleMutex    sync.RWMutex

	// Check interface
	_ resource.ResourceWithImportState = (*resourceLBACRuleItem)(nil)
)

func makeResourceDataSourceLBACRuleItem() *common.Resource {
	return common.NewResource(
		common.CategoryGrafanaEnterprise,
		resourceLBACRuleItemName,
		resourceLBACRuleID,
		&resourceLBACRuleItem{},
	)
}

type resourceLBACRuleItemModel struct {
	ID            types.String `tfsdk:"id"`
	OrgID         types.String `tfsdk:"org_id"`
	DatasourceUID types.String `tfsdk:"datasource_uid"`
	TeamID        types.String `tfsdk:"team_id"`
	UserID        types.String `tfsdk:"user_id"`
	Rules         types.List   `tfsdk:"rules"`
}

type resourceLBACRuleItem struct {
	basePluginFrameworkResource
}

func (r *resourceLBACRuleItem) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceLBACRuleItemName
}

func (r *resourceLBACRuleItem) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	targetOneOf := stringvalidator.ExactlyOneOf(
		path.MatchRoot("team_id"),
		// path.MatchRoot("user_id"),
		// path.MatchRoot("service_account_id"),
	)

	resp.Schema = schema.Schema{
		MarkdownDescription: `
* [Official documentation](https://grafana.com/docs/grafana/latest/datasources/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/data_source/)

The required arguments for this resource vary depending on the type of data
source selected (via the 'type' argument).

Example usage:
resource "grafana_data_source_lbac_rule" "team_rule" {
  datasource_uid = "some-unique-datasource-uid"
  team_id        = "team1"
  rules          = [
    "{ foo != \"bar\", foo !~ \"baz\" }",
    "{ foo = \"qux\", bar ~ \"quux\" }"
  ]
}
`,
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
				Description: "the datasource UID onto which to assign an actor",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"rules": schema.ListAttribute{
				ElementType: types.StringType,
				Required:    true,
				Description: "List of LBAC rules to apply",
			},
			"team_id": schema.StringAttribute{
				Optional:    true,
				Description: "the team onto which the rules should be added",
				Validators: []validator.String{
					targetOneOf,
				},
				PlanModifiers: []planmodifier.String{
					&orgScopedAttributePlanModifier{},
					stringplanmodifier.RequiresReplace(),
				},
			},
			// "user_id": schema.StringAttribute{
			// 	Optional:    true,
			// 	Description: "the user onto which the rules should be added",
			// 	Validators: []validator.String{
			// 		targetOneOf,
			// 	},
			// 	PlanModifiers: []planmodifier.String{
			// 		stringplanmodifier.RequiresReplace(),
			// 	},
			// },
			// "service_account_id": schema.StringAttribute{
			// 	Optional:    true,
			// 	Description: "the service account onto which the rules should be added",
			// 	Validators: []validator.String{
			// 		targetOneOf,
			// 	},
			// 	PlanModifiers: []planmodifier.String{
			// 		&orgScopedAttributePlanModifier{},
			// 		stringplanmodifier.RequiresReplace(),
			// 	},
			// },
		},
	}
}

func (r *resourceLBACRuleItem) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resourceLBACRuleMutex.RLock()
	defer resourceLBACRuleMutex.RUnlock()
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

func (r *resourceLBACRuleItem) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Read Terraform plan data into the model
	var data resourceLBACRuleItemModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, _, err := r.clientFromNewOrgResource(data.OrgID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get client", err.Error())
		return
	}

	// Get existing role assignments
	resourceLBACRuleMutex.Lock()
	defer resourceLBACRuleMutex.Unlock()
	getResp, err := client.Enterprise.GetTeamLBACRulesAPI(data.DatasourceUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get LBAC rules", err.Error())
		return
	}
	lbacRules := getResp.Payload
	b, err := lbacRules.MarshalBinary()
	if err != nil {
		resp.Diagnostics.AddError("Failed to marshal LBAC rules", err.Error())
		return
	}
	fmt.Println(string(b))

	// assignmentType := ""
	// resourceID := ""
	switch {
	case !data.TeamID.IsNull():
		_, teamIDStr := SplitOrgResourceID(data.TeamID.ValueString())
		_, err := strconv.ParseInt(teamIDStr, 10, 64)
		if err != nil {
			resp.Diagnostics.AddError("Failed to parse team ID", err.Error())
			return
		}
		// lbacRules.Teams = append(lbacRules.Teams, teamID)
		// assignmentType = "team"
		// resourceID = teamIDStr
	}
	// TODO: add support for users
	// case !data.UserID.IsNull():
	// 	userID, err := strconv.ParseInt(data.UserID.ValueString(), 10, 64)
	// 	if err != nil {
	// 		resp.Diagnostics.AddError("Failed to parse user ID", err.Error())
	// 		return
	// 	}
	// 	lbacRules.Users = append(lbacRules.Users, userID)
	// 	assignmentType = "user"
	// 	resourceID = data.UserID.ValueString()
	// case !data.ServiceAccountID.IsNull():
	// 	_, serviceAccountIDStr := SplitOrgResourceID(data.ServiceAccountID.ValueString())
	// 	serviceAccountID, err := strconv.ParseInt(serviceAccountIDStr, 10, 64)
	// 	if err != nil {
	// 		resp.Diagnostics.AddError("Failed to parse service account ID", err.Error())
	// 		return
	// 	}
	// 	lbacRules.ServiceAccounts = append(lbacRules.ServiceAccounts, serviceAccountID)
	// 	assignmentType = "service_account"
	// 	resourceID = serviceAccountIDStr
	// }

	// _, err = client.Enterprise.UpdateTeamLBACRulesAPI()
	// if err != nil {
	// 	resp.Diagnostics.AddError("Failed to update LBAC rules", err.Error())
	// 	return
	// }

	// Save data into Terraform state
	// data.ID = types.StringValue(resourceLBACRuleID.Make(orgID, data.RoleUID.ValueString(), assignmentType, resourceID))
	// data.OrgID = types.StringValue(strconv.FormatInt(orgID, 10))
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *resourceLBACRuleItem) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Read Terraform state data into the model
	var data resourceLBACRuleItemModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	// Read from API
	resourceLBACRuleMutex.RLock()
	defer resourceLBACRuleMutex.RUnlock()
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

func (r *resourceLBACRuleItem) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Update shouldn't happen as all attributes require replacement
	resp.Diagnostics.AddError("Update not supported", "Update not supported")
}

func (r *resourceLBACRuleItem) read(id string) (*resourceLBACRuleItemModel, diag.Diagnostics) {
	client, orgID, idFields, err := r.clientFromExistingOrgResource(resourceLBACRuleID, id)
	if err != nil {
		return nil, diag.Diagnostics{diag.NewErrorDiagnostic("Failed to get client", err.Error())}
	}
	datasourceUID, assignmentType, _ := idFields[0].(string), idFields[1].(string), idFields[2].(string)

	// Try to get the role
	_, err = client.Datasources.GetDataSourceByID(datasourceUID)
	if err != nil {
		if common.IsNotFoundError(err) {
			return nil, nil
		}
		return nil, diag.Diagnostics{diag.NewErrorDiagnostic("Failed to get datasource", err.Error())}
	}

	// Get existing role assignments
	getResp, err := client.Enterprise.GetTeamLBACRulesAPI(datasourceUID)
	if err != nil {
		return nil, diag.Diagnostics{diag.NewErrorDiagnostic("Failed to get datasource LBAC rules", err.Error())}
	}

	// Find the assignment
	data := &resourceLBACRuleItemModel{
		ID:            types.StringValue(id),
		OrgID:         types.StringValue(strconv.FormatInt(orgID, 10)),
		DatasourceUID: types.StringValue(datasourceUID),
		Rules:         types.List{},
		TeamID:        types.String{},
	}
	switch assignmentType {
	case "team":
		fmt.Printf("getResp.Payload: %+v\n", getResp.Payload)
		// for _, team := range getResp.Payload.Teams {
		// 	if teamIDStr := strconv.FormatInt(team, 10); teamIDStr == identifier {
		// 		data.TeamID = types.StringValue(teamIDStr)
		// 		break
		// 	}
		// }
	// TODO: add support for users and service accounts
	// case "user":
	// 	for _, user := range getResp.Payload.Users {
	// 		if userIDStr := strconv.FormatInt(user, 10); userIDStr == identifier {
	// 			data.UserID = types.StringValue(userIDStr)
	// 			break
	// 		}
	// 	}
	// case "service_account":
	// 	for _, serviceAccount := range getResp.Payload.ServiceAccounts {
	// 		if saIDStr := strconv.FormatInt(serviceAccount, 10); saIDStr == identifier {
	// 			data.ServiceAccountID = types.StringValue(saIDStr)
	// 			break
	// 		}
	// 	}
	default:
		// Should never happen due to the schema validation, but include for completeness
		return nil, diag.Diagnostics{diag.NewErrorDiagnostic("Invalid assignment type", assignmentType)}
	}

	// TODO: add support for users and service accounts
	if data.TeamID.IsNull() {
		return nil, nil
	}

	return data, nil
}

func (r *resourceLBACRuleItem) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Read Terraform prior state data into the model
	var data resourceLBACRuleItemModel
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
	// case "user":
	// 	for i, user := range roleAssignments.Users {
	// 		if strconv.FormatInt(user, 10) == identifier {
	// 			roleAssignments.Users = append(roleAssignments.Users[:i], roleAssignments.Users[i+1:]...)
	// 			break
	// 		}
	// 	}
	// case "service_account":
	// 	for i, serviceAccount := range roleAssignments.ServiceAccounts {
	// 		if strconv.FormatInt(serviceAccount, 10) == identifier {
	// 			roleAssignments.ServiceAccounts = append(roleAssignments.ServiceAccounts[:i], roleAssignments.ServiceAccounts[i+1:]...)
	// 			break
	// 		}
	// 	}
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
