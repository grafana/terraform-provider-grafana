package appplatform

import (
	"context"
	"testing"

	"github.com/grafana/grafana/apps/alerting/rules/pkg/apis/alerting/v0alpha1"
	"github.com/grafana/terraform-provider-grafana/v4/internal/util"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

// TestAlertRuleV0Alpha1ImportedExpressionsSurviveApply covers the import path: the saver writes
// what the API returned, and parsing that state must send the same expressions back.
func TestAlertRuleV0Alpha1ImportedExpressionsSurviveApply(t *testing.T) {
	ctx := context.Background()
	imported := v0alpha1.AlertRule{Spec: v0alpha1.AlertRuleSpec{
		Title:   "Imported",
		Trigger: v0alpha1.AlertRuleIntervalTrigger{Interval: "1m"},
		Expressions: v0alpha1.AlertRuleExpressionMap{
			"A": {
				QueryType:         util.Ptr("range"),
				RelativeTimeRange: &v0alpha1.AlertRuleRelativeTimeRange{From: "10m", To: "0s"},
				DatasourceUID:     util.Ptr(v0alpha1.AlertRuleDatasourceUID("ds_uid")),
				Model:             map[string]any{"expr": "up", "refId": "A"},
			},
			"B": {
				Model:  map[string]any{"expression": "A", "refId": "B", "type": "threshold"},
				Source: util.Ptr(true),
			},
		},
		NoDataState:  "NoData",
		ExecErrState: "Error",
	}}

	var state ResourceModel
	diags := saveAlertRuleSpec(ctx, &imported, &state)
	require.False(t, diags.HasError(), diags)

	// The state uses the snake_case keys users write, in jsonencode's key order.
	expressions := state.Spec.Attributes()["expressions"].(types.Map).Elements()
	require.Equal(t,
		`{"datasource_uid":"ds_uid","model":{"expr":"up","refId":"A"},"query_type":"range","relative_time_range":{"from":"10m","to":"0s"}}`,
		expressions["A"].(types.String).ValueString(),
	)

	var applied v0alpha1.AlertRule
	diags = parseAlertRuleSpec(ctx, state.Spec, &applied)
	require.False(t, diags.HasError(), diags)
	require.Equal(t, imported.Spec.Expressions, applied.Spec.Expressions)
}

// TestRecordingRuleV0Alpha1ImportedExpressionsSurviveApply is the recording rule counterpart.
func TestRecordingRuleV0Alpha1ImportedExpressionsSurviveApply(t *testing.T) {
	ctx := context.Background()
	imported := v0alpha1.RecordingRule{Spec: v0alpha1.RecordingRuleSpec{
		Title:               "Imported",
		Trigger:             v0alpha1.RecordingRuleIntervalTrigger{Interval: "1m"},
		Metric:              "test_metric",
		TargetDatasourceUID: "target_uid",
		Expressions: v0alpha1.RecordingRuleExpressionMap{
			"A": {
				QueryType:         util.Ptr("range"),
				RelativeTimeRange: &v0alpha1.RecordingRuleRelativeTimeRange{From: "10m", To: "0s"},
				DatasourceUID:     util.Ptr(v0alpha1.RecordingRuleDatasourceUID("ds_uid")),
				Model:             map[string]any{"expr": "up", "refId": "A"},
				Source:            util.Ptr(true),
			},
		},
	}}

	var state ResourceModel
	diags := saveRecordingRuleSpec(ctx, &imported, &state)
	require.False(t, diags.HasError(), diags)

	var applied v0alpha1.RecordingRule
	diags = parseRecordingRuleSpec(ctx, state.Spec, &applied)
	require.False(t, diags.HasError(), diags)
	require.Equal(t, imported.Spec.Expressions, applied.Spec.Expressions)
}

// TestConvertJSONToExpressionV0Alpha1AcceptsCamelCase covers state saved by earlier versions,
// whose import wrote the API's camelCase keys.
func TestConvertJSONToExpressionV0Alpha1AcceptsCamelCase(t *testing.T) {
	ctx := context.Background()
	camelCase := map[string]any{
		"model":             map[string]any{"expr": "up", "refId": "A"},
		"queryType":         "range",
		"datasourceUID":     "ds_uid",
		"relativeTimeRange": map[string]any{"from": "10m", "to": "0s"},
		"source":            true,
	}

	alertExpr, diags := convertJSONToAlertRuleExpression(ctx, camelCase)
	require.False(t, diags.HasError(), diags)
	require.Equal(t, v0alpha1.AlertRuleExpression{
		QueryType:         util.Ptr("range"),
		RelativeTimeRange: &v0alpha1.AlertRuleRelativeTimeRange{From: "10m", To: "0s"},
		DatasourceUID:     util.Ptr(v0alpha1.AlertRuleDatasourceUID("ds_uid")),
		Model:             map[string]any{"expr": "up", "refId": "A"},
		Source:            util.Ptr(true),
	}, alertExpr)

	recordingExpr, diags := convertJSONToRecordingRuleExpression(ctx, camelCase)
	require.False(t, diags.HasError(), diags)
	require.Equal(t, v0alpha1.RecordingRuleExpression{
		QueryType:         util.Ptr("range"),
		RelativeTimeRange: &v0alpha1.RecordingRuleRelativeTimeRange{From: "10m", To: "0s"},
		DatasourceUID:     util.Ptr(v0alpha1.RecordingRuleDatasourceUID("ds_uid")),
		Model:             map[string]any{"expr": "up", "refId": "A"},
		Source:            util.Ptr(true),
	}, recordingExpr)
}
