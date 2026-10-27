package appplatform

import (
	"context"

	"github.com/grafana/grafana/apps/alerting/rules/pkg/apis/alerting/v1beta1"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// v1beta1 renamed the rule sequence spec field alertingRules to alertRules. The Terraform
// attribute follows it, from alerting_rules to alert_rules.
const ruleSequenceV1Beta1AlertRulesAttr = "alert_rules"

var ruleSequenceSpecV1Beta1Type = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"trigger":         ruleTriggerType,
		"recording_rules": types.ListType{ElemType: ruleSequenceRuleRefType},
		"alert_rules":     types.ListType{ElemType: ruleSequenceRuleRefType},
	},
}

type ruleSequenceSpecV1Beta1Model struct {
	Trigger        types.Object `tfsdk:"trigger"`
	RecordingRules types.List   `tfsdk:"recording_rules"`
	AlertRules     types.List   `tfsdk:"alert_rules"`
}

// RuleSequenceV1Beta1 creates a new Grafana Rule Sequence resource backed by the v1beta1 API.
func RuleSequenceV1Beta1() NamedResource {
	spec := ruleSequenceSpecSchema(ruleSequenceV1Beta1AlertRulesAttr)
	spec.Description = "Manages Grafana Rule Sequences."
	spec.MarkdownDescription = `
Manages Grafana Rule Sequences.

A rule sequence groups existing alert rules and recording rules so they are evaluated in order, at a shared interval, within a single folder. All referenced rules must live in the same folder as the sequence (set ` + "`metadata.folder_uid`" + `).

This resource uses the ` + "`rules.alerting.grafana.app/v1beta1`" + ` API, which requires Grafana 13.3 or later.
`

	return NewNamedResource[*v1beta1.RuleSequence, *v1beta1.RuleSequenceList](
		common.CategoryAlerting,
		ResourceConfig[*v1beta1.RuleSequence]{
			Kind:       v1beta1.RuleSequenceKind(),
			Schema:     spec,
			SpecParser: parseRuleSequenceV1Beta1Spec,
			SpecSaver:  saveRuleSequenceV1Beta1Spec,
		})
}

func parseRuleSequenceV1Beta1Spec(ctx context.Context, src types.Object, dst *v1beta1.RuleSequence) diag.Diagnostics {
	var data ruleSequenceSpecV1Beta1Model
	if diags := src.As(ctx, &data, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	}); diags.HasError() {
		return diags
	}

	spec := v1beta1.RuleSequenceSpec{}

	if isNotNullOrUknown(data.Trigger) {
		var trigger RuleTriggerModel
		if diags := data.Trigger.As(ctx, &trigger, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty:    true,
			UnhandledUnknownAsEmpty: true,
		}); diags.HasError() {
			return diags
		}
		spec.Trigger = v1beta1.RuleSequenceIntervalTrigger{
			Interval: v1beta1.RuleSequencePromDuration(trigger.Interval.ValueString()),
		}
	}

	if isNotNullOrUknown(data.RecordingRules) {
		refs, diags := parseRuleSequenceRefsV1Beta1(ctx, data.RecordingRules)
		if diags.HasError() {
			return diags
		}
		spec.RecordingRules = refs
	}

	if isNotNullOrUknown(data.AlertRules) {
		refs, diags := parseRuleSequenceRefsV1Beta1(ctx, data.AlertRules)
		if diags.HasError() {
			return diags
		}
		spec.AlertRules = refs
	}

	if err := dst.SetSpec(spec); err != nil {
		return diag.Diagnostics{diag.NewErrorDiagnostic("failed to set spec", err.Error())}
	}
	return nil
}

func parseRuleSequenceRefsV1Beta1(ctx context.Context, src types.List) ([]v1beta1.RuleSequenceRuleRef, diag.Diagnostics) {
	var models []ruleSequenceRuleRefModel
	if diags := src.ElementsAs(ctx, &models, false); diags.HasError() {
		return nil, diags
	}

	refs := make([]v1beta1.RuleSequenceRuleRef, 0, len(models))
	for _, m := range models {
		refs = append(refs, v1beta1.RuleSequenceRuleRef{
			Name: v1beta1.RuleSequenceRuleUID(m.Name.ValueString()),
		})
	}
	return refs, nil
}

func saveRuleSequenceV1Beta1Spec(ctx context.Context, src *v1beta1.RuleSequence, dst *ResourceModel) diag.Diagnostics {
	values := make(map[string]attr.Value)

	trigger, diags := types.ObjectValueFrom(ctx, ruleTriggerType.AttrTypes, RuleTriggerModel{
		Interval: types.StringValue(string(src.Spec.Trigger.Interval)),
	})
	if diags.HasError() {
		return diags
	}
	values["trigger"] = trigger

	recordingRules, diags := ruleSequenceRefsV1Beta1ToTf(ctx, src.Spec.RecordingRules)
	if diags.HasError() {
		return diags
	}
	values["recording_rules"] = recordingRules

	values["alert_rules"] = types.ListNull(ruleSequenceRuleRefType)
	if len(src.Spec.AlertRules) > 0 {
		alertRules, diags := ruleSequenceRefsV1Beta1ToTf(ctx, src.Spec.AlertRules)
		if diags.HasError() {
			return diags
		}
		values["alert_rules"] = alertRules
	}

	spec, diags := types.ObjectValue(ruleSequenceSpecV1Beta1Type.AttrTypes, values)
	if diags.HasError() {
		return diags
	}
	dst.Spec = spec
	return nil
}

func ruleSequenceRefsV1Beta1ToTf(ctx context.Context, refs []v1beta1.RuleSequenceRuleRef) (types.List, diag.Diagnostics) {
	models := make([]ruleSequenceRuleRefModel, 0, len(refs))
	for _, ref := range refs {
		models = append(models, ruleSequenceRuleRefModel{
			Name: types.StringValue(string(ref.Name)),
		})
	}
	return types.ListValueFrom(ctx, ruleSequenceRuleRefType, models)
}
