package assistant

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common/assistantapi"
	"github.com/grafana/terraform-provider-grafana/v4/internal/util"
)

const resourceRuleName = "grafana_assistant_rule"

var resourceRuleID = common.NewResourceID(common.StringIDField("id"))

type ruleResource struct {
	client *assistantapi.Client
}

type ruleModel struct {
	ID           types.String `tfsdk:"id"`
	Scope        types.String `tfsdk:"scope"`
	Name         types.String `tfsdk:"name"`
	Description  types.String `tfsdk:"description"`
	RuleContent  types.String `tfsdk:"rule_content"`
	Enabled      types.Bool   `tfsdk:"enabled"`
	Priority     types.Int64  `tfsdk:"priority"`
	Applications types.List   `tfsdk:"applications"`
}

func makeResourceRule() *common.Resource {
	return common.NewResource(
		common.CategoryGrafanaAssistant,
		resourceRuleName,
		resourceRuleID,
		&ruleResource{},
	)
}

func (r *ruleResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceRuleName
}

func (r *ruleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Grafana Assistant rule that is injected into the assistant system prompt.",
		Attributes: map[string]schema.Attribute{
			"id":          idAttribute(),
			"scope":       scopeAttribute(),
			"name":        schema.StringAttribute{Description: "The rule name.", Required: true},
			"description": schema.StringAttribute{Description: "Optional description of the rule.", Optional: true},
			"rule_content": schema.StringAttribute{
				Description: "The rule text included in the assistant system prompt.",
				Required:    true,
			},
			"enabled":      enabledAttribute(),
			"priority":     schema.Int64Attribute{Description: "Rule priority (lower values apply first).", Optional: true, Computed: true, Default: int64default.StaticInt64(0)},
			"applications": applicationsAttribute(),
		},
	}
}

func (r *ruleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil || r.client != nil {
		return
	}
	client, err := withClientForResource(req, resp)
	if err != nil {
		return
	}
	r.client = client
}

func (r *ruleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ruleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	applications, diags := listValueToStrings(ctx, plan.Applications)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	enabled := plan.Enabled.ValueBool()
	created, err := r.client.CreateRule(ctx, assistantapi.RuleCreate{
		Scope:        plan.Scope.ValueString(),
		Name:         plan.Name.ValueString(),
		Description:  plan.Description.ValueString(),
		RuleContent:  plan.RuleContent.ValueString(),
		Enabled:      util.Ptr(enabled),
		Priority:     int(plan.Priority.ValueInt64()),
		Applications: applications,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create assistant rule", err.Error())
		return
	}

	state, diags := ruleToModel(ctx, created)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *ruleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ruleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rule, err := r.client.GetRule(ctx, state.ID.ValueString())
	if err != nil {
		if errors.Is(err, assistantapi.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read assistant rule", err.Error())
		return
	}

	model, diags := ruleToModel(ctx, rule)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *ruleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ruleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	applications, diags := listValueToStrings(ctx, plan.Applications)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	enabled := plan.Enabled.ValueBool()
	priority := int(plan.Priority.ValueInt64())
	updated, err := r.client.UpdateRule(ctx, state.ID.ValueString(), state.Scope.ValueString(), assistantapi.RuleUpdate{
		Scope:        plan.Scope.ValueString(),
		Name:         util.Ptr(plan.Name.ValueString()),
		Description:  util.Ptr(plan.Description.ValueString()),
		RuleContent:  util.Ptr(plan.RuleContent.ValueString()),
		Enabled:      util.Ptr(enabled),
		Priority:     &priority,
		Applications: &applications,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to update assistant rule", err.Error())
		return
	}

	model, diags := ruleToModel(ctx, updated)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *ruleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ruleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteRule(ctx, state.ID.ValueString(), state.Scope.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete assistant rule", err.Error())
		return
	}
}

func (r *ruleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	rule, err := r.client.GetRule(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import assistant rule", err.Error())
		return
	}
	model, diags := ruleToModel(ctx, rule)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func ruleToModel(ctx context.Context, rule assistantapi.Rule) (ruleModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	applications, appDiags := stringsToListValue(ctx, rule.Applications)
	diags.Append(appDiags...)

	enabled := true
	if rule.Enabled != nil {
		enabled = *rule.Enabled
	}

	return ruleModel{
		ID:           types.StringValue(rule.ID),
		Scope:        types.StringValue(rule.Scope),
		Name:         types.StringValue(rule.Name),
		Description:  stringValueOrNull(rule.Description),
		RuleContent:  types.StringValue(rule.RuleContent),
		Enabled:      types.BoolValue(enabled),
		Priority:     types.Int64Value(int64(rule.Priority)),
		Applications: applications,
	}, diags
}
