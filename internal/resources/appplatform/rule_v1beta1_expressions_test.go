package appplatform

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

// expressionV1Beta1Value builds one v1beta1 expression object. Fields left out of values are null.
func expressionV1Beta1Value(t *testing.T, values map[string]attr.Value) types.Object {
	t.Helper()
	attrs := map[string]attr.Value{
		"query_type":          types.StringNull(),
		"relative_time_range": types.ObjectNull(relativeTimeRangeType.AttrTypes),
		"datasource_uid":      types.StringNull(),
		"model":               types.StringNull(),
		"source":              types.BoolNull(),
	}
	for name, value := range values {
		attrs[name] = value
	}
	return objectValue(t, ruleExpressionType.AttrTypes, attrs)
}

func timeRangeValue(t *testing.T, from, to string) types.Object {
	t.Helper()
	return objectValue(t, relativeTimeRangeType.AttrTypes, map[string]attr.Value{
		"from": types.StringValue(from),
		"to":   types.StringValue(to),
	})
}

func expressionsV1Beta1Value(t *testing.T, expressions map[string]types.Object) types.Map {
	t.Helper()
	values := make(map[string]attr.Value, len(expressions))
	for ref, expr := range expressions {
		values[ref] = expr
	}
	v, diags := types.MapValue(ruleExpressionType, values)
	require.False(t, diags.HasError(), diags)
	return v
}

func TestRuleExpressionsV1Beta1Validator(t *testing.T) {
	ctx := context.Background()
	query := func(source attr.Value) types.Object {
		return expressionV1Beta1Value(t, map[string]attr.Value{
			"datasource_uid":      types.StringValue("ds_uid"),
			"relative_time_range": timeRangeValue(t, "10m", "0s"),
			"model":               types.StringValue(`{"expr":"up","refId":"A"}`),
			"source":              source,
		})
	}

	for _, tc := range []struct {
		name        string
		expressions map[string]types.Object
		errPath     string
	}{
		{
			name:        "one source",
			expressions: map[string]types.Object{"A": query(types.BoolValue(true)), "B": query(types.BoolValue(false))},
		},
		{
			name: "only model and source set",
			expressions: map[string]types.Object{"A": expressionV1Beta1Value(t, map[string]attr.Value{
				"model":  types.StringValue(`{"type":"threshold"}`),
				"source": types.BoolValue(true),
			})},
		},
		{
			name: "millisecond bounds",
			expressions: map[string]types.Object{"A": expressionV1Beta1Value(t, map[string]attr.Value{
				"relative_time_range": timeRangeValue(t, "1500ms", "0s"),
				"model":               types.StringValue(`{}`),
				"source":              types.BoolValue(true),
			})},
		},
		{
			name:        "unknown source skips the count",
			expressions: map[string]types.Object{"A": query(types.BoolUnknown()), "B": query(types.BoolNull())},
		},
		{
			name:        "no source",
			expressions: map[string]types.Object{"A": query(types.BoolNull()), "B": query(types.BoolValue(false))},
			errPath:     "spec.expressions",
		},
		{
			name:        "two sources",
			expressions: map[string]types.Object{"A": query(types.BoolValue(true)), "B": query(types.BoolValue(true))},
			errPath:     `spec.expressions["B"].source`,
		},
		{
			name: "model is not JSON",
			expressions: map[string]types.Object{"A": expressionV1Beta1Value(t, map[string]attr.Value{
				"model":  types.StringValue(`{"expr":`),
				"source": types.BoolValue(true),
			})},
			errPath: `spec.expressions["A"].model`,
		},
		{
			name: "model is a JSON array",
			expressions: map[string]types.Object{"A": expressionV1Beta1Value(t, map[string]attr.Value{
				"model":  types.StringValue(`[1, 2]`),
				"source": types.BoolValue(true),
			})},
			errPath: `spec.expressions["A"].model`,
		},
		{
			name: "model is missing",
			expressions: map[string]types.Object{"A": expressionV1Beta1Value(t, map[string]attr.Value{
				"source": types.BoolValue(true),
			})},
			errPath: `spec.expressions["A"].model`,
		},
		{
			name: "bad duration",
			expressions: map[string]types.Object{"A": expressionV1Beta1Value(t, map[string]attr.Value{
				"relative_time_range": timeRangeValue(t, "ten minutes", "0s"),
				"model":               types.StringValue(`{}`),
				"source":              types.BoolValue(true),
			})},
			errPath: `spec.expressions["A"].relative_time_range.from`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp := &validator.MapResponse{}
			ruleExpressionsV1Beta1Validator{}.ValidateMap(ctx, validator.MapRequest{
				Path:        path.Root("spec").AtName("expressions"),
				ConfigValue: expressionsV1Beta1Value(t, tc.expressions),
			}, resp)

			if tc.errPath == "" {
				require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)
				return
			}
			require.Len(t, resp.Diagnostics.Errors(), 1, resp.Diagnostics)
			withPath, ok := resp.Diagnostics.Errors()[0].(interface{ Path() path.Path })
			require.True(t, ok)
			require.Equal(t, tc.errPath, withPath.Path().String())
		})
	}
}

func TestRuleV1Beta1SchemaOverridesLeaveSharedSchemaAlone(t *testing.T) {
	AlertRuleV1Beta1()
	RecordingRuleV1Beta1()

	alertAttrs := alertRuleSpecSchema(nfSettingsV1Beta1Block()).SpecAttributes
	require.Equal(t, types.StringType, alertAttrs["expressions"].(schema.MapAttribute).ElementType)
	require.Equal(t, types.StringType, alertAttrs["panel_ref"].(schema.MapAttribute).ElementType)
	require.Equal(t, types.StringType, recordingRuleSpecSchema().SpecAttributes["expressions"].(schema.MapAttribute).ElementType)
}

func TestRuleExpressionsV1Beta1ModelMatchesJSONEncode(t *testing.T) {
	// Terraform's jsonencode writes compact JSON with sorted keys and escapes <, > and & as
	// <, > and &, which is also what json.Marshal produces for the decoded model,
	// so an imported model matches the config.
	model := "{\"datasource\":{\"type\":\"prometheus\",\"uid\":\"ds_uid\"},\"expr\":\"count(up{}) \\u003e 0\",\"intervalMs\":1000,\"maxDataPoints\":43200,\"refId\":\"A\"}"
	src := expressionsV1Beta1Value(t, map[string]types.Object{"A": expressionV1Beta1Value(t, map[string]attr.Value{
		"model":  types.StringValue(model),
		"source": types.BoolValue(true),
	})})

	parsed, diags := parseRuleExpressionsV1Beta1(context.Background(), src)
	require.False(t, diags.HasError(), diags)
	saved, diags := saveRuleExpressionsV1Beta1(parsed)
	require.False(t, diags.HasError(), diags)
	require.True(t, saved.Equal(src), "got %s, want %s", saved, src)
}
