package agento11y

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common/agento11yapi"
	"github.com/grafana/terraform-provider-grafana/v4/internal/util"
)

const resourceHookRuleName = "grafana_agento11y_hook_rule"

var resourceHookRuleID = common.NewResourceID(common.StringIDField("rule_id"))

type hookRuleResource struct {
	client *agento11yapi.Client
}

type hookRedactPatternModel struct {
	ID    types.String `tfsdk:"id"`
	Regex types.String `tfsdk:"regex"`
}

type hookRuleModel struct {
	ID           types.String         `tfsdk:"id"`
	RuleID       types.String         `tfsdk:"rule_id"`
	Enabled      types.Bool           `tfsdk:"enabled"`
	Phase        types.String         `tfsdk:"phase"`
	Priority     types.Int64          `tfsdk:"priority"`
	Selector     types.String         `tfsdk:"selector"`
	Match        jsontypes.Normalized `tfsdk:"match"`
	EvaluatorIDs types.List           `tfsdk:"evaluator_ids"`
	ActionOnFail types.String         `tfsdk:"action_on_fail"`
	ShortCircuit types.Bool           `tfsdk:"short_circuit"`
	BlockedTools types.List           `tfsdk:"blocked_tools"`
	Redact       types.List           `tfsdk:"redact"`
}

func redactAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":    types.StringType,
		"regex": types.StringType,
	}
}

func makeResourceHookRule() *common.Resource {
	return common.NewResource(
		common.CategoryAgentObservability,
		resourceHookRuleName,
		resourceHookRuleID,
		&hookRuleResource{},
	)
}

func (r *hookRuleResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceHookRuleName
}

func (r *hookRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Grafana Agent Observability hook (guard) rule. Hook rules run synchronously on the request path and can deny or warn on matching generations, block tool calls, or redact content.\n\nAt least one of `evaluator_ids`, `blocked_tools`, or `redact` must be set. Requires a Grafana instance with the `grafana-agento11y-app` plugin installed.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The hook rule identifier (equal to `rule_id`).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"rule_id": schema.StringAttribute{
				Description: "Tenant-unique identifier of the hook rule. Changing this forces a new resource.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the hook rule is enabled. Defaults to `true`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"phase": schema.StringAttribute{
				Description: "When the hook runs. One of `preflight`, `postflight`. Defaults to `preflight`.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("preflight"),
				Validators: []validator.String{
					stringvalidator.OneOf("preflight", "postflight"),
				},
			},
			"priority": schema.Int64Attribute{
				Description: "Evaluation priority; lower priority rules run first. Defaults to `0`.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
			},
			"selector": schema.StringAttribute{
				Description: "Which generations the hook applies to. One of `all`, `user_visible_turn`, `all_assistant_generations`, `tool_call_steps`, `errored_generations`. Defaults to `all`.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("all"),
				Validators: []validator.String{
					stringvalidator.OneOf("all", "user_visible_turn", "all_assistant_generations", "tool_call_steps", "errored_generations"),
				},
			},
			"match": schema.StringAttribute{
				CustomType:  jsontypes.NormalizedType{},
				Description: "Optional JSON object of match filters (for example `{\"agent_name\":\"checkout-*\"}`). Omit to match everything.",
				Optional:    true,
			},
			"evaluator_ids": schema.ListAttribute{
				Description: "IDs of the evaluators to run synchronously. Optional when `blocked_tools` or `redact` is set.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"action_on_fail": schema.StringAttribute{
				Description: "Action taken when the hook fails. One of `deny`, `warn`. Defaults to `deny`.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("deny"),
				Validators: []validator.String{
					stringvalidator.OneOf("deny", "warn"),
				},
			},
			"short_circuit": schema.BoolAttribute{
				Description: "When `true` (default), stop at the first failed rule. When `false`, run all evaluators and deny if any failed.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"blocked_tools": schema.ListAttribute{
				Description: "Glob patterns of tool call names to block (for example `[\"delete_*\"]`).",
				Optional:    true,
				ElementType: types.StringType,
			},
		},
		Blocks: map[string]schema.Block{
			"redact": schema.ListNestedBlock{
				Description: "Ordered regex redaction patterns applied to request/response text.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"regex": schema.StringAttribute{
							Description: "Regular expression to redact.",
							Required:    true,
						},
						"id": schema.StringAttribute{
							Description: "Optional stable identifier for the pattern.",
							Optional:    true,
						},
					},
				},
			},
		},
	}
}

func (r *hookRuleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil || r.client != nil {
		return
	}
	client, err := withClientForResource(req, resp)
	if err != nil {
		return
	}
	r.client = client
}

func (r *hookRuleResource) buildWrite(ctx context.Context, plan hookRuleModel) (agento11yapi.HookRuleWrite, diag.Diagnostics) {
	var diags diag.Diagnostics

	evaluatorIDs, d := listValueToStrings(ctx, plan.EvaluatorIDs)
	diags.Append(d...)
	blockedTools, d := listValueToStrings(ctx, plan.BlockedTools)
	diags.Append(d...)
	if diags.HasError() {
		return agento11yapi.HookRuleWrite{}, diags
	}

	body := agento11yapi.HookRuleWrite{
		RuleID:       plan.RuleID.ValueString(),
		Enabled:      util.Ptr(plan.Enabled.ValueBool()),
		Phase:        plan.Phase.ValueString(),
		Priority:     util.Ptr(int(plan.Priority.ValueInt64())),
		Selector:     plan.Selector.ValueString(),
		Match:        rawJSONFromNormalized(plan.Match),
		EvaluatorIDs: evaluatorIDs,
		ActionOnFail: plan.ActionOnFail.ValueString(),
		ShortCircuit: util.Ptr(plan.ShortCircuit.ValueBool()),
	}
	if len(blockedTools) > 0 {
		body.ToolFilter = &agento11yapi.HookToolFilter{BlockedNames: blockedTools}
	}

	if !plan.Redact.IsNull() && !plan.Redact.IsUnknown() {
		var patterns []hookRedactPatternModel
		diags.Append(plan.Redact.ElementsAs(ctx, &patterns, false)...)
		if diags.HasError() {
			return agento11yapi.HookRuleWrite{}, diags
		}
		if len(patterns) > 0 {
			redact := &agento11yapi.HookRedactConfig{Patterns: make([]agento11yapi.HookRedactPattern, 0, len(patterns))}
			for _, p := range patterns {
				redact.Patterns = append(redact.Patterns, agento11yapi.HookRedactPattern{
					ID:    p.ID.ValueString(),
					Regex: p.Regex.ValueString(),
				})
			}
			body.Redact = redact
		}
	}

	return body, diags
}

func (r *hookRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan hookRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, diags := r.buildWrite(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateHookRule(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create agent observability hook rule", err.Error())
		return
	}

	model, diags := hookRuleToModel(ctx, created)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *hookRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state hookRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rule, err := r.client.GetHookRule(ctx, state.RuleID.ValueString())
	if err != nil {
		if errors.Is(err, agento11yapi.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read agent observability hook rule", err.Error())
		return
	}

	model, diags := hookRuleToModel(ctx, rule)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *hookRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state hookRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, diags := r.buildWrite(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.client.UpsertHookRule(ctx, state.RuleID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update agent observability hook rule", err.Error())
		return
	}

	model, diags := hookRuleToModel(ctx, updated)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *hookRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state hookRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteHookRule(ctx, state.RuleID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete agent observability hook rule", err.Error())
		return
	}
}

func (r *hookRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	rule, err := r.client.GetHookRule(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import agent observability hook rule", err.Error())
		return
	}
	model, diags := hookRuleToModel(ctx, rule)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func hookRuleToModel(ctx context.Context, rule agento11yapi.HookRule) (hookRuleModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	evaluatorIDs, d := stringsToListValue(ctx, rule.EvaluatorIDs)
	diags.Append(d...)

	blockedTools := types.ListNull(types.StringType)
	if rule.ToolFilter != nil && len(rule.ToolFilter.BlockedNames) > 0 {
		blockedTools, d = stringsToListValue(ctx, rule.ToolFilter.BlockedNames)
		diags.Append(d...)
	}

	// redact is a nested block, so its absence is represented as an empty list
	// (not null) to match how Terraform stores zero blocks.
	redact := types.ListValueMust(types.ObjectType{AttrTypes: redactAttrTypes()}, []attr.Value{})
	if rule.Redact != nil && len(rule.Redact.Patterns) > 0 {
		patterns := make([]hookRedactPatternModel, 0, len(rule.Redact.Patterns))
		for _, p := range rule.Redact.Patterns {
			patterns = append(patterns, hookRedactPatternModel{
				ID:    stringValueOrNull(p.ID),
				Regex: types.StringValue(p.Regex),
			})
		}
		var listDiags diag.Diagnostics
		redact, listDiags = types.ListValueFrom(ctx, types.ObjectType{AttrTypes: redactAttrTypes()}, patterns)
		diags.Append(listDiags...)
	}

	return hookRuleModel{
		ID:           types.StringValue(rule.RuleID),
		RuleID:       types.StringValue(rule.RuleID),
		Enabled:      types.BoolValue(rule.Enabled),
		Phase:        types.StringValue(rule.Phase),
		Priority:     types.Int64Value(int64(rule.Priority)),
		Selector:     types.StringValue(rule.Selector),
		Match:        normalizedMatchOrNull(rule.Match),
		EvaluatorIDs: evaluatorIDs,
		ActionOnFail: types.StringValue(rule.ActionOnFail),
		ShortCircuit: types.BoolValue(rule.ShortCircuit),
		BlockedTools: blockedTools,
		Redact:       redact,
	}, diags
}
