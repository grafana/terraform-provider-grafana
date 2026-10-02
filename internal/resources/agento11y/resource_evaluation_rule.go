package agento11y

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common/agento11yapi"
	"github.com/grafana/terraform-provider-grafana/v4/internal/util"
)

const resourceEvaluationRuleName = "grafana_agento11y_evaluation_rule"

const (
	selectorConversation        = "conversation"
	executionModeParallel       = "parallel"
	executionModeSequential     = "sequential"
	maxFilterableTagKeysPerRule = 10
)

var resourceEvaluationRuleID = common.NewResourceID(common.StringIDField("rule_id"))

type evaluationRuleResource struct {
	client *agento11yapi.Client
}

type evaluationRuleModel struct {
	ID                types.String         `tfsdk:"id"`
	RuleID            types.String         `tfsdk:"rule_id"`
	Enabled           types.Bool           `tfsdk:"enabled"`
	Selector          types.String         `tfsdk:"selector"`
	Match             jsontypes.Normalized `tfsdk:"match"`
	SampleRate        types.Float64        `tfsdk:"sample_rate"`
	EvaluatorIDs      types.List           `tfsdk:"evaluator_ids"`
	ExecutionMode     types.String         `tfsdk:"execution_mode"`
	AlertRuleUIDs     types.List           `tfsdk:"alert_rule_uids"`
	FilterableTagKeys types.List           `tfsdk:"filterable_tag_keys"`
	MinIdleSeconds    types.Int64          `tfsdk:"min_idle_seconds"`
}

func makeResourceEvaluationRule() *common.Resource {
	return common.NewResource(
		common.CategoryAgentObservability,
		resourceEvaluationRuleName,
		resourceEvaluationRuleID,
		&evaluationRuleResource{},
	)
}

func (r *evaluationRuleResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceEvaluationRuleName
}

func (r *evaluationRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Grafana Agent Observability online evaluation rule. Rules select which agent generations (or whole conversations) are sampled and scored by one or more evaluators.\n\nRequires a Grafana instance with the `grafana-agento11y-app` plugin installed. " + writePermissionDescription,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The rule identifier (equal to `rule_id`).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"rule_id": schema.StringAttribute{
				Description: "Tenant-unique identifier of the rule. Changing this forces a new resource.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the rule is enabled. Defaults to `true`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"selector": schema.StringAttribute{
				Description: "Which generations the rule applies to. One of `user_visible_turn`, `all_assistant_generations`, `tool_call_steps`, `errored_generations`, `conversation`. Defaults to `user_visible_turn`.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("user_visible_turn"),
				Validators: []validator.String{
					stringvalidator.OneOf("user_visible_turn", "all_assistant_generations", "tool_call_steps", "errored_generations", selectorConversation),
				},
			},
			"match": schema.StringAttribute{
				CustomType:  jsontypes.NormalizedType{},
				Description: "Optional JSON object of match filters (for example `{\"agent_name\":\"checkout-*\"}`). Omit to match everything.",
				Optional:    true,
			},
			"sample_rate": schema.Float64Attribute{
				Description: "Fraction of matching generations to evaluate, in `[0,1]`. Defaults to `0.01`.",
				Optional:    true,
				Computed:    true,
				Default:     float64default.StaticFloat64(0.01),
				Validators: []validator.Float64{
					float64validator.Between(0, 1),
				},
			},
			"evaluator_ids": schema.ListAttribute{
				Description: "IDs of the evaluators to run against matching generations. Must be non-empty.",
				Required:    true,
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
			},
			"execution_mode": schema.StringAttribute{
				Description: "How evaluators execute. `parallel` runs all evaluators independently; `sequential` treats `evaluator_ids` as an ordered gate chain. Defaults to `parallel`.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(executionModeParallel),
				Validators: []validator.String{
					stringvalidator.OneOf(executionModeParallel, executionModeSequential),
				},
			},
			"alert_rule_uids": schema.ListAttribute{
				Description: "Optional Grafana alert rule UIDs associated with this evaluation rule.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"filterable_tag_keys": schema.ListAttribute{
				Description: "Generation tag keys to promote to Prometheus labels on evaluation metrics. Supports at most 10 unique, non-empty keys and cannot be set when `selector` is `conversation`.",
				Optional:    true,
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.SizeAtMost(maxFilterableTagKeysPerRule),
					listvalidator.UniqueValues(),
					listvalidator.ValueStringsAre(
						stringvalidator.LengthAtLeast(1),
						trimmedStringValidator{},
					),
				},
			},
			"min_idle_seconds": schema.Int64Attribute{
				Description: "Idle window, in seconds, before a conversation-scope rule runs. Required when `selector` is `conversation`; must be unset otherwise.",
				Optional:    true,
			},
		},
	}
}

func (r *evaluationRuleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil || r.client != nil {
		return
	}
	client, err := withClientForResource(req, resp)
	if err != nil {
		return
	}
	r.client = client
}

// validateRuleSelection enforces the selector-specific rule invariants.
func validateRuleSelection(selector string, minIdle types.Int64, filterableTagKeys types.List) diag.Diagnostics {
	var diags diag.Diagnostics
	if selector == selectorConversation {
		if minIdle.IsNull() || minIdle.IsUnknown() {
			diags.AddError("Invalid evaluation rule", "min_idle_seconds is required when selector is \"conversation\"")
		}
		if !filterableTagKeys.IsNull() && !filterableTagKeys.IsUnknown() && len(filterableTagKeys.Elements()) > 0 {
			diags.AddError("Invalid evaluation rule", "filterable_tag_keys cannot be set when selector is \"conversation\"")
		}
		return diags
	}
	if !minIdle.IsNull() && !minIdle.IsUnknown() {
		diags.AddError("Invalid evaluation rule", "min_idle_seconds may only be set when selector is \"conversation\"")
	}
	return diags
}

func (r *evaluationRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan evaluationRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(validateRuleSelection(plan.Selector.ValueString(), plan.MinIdleSeconds, plan.FilterableTagKeys)...)
	if resp.Diagnostics.HasError() {
		return
	}

	evaluatorIDs, diags := listValueToStrings(ctx, plan.EvaluatorIDs)
	resp.Diagnostics.Append(diags...)
	alertUIDs, diags := listValueToStrings(ctx, plan.AlertRuleUIDs)
	resp.Diagnostics.Append(diags...)
	filterableTagKeys, diags := listValueToStrings(ctx, plan.FilterableTagKeys)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := agento11yapi.RuleWrite{
		RuleID:            plan.RuleID.ValueString(),
		Enabled:           util.Ptr(plan.Enabled.ValueBool()),
		Selector:          plan.Selector.ValueString(),
		Match:             rawJSONFromNormalized(plan.Match),
		SampleRate:        util.Ptr(plan.SampleRate.ValueFloat64()),
		EvaluatorIDs:      evaluatorIDs,
		ExecutionMode:     plan.ExecutionMode.ValueString(),
		AlertRuleUIDs:     alertUIDs,
		FilterableTagKeys: filterableTagKeys,
	}
	if plan.Selector.ValueString() == selectorConversation && !plan.MinIdleSeconds.IsNull() {
		body.MinIdleSeconds = util.Ptr(int(plan.MinIdleSeconds.ValueInt64()))
	}

	created, err := r.client.CreateRule(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create agent observability evaluation rule", err.Error())
		return
	}

	model, diags := ruleToModel(ctx, created)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *evaluationRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state evaluationRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rule, err := r.client.GetRule(ctx, state.RuleID.ValueString())
	if err != nil {
		if errors.Is(err, agento11yapi.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read agent observability evaluation rule", err.Error())
		return
	}

	model, diags := ruleToModel(ctx, rule)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *evaluationRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state evaluationRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(validateRuleSelection(plan.Selector.ValueString(), plan.MinIdleSeconds, plan.FilterableTagKeys)...)
	if resp.Diagnostics.HasError() {
		return
	}

	evaluatorIDs, diags := listValueToStrings(ctx, plan.EvaluatorIDs)
	resp.Diagnostics.Append(diags...)
	alertUIDs, diags := listValueToStrings(ctx, plan.AlertRuleUIDs)
	resp.Diagnostics.Append(diags...)
	filterableTagKeys, diags := listValueToStrings(ctx, plan.FilterableTagKeys)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if filterableTagKeys == nil {
		// PATCH requires an explicit empty list to clear previously configured keys.
		filterableTagKeys = []string{}
	}

	selector := plan.Selector.ValueString()
	executionMode := plan.ExecutionMode.ValueString()
	match := rawJSONFromNormalized(plan.Match)
	if match == nil {
		// Send an explicit empty object so PATCH clears a previously-set match.
		match = []byte("{}")
	}
	patch := agento11yapi.RulePatch{
		Enabled:           util.Ptr(plan.Enabled.ValueBool()),
		Selector:          &selector,
		Match:             match,
		SampleRate:        util.Ptr(plan.SampleRate.ValueFloat64()),
		EvaluatorIDs:      evaluatorIDs,
		ExecutionMode:     &executionMode,
		AlertRuleUIDs:     alertUIDs,
		FilterableTagKeys: filterableTagKeys,
	}
	if selector == selectorConversation && !plan.MinIdleSeconds.IsNull() {
		patch.MinIdleSeconds = util.Ptr(int(plan.MinIdleSeconds.ValueInt64()))
	}

	updated, err := r.client.UpdateRule(ctx, state.RuleID.ValueString(), patch)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update agent observability evaluation rule", err.Error())
		return
	}

	model, diags := ruleToModel(ctx, updated)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *evaluationRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state evaluationRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteRule(ctx, state.RuleID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete agent observability evaluation rule", err.Error())
		return
	}
}

func (r *evaluationRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	rule, err := r.client.GetRule(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import agent observability evaluation rule", err.Error())
		return
	}
	model, diags := ruleToModel(ctx, rule)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func ruleToModel(ctx context.Context, rule agento11yapi.Rule) (evaluationRuleModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	evaluatorIDs, d := stringsToListValue(ctx, rule.EvaluatorIDs)
	diags.Append(d...)
	alertUIDs, d := stringsToListValue(ctx, rule.AlertRuleUIDs)
	diags.Append(d...)
	filterableTagKeys, d := stringsToListValue(ctx, rule.FilterableTagKeys)
	diags.Append(d...)

	return evaluationRuleModel{
		ID:                types.StringValue(rule.RuleID),
		RuleID:            types.StringValue(rule.RuleID),
		Enabled:           types.BoolValue(rule.Enabled),
		Selector:          types.StringValue(rule.Selector),
		Match:             normalizedMatchOrNull(rule.Match),
		SampleRate:        types.Float64Value(rule.SampleRate),
		EvaluatorIDs:      evaluatorIDs,
		ExecutionMode:     types.StringValue(rule.ExecutionMode),
		AlertRuleUIDs:     alertUIDs,
		FilterableTagKeys: filterableTagKeys,
		MinIdleSeconds:    int64PtrValue(rule.MinIdleSeconds),
	}, diags
}
