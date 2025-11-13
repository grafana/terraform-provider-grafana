package appplatform

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
		"model":               types.StringType, // JSON string representation of the model
		"source":              types.BoolType,
	},
}

type RuleExpressionModel struct {
	QueryType         types.String `tfsdk:"query_type"`
	RelativeTimeRange types.Object `tfsdk:"relative_time_range"`
	DatasourceUID     types.String `tfsdk:"datasource_uid"`
	Model             types.String `tfsdk:"model"` // JSON string representation of the model
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

// TEMPORARY: Not currently used - switched to map[string]string for compatibility with dependent projects
// that don't yet support dynamic types. This can be re-enabled once plugin framework support is universal.
// ExpressionsDynamicValidator validates that the dynamic value is a valid expressions map
type ExpressionsDynamicValidator struct{}

func (v ExpressionsDynamicValidator) Description(_ context.Context) string {
	return "expressions must be a map of HCL expression objects with exactly one marked as source"
}

func (v ExpressionsDynamicValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v ExpressionsDynamicValidator) ValidateDynamic(ctx context.Context, req validator.DynamicRequest, resp *validator.DynamicResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	// Extract the underlying value
	underlyingValue := req.ConfigValue.UnderlyingValue()

	// Handle both Map and Object types (HCL can parse as either depending on syntax)
	var elements map[string]attr.Value

	switch v := underlyingValue.(type) {
	case types.Map:
		elements = v.Elements()
	case types.Object:
		elements = v.Attributes()
	default:
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Expressions Type",
			"Expressions must be a map of HCL objects",
		)
		return
	}

	if len(elements) == 0 {
		return
	}

	// Validate that exactly one expression is marked as source
	sourceCount := 0
	for key, rawVal := range elements {
		obj, ok := rawVal.(types.Object)
		if !ok {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtMapKey(key),
				"Invalid Expression Type",
				fmt.Sprintf("Expression '%s' must be an object", key),
			)
			continue
		}

		// Check if this expression has source = true
		// Extract source field directly from attributes
		attrs := obj.Attributes()
		if sourceAttr, ok := attrs["source"]; ok && !sourceAttr.IsNull() && !sourceAttr.IsUnknown() {
			if sourceBool, ok := sourceAttr.(types.Bool); ok && sourceBool.ValueBool() {
				sourceCount++
			}
		}

		// Validate that required fields are present
		// Check if model attribute exists and is valid JSON
		if modelAttr, ok := attrs["model"]; !ok || modelAttr.IsNull() || modelAttr.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtMapKey(key).AtName("model"),
				"Missing Required Field",
				fmt.Sprintf("Expression '%s' must have a 'model' field", key),
			)
		} else if strModel, ok := modelAttr.(types.String); ok {
			// Validate that it's valid JSON
			modelStr := strModel.ValueString()
			if modelStr != "" {
				var temp map[string]any
				if err := json.Unmarshal([]byte(modelStr), &temp); err != nil {
					resp.Diagnostics.AddAttributeError(
						req.Path.AtMapKey(key).AtName("model"),
						"Invalid Model JSON",
						fmt.Sprintf("Expression '%s' model must be valid JSON: %v", key, err),
					)
				}
			}
		}
	}

	if sourceCount != 1 {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Expression Map",
			fmt.Sprintf("Exactly one expression must be marked as source, found %d", sourceCount),
		)
	}
}

// TEMPORARY: Not currently used - switched to map[string]string for compatibility with dependent projects
// that don't yet support dynamic types. This can be re-enabled once plugin framework support is universal.
// ParseExpressionsFromDynamic extracts and validates expressions from a dynamic type value
func ParseExpressionsFromDynamic(ctx context.Context, dynamicValue types.Dynamic) (map[string]types.Object, diag.Diagnostics) {
	diags := diag.Diagnostics{}

	if dynamicValue.IsNull() || dynamicValue.IsUnknown() {
		return nil, diags
	}

	// Extract the underlying value
	underlyingValue := dynamicValue.UnderlyingValue()

	// Handle both Map and Object types (HCL can parse as either depending on syntax)
	var elements map[string]attr.Value

	switch v := underlyingValue.(type) {
	case types.Map:
		elements = v.Elements()
	case types.Object:
		elements = v.Attributes()
	default:
		diags.AddError("Invalid data", "Expressions must be a map or object of HCL objects")
		return nil, diags
	}

	result := make(map[string]types.Object)
	for key, rawVal := range elements {
		obj, ok := rawVal.(types.Object)
		if !ok {
			diags.AddError("Invalid data", fmt.Sprintf("Expression '%s' is not an object", key))
			continue
		}

		attrs := obj.Attributes()
		newAttrs := make(map[string]attr.Value)

		for attrName := range ruleExpressionType.AttrTypes {
			if val, ok := attrs[attrName]; ok {
				newAttrs[attrName] = val
			} else {
				switch attrName {
				case "query_type", "datasource_uid", "model":
					newAttrs[attrName] = types.StringNull()
				case "source":
					newAttrs[attrName] = types.BoolNull()
				case "relative_time_range":
					newAttrs[attrName] = types.ObjectNull(relativeTimeRangeType.AttrTypes)
				default:
					newAttrs[attrName] = types.StringNull()
				}
			}
		}

		if modelAttr, ok := attrs["model"]; ok && !modelAttr.IsNull() && !modelAttr.IsUnknown() {
			var jsonStr string
			if dynVal, ok := modelAttr.(types.Dynamic); ok {
				jsonStr, _ = ConvertModelToJSON(ctx, dynVal)
			} else if objVal, ok := modelAttr.(types.Object); ok {
				tempDyn := types.DynamicValue(objVal)
				jsonStr, _ = ConvertModelToJSON(ctx, tempDyn)
			}
			newAttrs["model"] = types.StringValue(jsonStr)
		}

		convertedObj, d := types.ObjectValue(ruleExpressionType.AttrTypes, newAttrs)
		if d.HasError() {
			diags.Append(d...)
			continue
		}
		result[key] = convertedObj
	}

	return result, diags
}

// TEMPORARY: Not currently used - switched to map[string]string for compatibility with dependent projects
// that don't yet support dynamic types. This can be re-enabled once plugin framework support is universal.
// ConvertExpressionsMapToDynamic converts a map of expression objects to a dynamic value
func ConvertExpressionsMapToDynamic(ctx context.Context, expressions map[string]attr.Value) (types.Dynamic, diag.Diagnostics) {
	if len(expressions) == 0 {
		return types.DynamicNull(), diag.Diagnostics{}
	}

	exprMapValue, d := types.MapValue(ruleExpressionType, expressions)
	if d.HasError() {
		return types.DynamicNull(), d
	}

	return types.DynamicValue(exprMapValue), diag.Diagnostics{}
}

// convertTerraformValueToGo converts a Terraform attr.Value to a Go any type for JSON marshaling
func convertTerraformValueToGo(ctx context.Context, val attr.Value, diags *diag.Diagnostics) any {
	if val.IsNull() || val.IsUnknown() {
		return nil
	}

	switch v := val.(type) {
	case types.String:
		return v.ValueString()
	case types.Bool:
		return v.ValueBool()
	case types.Number:
		f, _ := v.ValueBigFloat().Float64()
		return f
	case types.Int64:
		return v.ValueInt64()
	case types.Dynamic:
		// Recursively convert dynamic values
		jsonStr, d := ConvertModelToJSON(ctx, v)
		if d.HasError() {
			if diags != nil {
				diags.Append(d...)
			}
			return nil
		}
		var result any
		if err := json.Unmarshal([]byte(jsonStr), &result); err == nil {
			return result
		}
		return nil
	case types.Object:
		// Convert to dynamic and recurse
		dynVal := types.DynamicValue(v)
		jsonStr, d := ConvertModelToJSON(ctx, dynVal)
		if d.HasError() {
			if diags != nil {
				diags.Append(d...)
			}
			return nil
		}
		var result any
		if err := json.Unmarshal([]byte(jsonStr), &result); err == nil {
			return result
		}
		return nil
	case types.List:
		elements := v.Elements()
		array := make([]any, len(elements))
		for i, elem := range elements {
			array[i] = convertTerraformValueToGo(ctx, elem, diags)
		}
		return array
	case types.Map:
		elements := v.Elements()
		result := make(map[string]any)
		for k, elem := range elements {
			result[k] = convertTerraformValueToGo(ctx, elem, diags)
		}
		return result
	default:
		// Unknown type, return nil
		return nil
	}
}

func convertValueToGoType(ctx context.Context, val attr.Value, diags *diag.Diagnostics) any {
	if val.IsNull() || val.IsUnknown() {
		return nil
	}

	switch v := val.(type) {
	case types.String:
		return v.ValueString()
	case types.Bool:
		return v.ValueBool()
	case types.Number:
		f, _ := v.ValueBigFloat().Float64()
		return f
	case types.Int64:
		return v.ValueInt64()
	case types.Dynamic:
		jsonStr, d := ConvertModelToJSON(ctx, v)
		if d.HasError() {
			diags.Append(d...)
			return nil
		}
		var nested any
		if err := json.Unmarshal([]byte(jsonStr), &nested); err == nil {
			return nested
		}
		return nil
	case types.Object:
		nestedDyn := types.DynamicValue(v)
		jsonStr, d := ConvertModelToJSON(ctx, nestedDyn)
		if d.HasError() {
			diags.Append(d...)
			return nil
		}
		var nested any
		if err := json.Unmarshal([]byte(jsonStr), &nested); err == nil {
			return nested
		}
		return nil
	case types.List:
		listElements := v.Elements()
		array := make([]any, len(listElements))
		for i, elem := range listElements {
			array[i] = convertTerraformValueToGo(ctx, elem, diags)
		}
		return array
	default:
		return nil
	}
}

func convertMapToGoMap(ctx context.Context, elements map[string]attr.Value, diags *diag.Diagnostics) map[string]any {
	modelMap := make(map[string]any)
	for key, val := range elements {
		modelMap[key] = convertValueToGoType(ctx, val, diags)
	}
	return modelMap
}

// ConvertModelToJSON converts a dynamic value containing an HCL object/map to a JSON string for the API
func ConvertModelToJSON(ctx context.Context, modelValue types.Dynamic) (string, diag.Diagnostics) {
	diags := diag.Diagnostics{}

	if modelValue.IsNull() || modelValue.IsUnknown() {
		return "", diags
	}

	underlyingValue := modelValue.UnderlyingValue()

	var modelData any

	switch v := underlyingValue.(type) {
	case types.Map:
		modelData = convertMapToGoMap(ctx, v.Elements(), &diags)
	case types.Object:
		modelData = convertMapToGoMap(ctx, v.Attributes(), &diags)
	default:
		diags.AddError("Invalid model type", "Model must be a map or object")
		return "", diags
	}

	jsonBytes, err := json.Marshal(modelData)
	if err != nil {
		diags.AddError("Failed to marshal model to JSON", err.Error())
		return "", diags
	}

	return string(jsonBytes), diags
}

// ParseModelFromJSON parses a JSON string from the API to a dynamic value containing an HCL-compatible map
func ParseModelFromJSON(ctx context.Context, jsonStr string) (types.Dynamic, diag.Diagnostics) {
	diags := diag.Diagnostics{}

	if jsonStr == "" {
		return types.DynamicNull(), diags
	}

	// Parse JSON string to map
	var modelData map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &modelData); err != nil {
		diags.AddError("Failed to parse model JSON", err.Error())
		return types.DynamicNull(), diags
	}

	// Convert to a types.Map
	mapElements := make(map[string]attr.Value)
	for key, val := range modelData {
		mapElements[key] = convertInterfaceToAttrValue(ctx, val)
	}

	mapValue, d := types.MapValue(types.DynamicType, mapElements)
	if d.HasError() {
		diags.Append(d...)
		return types.DynamicNull(), diags
	}

	return types.DynamicValue(mapValue), diags
}

// Helper function to convert any to attr.Value
func convertInterfaceToAttrValue(ctx context.Context, val any) attr.Value {
	switch v := val.(type) {
	case string:
		return types.StringValue(v)
	case bool:
		return types.BoolValue(v)
	case float64:
		// Check if it's actually an integer
		if v == float64(int64(v)) {
			return types.Int64Value(int64(v))
		}
		return types.NumberValue(new(big.Float).SetFloat64(v))
	case map[string]any:
		// Nested object - return as a map without extra Dynamic wrapper
		nested := make(map[string]attr.Value)
		for k, v := range v {
			nested[k] = convertInterfaceToAttrValue(ctx, v)
		}
		// Just return the map value, not wrapped in Dynamic
		mapVal, _ := types.MapValue(types.StringType, nested)
		return mapVal
	case []any:
		// Array - return as a list without extra Dynamic wrapper
		items := make([]attr.Value, len(v))
		for i, item := range v {
			items[i] = convertInterfaceToAttrValue(ctx, item)
		}
		// Just return the list value, not wrapped in Dynamic
		listVal, _ := types.ListValue(types.StringType, items)
		return listVal
	case nil:
		return types.StringNull()
	default:
		// Fallback to string representation
		return types.StringValue(fmt.Sprintf("%v", v))
	}
}

// ConvertAPIExpressionToTerraform converts an API expression (with model as JSON string) to a Terraform object (with model as HCL object/map)
func ConvertAPIExpressionToTerraform(ctx context.Context, apiExpr any, attrTypes map[string]attr.Type) (types.Object, diag.Diagnostics) {
	// First convert the whole expression to a generic map
	exprMap := make(map[string]attr.Value)

	// Use reflection to handle the API expression struct
	// This is a bit hacky but necessary to handle the any model field
	jsonBytes, err := json.Marshal(apiExpr)
	if err != nil {
		return types.ObjectNull(attrTypes), diag.Diagnostics{
			diag.NewErrorDiagnostic("Failed to marshal expression", err.Error()),
		}
	}

	var data map[string]any
	if err := json.Unmarshal(jsonBytes, &data); err != nil {
		return types.ObjectNull(attrTypes), diag.Diagnostics{
			diag.NewErrorDiagnostic("Failed to unmarshal expression", err.Error()),
		}
	}

	// Process each field
	for key, value := range data {
		switch key {
		case "model":
			// Keep model as JSON string
			if modelStr, ok := value.(string); ok {
				exprMap[key] = types.StringValue(modelStr)
			} else {
				exprMap[key] = types.StringNull()
			}
		case "relativeTimeRange":
			// Handle relative time range
			if rtRange, ok := value.(map[string]any); ok {
				// Convert from/to values to strings
				fromStr := ""
				toStr := ""

				if from, ok := rtRange["from"]; ok {
					switch v := from.(type) {
					case string:
						fromStr = v
					case float64:
						fromStr = fmt.Sprintf("%.0f", v)
					}
				}

				if to, ok := rtRange["to"]; ok {
					switch v := to.(type) {
					case string:
						toStr = v
					case float64:
						toStr = fmt.Sprintf("%.0f", v)
					}
				}

				rtObj, _ := types.ObjectValue(relativeTimeRangeType.AttrTypes, map[string]attr.Value{
					"from": types.StringValue(fromStr),
					"to":   types.StringValue(toStr),
				})
				exprMap["relative_time_range"] = rtObj
			} else {
				exprMap["relative_time_range"] = types.ObjectNull(relativeTimeRangeType.AttrTypes)
			}
		case "queryType":
			if str, ok := value.(string); ok {
				exprMap["query_type"] = types.StringValue(str)
			} else {
				exprMap["query_type"] = types.StringNull()
			}
		case "datasourceUID":
			if str, ok := value.(string); ok {
				exprMap["datasource_uid"] = types.StringValue(str)
			} else {
				exprMap["datasource_uid"] = types.StringNull()
			}
		case "source":
			if b, ok := value.(bool); ok {
				exprMap["source"] = types.BoolValue(b)
			} else {
				exprMap["source"] = types.BoolNull()
			}
		}
	}

	// Ensure all required attributes are present
	if _, ok := exprMap["query_type"]; !ok {
		exprMap["query_type"] = types.StringNull()
	}
	if _, ok := exprMap["relative_time_range"]; !ok {
		exprMap["relative_time_range"] = types.ObjectNull(relativeTimeRangeType.AttrTypes)
	}
	if _, ok := exprMap["datasource_uid"]; !ok {
		exprMap["datasource_uid"] = types.StringNull()
	}
	if _, ok := exprMap["model"]; !ok {
		exprMap["model"] = types.StringNull()
	}
	if _, ok := exprMap["source"]; !ok {
		exprMap["source"] = types.BoolNull()
	}

	return types.ObjectValue(ruleExpressionType.AttrTypes, exprMap)
}
