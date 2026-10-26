package assistant

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
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

type mcpGitHubAppModel struct {
	AppID          types.Int64  `tfsdk:"app_id"`
	InstallationID types.Int64  `tfsdk:"installation_id"`
	PrivateKey     types.String `tfsdk:"private_key"`
	PrivateKeyHash types.String `tfsdk:"private_key_hash"`
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
	GitHubApp     *mcpGitHubAppModel    `tfsdk:"github_app"`
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
			"github_app": schema.SingleNestedBlock{
				Description: "GitHub App installation credentials, used instead of a personal access token for `tenant` scoped GitHub MCP servers (`https://api.githubcopilot.com/mcp` or `https://<HOSTNAME>.ghe.com/mcp`). " +
					"The Assistant mints short-lived installation tokens from these credentials and uses them in place of any `Authorization` custom header. All three attributes are required when the block is set. Removing the block clears the credentials.",
				Attributes: map[string]schema.Attribute{
					"app_id": schema.Int64Attribute{
						Description: "The GitHub App ID.",
						Optional:    true,
						Validators:  []validator.Int64{int64validator.AtLeast(1)},
					},
					"installation_id": schema.Int64Attribute{
						Description: "The ID of the GitHub App installation on your organization.",
						Optional:    true,
						Validators:  []validator.Int64{int64validator.AtLeast(1)},
					},
					"private_key": schema.StringAttribute{
						Description: "The GitHub App private key in PEM format. Terraform sends it to the API but never stores it in state.",
						Optional:    true,
						WriteOnly:   true,
						Sensitive:   true,
						Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
					},
					"private_key_hash": schema.StringAttribute{
						Description: "SHA-256 hash of the private key Terraform last sent, used to detect key changes in configuration. " +
							"A key changed outside Terraform is not detected, because the API never returns the key.",
						Computed:  true,
						Sensitive: true,
					},
				},
				Validators: []validator.Object{
					objectvalidator.AlsoRequires(
						path.MatchRelative().AtName("app_id"),
						path.MatchRelative().AtName("installation_id"),
						path.MatchRelative().AtName("private_key"),
					),
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
	resp.Diagnostics.Append(setGitHubAppPrivateKeyFromConfig(ctx, req.Config, &plan)...)
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

	state, stateDiags := mcpToModel(ctx, created, plan.CustomHeaders, githubAppPrivateKeyHash(plan.GitHubApp))
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

	model, diags := mcpToModel(ctx, integration, state.CustomHeaders, githubAppPrivateKeyHash(state.GitHubApp))
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *mcpServerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state mcpServerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(setGitHubAppPrivateKeyFromConfig(ctx, req.Config, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, diags := mcpPlanToUpdate(ctx, plan, state)
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
	model, stateDiags := mcpToModel(ctx, updated, customHeaders, githubAppPrivateKeyHash(plan.GitHubApp))
	resp.Diagnostics.Append(stateDiags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

// ValidateConfig rejects GitHub App credentials the backend would store but
// never use: they only apply to tenant-scoped GitHub MCP servers.
func (r *mcpServerResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var githubApp types.Object
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("github_app"), &githubApp)...)
	if resp.Diagnostics.HasError() || githubApp.IsNull() || githubApp.IsUnknown() {
		return
	}

	var scope, mcpURL types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("scope"), &scope)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("configuration").AtName("url"), &mcpURL)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !scope.IsNull() && !scope.IsUnknown() && scope.ValueString() != "tenant" {
		resp.Diagnostics.AddAttributeError(path.Root("github_app"), "GitHub App credentials require tenant scope",
			fmt.Sprintf("github_app can only be used with scope = \"tenant\", got %q.", scope.ValueString()))
	}
	switch {
	case mcpURL.IsUnknown():
	case mcpURL.IsNull():
		resp.Diagnostics.AddAttributeError(path.Root("github_app"), "GitHub App credentials require a GitHub MCP server URL",
			"github_app requires configuration.url to be set to a GitHub MCP server URL, such as "+gitHubMCPURLExamples+".")
	case !isGitHubMCPURL(mcpURL.ValueString()):
		resp.Diagnostics.AddAttributeError(path.Root("github_app"), "GitHub App credentials require a GitHub MCP server URL",
			fmt.Sprintf("github_app can only be used with a GitHub MCP server URL, such as %s, got %q.", gitHubMCPURLExamples, mcpURL.ValueString()))
	}
}

// ModifyPlan plans private_key_hash from the write-only private key, which is
// only available in config, so that a changed key shows as a diff.
func (r *mcpServerResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	var githubApp types.Object
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("github_app"), &githubApp)...)
	if resp.Diagnostics.HasError() || githubApp.IsNull() || githubApp.IsUnknown() {
		return
	}

	var privateKey types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("github_app").AtName("private_key"), &privateKey)...)
	if resp.Diagnostics.HasError() {
		return
	}
	hash := types.StringNull()
	switch {
	case privateKey.IsUnknown():
		hash = types.StringUnknown()
	case !privateKey.IsNull():
		hash = types.StringValue(hashPrivateKey(privateKey.ValueString()))
	}
	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("github_app").AtName("private_key_hash"), hash)...)
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
	model, diags := mcpToModel(ctx, integration, types.MapNull(types.StringType), types.StringNull())
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
	body := assistantapi.IntegrationCreate{
		Scope:         plan.Scope.ValueString(),
		Name:          plan.Name.ValueString(),
		Description:   plan.Description.ValueString(),
		Type:          "mcp",
		Enabled:       util.Ptr(enabled),
		Applications:  applications,
		Configuration: configJSON,
		CustomHeaders: headers,
	}
	if plan.GitHubApp != nil {
		body.GitHubAppID = plan.GitHubApp.AppID.ValueInt64Pointer()
		body.GitHubAppInstallationID = plan.GitHubApp.InstallationID.ValueInt64Pointer()
		body.GitHubAppPrivateKey = plan.GitHubApp.PrivateKey.ValueStringPointer()
	}
	return body, diags
}

func mcpPlanToUpdate(ctx context.Context, plan, state mcpServerModel) (assistantapi.IntegrationUpdate, diag.Diagnostics) {
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
	body := assistantapi.IntegrationUpdate{
		Scope:         plan.Scope.ValueString(),
		Name:          util.Ptr(plan.Name.ValueString()),
		Description:   util.Ptr(plan.Description.ValueString()),
		Enabled:       util.Ptr(enabled),
		Applications:  &applications,
		Configuration: &configJSON,
		CustomHeaders: headers,
	}
	switch {
	case plan.GitHubApp != nil:
		body.GitHubAppID = plan.GitHubApp.AppID.ValueInt64Pointer()
		body.GitHubAppInstallationID = plan.GitHubApp.InstallationID.ValueInt64Pointer()
		// The API treats every resubmitted key as a credential change, so only
		// send it when it differs from the key Terraform last sent, or when the
		// IDs differ (e.g. changed outside Terraform), since the API keeps the
		// stored key when none is sent and it could belong to another App.
		if state.GitHubApp == nil ||
			!plan.GitHubApp.PrivateKeyHash.Equal(state.GitHubApp.PrivateKeyHash) ||
			!plan.GitHubApp.AppID.Equal(state.GitHubApp.AppID) ||
			!plan.GitHubApp.InstallationID.Equal(state.GitHubApp.InstallationID) {
			body.GitHubAppPrivateKey = plan.GitHubApp.PrivateKey.ValueStringPointer()
		}
	case state.GitHubApp != nil:
		// A zero app ID clears all GitHub App credentials.
		body.GitHubAppID = util.Ptr(int64(0))
	}
	return body, diags
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

func mcpToModel(ctx context.Context, integration assistantapi.Integration, customHeaders types.Map, githubAppKeyHash types.String) (mcpServerModel, diag.Diagnostics) {
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

	var githubApp *mcpGitHubAppModel
	if integration.GitHubAppID != nil && *integration.GitHubAppID > 0 {
		githubApp = &mcpGitHubAppModel{
			AppID:          types.Int64Value(*integration.GitHubAppID),
			InstallationID: types.Int64PointerValue(integration.GitHubAppInstallationID),
			PrivateKey:     types.StringNull(),
			PrivateKeyHash: githubAppKeyHash,
		}
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
		GitHubApp:     githubApp,
	}, diags
}

// setGitHubAppPrivateKeyFromConfig copies the write-only private key, which is
// null in the plan, from config into the plan model.
func setGitHubAppPrivateKeyFromConfig(ctx context.Context, config tfsdk.Config, plan *mcpServerModel) diag.Diagnostics {
	if plan.GitHubApp == nil {
		return nil
	}
	return config.GetAttribute(ctx, path.Root("github_app").AtName("private_key"), &plan.GitHubApp.PrivateKey)
}

// githubAppPrivateKeyHash returns the hash of the key Terraform last sent, which
// must be carried over from plan or state because the API never returns the key.
func githubAppPrivateKeyHash(githubApp *mcpGitHubAppModel) types.String {
	if githubApp == nil {
		return types.StringNull()
	}
	return githubApp.PrivateKeyHash
}

const gitHubMCPURLExamples = "https://api.githubcopilot.com/mcp/ or https://<HOSTNAME>.ghe.com/mcp"

// isGitHubMCPURL reports whether rawURL is a GitHub MCP endpoint, such as
// https://api.githubcopilot.com/mcp/ or https://<HOSTNAME>.ghe.com/mcp.
func isGitHubMCPURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	if host != "api.githubcopilot.com" && !strings.HasSuffix(host, ".ghe.com") {
		return false
	}
	return u.Path == "/mcp" || strings.HasPrefix(u.Path, "/mcp/")
}

func hashPrivateKey(privateKey string) string {
	sum := sha256.Sum256([]byte(privateKey))
	return hex.EncodeToString(sum[:])
}

func mcpConfigToModel(ctx context.Context, cfg assistantapi.MCPConfig) (mcpConfigurationModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	toolPrefs, prefsDiags := stringMapToTypesMap(ctx, cfg.ToolPreferences)
	diags.Append(prefsDiags...)
	toolPolicies, polDiags := stringMapToTypesMap(ctx, cfg.ToolApprovalPolicies)
	diags.Append(polDiags...)

	mcpURL := types.StringNull()
	if cfg.URL != "" {
		mcpURL = types.StringValue(cfg.URL)
	}
	builtinID := types.StringNull()
	if cfg.BuiltinID != "" {
		builtinID = types.StringValue(cfg.BuiltinID)
	}

	return mcpConfigurationModel{
		URL:                  mcpURL,
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
