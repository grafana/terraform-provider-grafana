package grafana

import (
	"context"
	"strconv"

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
	OrgID         types.Int64  `tfsdk:"org_id"`
	DatasourceUID types.String `tfsdk:"datasource_uid"`
	Rules         types.List   `tfsdk:"rules"`
	TeamID        types.String `tfsdk:"team_id"`
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
  uid = "some-unique-datasource-uid"
  team_id        = "team1"
  rules          = [
    "{ foo != \"bar\", foo !~ \"baz\" }",
    "{ foo = \"qux\", bar ~ \"quux\" }"
  ]
}
`,
		Attributes: map[string]schema.Attribute{
			"org_id": pluginFrameworkOrgIDAttribute(),
			"uid": schema.StringAttribute{
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
	var data resourceLBACRuleItemModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, orgID, err := r.clientFromNewOrgResource(string(data.OrgID.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError("Failed to get client", err.Error())
		return
	}

	// Get existing lbacRules
	getResp, err := client.Enterprise.GetTeamLBACRulesAPI(data.DatasourceUID.String())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get role assignments", err.Error())
		return
	}
	lbacRules := getResp.Payload

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
		lbacRules.Teams = append(lbacRules.Teams, teamID)
		assignmentType = "team"
		resourceID = teamIDStr
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
	}

	_, err = client.Enterprise.UpdateTeamLBACRulesAPI(data.DatasourceUID.ValueString(), &models.UpdateTeamLBACRulesCommand{
		LBACRules: lbacRules,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to set lbac rules", err.Error())
		return
	}

	// Save data into Terraform state
	data.ID = types.StringValue(resourceRoleAssignmentItemID.Make(orgID, data.RoleUID.ValueString(), assignmentType, resourceID))
	data.OrgID = types.StringValue(strconv.FormatInt(orgID, 10))
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *resourceLBACRuleItem) read(id string) (*resourceLBACRuleItemModel, diag.Diagnostics) {
	client, orgID, idFields, err := r.clientFromExistingOrgResource(resourceLBACRuleID, id)
	if err != nil {
		return nil, diag.Diagnostics{diag.NewErrorDiagnostic("Failed to get client", err.Error())}
	}
	datasourceUID, assignmentType, identifier := idFields[0].(string), idFields[1].(string), idFields[2].(string)

	// Try to get the datasource lbac rules
	getResp, err := client.Enterprise.GetTeamLBACRulesAPI(datasourceUID)
	if err != nil {
		if common.IsNotFoundError(err) {
			return nil, nil
		}
		return nil, diag.Diagnostics{diag.NewErrorDiagnostic("Failed to get role", err.Error())}
	}

	data := &resourceLBACRuleItemModel{
		OrgID:         types.Int64Value(orgID),
		DatasourceUID: types.StringValue(datasourceUID),
	}
	switch assignmentType {
	case "team":
		for team, rules := range getResp.Payload.LBACRules {
			if teamIDStr := strconv.FormatInt(team, 10); teamIDStr == identifier {
				data.TeamID = types.StringValue(teamIDStr)
				teamRules, _ := types.ListValue(types.StringType, rules)
				data.Rules = teamRules
				break
			}
		}
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

	if data.TeamID.IsNull() {
		return nil, nil
	}

	return data, nil
}

func (r *resourceLBACRuleItem) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Read Terraform state data into the model
	var data resourceLBACRuleItemModel
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

//	return response.JSON(http.StatusOK, util.DynMap{
//		"message":   "Data source LBAC rules updated",
//		"id":        dsUpdated.ID,
//		"uid":       dsUpdated.UID,
//		"name":      dsUpdated.Name,
//		"lbacRules": teamLBACRulesUpdated,
//	})
