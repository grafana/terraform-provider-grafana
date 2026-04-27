package appplatform

import (
	"context"
	"testing"

	"github.com/grafana/grafana/apps/alerting/rules/pkg/apis/alerting/v0alpha1"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/stretchr/testify/require"
)

func TestParseNotificationSettingsLegacyFlatShape(t *testing.T) {
	ctx := context.Background()

	src := notificationSettingsObject(t, map[string]attr.Value{
		"contact_point":      types.StringValue("grafana-default-email"),
		"group_by":           stringListValue(t, "alertname", "grafana_folder"),
		"mute_timings":       stringListValue(t, "mute-a"),
		"active_timings":     stringListValue(t, "active-a"),
		"group_wait":         types.StringValue("45s"),
		"group_interval":     types.StringValue("6m"),
		"repeat_interval":    types.StringValue("3h"),
		"simplified_routing": types.ObjectNull(simplifiedRoutingType.AttrTypes),
		"named_routing_tree": types.ObjectNull(namedRoutingTreeType.AttrTypes),
	})

	settings, diags := parseNotificationSettings(ctx, src)
	require.False(t, diags.HasError())
	require.NotNil(t, settings.SimplifiedRouting)
	require.Nil(t, settings.NamedRoutingTree)
	require.Equal(t, "grafana-default-email", settings.SimplifiedRouting.Receiver)
	require.Equal(t, []string{"alertname", "grafana_folder"}, settings.SimplifiedRouting.GroupBy)
	require.Equal(t, "mute-a", string(settings.SimplifiedRouting.MuteTimeIntervals[0]))
	require.Equal(t, "active-a", string(settings.SimplifiedRouting.ActiveTimeIntervals[0]))
	require.Equal(t, "45s", string(*settings.SimplifiedRouting.GroupWait))
	require.Equal(t, "6m", string(*settings.SimplifiedRouting.GroupInterval))
	require.Equal(t, "3h", string(*settings.SimplifiedRouting.RepeatInterval))
}

func TestParseNotificationSettingsSimplifiedRoutingShape(t *testing.T) {
	ctx := context.Background()

	simplified := objectValue(t, simplifiedRoutingType.AttrTypes, map[string]attr.Value{
		"contact_point":   types.StringValue("grafana-default-email"),
		"group_by":        stringListValue(t, "alertname"),
		"mute_timings":    types.ListNull(types.StringType),
		"active_timings":  types.ListNull(types.StringType),
		"group_wait":      types.StringValue("45s"),
		"group_interval":  types.StringNull(),
		"repeat_interval": types.StringNull(),
	})

	src := notificationSettingsObject(t, map[string]attr.Value{
		"contact_point":      types.StringNull(),
		"group_by":           types.ListNull(types.StringType),
		"mute_timings":       types.ListNull(types.StringType),
		"active_timings":     types.ListNull(types.StringType),
		"group_wait":         types.StringNull(),
		"group_interval":     types.StringNull(),
		"repeat_interval":    types.StringNull(),
		"simplified_routing": simplified,
		"named_routing_tree": types.ObjectNull(namedRoutingTreeType.AttrTypes),
	})

	settings, diags := parseNotificationSettings(ctx, src)
	require.False(t, diags.HasError())
	require.NotNil(t, settings.SimplifiedRouting)
	require.Equal(t, "grafana-default-email", settings.SimplifiedRouting.Receiver)
	require.Equal(t, []string{"alertname"}, settings.SimplifiedRouting.GroupBy)
	require.Equal(t, "45s", string(*settings.SimplifiedRouting.GroupWait))
}

func TestParseNotificationSettingsNamedRoutingTreeShape(t *testing.T) {
	ctx := context.Background()

	namedRoutingTree := objectValue(t, namedRoutingTreeType.AttrTypes, map[string]attr.Value{
		"routing_tree": types.StringValue("team-a"),
	})

	src := notificationSettingsObject(t, map[string]attr.Value{
		"contact_point":      types.StringNull(),
		"group_by":           types.ListNull(types.StringType),
		"mute_timings":       types.ListNull(types.StringType),
		"active_timings":     types.ListNull(types.StringType),
		"group_wait":         types.StringNull(),
		"group_interval":     types.StringNull(),
		"repeat_interval":    types.StringNull(),
		"simplified_routing": types.ObjectNull(simplifiedRoutingType.AttrTypes),
		"named_routing_tree": namedRoutingTree,
	})

	settings, diags := parseNotificationSettings(ctx, src)
	require.False(t, diags.HasError())
	require.Nil(t, settings.SimplifiedRouting)
	require.NotNil(t, settings.NamedRoutingTree)
	require.Equal(t, "team-a", settings.NamedRoutingTree.RoutingTree)
}

func TestParseNotificationSettingsRejectsMixedShapes(t *testing.T) {
	ctx := context.Background()

	simplified := objectValue(t, simplifiedRoutingType.AttrTypes, map[string]attr.Value{
		"contact_point":   types.StringValue("grafana-default-email"),
		"group_by":        types.ListNull(types.StringType),
		"mute_timings":    types.ListNull(types.StringType),
		"active_timings":  types.ListNull(types.StringType),
		"group_wait":      types.StringNull(),
		"group_interval":  types.StringNull(),
		"repeat_interval": types.StringNull(),
	})

	src := notificationSettingsObject(t, map[string]attr.Value{
		"contact_point":      types.StringValue("legacy-contact-point"),
		"group_by":           types.ListNull(types.StringType),
		"mute_timings":       types.ListNull(types.StringType),
		"active_timings":     types.ListNull(types.StringType),
		"group_wait":         types.StringNull(),
		"group_interval":     types.StringNull(),
		"repeat_interval":    types.StringNull(),
		"simplified_routing": simplified,
		"named_routing_tree": types.ObjectNull(namedRoutingTreeType.AttrTypes),
	})

	_, diags := parseNotificationSettings(ctx, src)
	require.True(t, diags.HasError())
}

func TestParseNotificationSettingsLegacyFlatShapeRequiresContactPoint(t *testing.T) {
	ctx := context.Background()

	src := notificationSettingsObject(t, map[string]attr.Value{
		"contact_point":      types.StringNull(),
		"group_by":           stringListValue(t, "alertname"),
		"mute_timings":       types.ListNull(types.StringType),
		"active_timings":     types.ListNull(types.StringType),
		"group_wait":         types.StringNull(),
		"group_interval":     types.StringNull(),
		"repeat_interval":    types.StringNull(),
		"simplified_routing": types.ObjectNull(simplifiedRoutingType.AttrTypes),
		"named_routing_tree": types.ObjectNull(namedRoutingTreeType.AttrTypes),
	})

	_, diags := parseNotificationSettings(ctx, src)
	require.True(t, diags.HasError())
}

func TestSaveNotificationSettingsCanonicalizesToNestedShape(t *testing.T) {
	ctx := context.Background()

	saved, diags := saveNotificationSettings(ctx, &v0alpha1.AlertRuleNotificationSettings{
		SimplifiedRouting: &v0alpha1.AlertRuleSimplifiedRouting{
			Type:     v0alpha1.AlertRuleNotificationSettingsTypeSimplifiedRouting,
			Receiver: "grafana-default-email",
			GroupBy:  []string{"alertname"},
		},
	})
	require.False(t, diags.HasError())

	var data NotificationSettingsModel
	readDiags := saved.As(ctx, &data, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	})
	require.False(t, readDiags.HasError())
	require.True(t, data.ContactPoint.IsNull())
	require.True(t, data.GroupBy.IsNull())
	require.True(t, data.NamedRoutingTree.IsNull())
	require.False(t, data.SimplifiedRouting.IsNull())
}

func TestConfiguredNotificationSettingsShapes(t *testing.T) {
	legacyOnly := NotificationSettingsModel{ContactPoint: types.StringValue("legacy")}
	count, legacy, simplified, named := configuredNotificationSettingsShapes(legacyOnly)
	require.Equal(t, 1, count)
	require.True(t, legacy)
	require.False(t, simplified)
	require.False(t, named)
}

func notificationSettingsObject(t *testing.T, values map[string]attr.Value) types.Object {
	t.Helper()
	return objectValue(t, notificationSettingsType.AttrTypes, values)
}

func objectValue(t *testing.T, attrTypes map[string]attr.Type, values map[string]attr.Value) types.Object {
	t.Helper()
	v, diags := types.ObjectValue(attrTypes, values)
	require.False(t, diags.HasError())
	return v
}

func stringListValue(t *testing.T, values ...string) types.List {
	t.Helper()
	v, diags := types.ListValueFrom(context.Background(), types.StringType, values)
	require.False(t, diags.HasError())
	return v
}
