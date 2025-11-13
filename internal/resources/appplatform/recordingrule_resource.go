package appplatform

import (
	"context"
	"encoding/json"

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
		"expressions":           types.MapType{ElemType: types.StringType},
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
						ElementType: types.StringType,
						Description: "A sequence of stages that describe the contents of the rule. Each value is a JSON string representing an expression object.",
					},
					"paused": schema.BoolAttribute{
						Optional:    true,
						Description: "Sets whether the recording rule should be paused or not.",
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
				SpecBlocks: map[string]schema.Block{
					"trigger": schema.SingleNestedBlock{
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
		// Parse map[string]string where each string is a JSON-encoded expression
		spec.Expressions = make(map[string]v0alpha1.RecordingRuleExpression)
		for ref, val := range data.Expressions.Elements() {
			strVal, ok := val.(types.String)
			if !ok || strVal.IsNull() || strVal.IsUnknown() {
				continue
			}

			// Parse JSON string to expression data
			var exprJSON map[string]interface{}
			if err := json.Unmarshal([]byte(strVal.ValueString()), &exprJSON); err != nil {
				return diag.Diagnostics{
					diag.NewErrorDiagnostic("Failed to parse expression JSON", err.Error()),
				}
			}

			// Convert JSON to expression object
			exprObj, d := convertJSONToRecordingRuleExpression(ctx, exprJSON)
			if d.HasError() {
				return d
			}
			spec.Expressions[ref] = exprObj
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
	if src.Spec.Paused != nil {
		values["paused"] = types.BoolValue(*src.Spec.Paused)
	} else {
		values["paused"] = types.BoolNull()
	}
	values["metric"] = types.StringValue(src.Spec.Metric)
	if src.Spec.Labels != nil {
		labels, d := types.MapValueFrom(ctx, types.StringType, src.Spec.Labels)
		if d.HasError() {
			return d
		}
		values["labels"] = labels
	} else {
		values["labels"] = types.MapNull(types.StringType)
	}
	values["target_datasource_uid"] = types.StringValue(src.Spec.TargetDatasourceUID)

	if len(src.Spec.Expressions) > 0 {
		// Convert expressions to map[string]string where each string is JSON
		expressionsMap := make(map[string]attr.Value)
		for ref, expr := range src.Spec.Expressions {
			// Marshal expression to JSON
			jsonBytes, err := json.Marshal(expr)
			if err != nil {
				return diag.Diagnostics{
					diag.NewErrorDiagnostic("Failed to marshal expression to JSON", err.Error()),
				}
			}
			expressionsMap[ref] = types.StringValue(string(jsonBytes))
		}
		mapValue, d := types.MapValue(types.StringType, expressionsMap)
		if d.HasError() {
			return d
		}
		values["expressions"] = mapValue
	} else {
		// Set to null if no expressions
		values["expressions"] = types.MapNull(types.StringType)
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

// convertJSONToRecordingRuleExpression converts a JSON map to a RecordingRuleExpression
func convertJSONToRecordingRuleExpression(ctx context.Context, exprJSON map[string]interface{}) (v0alpha1.RecordingRuleExpression, diag.Diagnostics) {
	dstExpr := v0alpha1.RecordingRuleExpression{}

	// Extract model
	if model, ok := exprJSON["model"].(map[string]interface{}); ok {
		dstExpr.Model = model
	}

	// Extract query_type
	if queryType, ok := exprJSON["query_type"].(string); ok && queryType != "" {
		dstExpr.QueryType = util.Ptr(queryType)
	}

	// Extract datasource_uid
	if datasourceUID, ok := exprJSON["datasource_uid"].(string); ok && datasourceUID != "" {
		dstExpr.DatasourceUID = util.Ptr(v0alpha1.RecordingRuleDatasourceUID(datasourceUID))
	}

	// Extract source
	if source, ok := exprJSON["source"].(bool); ok {
		dstExpr.Source = util.Ptr(source)
	}

	// Extract relative_time_range
	if relTimeRange, ok := exprJSON["relative_time_range"].(map[string]interface{}); ok {
		from, _ := relTimeRange["from"].(string)
		to, _ := relTimeRange["to"].(string)
		if from != "" || to != "" {
			dstExpr.RelativeTimeRange = &v0alpha1.RecordingRuleRelativeTimeRange{
				From: v0alpha1.RecordingRulePromDurationWMillis(from),
				To:   v0alpha1.RecordingRulePromDurationWMillis(to),
			}
		}
	}

	return dstExpr, diag.Diagnostics{}
}
