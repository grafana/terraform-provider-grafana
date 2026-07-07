package assistant

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common/assistantapi"
	"github.com/grafana/terraform-provider-grafana/v4/internal/util"
)

const resourceMCPServerName = "grafana_assistant_mcp_server"

var resourceMCPServerID = common.NewResourceID(common.StringIDField("id"))

type mcpServerResource struct {
	client *assistantapi.Client
}

type mcpConfigurationModel struct {
	URL                  types.String `tfsdk:"url"`
	BuiltinID            types.String `tfsdk:"builtin_id"`
	ToolPreferences      types.Map    `tfsdk:"tool_preferences"`
	ToolApprovalPolicies types.Map    `tfsdk:"tool_approval_policies"`
}

type mcpServerModel struct {
	ID            types.String          `tfsdk:"id"`
	Scope         types.String          `tfsdk:"scope"`
	Name          types.String          `tfsdk:"name"`
	Description   types.String          `tfsdk:"description"`
	Enabled       types.Bool            `tfsdk:"enabled"`
	Applications  types.List            `tfsdk:"applications"`
	Configuration mcpConfigurationModel `tfsdk:"configuration"`
	CustomHeaders types.Map             `tfsdk:"custom_headers"`
}

func makeResourceMCPServer() *common.Resource {
	return common.NewResource(
		common.CategoryGrafanaAssistant,
		resourceMCPServerName,
		resourceMCPServerID,
		&mcpServerResource{},
	)
}

func (r *mcpServerResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceMCPServerName
}

func (r *mcpServerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Grafana Assistant MCP server integration.",
		Attributes: map[string]schema.Attribute{
			"id":           idAttribute(),
			"scope":        scopeAttribute(),
			"name":         schema.StringAttribute{Description: "The MCP server integration name.", Required: true},
			"description":  schema.StringAttribute{Description: "Optional description.", Optional: true},
			"enabled":      enabledAttribute(),
			"applications": applicationsAttributeMCP(),
			"custom_headers": schema.MapAttribute{
				Description: "Custom HTTP headers sent to the MCP server. Values are write-only and not returned by the API.",
				Optional:    true,
				Sensitive:   true,
				ElementType: types.StringType,
				// Write-only: use UseStateForUnknown on read we skip setting this
			},
		},
		Blocks: map[string]schema.Block{
			"configuration": schema.SingleNestedBlock{
				Description: "MCP server configuration.",
				Attributes: map[string]schema.Attribute{
					"url": schema.StringAttribute{
						Description: "MCP server URL.",
						Optional:    true,
					},
					"builtin_id": schema.StringAttribute{
						Description: "Built-in provider ID (e.g. cursor). When set, tools are provided locally.",
						Optional:    true,
					},
					"tool_preferences": schema.MapAttribute{
						Description: "Tool preferences keyed by tool name (`enabled` or `disabled`).",
						Optional:    true,
						ElementType: types.StringType,
					},
					"tool_approval_policies": schema.MapAttribute{
						Description: "Tool approval policies keyed by tool name (`auto_approve`, `always_ask`, or empty for default).",
						Optional:    true,
						ElementType: types.StringType,
					},
				},
			},
		},
	}
}

func (r *mcpServerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil || r.client != nil {
		return
	}
	client, err := withClientForResource(req, resp)
	if err != nil {
		return
	}
	r.client = client
}

func (r *mcpServerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan mcpServerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, diags := mcpPlanToCreate(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateIntegration(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create assistant MCP server", err.Error())
		return
	}

	state, stateDiags := mcpToModel(ctx, created, plan.CustomHeaders)
	resp.Diagnostics.Append(stateDiags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *mcpServerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state mcpServerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	integration, err := r.client.GetIntegration(ctx, state.ID.ValueString())
	if err != nil {
		if errors.Is(err, assistantapi.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read assistant MCP server", err.Error())
		return
	}

	model, diags := mcpToModel(ctx, integration, state.CustomHeaders)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *mcpServerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state mcpServerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, diags := mcpPlanToUpdate(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.client.UpdateIntegration(ctx, state.ID.ValueString(), state.Scope.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update assistant MCP server", err.Error())
		return
	}

	// Preserve write-only custom headers from plan when API redacts them.
	customHeaders := state.CustomHeaders
	if !plan.CustomHeaders.IsNull() && !plan.CustomHeaders.IsUnknown() {
		customHeaders = plan.CustomHeaders
	}
	model, stateDiags := mcpToModel(ctx, updated, customHeaders)
	resp.Diagnostics.Append(stateDiags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *mcpServerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state mcpServerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteIntegration(ctx, state.ID.ValueString(), state.Scope.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete assistant MCP server", err.Error())
	}
}

func (r *mcpServerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	integration, err := r.client.GetIntegration(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import assistant MCP server", err.Error())
		return
	}
	model, diags := mcpToModel(ctx, integration, types.MapNull(types.StringType))
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func mcpPlanToCreate(ctx context.Context, plan mcpServerModel) (assistantapi.IntegrationCreate, diag.Diagnostics) {
	var diags diag.Diagnostics
	configJSON, configDiags := mcpConfigToJSON(ctx, plan.Configuration)
	diags.Append(configDiags...)
	applications, appDiags := listValueToStrings(ctx, plan.Applications)
	diags.Append(appDiags...)
	headers, headerDiags := headersFromMap(plan.CustomHeaders)
	diags.Append(headerDiags...)

	enabled := plan.Enabled.ValueBool()
	return assistantapi.IntegrationCreate{
		Scope:         plan.Scope.ValueString(),
		Name:          plan.Name.ValueString(),
		Description:   plan.Description.ValueString(),
		Type:          "mcp",
		Enabled:       util.Ptr(enabled),
		Applications:  applications,
		Configuration: configJSON,
		CustomHeaders: headers,
	}, diags
}

func mcpPlanToUpdate(ctx context.Context, plan mcpServerModel) (assistantapi.IntegrationUpdate, diag.Diagnostics) {
	var diags diag.Diagnostics
	configJSON, configDiags := mcpConfigToJSON(ctx, plan.Configuration)
	diags.Append(configDiags...)
	applications, appDiags := listValueToStrings(ctx, plan.Applications)
	diags.Append(appDiags...)

	var headers *[]assistantapi.Header
	if !plan.CustomHeaders.IsNull() && !plan.CustomHeaders.IsUnknown() {
		h, headerDiags := headersFromMap(plan.CustomHeaders)
		diags.Append(headerDiags...)
		headers = &h
	}

	enabled := plan.Enabled.ValueBool()
	return assistantapi.IntegrationUpdate{
		Scope:         plan.Scope.ValueString(),
		Name:          util.Ptr(plan.Name.ValueString()),
		Description:   util.Ptr(plan.Description.ValueString()),
		Enabled:       util.Ptr(enabled),
		Applications:  &applications,
		Configuration: &configJSON,
		CustomHeaders: headers,
	}, diags
}

func mcpConfigToJSON(ctx context.Context, cfg mcpConfigurationModel) (json.RawMessage, diag.Diagnostics) {
	mcpCfg, diags := mcpConfigFromModel(ctx, cfg)
	if diags.HasError() {
		return nil, diags
	}
	raw, err := assistantapi.MarshalMCPConfig(mcpCfg)
	if err != nil {
		return nil, diag.Diagnostics{diag.NewErrorDiagnostic("Failed to marshal MCP configuration", err.Error())}
	}
	return raw, diags
}

func mcpConfigFromModel(ctx context.Context, cfg mcpConfigurationModel) (assistantapi.MCPConfig, diag.Diagnostics) {
	var diags diag.Diagnostics
	result := assistantapi.MCPConfig{
		URL:       cfg.URL.ValueString(),
		BuiltinID: cfg.BuiltinID.ValueString(),
	}
	if !cfg.ToolPreferences.IsNull() && !cfg.ToolPreferences.IsUnknown() {
		prefs := make(map[string]string)
		for k, v := range cfg.ToolPreferences.Elements() {
			if s, ok := v.(types.String); ok {
				prefs[k] = s.ValueString()
			}
		}
		result.ToolPreferences = prefs
	}
	if !cfg.ToolApprovalPolicies.IsNull() && !cfg.ToolApprovalPolicies.IsUnknown() {
		policies := make(map[string]string)
		for k, v := range cfg.ToolApprovalPolicies.Elements() {
			if s, ok := v.(types.String); ok {
				policies[k] = s.ValueString()
			}
		}
		result.ToolApprovalPolicies = policies
	}
	return result, diags
}

func mcpToModel(ctx context.Context, integration assistantapi.Integration, customHeaders types.Map) (mcpServerModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	applications, appDiags := stringsToListValue(ctx, integration.Applications)
	diags.Append(appDiags...)

	cfg, err := assistantapi.ParseMCPConfig(integration.Configuration)
	if err != nil {
		diags.AddError("Failed to parse MCP configuration", err.Error())
		return mcpServerModel{}, diags
	}
	configModel, cfgDiags := mcpConfigToModel(ctx, cfg)
	diags.Append(cfgDiags...)

	enabled := true
	if integration.Enabled != nil {
		enabled = *integration.Enabled
	}

	return mcpServerModel{
		ID:            types.StringValue(integration.ID),
		Scope:         types.StringValue(integration.Scope),
		Name:          types.StringValue(integration.Name),
		Description:   stringValueOrNull(integration.Description),
		Enabled:       types.BoolValue(enabled),
		Applications:  applications,
		Configuration: configModel,
		CustomHeaders: customHeaders,
	}, diags
}

func mcpConfigToModel(ctx context.Context, cfg assistantapi.MCPConfig) (mcpConfigurationModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	toolPrefs, prefsDiags := stringMapToTypesMap(ctx, cfg.ToolPreferences)
	diags.Append(prefsDiags...)
	toolPolicies, polDiags := stringMapToTypesMap(ctx, cfg.ToolApprovalPolicies)
	diags.Append(polDiags...)

	url := types.StringNull()
	if cfg.URL != "" {
		url = types.StringValue(cfg.URL)
	}
	builtinID := types.StringNull()
	if cfg.BuiltinID != "" {
		builtinID = types.StringValue(cfg.BuiltinID)
	}

	return mcpConfigurationModel{
		URL:                  url,
		BuiltinID:            builtinID,
		ToolPreferences:      toolPrefs,
		ToolApprovalPolicies: toolPolicies,
	}, diags
}

func stringMapToTypesMap(ctx context.Context, m map[string]string) (types.Map, diag.Diagnostics) {
	if len(m) == 0 {
		return types.MapNull(types.StringType), nil
	}
	return types.MapValueFrom(ctx, types.StringType, m)
}
