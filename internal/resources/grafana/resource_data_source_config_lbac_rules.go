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

// Note: The LBAC Rules API only supports GET and UPDATE operations.
// There is no CREATE or DELETE API endpoint. The UPDATE operation is used
// for all modifications (create/update/delete) by sending the complete desired
// state. Deleting rules is done by sending an empty rules list.

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

This resource requires Grafana >=11.5.0.
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
				Description: "JSON-encoded LBAC rules for the data source. Map of team UIDs to lists of rule strings.",
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

// Add this helper function to handle the common update logic
func (r *resourceDataSourceConfigLBACRules) updateRules(ctx context.Context, data *resourceDataSourceConfigLBACRulesModel, rules map[string][]string) error {
	apiRules := make([]*models.TeamLBACRule, 0, len(rules))
	for teamUID, ruleList := range rules {
		apiRules = append(apiRules, &models.TeamLBACRule{
			TeamUID: teamUID,
			Rules:   ruleList,
		})
	}

	params := &enterprise.UpdateTeamLBACRulesAPIParams{
		Context: ctx,
		UID:     data.DatasourceUID.ValueString(),
		Body:    &models.UpdateTeamLBACCommand{Rules: apiRules},
	}

	_, err := r.client.GrafanaAPI.Enterprise.UpdateTeamLBACRulesAPI(params)
	if err != nil {
		return fmt.Errorf("failed to update LBAC rules for datasource %q: %w", data.DatasourceUID.ValueString(), err)
	}
	return nil
}

func (r *resourceDataSourceConfigLBACRules) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resourceDataSourceConfigLBACRulesModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rulesMap := make(map[string][]string)
	if err := json.Unmarshal([]byte(data.Rules.ValueString()), &rulesMap); err != nil {
		resp.Diagnostics.AddError(
			"Invalid rules JSON",
			fmt.Sprintf("Failed to parse rules for datasource %q: %v. Please ensure the rules are valid JSON.", data.DatasourceUID.ValueString(), err),
		)
		return
	}

	if err := r.updateRules(ctx, &data, rulesMap); err != nil {
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
		resp.Diagnostics.AddError(
			"Failed to get LBAC rules",
			fmt.Sprintf("Could not read LBAC rules for datasource %q: %v", datasourceUID, err),
		)
		return
	}

	rulesMap := make(map[string][]string)
	for _, rule := range getResp.Payload.Rules {
		rulesMap[rule.TeamUID] = rule.Rules
	}

	rulesJSON, err := json.Marshal(rulesMap)
	if err != nil {
		// Marshal error should never happen for a valid map
		resp.Diagnostics.AddError(
			"Failed to encode rules",
			fmt.Sprintf("Could not encode LBAC rules for datasource %q: %v. This is an internal error, please report it.", datasourceUID, err),
		)
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
	if err := json.Unmarshal([]byte(data.Rules.ValueString()), &rulesMap); err != nil {
		resp.Diagnostics.AddError(
			"Invalid rules JSON",
			fmt.Sprintf("Failed to parse updated rules for datasource %q: %v. Please ensure the rules are valid JSON.", data.DatasourceUID.ValueString(), err),
		)
		return
	}

	if err := r.updateRules(ctx, &data, rulesMap); err != nil {
		resp.Diagnostics.AddError("Failed to update LBAC rules", err.Error())
		return
	}

	data.ID = types.StringValue(data.DatasourceUID.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *resourceDataSourceConfigLBACRules) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state resourceDataSourceConfigLBACRulesModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Pass empty rules map to clear all rules
	if err := r.updateRules(ctx, &state, make(map[string][]string)); err != nil {
		resp.Diagnostics.AddError(
			"Failed to delete LBAC rules",
			fmt.Sprintf("Could not delete LBAC rules for datasource %q: %v", state.DatasourceUID.ValueString(), err),
		)
		return
	}
}

func (r *resourceDataSourceConfigLBACRules) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	datasourceUID := req.ID

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), datasourceUID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("datasource_uid"), datasourceUID)...)
}
