package appplatform

import (
	"context"
	"testing"

	"github.com/grafana/grafana/apps/alerting/rules/pkg/apis/alerting/v1beta1"
	"github.com/grafana/terraform-provider-grafana/v4/internal/util"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

func TestParseRecordingRuleV1Beta1Spec(t *testing.T) {
	var dst v1beta1.RecordingRule
	diags := parseRecordingRuleV1Beta1Spec(context.Background(), recordingRuleV1Beta1SpecObject(t), &dst)
	require.False(t, diags.HasError(), diags)

	require.Equal(t, "Test rule", dst.Spec.Title)
	require.Equal(t, v1beta1.RecordingRulePromDuration("1m"), dst.Spec.Trigger.Interval)
	require.Equal(t, v1beta1.RecordingRuleMetricName("test_metric"), dst.Spec.Metric)
	require.Equal(t, v1beta1.RecordingRuleDatasourceUID("target_uid"), dst.Spec.TargetDatasourceUID)
	require.Equal(t, map[string]v1beta1.RecordingRuleTemplateString{"team": "alerting"}, dst.Spec.Labels)
	require.NotNil(t, dst.Spec.Paused)
	require.True(t, *dst.Spec.Paused)

	require.Equal(t, v1beta1.RecordingRuleExpression{
		RelativeTimeRange: &v1beta1.RecordingRuleRelativeTimeRange{From: "10m", To: "0s"},
		DatasourceUID:     util.Ptr(v1beta1.RecordingRuleDatasourceUID("ds_uid")),
		Model:             map[string]any{"expr": "up", "refId": "A"},
		Source:            util.Ptr(true),
	}, dst.Spec.Expressions["A"])
}

func TestSaveRecordingRuleV1Beta1SpecRoundTrip(t *testing.T) {
	ctx := context.Background()
	src := recordingRuleV1Beta1SpecObject(t)

	var rule v1beta1.RecordingRule
	diags := parseRecordingRuleV1Beta1Spec(ctx, src, &rule)
	require.False(t, diags.HasError(), diags)

	var dst ResourceModel
	diags = saveRecordingRuleV1Beta1Spec(ctx, &rule, &dst)
	require.False(t, diags.HasError(), diags)
	require.True(t, dst.Spec.Type(ctx).Equal(recordingRuleSpecV1Beta1Type))

	got := dst.Spec.Attributes()
	for name, want := range src.Attributes() {
		require.True(t, got[name].Equal(want), "%s: got %s, want %s", name, got[name], want)
	}
}

// TestRecordingRuleV1Beta1ImportedExpressionsSurviveApply covers the import path: the saver writes
// what the API returned, and parsing that state must send the same expressions back.
func TestRecordingRuleV1Beta1ImportedExpressionsSurviveApply(t *testing.T) {
	ctx := context.Background()
	imported := v1beta1.RecordingRule{Spec: v1beta1.RecordingRuleSpec{
		Title:               "Imported",
		Trigger:             v1beta1.RecordingRuleIntervalTrigger{Interval: "1m"},
		Metric:              "test_metric",
		TargetDatasourceUID: "target_uid",
		Expressions: v1beta1.RecordingRuleExpressionMap{
			"A": {
				QueryType:         util.Ptr("range"),
				RelativeTimeRange: &v1beta1.RecordingRuleRelativeTimeRange{From: "10m", To: "0s"},
				DatasourceUID:     util.Ptr(v1beta1.RecordingRuleDatasourceUID("ds_uid")),
				Model:             map[string]any{"expr": "up", "refId": "A"},
				Source:            util.Ptr(true),
			},
		},
	}}

	var state ResourceModel
	diags := saveRecordingRuleV1Beta1Spec(ctx, &imported, &state)
	require.False(t, diags.HasError(), diags)

	var applied v1beta1.RecordingRule
	diags = parseRecordingRuleV1Beta1Spec(ctx, state.Spec, &applied)
	require.False(t, diags.HasError(), diags)
	require.Equal(t, imported.Spec.Expressions, applied.Spec.Expressions)
}

func recordingRuleV1Beta1SpecObject(t *testing.T) types.Object {
	t.Helper()
	expressions := expressionsV1Beta1Value(t, map[string]types.Object{
		"A": expressionV1Beta1Value(t, map[string]attr.Value{
			"model":               types.StringValue(`{"expr":"up","refId":"A"}`),
			"datasource_uid":      types.StringValue("ds_uid"),
			"relative_time_range": timeRangeValue(t, "10m", "0s"),
			"source":              types.BoolValue(true),
		}),
	})

	return objectValue(t, recordingRuleSpecV1Beta1Type.AttrTypes, map[string]attr.Value{
		"title":       types.StringValue("Test rule"),
		"expressions": expressions,
		"paused":      types.BoolValue(true),
		"trigger": objectValue(t, ruleTriggerType.AttrTypes, map[string]attr.Value{
			"interval": types.StringValue("1m"),
		}),
		"metric":                types.StringValue("test_metric"),
		"labels":                stringMapValue(t, map[string]string{"team": "alerting"}),
		"target_datasource_uid": types.StringValue("target_uid"),
	})
}
