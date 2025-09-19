package appplatform

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/prometheus/common/model"
)

var ruleTriggerType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"interval": types.StringType,
	},
}

type RuleTriggerModel struct {
	Interval types.String `tfsdk:"interval"`
}

var relativeTimeRangeType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"from": types.StringType,
		"to":   types.StringType,
	},
}

type RelativeTimeRangeModel struct {
	From types.String `tfsdk:"from"`
	To   types.String `tfsdk:"to"`
}

var ruleExpressionType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"query_type":          types.StringType,
		"relative_time_range": relativeTimeRangeType,
		"datasource_uid":      types.StringType,
		"model":               types.StringType,
		"source":              types.BoolType,
	},
}

type RuleExpressionModel struct {
	QueryType         types.String `tfsdk:"query_type"`
	RelativeTimeRange types.Object `tfsdk:"relative_time_range"`
	DatasourceUid     types.String `tfsdk:"datasource_uid"`
	Model             types.String `tfsdk:"model"`
	Source            types.Bool   `tfsdk:"source"`
}

type ExpressionMapValidator struct{}

func (v ExpressionMapValidator) Description(_ context.Context) string {
	return "expressions must meet validation requirements"
}

func (v ExpressionMapValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v ExpressionMapValidator) ValidateMap(ctx context.Context, req validator.MapRequest, resp *validator.MapResponse) {

	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	val := req.ConfigValue.Elements()

	if len(val) == 0 {
		return
	}

	sourceCount := 0
	for _, v := range val {
		obj, ok := v.(types.Object)
		if !ok {
			return
		}
		var data RuleExpressionModel
		if diag := obj.As(ctx, &data, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty:    true,
			UnhandledUnknownAsEmpty: true,
		}); diag.HasError() {
			return
		}
		if data.Source.ValueBool() {
			if sourceCount > 0 {
				// skip continuing to check other elements, we've reached an invalid state
				break
			}
			sourceCount++
		}
	}

	if sourceCount != 1 {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Expression Map",
			"Exactly one expression must be marked as source",
		)
	}
}

// PrometheusDurationValidator validates that a string is a valid Prometheus duration
type PrometheusDurationValidator struct{}

// Description returns the validator's description.
func (v PrometheusDurationValidator) Description(_ context.Context) string {
	return "string must be a valid Prometheus duration with at most seconds precision (e.g., 30s, 1m, 2h, 3d, 4w)"
}

// MarkdownDescription returns the validator's description in Markdown format.
func (v PrometheusDurationValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateString performs the validation.
func (v PrometheusDurationValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	val := req.ConfigValue.ValueString()
	if val == "" {
		return
	}

	duration, err := model.ParseDuration(val)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Duration",
			fmt.Sprintf("String %q is not a valid Prometheus duration. Example valid values: 30s, 1m, 2h, 3d, 4w. Error: %s", val, err),
		)
		return
	}

	// Convert to time.Duration for easier comparison
	d := time.Duration(duration)

	// Ensure the duration is positive
	if d <= 0 {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Duration",
			fmt.Sprintf("Duration %q must be positive", val),
		)
		return
	}

	// Check that duration is not using millisecond precision
	if d%time.Second != 0 {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Duration",
			fmt.Sprintf("Duration %q cannot use millisecond precision. Use seconds or larger units (e.g., 30s, 1m, 2h, 3d, 4w)", val),
		)
	}
}

// PrometheusDurationWithMillisValidator validates that a string is a valid Prometheus duration with millisecond precision
type PrometheusDurationWithMillisValidator struct{}

// Description returns the validator's description.
func (v PrometheusDurationWithMillisValidator) Description(_ context.Context) string {
	return "string must be a valid Prometheus duration (e.g., 500ms, 30s, 1m, 2h, 3d, 4w)"
}

// MarkdownDescription returns the validator's description in Markdown format.
func (v PrometheusDurationWithMillisValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateString performs the validation.
func (v PrometheusDurationWithMillisValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	val := req.ConfigValue.ValueString()
	if val == "" {
		return
	}

	duration, err := model.ParseDuration(val)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Duration",
			fmt.Sprintf("String %q is not a valid Prometheus duration. Example valid values: 500ms, 30s, 1m, 2h, 3d, 4w. Error: %s", val, err),
		)
		return
	}

	// Ensure the duration is positive
	if time.Duration(duration) <= 0 {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Duration",
			fmt.Sprintf("Duration %q must be positive", val),
		)
	}
}
