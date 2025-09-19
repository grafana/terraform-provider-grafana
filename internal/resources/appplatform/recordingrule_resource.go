package appplatform

import (
	"context"

	"github.com/grafana/grafana/apps/alerting/rules/pkg/apis/alerting/v0alpha1"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/util"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var recordingRuleSpecType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"title":                 types.StringType,
		"data":                  types.MapType{ElemType: ruleExpressionType},
		"paused":                types.BoolType,
		"trigger":               ruleTriggerType,
		"metric":                types.StringType,
		"labels":                types.MapType{ElemType: types.StringType},
		"target_datasource_uid": types.StringType,
	},
}

type RecordingRuleSpecModel struct {
	Title               types.String `tfsdk:"title"`
	Expressions         types.Map    `tfsdk:"expressions"`
	Paused              types.Bool   `tfsdk:"paused"`
	Trigger             types.Object `tfsdk:"trigger"`
	Metric              types.String `tfsdk:"metric"`
	Labels              types.Map    `tfsdk:"labels"`
	TargetDatasourceUID types.String `tfsdk:"target_datasource_uid"`
}

func RecordingRule() NamedResource {
	return NewNamedResource[*v0alpha1.RecordingRule, *v0alpha1.RecordingRuleList](
		common.CategoryAlerting,
		ResourceConfig[*v0alpha1.RecordingRule]{
			Kind: v0alpha1.RecordingRuleKind(),
			Schema: ResourceSpecSchema{
				Description: "Manages Grafana Recording Rules.",
				MarkdownDescription: `
Manages Grafana Recording Rules.
`,
				SpecAttributes: map[string]schema.Attribute{
					"title": schema.StringAttribute{
						Required:    true,
						Description: "The title of the recording rule.",
					},
					"expressions": schema.MapAttribute{
						Required:    true,
						Description: "A sequence of stages that describe the contents of the rule.",
						ElementType: ruleExpressionType,
						Validators: []validator.Map{
							ExpressionMapValidator{},
						},
					},
					"paused": schema.BoolAttribute{
						Optional:    true,
						Description: "Sets whether the recording rule should be paused or not.",
					},
					"trigger": schema.SingleNestedAttribute{
						Required:    true,
						Description: "The trigger configuration for the recording rule.",
						Attributes: map[string]schema.Attribute{
							"interval": schema.StringAttribute{
								Required:    true,
								Description: "The interval at which the recording rule should be evaluated.",
								Validators: []validator.String{
									PrometheusDurationValidator{},
								},
							},
						},
					},
					"metric": schema.StringAttribute{
						Required:    true,
						Description: "The name of the metric to write to.",
					},
					"labels": schema.MapAttribute{
						Optional:    true,
						ElementType: types.StringType,
						Description: "Key-value pairs to attach to the recorded metric.",
					},
					"target_datasource_uid": schema.StringAttribute{
						Required:    true,
						Description: "The UID of the datasource to write the metric to.",
					},
				},
			},
			SpecParser: parseRecordingRuleSpec,
			SpecSaver:  saveRecordingRuleSpec,
		})
}

func parseRecordingRuleSpec(ctx context.Context, src types.Object, dst *v0alpha1.RecordingRule) diag.Diagnostics {
	var data RecordingRuleSpecModel
	if diag := src.As(ctx, &data, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	}); diag.HasError() {
		return diag
	}

	spec := v0alpha1.RecordingRuleSpec{
		Title: data.Title.ValueString(),
	}

	if !data.Paused.IsNull() && !data.Paused.IsUnknown() {
		spec.Paused = util.Ptr(data.Paused.ValueBool())
	}

	if !data.Trigger.IsNull() && !data.Trigger.IsUnknown() {
		trigger, diags := parseRecordingRuleTrigger(ctx, data.Trigger)
		if diags.HasError() {
			return diags
		}
		spec.Trigger = trigger
	}

	if !data.Metric.IsNull() && !data.Metric.IsUnknown() {
		spec.Metric = data.Metric.ValueString()
	}

	if !data.Labels.IsNull() && !data.Labels.IsUnknown() {
		labels := make(map[string]string)
		if diag := data.Labels.ElementsAs(ctx, &labels, false); diag.HasError() {
			return diag
		}
		spec.Labels = make(map[string]v0alpha1.RecordingRuleTemplateString)
		for k, v := range labels {
			spec.Labels[k] = v0alpha1.RecordingRuleTemplateString(v)
		}
	}

	if !data.TargetDatasourceUID.IsNull() && !data.TargetDatasourceUID.IsUnknown() {
		spec.TargetDatasourceUID = data.TargetDatasourceUID.ValueString()
	}

	if !data.Expressions.IsNull() && !data.Expressions.IsUnknown() {
		for ref, rawVal := range data.Expressions.Elements() {
			obj, ok := rawVal.(types.Object)
			if !ok {
				return diag.Diagnostics{
					diag.NewErrorDiagnostic("Invalid data", "Expression is not an object"),
				}
			}
			data, diags := parseRecordingRuleDataModel(ctx, obj)
			if diags.HasError() {
				return diags
			}
			spec.Expressions[ref] = data
		}
	}

	if err := dst.SetSpec(spec); err != nil {
		return diag.Diagnostics{
			diag.NewErrorDiagnostic("failed to set spec", err.Error()),
		}
	}

	return diag.Diagnostics{}
}

func saveRecordingRuleSpec(ctx context.Context, src *v0alpha1.RecordingRule, dst *ResourceModel) diag.Diagnostics {
	values := make(map[string]attr.Value)

	values["title"] = types.StringValue(src.Spec.Title)
	trigger, d := types.ObjectValueFrom(ctx, ruleTriggerType.AttrTypes, src.Spec.Trigger)
	if d.HasError() {
		return d
	}
	values["trigger"] = trigger
	values["paused"] = types.BoolValue(*src.Spec.Paused)
	values["metric"] = types.StringValue(src.Spec.Metric)
	if src.Spec.Labels != nil {
		labels, d := types.MapValueFrom(ctx, types.StringType, src.Spec.Labels)
		if d.HasError() {
			return d
		}
		values["labels"] = labels
	}
	values["target_datasource_uid"] = types.StringValue(src.Spec.TargetDatasourceUID)

	if len(src.Spec.Expressions) > 0 {
		data, d := types.MapValueFrom(ctx, ruleExpressionType, src.Spec.Expressions)
		if d.HasError() {
			return d
		}
		values["expressions"] = data
	}

	spec, d := types.ObjectValue(recordingRuleSpecType.AttrTypes, values)
	if d.HasError() {
		return d
	}
	dst.Spec = spec
	return diag.Diagnostics{}
}

// Parser helpers

func parseRecordingRuleTrigger(ctx context.Context, src types.Object) (v0alpha1.RecordingRuleIntervalTrigger, diag.Diagnostics) {
	var data RuleTriggerModel
	if diag := src.As(ctx, &data, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	}); diag.HasError() {
		return v0alpha1.RecordingRuleIntervalTrigger{}, diag
	}
	return v0alpha1.RecordingRuleIntervalTrigger{
		Interval: v0alpha1.RecordingRulePromDuration(data.Interval.ValueString()),
	}, diag.Diagnostics{}
}

func parseRecordingRuleRelativeTimeRange(ctx context.Context, src types.Object) (v0alpha1.RecordingRuleRelativeTimeRange, diag.Diagnostics) {
	var data RelativeTimeRangeModel
	if diag := src.As(ctx, &data, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	}); diag.HasError() {
		return v0alpha1.RecordingRuleRelativeTimeRange{}, diag
	}
	return v0alpha1.RecordingRuleRelativeTimeRange{
		From: v0alpha1.RecordingRulePromDurationWMillis(data.From.ValueString()),
		To:   v0alpha1.RecordingRulePromDurationWMillis(data.To.ValueString()),
	}, diag.Diagnostics{}
}

func parseRecordingRuleDataModel(ctx context.Context, src types.Object) (v0alpha1.RecordingRuleExpression, diag.Diagnostics) {
	var srcExpr RuleExpressionModel
	if diag := src.As(ctx, &srcExpr, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	}); diag.HasError() {
		return v0alpha1.RecordingRuleExpression{}, diag
	}

	relativeTimeRange, diags := parseRecordingRuleRelativeTimeRange(ctx, srcExpr.RelativeTimeRange)
	if diags.HasError() {
		return v0alpha1.RecordingRuleExpression{}, diags
	}

	dstExpr := v0alpha1.RecordingRuleExpression{
		Model: srcExpr.Model.ValueString(),
		RelativeTimeRange: &v0alpha1.RecordingRuleRelativeTimeRange{
			From: relativeTimeRange.From,
			To:   relativeTimeRange.To,
		},
	}
	if srcExpr.QueryType.ValueString() != "" {
		dstExpr.QueryType = util.Ptr(srcExpr.QueryType.ValueString())
	}
	if srcExpr.DatasourceUid.ValueString() != "" {
		dstExpr.DatasourceUID = util.Ptr(v0alpha1.RecordingRuleDatasourceUID(srcExpr.DatasourceUid.ValueString()))
	}
	if srcExpr.Source.ValueBool() {
		dstExpr.Source = util.Ptr(srcExpr.Source.ValueBool())
	}
	return dstExpr, diag.Diagnostics{}
}
