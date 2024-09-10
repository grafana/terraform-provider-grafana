package grafana

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/grafana/grafana-openapi-client-go/client/enterprise"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
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
		resourceDataSourceConfigLBACRulesName,
		resourceDataSourceConfigLBACRulesID,
		resourceStruct,
	)
}

type LBACRule struct {
	TeamID types.String   `tfsdk:"team_id"`
	Rules  []types.String `tfsdk:"rules"`
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
			"rules": schema.StringAttribute{
				Required:    true,
				Description: "JSON-encoded LBAC rules for the data source. Map of team IDs to lists of rule strings.",
			},
		},
	}
}

func (r *resourceDataSourceConfigLBACRules) Configure(ctx context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	tflog.Info(ctx, "Configuring LBAC rules")
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*common.Client)
}

func (r *resourceDataSourceConfigLBACRules) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "Creating LBAC rules")
	var data resourceDataSourceConfigLBACRulesModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Creating LBAC rules", map[string]interface{}{"datasource_uid": data.DatasourceUID.ValueString()})

	var rulesMap map[string][]string
	err := json.Unmarshal([]byte(data.Rules.ValueString()), &rulesMap)
	if err != nil {
		resp.Diagnostics.AddError("Invalid rules JSON", fmt.Sprintf("Failed to parse rules: %v", err))
		return
	}

	apiRules := make([]*models.TeamLBACRule, 0, len(rulesMap))
	for teamID, rules := range rulesMap {
		apiRules = append(apiRules, &models.TeamLBACRule{
			TeamID: teamID,
			Rules:  rules,
		})
	}
	_, err = r.client.GrafanaAPI.Enterprise.UpdateTeamLBACRulesAPI(&enterprise.UpdateTeamLBACRulesAPIParams{
		UID:     data.DatasourceUID.ValueString(),
		Context: ctx,
		Body:    &models.UpdateTeamLBACCommand{Rules: apiRules},
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create LBAC rules", err.Error())
		return
	}

	tflog.Info(ctx, "LBAC rules created successfully", map[string]interface{}{"datasource_uid": data.DatasourceUID.ValueString()})

	data.ID = data.DatasourceUID
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *resourceDataSourceConfigLBACRules) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "Reading LBAC rules")
	var data resourceDataSourceConfigLBACRulesModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Reading LBAC rules", map[string]interface{}{"datasource_uid": data.DatasourceUID.ValueString()})

	getResp, err := r.client.GrafanaAPI.Enterprise.GetTeamLBACRulesAPI(data.DatasourceUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get LBAC rules", err.Error())
		return
	}

	rulesMap := make(map[string][]string)
	for teamID, teamRules := range getResp.Payload.Rules {
		strTeamID := strconv.FormatInt(int64(teamID), 10)
		rulesMap[strTeamID] = teamRules.Rules
	}

	rulesJSON, err := json.Marshal(rulesMap)
	if err != nil {
		resp.Diagnostics.AddError("Failed to encode rules", err.Error())
		return
	}

	data.Rules = types.StringValue(string(rulesJSON))
	data.ID = data.DatasourceUID

	tflog.Info(ctx, "LBAC rules read successfully", map[string]interface{}{"datasource_uid": data.DatasourceUID.ValueString()})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *resourceDataSourceConfigLBACRules) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Info(ctx, "Updating LBAC rules")
	var data resourceDataSourceConfigLBACRulesModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Updating LBAC rules", map[string]interface{}{"datasource_uid": data.DatasourceUID.ValueString()})

	rulesMap := make(map[string][]string)
	err := json.Unmarshal([]byte(data.Rules.ValueString()), &rulesMap)
	if err != nil {
		resp.Diagnostics.AddError("Invalid rules JSON", fmt.Sprintf("Failed to parse rules: %v", err))
		return
	}

	apiRules := make([]*models.TeamLBACRule, 0, len(rulesMap))
	for teamID, rules := range rulesMap {
		_, err := strconv.ParseInt(teamID, 10, 64)
		if err != nil {
			resp.Diagnostics.AddError("Invalid team ID", fmt.Sprintf("Team ID %s is not a valid integer", teamID))
			return
		}
		apiRules = append(apiRules, &models.TeamLBACRule{
			TeamID: teamID,
			Rules:  rules,
		})
	}

	_, err = r.client.GrafanaAPI.Enterprise.UpdateTeamLBACRulesAPI(&enterprise.UpdateTeamLBACRulesAPIParams{
		UID:     data.DatasourceUID.ValueString(),
		Context: ctx,
		Body:    &models.UpdateTeamLBACCommand{Rules: apiRules},
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to update LBAC rules", err.Error())
		return
	}

	tflog.Info(ctx, "LBAC rules updated successfully", map[string]interface{}{"datasource_uid": data.DatasourceUID.ValueString()})

	data.ID = data.DatasourceUID
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *resourceDataSourceConfigLBACRules) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Warn(ctx, "Delete operation not supported for LBAC rules")
	resp.Diagnostics.AddWarning("Operation not supported", "Delete operation is not supported for LBAC rules")
}

func (r *resourceDataSourceConfigLBACRules) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Info(ctx, "Importing LBAC rules", map[string]interface{}{"datasource_uid": req.ID})
	resource.ImportStatePassthroughID(ctx, path.Root("datasource_uid"), req, resp)
}
