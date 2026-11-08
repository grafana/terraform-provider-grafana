package appplatform

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/prometheus/common/model"
)

// The v1beta1 alert and recording rules share the expression schema, because the API's
// AlertRuleExpression and RecordingRuleExpression have the same fields. Only model stays a JSON
// string.
//
// Expressions are a map attribute with an object element type rather than a MapNestedAttribute,
// because the provider serves protocol v5, which has no nested attributes. Protocol v5 object
// types have no optional attributes either, so every field must be set (null when unused). The
// names and types match what a MapNestedAttribute would use, so the schema can move to one once
// the provider serves protocol v6 without changing state or existing configs. Keep it that way:
// no blocks, no renames, and no validation stricter than what's here.

func ruleExpressionsV1Beta1Attribute() schema.MapAttribute {
	return schema.MapAttribute{
		Required:    true,
		ElementType: ruleExpressionType,
		Validators:  []validator.Map{ruleExpressionsV1Beta1Validator{}},
		Description: "The stages of the rule, keyed by reference ID. Each expression is an object with " +
			"`model` (the query or expression model as a JSON string, for example from `jsonencode`), " +
			"`datasource_uid` (the datasource to run against; `__expr__` for server-side expressions), " +
			"`query_type`, `relative_time_range` (an object with `from` and `to` durations) and " +
			"`source`. Exactly one expression must set `source = true`. Every field must be set; use `null` for unused fields.",
	}
}

// ruleExpressionsV1Beta1Validator checks what Terraform's type system can't: model is a JSON object,
// the time range bounds are durations, and exactly one expression is the source, which the API requires.
type ruleExpressionsV1Beta1Validator struct{}

func (v ruleExpressionsV1Beta1Validator) Description(context.Context) string {
	return "Each expression's model must be a JSON object, relative_time_range bounds must be durations, and exactly one expression must set source = true."
}

func (v ruleExpressionsV1Beta1Validator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v ruleExpressionsV1Beta1Validator) ValidateMap(ctx context.Context, req validator.MapRequest, resp *validator.MapResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	elements := req.ConfigValue.Elements()
	refs := make([]string, 0, len(elements))
	for ref := range elements {
		refs = append(refs, ref)
	}
	slices.Sort(refs)

	sources, sourcesKnown := 0, true
	for _, ref := range refs {
		exprPath := req.Path.AtMapKey(ref)
		if ref == "" {
			resp.Diagnostics.AddAttributeError(exprPath, "Invalid expression", "Expression reference IDs must not be empty.")
		}

		obj, ok := elements[ref].(types.Object)
		if !ok || obj.IsUnknown() {
			sourcesKnown = false
			continue
		}
		if obj.IsNull() {
			resp.Diagnostics.AddAttributeError(exprPath, "Invalid expression", "Expressions must not be null.")
			continue
		}

		var expr RuleExpressionModel
		if diags := obj.As(ctx, &expr, basetypes.ObjectAsOptions{}); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			continue
		}

		switch {
		case expr.Model.IsUnknown():
		case expr.Model.IsNull():
			resp.Diagnostics.AddAttributeError(exprPath.AtName("model"), "Invalid expression model", "model is required.")
		default:
			var m map[string]any
			if err := json.Unmarshal([]byte(expr.Model.ValueString()), &m); err != nil || m == nil {
				resp.Diagnostics.AddAttributeError(exprPath.AtName("model"), "Invalid expression model", "model must be a JSON object, for example from jsonencode({...}).")
			}
		}

		if isNotNullOrUknown(expr.RelativeTimeRange) {
			var timeRange RelativeTimeRangeModel
			if diags := expr.RelativeTimeRange.As(ctx, &timeRange, basetypes.ObjectAsOptions{}); diags.HasError() {
				resp.Diagnostics.Append(diags...)
			} else {
				for _, b := range []struct {
					name  string
					value types.String
				}{{"from", timeRange.From}, {"to", timeRange.To}} {
					name, bound := b.name, b.value
					boundPath := exprPath.AtName("relative_time_range").AtName(name)
					switch {
					case bound.IsUnknown():
					case bound.IsNull():
						resp.Diagnostics.AddAttributeError(boundPath, "Invalid relative_time_range", name+" is required.")
					default:
						if _, err := model.ParseDuration(bound.ValueString()); err != nil {
							resp.Diagnostics.AddAttributeError(boundPath, "Invalid relative_time_range",
								fmt.Sprintf("%q is not a valid duration. Example valid values: 0s, 500ms, 10m, 1h. Error: %s", bound.ValueString(), err))
						}
					}
				}
			}
		}

		switch {
		case expr.Source.IsUnknown():
			sourcesKnown = false
		case expr.Source.ValueBool():
			sources++
			if sources > 1 {
				resp.Diagnostics.AddAttributeError(exprPath.AtName("source"), "Invalid expressions", "Only one expression may set source = true.")
			}
		}
	}

	if sourcesKnown && sources == 0 {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid expressions", "Exactly one expression must set source = true.")
	}
}

// ruleExpressionV1Beta1 holds one expression's fields independently of the alert and recording
// rule API types, which name the same fields with different Go types.
type ruleExpressionV1Beta1 struct {
	QueryType         *string
	DatasourceUID     *string
	RelativeTimeRange *ruleRelativeTimeRangeV1Beta1
	Model             any
	Source            *bool
}

type ruleRelativeTimeRangeV1Beta1 struct {
	From string
	To   string
}

func parseRuleExpressionsV1Beta1(ctx context.Context, src types.Map) (map[string]ruleExpressionV1Beta1, diag.Diagnostics) {
	var data map[string]RuleExpressionModel
	if diags := src.ElementsAs(ctx, &data, false); diags.HasError() {
		return nil, diags
	}

	expressions := make(map[string]ruleExpressionV1Beta1, len(data))
	for ref, expr := range data {
		result := ruleExpressionV1Beta1{
			QueryType:     expr.QueryType.ValueStringPointer(),
			DatasourceUID: expr.DatasourceUID.ValueStringPointer(),
			Source:        expr.Source.ValueBoolPointer(),
		}

		if isNotNullOrUknown(expr.Model) {
			if err := json.Unmarshal([]byte(expr.Model.ValueString()), &result.Model); err != nil {
				return nil, diag.Diagnostics{diag.NewErrorDiagnostic(
					"Invalid expression model",
					fmt.Sprintf("expression %q: model must be a JSON object: %s", ref, err),
				)}
			}
		}

		if isNotNullOrUknown(expr.RelativeTimeRange) {
			var timeRange RelativeTimeRangeModel
			if diags := expr.RelativeTimeRange.As(ctx, &timeRange, basetypes.ObjectAsOptions{}); diags.HasError() {
				return nil, diags
			}
			result.RelativeTimeRange = &ruleRelativeTimeRangeV1Beta1{
				From: timeRange.From.ValueString(),
				To:   timeRange.To.ValueString(),
			}
		}

		expressions[ref] = result
	}
	return expressions, nil
}

func saveRuleExpressionsV1Beta1(expressions map[string]ruleExpressionV1Beta1) (types.Map, diag.Diagnostics) {
	if len(expressions) == 0 {
		return types.MapNull(ruleExpressionType), nil
	}

	values := make(map[string]attr.Value, len(expressions))
	for ref, expr := range expressions {
		exprModel := types.StringNull()
		if expr.Model != nil {
			raw, err := json.Marshal(expr.Model)
			if err != nil {
				return types.MapNull(ruleExpressionType), diag.Diagnostics{diag.NewErrorDiagnostic(
					"Failed to marshal expression model",
					fmt.Sprintf("expression %q: %s", ref, err),
				)}
			}
			exprModel = types.StringValue(string(raw))
		}

		timeRange := types.ObjectNull(relativeTimeRangeType.AttrTypes)
		if expr.RelativeTimeRange != nil {
			var diags diag.Diagnostics
			timeRange, diags = types.ObjectValue(relativeTimeRangeType.AttrTypes, map[string]attr.Value{
				"from": types.StringValue(expr.RelativeTimeRange.From),
				"to":   types.StringValue(expr.RelativeTimeRange.To),
			})
			if diags.HasError() {
				return types.MapNull(ruleExpressionType), diags
			}
		}

		value, diags := types.ObjectValue(ruleExpressionType.AttrTypes, map[string]attr.Value{
			"query_type":          types.StringPointerValue(expr.QueryType),
			"relative_time_range": timeRange,
			"datasource_uid":      types.StringPointerValue(expr.DatasourceUID),
			"model":               exprModel,
			"source":              types.BoolPointerValue(expr.Source),
		})
		if diags.HasError() {
			return types.MapNull(ruleExpressionType), diags
		}
		values[ref] = value
	}
	return types.MapValue(ruleExpressionType, values)
}

var panelRefV1Beta1Type = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"dashboard_uid": types.StringType,
		"panel_id":      types.Int64Type,
	},
}

type panelRefV1Beta1Model struct {
	DashboardUID types.String `tfsdk:"dashboard_uid"`
	PanelID      types.Int64  `tfsdk:"panel_id"`
}

func panelRefV1Beta1Attribute() schema.ObjectAttribute {
	return schema.ObjectAttribute{
		Optional:       true,
		AttributeTypes: panelRefV1Beta1Type.AttrTypes,
		Description:    "Reference to the dashboard panel this alert rule is associated with, as an object with `dashboard_uid` (string) and `panel_id` (number).",
	}
}
