package appplatform

import (
	"context"
	"maps"

	"github.com/grafana/grafana/apps/alerting/rules/pkg/apis/alerting/v0alpha1"
	"github.com/grafana/grafana/apps/alerting/rules/pkg/apis/alerting/v1beta1"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// The v1beta1 notification_settings block drops the deprecated flat attributes of the v0alpha1 block,
// leaving only simplified_routing and named_routing_tree. The API requires notificationSettings.type
// in v1beta1, and the configured block always determines it.
var notificationSettingsV1Beta1Type = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"simplified_routing": simplifiedRoutingType,
		"named_routing_tree": namedRoutingTreeType,
	},
}

var alertRuleSpecV1Beta1Type = func() types.ObjectType {
	attrTypes := maps.Clone(alertRuleSpecType.AttrTypes)
	attrTypes["notification_settings"] = notificationSettingsV1Beta1Type
	return types.ObjectType{AttrTypes: attrTypes}
}()

// legacyNotificationSettingsNulls holds a null value for each v0alpha1 notification_settings
// attribute that v1beta1 doesn't have.
var legacyNotificationSettingsNulls = map[string]attr.Value{
	"contact_point":   types.StringNull(),
	"group_by":        types.ListNull(types.StringType),
	"mute_timings":    types.ListNull(types.StringType),
	"active_timings":  types.ListNull(types.StringType),
	"group_wait":      types.StringNull(),
	"group_interval":  types.StringNull(),
	"repeat_interval": types.StringNull(),
}

// AlertRuleV1Beta1 creates a new Grafana Alert Rule resource backed by the v1beta1 API.
// It reuses the v0alpha1 parser and saver through spec conversion, widening its notification_settings
// to the v0alpha1 shape before parsing and narrowing it back after saving.
func AlertRuleV1Beta1() NamedResource {
	spec := alertRuleSpecSchema(nfSettingsV1Beta1Block())
	spec.Description = "Manages Grafana Alert Rules."
	spec.MarkdownDescription = `
Manages Grafana Alert Rules.

This resource uses the ` + "`rules.alerting.grafana.app/v1beta1`" + ` API, which requires Grafana 13.3 or later.
`

	return NewNamedResource[*v1beta1.AlertRule, *v1beta1.AlertRuleList](
		common.CategoryAlerting,
		ResourceConfig[*v1beta1.AlertRule]{
			Kind:       v1beta1.AlertRuleKind(),
			Schema:     spec,
			SpecParser: parseAlertRuleV1Beta1Spec,
			SpecSaver:  saveAlertRuleV1Beta1Spec,
		})
}

var parseAlertRuleV0Alpha1AsV1Beta1 = specConvertingParser[*v0alpha1.AlertRule, *v1beta1.AlertRule, v1beta1.AlertRuleSpec](
	func() *v0alpha1.AlertRule { return &v0alpha1.AlertRule{} },
	parseAlertRuleSpec,
)

var saveAlertRuleV1Beta1AsV0Alpha1 = specConvertingSaver[*v1beta1.AlertRule, *v0alpha1.AlertRule, v0alpha1.AlertRuleSpec](
	func() *v0alpha1.AlertRule { return &v0alpha1.AlertRule{} },
	saveAlertRuleSpec,
)

func parseAlertRuleV1Beta1Spec(ctx context.Context, src types.Object, dst *v1beta1.AlertRule) diag.Diagnostics {
	widened, diags := widenAlertRuleV1Beta1Spec(src)
	if diags.HasError() {
		return diags
	}
	return parseAlertRuleV0Alpha1AsV1Beta1(ctx, widened, dst)
}

func saveAlertRuleV1Beta1Spec(ctx context.Context, src *v1beta1.AlertRule, dst *ResourceModel) diag.Diagnostics {
	if diags := saveAlertRuleV1Beta1AsV0Alpha1(ctx, src, dst); diags.HasError() {
		return diags
	}
	narrowed, diags := narrowAlertRuleV1Beta1Spec(dst.Spec)
	if diags.HasError() {
		return diags
	}
	dst.Spec = narrowed
	return diags
}

func nfSettingsV1Beta1Block() schema.Block {
	return schema.SingleNestedBlock{
		Description: "Notification settings for the rule. If specified, it overrides the notification policies. Specify exactly one of simplified_routing or named_routing_tree.",
		Blocks:      notificationRoutingBlocks(),
		Validators:  []validator.Object{notificationRoutingValidator{}},
	}
}

// notificationRoutingValidator requires exactly one of simplified_routing or named_routing_tree.
type notificationRoutingValidator struct{}

func (v notificationRoutingValidator) Description(context.Context) string {
	return "Exactly one of simplified_routing or named_routing_tree must be configured."
}

func (v notificationRoutingValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v notificationRoutingValidator) ValidateObject(ctx context.Context, req validator.ObjectRequest, resp *validator.ObjectResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	attrs := req.ConfigValue.Attributes()
	simplified, named := attrs["simplified_routing"], attrs["named_routing_tree"]
	if simplified.IsUnknown() || named.IsUnknown() {
		return
	}
	if simplified.IsNull() != named.IsNull() {
		return
	}

	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Invalid notification_settings configuration",
		v.Description(ctx),
	)
}

// widenAlertRuleV1Beta1Spec converts a v1beta1 spec value to the v0alpha1 spec shape by adding
// the legacy notification_settings attributes as nulls.
func widenAlertRuleV1Beta1Spec(spec types.Object) (types.Object, diag.Diagnostics) {
	return replaceNotificationSettings(spec, alertRuleSpecType, notificationSettingsType, func(attrs map[string]attr.Value) {
		maps.Copy(attrs, legacyNotificationSettingsNulls)
	})
}

// narrowAlertRuleV1Beta1Spec converts a v0alpha1-shaped spec value to the v1beta1 shape by dropping
// the legacy notification_settings attributes. The v0alpha1 saver never sets them.
func narrowAlertRuleV1Beta1Spec(spec types.Object) (types.Object, diag.Diagnostics) {
	return replaceNotificationSettings(spec, alertRuleSpecV1Beta1Type, notificationSettingsV1Beta1Type, func(attrs map[string]attr.Value) {
		for name := range legacyNotificationSettingsNulls {
			delete(attrs, name)
		}
	})
}

func replaceNotificationSettings(
	spec types.Object, specType, settingsType types.ObjectType, edit func(map[string]attr.Value),
) (types.Object, diag.Diagnostics) {
	switch {
	case spec.IsNull():
		return types.ObjectNull(specType.AttrTypes), nil
	case spec.IsUnknown():
		return types.ObjectUnknown(specType.AttrTypes), nil
	}

	specAttrs := maps.Clone(spec.Attributes())
	settings, ok := specAttrs["notification_settings"].(types.Object)
	switch {
	case !ok || settings.IsNull():
		specAttrs["notification_settings"] = types.ObjectNull(settingsType.AttrTypes)
	case settings.IsUnknown():
		specAttrs["notification_settings"] = types.ObjectUnknown(settingsType.AttrTypes)
	default:
		settingsAttrs := maps.Clone(settings.Attributes())
		edit(settingsAttrs)
		value, diags := types.ObjectValue(settingsType.AttrTypes, settingsAttrs)
		if diags.HasError() {
			return types.ObjectNull(specType.AttrTypes), diags
		}
		specAttrs["notification_settings"] = value
	}

	return types.ObjectValue(specType.AttrTypes, specAttrs)
}
