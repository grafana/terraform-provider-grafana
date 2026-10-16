package appplatform

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/grafana/grafana/apps/alerting/rules/pkg/apis/alerting/v0alpha1"
	"github.com/grafana/grafana/apps/alerting/rules/pkg/apis/alerting/v1beta1"
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

func TestConvertRuleSpec(t *testing.T) {
	src := v0alpha1.RuleSequenceSpec{
		Trigger:        v0alpha1.RuleSequenceIntervalTrigger{Interval: "1m"},
		RecordingRules: []v0alpha1.RuleSequenceRuleRef{{Name: "recording"}},
		AlertingRules:  []v0alpha1.RuleSequenceRuleRef{{Name: "alerting"}},
	}

	dst, err := convertRuleSpec[v1beta1.RuleSequenceSpec](src)
	require.NoError(t, err)
	require.Equal(t, v1beta1.RuleSequencePromDuration("1m"), dst.Trigger.Interval)
	require.Equal(t, []v1beta1.RuleSequenceRuleRef{{Name: "recording"}}, dst.RecordingRules)
	require.Equal(t, []v1beta1.RuleSequenceRuleRef{{Name: "alerting"}}, dst.AlertingRules)
}

func alertRuleV1Beta1SpecObject(t *testing.T, settings types.Object) types.Object {
	t.Helper()
	expressions, diags := types.MapValue(types.StringType, map[string]attr.Value{
		"A": types.StringValue(`{"model":{"expr":"up","refId":"A"},"datasource_uid":"ds_uid","source":true}`),
	})
	require.False(t, diags.HasError())

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
		"panel_ref":                       types.MapNull(types.StringType),
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
