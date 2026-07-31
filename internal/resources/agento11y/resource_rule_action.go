package agento11y

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common/agento11yapi"
	"github.com/grafana/terraform-provider-grafana/v4/internal/util"
)

const resourceRuleActionName = "grafana_agento11y_rule_action"

// actionConfigKindAddToCollection is the only rule action config kind currently
// supported by the API.
const actionConfigKindAddToCollection = "add_to_collection"

var resourceRuleActionID = common.NewResourceID(
	common.StringIDField("rule_id"),
	common.StringIDField("action_id"),
)

type ruleActionResource struct {
	client *agento11yapi.Client
}

type ruleActionModel struct {
	ID            types.String `tfsdk:"id"`
	RuleID        types.String `tfsdk:"rule_id"`
	Condition     types.String `tfsdk:"condition"`
	CollectionIDs types.List   `tfsdk:"collection_ids"`
	Enabled       types.Bool   `tfsdk:"enabled"`
}

func makeResourceRuleAction() *common.Resource {
	return common.NewResource(
		common.CategoryAgentObservability,
		resourceRuleActionName,
		resourceRuleActionID,
		&ruleActionResource{},
	)
}

func (r *ruleActionResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceRuleActionName
}

func (r *ruleActionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an action attached to a Grafana Agent Observability evaluation rule. When the rule's aggregate verdict matches the configured condition, matching conversations are added to one or more collections.\n\nRequires a Grafana instance with the `grafana-agento11y-app` plugin installed.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The server-generated action identifier.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"rule_id": schema.StringAttribute{
				Description: "ID of the evaluation rule this action is attached to. Changing this forces a new resource.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"condition": schema.StringAttribute{
				Description: "Aggregate verdict that triggers the action. One of `all_evaluators_pass`, `all_evaluators_fail`.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("all_evaluators_pass", "all_evaluators_fail"),
				},
			},
			"collection_ids": schema.ListAttribute{
				Description: "IDs of the collections that matching conversations are added to. Must be non-empty.",
				Required:    true,
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the action is enabled. Defaults to `true`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
		},
	}
}

func (r *ruleActionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil || r.client != nil {
		return
	}
	client, err := withClientForResource(req, resp)
	if err != nil {
		return
	}
	r.client = client
}

func (r *ruleActionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ruleActionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	collectionIDs, diags := listValueToStrings(ctx, plan.CollectionIDs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateRuleAction(ctx, plan.RuleID.ValueString(), agento11yapi.RuleActionCreate{
		Condition: agento11yapi.RuleActionCondition{Kind: plan.Condition.ValueString()},
		ActionConfig: agento11yapi.RuleActionConfig{
			Kind:          actionConfigKindAddToCollection,
			CollectionIDs: collectionIDs,
		},
		Enabled: util.Ptr(plan.Enabled.ValueBool()),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create agent observability rule action", err.Error())
		return
	}

	model, diags := ruleActionToModel(ctx, created)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *ruleActionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ruleActionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	action, err := r.client.GetRuleAction(ctx, state.RuleID.ValueString(), state.ID.ValueString())
	if err != nil {
		if errors.Is(err, agento11yapi.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read agent observability rule action", err.Error())
		return
	}

	model, diags := ruleActionToModel(ctx, action)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *ruleActionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ruleActionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	collectionIDs, diags := listValueToStrings(ctx, plan.CollectionIDs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.client.UpdateRuleAction(ctx, state.RuleID.ValueString(), state.ID.ValueString(), agento11yapi.RuleActionUpdate{
		Condition: &agento11yapi.RuleActionCondition{Kind: plan.Condition.ValueString()},
		ActionConfig: &agento11yapi.RuleActionConfig{
			Kind:          actionConfigKindAddToCollection,
			CollectionIDs: collectionIDs,
		},
		Enabled: util.Ptr(plan.Enabled.ValueBool()),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to update agent observability rule action", err.Error())
		return
	}

	model, diags := ruleActionToModel(ctx, updated)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *ruleActionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ruleActionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteRuleAction(ctx, state.RuleID.ValueString(), state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete agent observability rule action", err.Error())
		return
	}
}

func (r *ruleActionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected import ID in the form \"{rule_id}:{action_id}\", got %q.", req.ID),
		)
		return
	}

	action, err := r.client.GetRuleAction(ctx, parts[0], parts[1])
	if err != nil {
		resp.Diagnostics.AddError("Failed to import agent observability rule action", err.Error())
		return
	}
	model, diags := ruleActionToModel(ctx, action)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func ruleActionToModel(ctx context.Context, action agento11yapi.RuleAction) (ruleActionModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	collectionIDs, d := stringsToListValue(ctx, action.ActionConfig.CollectionIDs)
	diags.Append(d...)

	return ruleActionModel{
		ID:            types.StringValue(action.ActionID),
		RuleID:        types.StringValue(action.RuleID),
		Condition:     types.StringValue(action.Condition.Kind),
		CollectionIDs: collectionIDs,
		Enabled:       types.BoolValue(action.Enabled),
	}, diags
}
