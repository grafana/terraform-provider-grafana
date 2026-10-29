package appplatform

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/grafana/grafana/apps/alerting/rules/pkg/apis/alerting/v1beta1"
	"github.com/grafana/terraform-provider-grafana/v4/internal/util"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

func TestParseAlertRuleV1Beta1SpecSetsNotificationSettingsType(t *testing.T) {
	ctx := context.Background()

	for _, tc := range []struct {
		name         string
		settings     types.Object
		expectedType string
	}{
		{
			name: "simplified routing",
			settings: notificationSettingsV1Beta1Object(t,
				simplifiedRoutingObject(t, "grafana-default-email"),
				types.ObjectNull(namedRoutingTreeType.AttrTypes),
			),
			expectedType: "SimplifiedRouting",
		},
		{
			name: "named routing tree",
			settings: notificationSettingsV1Beta1Object(t,
				types.ObjectNull(simplifiedRoutingType.AttrTypes),
				objectValue(t, namedRoutingTreeType.AttrTypes, map[string]attr.Value{
					"routing_tree": types.StringValue("my-tree"),
				}),
			),
			expectedType: "NamedRoutingTree",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var dst v1beta1.AlertRule
			diags := parseAlertRuleV1Beta1Spec(ctx, alertRuleV1Beta1SpecObject(t, tc.settings), &dst)
			require.False(t, diags.HasError(), diags)
			require.Equal(t, "Test rule", dst.Spec.Title)
			require.Equal(t, v1beta1.AlertRulePromDuration("1m"), dst.Spec.Trigger.Interval)
			require.Contains(t, dst.Spec.Expressions, "A")
			require.NotNil(t, dst.Spec.NotificationSettings)

			raw, err := json.Marshal(dst.Spec.NotificationSettings)
			require.NoError(t, err)
			var wire map[string]any
			require.NoError(t, json.Unmarshal(raw, &wire))
			require.Equal(t, tc.expectedType, wire["type"])
		})
	}
}

func TestParseAlertRuleV1Beta1SpecWithoutNotificationSettings(t *testing.T) {
	var dst v1beta1.AlertRule
	diags := parseAlertRuleV1Beta1Spec(context.Background(), alertRuleV1Beta1SpecObject(t, types.ObjectNull(notificationSettingsV1Beta1Type.AttrTypes)), &dst)
	require.False(t, diags.HasError(), diags)
	require.Nil(t, dst.Spec.NotificationSettings)
}

func TestSaveAlertRuleV1Beta1SpecRoundTrip(t *testing.T) {
	ctx := context.Background()
	settings := notificationSettingsV1Beta1Object(t,
		simplifiedRoutingObject(t, "grafana-default-email"),
		types.ObjectNull(namedRoutingTreeType.AttrTypes),
	)

	var rule v1beta1.AlertRule
	diags := parseAlertRuleV1Beta1Spec(ctx, alertRuleV1Beta1SpecObject(t, settings), &rule)
	require.False(t, diags.HasError(), diags)

	var dst ResourceModel
	diags = saveAlertRuleV1Beta1Spec(ctx, &rule, &dst)
	require.False(t, diags.HasError(), diags)
	require.True(t, dst.Spec.Type(ctx).Equal(alertRuleSpecV1Beta1Type))

	attrs := dst.Spec.Attributes()
	require.Equal(t, types.StringValue("Test rule"), attrs["title"])
	require.True(t, attrs["notification_settings"].Equal(settings))
}

func TestNotificationRoutingValidator(t *testing.T) {
	ctx := context.Background()
	simplified := simplifiedRoutingObject(t, "grafana-default-email")
	named := objectValue(t, namedRoutingTreeType.AttrTypes, map[string]attr.Value{
		"routing_tree": types.StringValue("my-tree"),
	})
	nullSimplified := types.ObjectNull(simplifiedRoutingType.AttrTypes)
	nullNamed := types.ObjectNull(namedRoutingTreeType.AttrTypes)

	for _, tc := range []struct {
		name      string
		value     types.Object
		expectErr bool
	}{
		{name: "block absent", value: types.ObjectNull(notificationSettingsV1Beta1Type.AttrTypes)},
		{name: "simplified routing", value: notificationSettingsV1Beta1Object(t, simplified, nullNamed)},
		{name: "named routing tree", value: notificationSettingsV1Beta1Object(t, nullSimplified, named)},
		{name: "both", value: notificationSettingsV1Beta1Object(t, simplified, named), expectErr: true},
		{name: "neither", value: notificationSettingsV1Beta1Object(t, nullSimplified, nullNamed), expectErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp := &validator.ObjectResponse{}
			notificationRoutingValidator{}.ValidateObject(ctx, validator.ObjectRequest{
				Path:        path.Root("spec").AtName("notification_settings"),
				ConfigValue: tc.value,
			}, resp)
			require.Equal(t, tc.expectErr, resp.Diagnostics.HasError())
		})
	}
}

func alertRuleV1Beta1SpecObject(t *testing.T, settings types.Object) types.Object {
	t.Helper()
	expressions := expressionsV1Beta1Value(t, map[string]types.Object{
		"A": expressionV1Beta1Value(t, map[string]attr.Value{
			"model":          types.StringValue(`{"expr":"up","refId":"A"}`),
			"datasource_uid": types.StringValue("ds_uid"),
			"source":         types.BoolValue(true),
		}),
	})

	return objectValue(t, alertRuleSpecV1Beta1Type.AttrTypes, map[string]attr.Value{
		"title":       types.StringValue("Test rule"),
		"expressions": expressions,
		"paused":      types.BoolNull(),
		"trigger": objectValue(t, ruleTriggerType.AttrTypes, map[string]attr.Value{
			"interval": types.StringValue("1m"),
		}),
		"no_data_state":                   types.StringValue("NoData"),
		"exec_err_state":                  types.StringValue("Error"),
		"for":                             types.StringNull(),
		"keep_firing_for":                 types.StringNull(),
		"missing_series_evals_to_resolve": types.Int64Null(),
		"notification_settings":           settings,
		"annotations":                     types.MapNull(types.StringType),
		"labels":                          types.MapNull(types.StringType),
		"panel_ref":                       types.ObjectNull(panelRefV1Beta1Type.AttrTypes),
	})
}

func notificationSettingsV1Beta1Object(t *testing.T, simplified, named types.Object) types.Object {
	t.Helper()
	return objectValue(t, notificationSettingsV1Beta1Type.AttrTypes, map[string]attr.Value{
		"simplified_routing": simplified,
		"named_routing_tree": named,
	})
}

func simplifiedRoutingObject(t *testing.T, contactPoint string) types.Object {
	t.Helper()
	return objectValue(t, simplifiedRoutingType.AttrTypes, map[string]attr.Value{
		"contact_point":   types.StringValue(contactPoint),
		"group_by":        types.ListNull(types.StringType),
		"mute_timings":    types.ListNull(types.StringType),
		"active_timings":  types.ListNull(types.StringType),
		"group_wait":      types.StringNull(),
		"group_interval":  types.StringNull(),
		"repeat_interval": types.StringNull(),
	})
}

func TestAlertRuleV1Beta1SpecAllFieldsRoundTrip(t *testing.T) {
	ctx := context.Background()

	for _, tc := range []struct {
		name     string
		settings types.Object
	}{
		{
			name: "simplified routing",
			settings: notificationSettingsV1Beta1Object(t,
				objectValue(t, simplifiedRoutingType.AttrTypes, map[string]attr.Value{
					"contact_point":   types.StringValue("grafana-default-email"),
					"group_by":        stringListValue(t, "alertname", "team"),
					"mute_timings":    stringListValue(t, "weekends"),
					"active_timings":  stringListValue(t, "business-hours"),
					"group_wait":      types.StringValue("30s"),
					"group_interval":  types.StringValue("5m"),
					"repeat_interval": types.StringValue("4h"),
				}),
				types.ObjectNull(namedRoutingTreeType.AttrTypes),
			),
		},
		{
			name: "named routing tree",
			settings: notificationSettingsV1Beta1Object(t,
				types.ObjectNull(simplifiedRoutingType.AttrTypes),
				objectValue(t, namedRoutingTreeType.AttrTypes, map[string]attr.Value{
					"routing_tree": types.StringValue("my-tree"),
				}),
			),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := alertRuleV1Beta1SpecObject(t, tc.settings)
			attrs := src.Attributes()
			attrs["paused"] = types.BoolValue(true)
			attrs["for"] = types.StringValue("5m")
			attrs["keep_firing_for"] = types.StringValue("1m")
			attrs["missing_series_evals_to_resolve"] = types.Int64Value(3)
			attrs["annotations"] = stringMapValue(t, map[string]string{"summary": "{{ $labels.instance }} is down"})
			attrs["labels"] = stringMapValue(t, map[string]string{"team": "alerting"})
			attrs["panel_ref"] = objectValue(t, panelRefV1Beta1Type.AttrTypes, map[string]attr.Value{
				"dashboard_uid": types.StringValue("dash"),
				"panel_id":      types.Int64Value(12),
			})
			attrs["expressions"] = expressionsV1Beta1Value(t, map[string]types.Object{
				"A": expressionV1Beta1Value(t, map[string]attr.Value{
					"query_type":          types.StringValue("instant"),
					"relative_time_range": timeRangeValue(t, "10m", "0s"),
					"datasource_uid":      types.StringValue("ds_uid"),
					"model":               types.StringValue(`{"expr":"up","refId":"A"}`),
					"source":              types.BoolValue(false),
				}),
				"B": expressionV1Beta1Value(t, map[string]attr.Value{
					"datasource_uid": types.StringValue("__expr__"),
					"model":          types.StringValue(`{"expression":"A","refId":"B","type":"threshold"}`),
					"source":         types.BoolValue(true),
				}),
			})
			src = objectValue(t, alertRuleSpecV1Beta1Type.AttrTypes, attrs)

			var rule v1beta1.AlertRule
			diags := parseAlertRuleV1Beta1Spec(ctx, src, &rule)
			require.False(t, diags.HasError(), diags)
			require.Equal(t, &v1beta1.AlertRulePanelRef{DashboardUID: "dash", PanelID: 12}, rule.Spec.PanelRef)
			require.Equal(t, v1beta1.AlertRuleTemplateString("alerting"), rule.Spec.Labels["team"])
			require.Equal(t, v1beta1.AlertRuleExpression{
				QueryType:         util.Ptr("instant"),
				RelativeTimeRange: &v1beta1.AlertRuleRelativeTimeRange{From: "10m", To: "0s"},
				DatasourceUID:     util.Ptr(v1beta1.AlertRuleDatasourceUID("ds_uid")),
				Model:             map[string]any{"expr": "up", "refId": "A"},
				Source:            util.Ptr(false),
			}, rule.Spec.Expressions["A"])

			var dst ResourceModel
			diags = saveAlertRuleV1Beta1Spec(ctx, &rule, &dst)
			require.False(t, diags.HasError(), diags)

			got := dst.Spec.Attributes()
			for name, want := range attrs {
				require.True(t, got[name].Equal(want), "%s: got %s, want %s", name, got[name], want)
			}
		})
	}
}

// TestAlertRuleV1Beta1ImportedExpressionsSurviveApply covers the import path: the saver writes
// what the API returned, and parsing that state must send the same expressions back.
func TestAlertRuleV1Beta1ImportedExpressionsSurviveApply(t *testing.T) {
	ctx := context.Background()
	imported := v1beta1.AlertRule{Spec: v1beta1.AlertRuleSpec{
		Title:   "Imported",
		Trigger: v1beta1.AlertRuleIntervalTrigger{Interval: "1m"},
		Expressions: v1beta1.AlertRuleExpressionMap{
			"A": {
				QueryType:         util.Ptr("range"),
				RelativeTimeRange: &v1beta1.AlertRuleRelativeTimeRange{From: "10m", To: "0s"},
				DatasourceUID:     util.Ptr(v1beta1.AlertRuleDatasourceUID("ds_uid")),
				Model:             map[string]any{"expr": "up", "refId": "A"},
			},
			"B": {
				Model:  map[string]any{"expression": "A", "refId": "B", "type": "threshold"},
				Source: util.Ptr(true),
			},
		},
		NoDataState:  "NoData",
		ExecErrState: "Error",
		PanelRef:     &v1beta1.AlertRulePanelRef{DashboardUID: "dash", PanelID: 5},
	}}

	var state ResourceModel
	diags := saveAlertRuleV1Beta1Spec(ctx, &imported, &state)
	require.False(t, diags.HasError(), diags)

	var applied v1beta1.AlertRule
	diags = parseAlertRuleV1Beta1Spec(ctx, state.Spec, &applied)
	require.False(t, diags.HasError(), diags)
	require.Equal(t, imported.Spec.Expressions, applied.Spec.Expressions)
	require.Equal(t, imported.Spec.PanelRef, applied.Spec.PanelRef)
}
