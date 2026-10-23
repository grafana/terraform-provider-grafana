package appplatform

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/grafana/grafana/apps/alerting/rules/pkg/apis/alerting/v1beta1"
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

	expr := dst.Spec.Expressions["A"]
	require.Equal(t, v1beta1.RecordingRuleDatasourceUID("ds_uid"), *expr.DatasourceUID)
	require.Equal(t, &v1beta1.RecordingRuleRelativeTimeRange{From: "10m", To: "0s"}, expr.RelativeTimeRange)
	require.True(t, *expr.Source)
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
	require.True(t, dst.Spec.Type(ctx).Equal(recordingRuleSpecType))

	// The saver writes expressions in the API's JSON shape, so they're checked separately.
	got := dst.Spec.Attributes()
	for name, want := range src.Attributes() {
		if name == "expressions" {
			continue
		}
		require.True(t, got[name].Equal(want), "%s: got %s, want %s", name, got[name], want)
	}

	expr := got["expressions"].(types.Map).Elements()["A"].(types.String).ValueString()
	var wire map[string]any
	require.NoError(t, json.Unmarshal([]byte(expr), &wire))
	require.Equal(t, "ds_uid", wire["datasourceUID"])
}

func recordingRuleV1Beta1SpecObject(t *testing.T) types.Object {
	t.Helper()
	expressions, diags := types.MapValue(types.StringType, map[string]attr.Value{
		"A": types.StringValue(`{"model":{"expr":"up","refId":"A"},"datasource_uid":"ds_uid","relative_time_range":{"from":"10m","to":"0s"},"source":true}`),
	})
	require.False(t, diags.HasError())

	return objectValue(t, recordingRuleSpecType.AttrTypes, map[string]attr.Value{
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
