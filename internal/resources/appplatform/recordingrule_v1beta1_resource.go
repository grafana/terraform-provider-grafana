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

// RecordingRuleV1Beta1 creates a new Grafana Recording Rule resource backed by the v1beta1 API.
func RecordingRuleV1Beta1() NamedResource {
	spec := recordingRuleSpecSchema()
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
		expressions, diags := parseRecordingRuleExpressionsV1Beta1(data.Expressions)
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

func parseRecordingRuleExpressionsV1Beta1(src types.Map) (v1beta1.RecordingRuleExpressionMap, diag.Diagnostics) {
	expressions := make(v1beta1.RecordingRuleExpressionMap)
	for ref, val := range src.Elements() {
		exprJSON, ok, diags := parseExpressionJSON(val)
		if diags.HasError() {
			return nil, diags
		}
		if !ok {
			continue
		}

		expr := v1beta1.RecordingRuleExpression{}
		if model, ok := exprJSON["model"].(map[string]any); ok {
			expr.Model = model
		}
		if queryType, ok := exprJSON["query_type"].(string); ok && queryType != "" {
			expr.QueryType = util.Ptr(queryType)
		}
		if datasourceUID, ok := exprJSON["datasource_uid"].(string); ok && datasourceUID != "" {
			expr.DatasourceUID = util.Ptr(v1beta1.RecordingRuleDatasourceUID(datasourceUID))
		}
		if source, ok := exprJSON["source"].(bool); ok {
			expr.Source = util.Ptr(source)
		}
		if relTimeRange, ok := exprJSON["relative_time_range"].(map[string]any); ok {
			from, _ := relTimeRange["from"].(string)
			to, _ := relTimeRange["to"].(string)
			if from != "" || to != "" {
				expr.RelativeTimeRange = &v1beta1.RecordingRuleRelativeTimeRange{
					From: v1beta1.RecordingRulePromDurationWMillis(from),
					To:   v1beta1.RecordingRulePromDurationWMillis(to),
				}
			}
		}
		expressions[ref] = expr
	}
	return expressions, nil
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

	expressions, diags := saveExpressionsJSON(src.Spec.Expressions)
	if diags.HasError() {
		return diags
	}
	values["expressions"] = expressions

	spec, diags := types.ObjectValue(recordingRuleSpecType.AttrTypes, values)
	if diags.HasError() {
		return diags
	}
	dst.Spec = spec
	return nil
}
