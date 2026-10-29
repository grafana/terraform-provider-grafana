package appplatform

import (
	"context"

	"github.com/grafana/grafana/apps/alerting/rules/pkg/apis/alerting/v1beta1"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/util"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var recordingRuleSpecV1Beta1Type = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"title":                 types.StringType,
		"expressions":           types.MapType{ElemType: ruleExpressionType},
		"paused":                types.BoolType,
		"trigger":               ruleTriggerType,
		"metric":                types.StringType,
		"labels":                types.MapType{ElemType: types.StringType},
		"target_datasource_uid": types.StringType,
	},
}

// RecordingRuleV1Beta1 creates a new Grafana Recording Rule resource backed by the v1beta1 API.
func RecordingRuleV1Beta1() NamedResource {
	spec := recordingRuleSpecSchema()
	spec.SpecAttributes["expressions"] = ruleExpressionsV1Beta1Attribute()
	spec.Description = "Manages Grafana Recording Rules."
	spec.MarkdownDescription = `
Manages Grafana Recording Rules.

This resource uses the ` + "`rules.alerting.grafana.app/v1beta1`" + ` API, which requires Grafana 13.3 or later.
`

	return NewNamedResource[*v1beta1.RecordingRule, *v1beta1.RecordingRuleList](
		common.CategoryAlerting,
		ResourceConfig[*v1beta1.RecordingRule]{
			Kind:       v1beta1.RecordingRuleKind(),
			Schema:     spec,
			SpecParser: parseRecordingRuleV1Beta1Spec,
			SpecSaver:  saveRecordingRuleV1Beta1Spec,
		})
}

func parseRecordingRuleV1Beta1Spec(ctx context.Context, src types.Object, dst *v1beta1.RecordingRule) diag.Diagnostics {
	var data RecordingRuleSpecModel
	if diags := src.As(ctx, &data, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	}); diags.HasError() {
		return diags
	}

	spec := v1beta1.RecordingRuleSpec{
		Title:               data.Title.ValueString(),
		Metric:              v1beta1.RecordingRuleMetricName(data.Metric.ValueString()),
		TargetDatasourceUID: v1beta1.RecordingRuleDatasourceUID(data.TargetDatasourceUID.ValueString()),
	}

	if isNotNullOrUknown(data.Paused) {
		spec.Paused = util.Ptr(data.Paused.ValueBool())
	}

	if isNotNullOrUknown(data.Trigger) {
		var trigger RuleTriggerModel
		if diags := data.Trigger.As(ctx, &trigger, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty:    true,
			UnhandledUnknownAsEmpty: true,
		}); diags.HasError() {
			return diags
		}
		spec.Trigger = v1beta1.RecordingRuleIntervalTrigger{
			Interval: v1beta1.RecordingRulePromDuration(trigger.Interval.ValueString()),
		}
	}

	if isNotNullOrUknown(data.Labels) {
		labels, diags := parseTemplateStringMap[v1beta1.RecordingRuleTemplateString](ctx, data.Labels)
		if diags.HasError() {
			return diags
		}
		spec.Labels = labels
	}

	if isNotNullOrUknown(data.Expressions) {
		expressions, diags := parseRecordingRuleExpressionsV1Beta1(ctx, data.Expressions)
		if diags.HasError() {
			return diags
		}
		spec.Expressions = expressions
	}

	if err := dst.SetSpec(spec); err != nil {
		return diag.Diagnostics{diag.NewErrorDiagnostic("failed to set spec", err.Error())}
	}
	return nil
}

func parseRecordingRuleExpressionsV1Beta1(ctx context.Context, src types.Map) (v1beta1.RecordingRuleExpressionMap, diag.Diagnostics) {
	parsed, diags := parseRuleExpressionsV1Beta1(ctx, src)
	if diags.HasError() {
		return nil, diags
	}

	expressions := make(v1beta1.RecordingRuleExpressionMap, len(parsed))
	for ref, expr := range parsed {
		result := v1beta1.RecordingRuleExpression{
			QueryType: expr.QueryType,
			Model:     expr.Model,
			Source:    expr.Source,
		}
		if expr.DatasourceUID != nil {
			result.DatasourceUID = util.Ptr(v1beta1.RecordingRuleDatasourceUID(*expr.DatasourceUID))
		}
		if expr.RelativeTimeRange != nil {
			result.RelativeTimeRange = &v1beta1.RecordingRuleRelativeTimeRange{
				From: v1beta1.RecordingRulePromDurationWMillis(expr.RelativeTimeRange.From),
				To:   v1beta1.RecordingRulePromDurationWMillis(expr.RelativeTimeRange.To),
			}
		}
		expressions[ref] = result
	}
	return expressions, nil
}

func saveRecordingRuleExpressionsV1Beta1(src v1beta1.RecordingRuleExpressionMap) (types.Map, diag.Diagnostics) {
	expressions := make(map[string]ruleExpressionV1Beta1, len(src))
	for ref, expr := range src {
		result := ruleExpressionV1Beta1{
			QueryType: expr.QueryType,
			Model:     expr.Model,
			Source:    expr.Source,
		}
		if expr.DatasourceUID != nil {
			result.DatasourceUID = util.Ptr(string(*expr.DatasourceUID))
		}
		if expr.RelativeTimeRange != nil {
			result.RelativeTimeRange = &ruleRelativeTimeRangeV1Beta1{
				From: string(expr.RelativeTimeRange.From),
				To:   string(expr.RelativeTimeRange.To),
			}
		}
		expressions[ref] = result
	}
	return saveRuleExpressionsV1Beta1(expressions)
}

func saveRecordingRuleV1Beta1Spec(ctx context.Context, src *v1beta1.RecordingRule, dst *ResourceModel) diag.Diagnostics {
	values := map[string]attr.Value{
		"title":                 types.StringValue(src.Spec.Title),
		"paused":                types.BoolPointerValue(src.Spec.Paused),
		"metric":                types.StringValue(string(src.Spec.Metric)),
		"target_datasource_uid": types.StringValue(string(src.Spec.TargetDatasourceUID)),
	}

	trigger, diags := types.ObjectValueFrom(ctx, ruleTriggerType.AttrTypes, RuleTriggerModel{
		Interval: types.StringValue(string(src.Spec.Trigger.Interval)),
	})
	if diags.HasError() {
		return diags
	}
	values["trigger"] = trigger

	labels, diags := saveTemplateStringMap(ctx, src.Spec.Labels)
	if diags.HasError() {
		return diags
	}
	values["labels"] = labels

	expressions, diags := saveRecordingRuleExpressionsV1Beta1(src.Spec.Expressions)
	if diags.HasError() {
		return diags
	}
	values["expressions"] = expressions

	spec, diags := types.ObjectValue(recordingRuleSpecV1Beta1Type.AttrTypes, values)
	if diags.HasError() {
		return diags
	}
	dst.Spec = spec
	return nil
}
