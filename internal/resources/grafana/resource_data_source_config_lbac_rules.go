package grafana

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/grafana/grafana-openapi-client-go/client/enterprise"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
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
	resourceDataSourceConfigLBACRulesID   = common.NewResourceID(
		common.StringIDField("datasource_uid"),
	)
)

func makeResourceDataSourceConfigLBACRules() *common.Resource {
	resourceStruct := &resourceDataSourceConfigLBACRules{}
	return common.NewResource(
		common.CategoryGrafanaEnterprise,
		resourceDataSourceConfigLBACRulesName,
		resourceDataSourceConfigLBACRulesID,
		resourceStruct,
	)
}

type LBACRule struct {
	TeamID  types.String   `tfsdk:"team_id"`
	TeamUID types.String   `tfsdk:"team_uid"`
	Rules   []types.String `tfsdk:"rules"`
}

type resourceDataSourceConfigLBACRulesModel struct {
	ID            types.String `tfsdk:"id"`
	DatasourceUID types.String `tfsdk:"datasource_uid"`
	Rules         types.String `tfsdk:"rules"`
}

type resourceDataSourceConfigLBACRules struct {
	client *common.Client
}

func (r *resourceDataSourceConfigLBACRules) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceDataSourceConfigLBACRulesName
}

func (r *resourceDataSourceConfigLBACRules) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Manages LBAC rules for a data source.

!> Warning: The resource is experimental and will be subject to change. This resource manages the entire LBAC rules tree, and will overwrite any existing rules.

* [Official documentation](https://grafana.com/docs/grafana/latest/administration/data-source-management/teamlbac/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/datasource_lbac_rules/)

This resource requires Grafana >=11.0.0.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"datasource_uid": schema.StringAttribute{
				Required:    true,
				Description: "The UID of the datasource.",
			},
			"rules": schema.StringAttribute{
				Required:    true,
				Description: "JSON-encoded LBAC rules for the data source. Map of team IDs to lists of rule strings.",
			},
		},
	}
}

func (r *resourceDataSourceConfigLBACRules) Configure(ctx context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*common.Client)
}

func (r *resourceDataSourceConfigLBACRules) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resourceDataSourceConfigLBACRulesModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rulesMap := make(map[string][]string)
	err := json.Unmarshal([]byte(data.Rules.ValueString()), &rulesMap)
	if err != nil {
		resp.Diagnostics.AddError("Invalid rules JSON", fmt.Sprintf("Failed to parse rule s: %v", err))
	}

	apiRules := make([]*models.TeamLBACRule, 0, len(rulesMap))
	for teamUID, rules := range rulesMap {
		apiRules = append(apiRules, &models.TeamLBACRule{
			TeamID:  "",
			TeamUID: teamUID,
			Rules:   rules,
		})
	}

	client := r.client.GrafanaAPI

	params := &enterprise.UpdateTeamLBACRulesAPIParams{
		Context: ctx,
		UID:     data.DatasourceUID.ValueString(),
		Body:    &models.UpdateTeamLBACCommand{Rules: apiRules},
	}

	_, err = client.Enterprise.UpdateTeamLBACRulesAPI(params)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create LBAC rules", err.Error())
		return
	}

	data.ID = types.StringValue(data.DatasourceUID.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *resourceDataSourceConfigLBACRules) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resourceDataSourceConfigLBACRulesModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	datasourceUID := data.DatasourceUID.ValueString()
	client := r.client.GrafanaAPI

	getResp, err := client.Enterprise.GetTeamLBACRulesAPI(datasourceUID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get LBAC rules", err.Error())
		return
	}

	rulesMap := make(map[string][]string)
	for _, rule := range getResp.Payload.Rules {
		rulesMap[rule.TeamUID] = rule.Rules
	}

	rulesJSON, err := json.Marshal(rulesMap)
	if err != nil {
		resp.Diagnostics.AddError("Failed to encode rules", err.Error())
		return
	}

	data = resourceDataSourceConfigLBACRulesModel{
		ID:            types.StringValue(datasourceUID),
		DatasourceUID: types.StringValue(datasourceUID),
		Rules:         types.StringValue(string(rulesJSON)),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *resourceDataSourceConfigLBACRules) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resourceDataSourceConfigLBACRulesModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rulesMap := make(map[string][]string)
	err := json.Unmarshal([]byte(data.Rules.ValueString()), &rulesMap)
	if err != nil {
		resp.Diagnostics.AddError("Invalid rules JSON", fmt.Sprintf("Failed to parse rules: %v", err))
		return
	}

	apiRules := make([]*models.TeamLBACRule, 0, len(rulesMap))
	for teamUID, rules := range rulesMap {
		apiRules = append(apiRules, &models.TeamLBACRule{
			TeamID:  "",
			TeamUID: teamUID,
			Rules:   rules,
		})
	}
	datasourceUID := data.DatasourceUID.ValueString()
	client := r.client.GrafanaAPI

	params := &enterprise.UpdateTeamLBACRulesAPIParams{
		Context: ctx,
		UID:     datasourceUID,
		Body:    &models.UpdateTeamLBACCommand{Rules: apiRules},
	}

	_, err = client.Enterprise.UpdateTeamLBACRulesAPI(params)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update LBAC rules", err.Error())
		return
	}

	data.ID = types.StringValue(datasourceUID)
	data.DatasourceUID = types.StringValue(datasourceUID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *resourceDataSourceConfigLBACRules) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning("Operation not supported", "Delete operation is not supported for LBAC rules")
}

func (r *resourceDataSourceConfigLBACRules) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	datasourceUID := req.ID

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), datasourceUID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("datasource_uid"), datasourceUID)...)
}
