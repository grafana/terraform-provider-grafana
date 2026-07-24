package appplatform

import (
	"context"

	"github.com/grafana/grafana/apps/alerting/rules/pkg/apis/alerting/v0alpha1"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var ruleSequenceRuleRefType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"name": types.StringType,
	},
}

type ruleSequenceRuleRefModel struct {
	Name types.String `tfsdk:"name"`
}

var ruleSequenceSpecType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"trigger":         ruleTriggerType,
		"recording_rules": types.ListType{ElemType: ruleSequenceRuleRefType},
		"alerting_rules":  types.ListType{ElemType: ruleSequenceRuleRefType},
	},
}

type ruleSequenceSpecModel struct {
	Trigger        types.Object `tfsdk:"trigger"`
	RecordingRules types.List   `tfsdk:"recording_rules"`
	AlertingRules  types.List   `tfsdk:"alerting_rules"`
}

// RuleSequence creates a new Grafana Rule Sequence resource.
func RuleSequence() NamedResource {
	return NewNamedResource[*v0alpha1.RuleSequence, *v0alpha1.RuleSequenceList](
		common.CategoryAlerting,
		ResourceConfig[*v0alpha1.RuleSequence]{
			Kind: v0alpha1.RuleSequenceKind(),
			Schema: ResourceSpecSchema{
				Description: "Manages Grafana Rule Sequences.",
				MarkdownDescription: `
Manages Grafana Rule Sequences.

A rule sequence groups existing alert rules and recording rules so they are evaluated in order, at a shared interval, within a single folder. All referenced rules must live in the same folder as the sequence (set ` + "`metadata.folder_uid`" + `).

This resource is currently in alpha and is subject to change.
`,
				SpecAttributes: map[string]schema.Attribute{
					"recording_rules": schema.ListAttribute{
						Required:    true,
						ElementType: ruleSequenceRuleRefType,
						Description: "The recording rules that belong to this sequence, evaluated in the order listed. At least one recording rule is required. Each entry references a recording rule by its `name` (the rule's UID).",
					},
					"alerting_rules": schema.ListAttribute{
						Optional:    true,
						ElementType: ruleSequenceRuleRefType,
						Description: "The alert rules that belong to this sequence, evaluated in the order listed. Each entry references an alert rule by its `name` (the rule's UID).",
					},
				},
				SpecBlocks: map[string]schema.Block{
					"trigger": schema.SingleNestedBlock{
						Description: "The trigger configuration shared by every rule in the sequence.",
						Attributes: map[string]schema.Attribute{
							"interval": schema.StringAttribute{
								Required:    true,
								Description: "The interval at which the rules in the sequence should be evaluated.",
								Validators: []validator.String{
									PrometheusDurationValidator{},
								},
							},
						},
					},
				},
			},
			SpecParser: parseRuleSequenceSpec,
			SpecSaver:  saveRuleSequenceSpec,
		})
}

func parseRuleSequenceSpec(ctx context.Context, src types.Object, dst *v0alpha1.RuleSequence) diag.Diagnostics {
	var data ruleSequenceSpecModel
	if diag := src.As(ctx, &data, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	}); diag.HasError() {
		return diag
	}

	spec := v0alpha1.RuleSequenceSpec{}

	if !data.Trigger.IsNull() && !data.Trigger.IsUnknown() {
		trigger, diags := parseRuleSequenceTrigger(ctx, data.Trigger)
		if diags.HasError() {
			return diags
		}
		spec.Trigger = trigger
	}

	if !data.RecordingRules.IsNull() && !data.RecordingRules.IsUnknown() {
		refs, diags := parseRuleSequenceRefs(ctx, data.RecordingRules)
		if diags.HasError() {
			return diags
		}
		spec.RecordingRules = refs
	}

	if !data.AlertingRules.IsNull() && !data.AlertingRules.IsUnknown() {
		refs, diags := parseRuleSequenceRefs(ctx, data.AlertingRules)
		if diags.HasError() {
			return diags
		}
		spec.AlertingRules = refs
	}

	if err := dst.SetSpec(spec); err != nil {
		return diag.Diagnostics{
			diag.NewErrorDiagnostic("failed to set spec", err.Error()),
		}
	}

	return diag.Diagnostics{}
}

func parseRuleSequenceTrigger(ctx context.Context, src types.Object) (v0alpha1.RuleSequenceIntervalTrigger, diag.Diagnostics) {
	var data RuleTriggerModel
	if diag := src.As(ctx, &data, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	}); diag.HasError() {
		return v0alpha1.RuleSequenceIntervalTrigger{}, diag
	}
	return v0alpha1.RuleSequenceIntervalTrigger{
		Interval: v0alpha1.RuleSequencePromDuration(data.Interval.ValueString()),
	}, diag.Diagnostics{}
}

func parseRuleSequenceRefs(ctx context.Context, src types.List) ([]v0alpha1.RuleSequenceRuleRef, diag.Diagnostics) {
	var models []ruleSequenceRuleRefModel
	if diag := src.ElementsAs(ctx, &models, false); diag.HasError() {
		return nil, diag
	}

	refs := make([]v0alpha1.RuleSequenceRuleRef, 0, len(models))
	for _, m := range models {
		refs = append(refs, v0alpha1.RuleSequenceRuleRef{
			Name: v0alpha1.RuleSequenceRuleUID(m.Name.ValueString()),
		})
	}
	return refs, nil
}

func saveRuleSequenceSpec(ctx context.Context, src *v0alpha1.RuleSequence, dst *ResourceModel) diag.Diagnostics {
	values := make(map[string]attr.Value)

	trigger, d := types.ObjectValueFrom(ctx, ruleTriggerType.AttrTypes, RuleTriggerModel{
		Interval: types.StringValue(string(src.Spec.Trigger.Interval)),
	})
	if d.HasError() {
		return d
	}
	values["trigger"] = trigger

	recordingRules, d := ruleSequenceRefsToTf(ctx, src.Spec.RecordingRules)
	if d.HasError() {
		return d
	}
	values["recording_rules"] = recordingRules

	if len(src.Spec.AlertingRules) > 0 {
		alertingRules, d := ruleSequenceRefsToTf(ctx, src.Spec.AlertingRules)
		if d.HasError() {
			return d
		}
		values["alerting_rules"] = alertingRules
	} else {
		values["alerting_rules"] = types.ListNull(ruleSequenceRuleRefType)
	}

	spec, d := types.ObjectValue(ruleSequenceSpecType.AttrTypes, values)
	if d.HasError() {
		return d
	}
	dst.Spec = spec

	return diag.Diagnostics{}
}

func ruleSequenceRefsToTf(ctx context.Context, refs []v0alpha1.RuleSequenceRuleRef) (types.List, diag.Diagnostics) {
	models := make([]ruleSequenceRuleRefModel, 0, len(refs))
	for _, ref := range refs {
		models = append(models, ruleSequenceRuleRefModel{
			Name: types.StringValue(string(ref.Name)),
		})
	}
	return types.ListValueFrom(ctx, ruleSequenceRuleRefType, models)
}
