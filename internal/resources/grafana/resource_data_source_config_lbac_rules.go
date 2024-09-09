package grafana

import (
	"context"
	"strconv"

	"github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/client/enterprise"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	// Check interface
	_ resource.ResourceWithImportState = (*resourceDataSourceConfigLBACRules)(nil)
)

var (
	resourceDataSourceConfigLBACRulesName = "grafana_data_source_config_lbac_rules"
	resourceDataSourceConfigLBACRulesID   = common.NewResourceID(common.StringIDField("datasource_uid"))
)

func makeResourceDataSourceConfigLBACRules() *common.Resource {
	resourceStruct := &resourceDataSourceConfigLBACRules{}
	return common.NewResource(
		common.CategoryGrafanaEnterprise,
		resourceDatasourcePermissionItemName,
		resourceDatasourcePermissionItemID,
		resourceStruct,
	)
}

type resourceDataSourceConfigLBACRulesModel struct {
	ID            types.String `tfsdk:"id"`
	DatasourceUID types.String `tfsdk:"datasource_uid"`
	Rules         types.Map    `tfsdk:"rules"`
}

type resourceDataSourceConfigLBACRules struct {
	client *client.GrafanaHTTPAPI
}

func (r *resourceDataSourceConfigLBACRules) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceDataSourceConfigLBACRulesName
}

func (r *resourceDataSourceConfigLBACRules) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages LBAC rules for a data source.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"datasource_uid": schema.StringAttribute{
				Required:    true,
				Description: "The UID of the datasource.",
			},
			"rules": schema.MapAttribute{
				Required:    true,
				Description: "LBAC rules for the data source. Map of team IDs to lists of rule strings.",
				ElementType: types.ListType{
					ElemType: types.StringType,
				},
			},
		},
	}
}

func (r *resourceDataSourceConfigLBACRules) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*client.GrafanaHTTPAPI)
}

func (r *resourceDataSourceConfigLBACRules) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Not implemented, but satisfies the interface
	resp.Diagnostics.AddWarning("Operation not supported", "Create operation is not supported for LBAC rules")
}

func (r *resourceDataSourceConfigLBACRules) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resourceDataSourceConfigLBACRulesModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	getResp, err := r.client.Enterprise.GetTeamLBACRulesAPI(data.DatasourceUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get LBAC rules", err.Error())
		return
	}

	rules := make(map[string]types.List)
	for teamID, teamRules := range getResp.Payload.Rules {
		stringRules := make([]attr.Value, len(teamRules.Rules))
		for i, rule := range teamRules.Rules {
			stringRules[i] = types.StringValue(rule)
		}
		rules[strconv.Itoa(int(teamID))] = types.ListValueMust(types.StringType, stringRules)
	}

	rulesAttr := make(map[string]attr.Value)
	for k, v := range rules {
		rulesAttr[k] = v
	}
	data.Rules = types.MapValueMust(types.ListType{ElemType: types.StringType}, rulesAttr)
	data.ID = data.DatasourceUID

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *resourceDataSourceConfigLBACRules) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resourceDataSourceConfigLBACRulesModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updatedRules := make(map[string][]string)
	data.Rules.ElementsAs(ctx, &updatedRules, false)

	apiRules := make([]*models.TeamLBACRule, 0)
	for teamID, rules := range updatedRules {
		teamRule := &models.TeamLBACRule{
			TeamID: teamID,
			Rules:  rules, // Change this line
		}
		apiRules = append(apiRules, teamRule)
	}

	_, err := r.client.Enterprise.UpdateTeamLBACRulesAPI(&enterprise.UpdateTeamLBACRulesAPIParams{
		UID:     data.DatasourceUID.ValueString(),
		Context: ctx,
		Body:    &models.UpdateTeamLBACCommand{Rules: apiRules},
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to update LBAC rules", err.Error())
		return
	}

	data.ID = data.DatasourceUID
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *resourceDataSourceConfigLBACRules) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Not implemented, but satisfies the interface
	resp.Diagnostics.AddWarning("Operation not supported", "Delete operation is not supported for LBAC rules")
}

func (r *resourceDataSourceConfigLBACRules) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("datasource_uid"), req, resp)
}
