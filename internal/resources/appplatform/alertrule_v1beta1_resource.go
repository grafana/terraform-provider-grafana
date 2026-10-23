package appplatform

import (
	"context"
	"strconv"

	"github.com/grafana/grafana/apps/alerting/rules/pkg/apis/alerting/v1beta1"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/util"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
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

type notificationSettingsV1Beta1Model struct {
	SimplifiedRouting types.Object `tfsdk:"simplified_routing"`
	NamedRoutingTree  types.Object `tfsdk:"named_routing_tree"`
}

var alertRuleSpecV1Beta1Type = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"title":                           types.StringType,
		"expressions":                     types.MapType{ElemType: types.StringType},
		"paused":                          types.BoolType,
		"trigger":                         ruleTriggerType,
		"no_data_state":                   types.StringType,
		"exec_err_state":                  types.StringType,
		"for":                             types.StringType,
		"keep_firing_for":                 types.StringType,
		"missing_series_evals_to_resolve": types.Int64Type,
		"notification_settings":           notificationSettingsV1Beta1Type,
		"annotations":                     types.MapType{ElemType: types.StringType},
		"labels":                          types.MapType{ElemType: types.StringType},
		"panel_ref":                       types.MapType{ElemType: types.StringType},
	},
}

// AlertRuleV1Beta1 creates a new Grafana Alert Rule resource backed by the v1beta1 API.
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

func parseAlertRuleV1Beta1Spec(ctx context.Context, src types.Object, dst *v1beta1.AlertRule) diag.Diagnostics {
	var data AlertRuleSpecModel
	if diags := src.As(ctx, &data, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	}); diags.HasError() {
		return diags
	}

	spec := v1beta1.AlertRuleSpec{
		Title: data.Title.ValueString(),
	}

	if isNotNullOrUknown(data.Trigger) {
		var trigger RuleTriggerModel
		if diags := data.Trigger.As(ctx, &trigger, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty:    true,
			UnhandledUnknownAsEmpty: true,
		}); diags.HasError() {
			return diags
		}
		spec.Trigger = v1beta1.AlertRuleIntervalTrigger{
			Interval: v1beta1.AlertRulePromDuration(trigger.Interval.ValueString()),
		}
	}

	if isNotNullOrUknown(data.Paused) {
		spec.Paused = util.Ptr(data.Paused.ValueBool())
	}
	if isNotNullOrUknown(data.NoDataState) {
		spec.NoDataState = v1beta1.AlertRuleNoDataState(data.NoDataState.ValueString())
	}
	if isNotNullOrUknown(data.ExecErrState) {
		spec.ExecErrState = v1beta1.AlertRuleExecErrState(data.ExecErrState.ValueString())
	}
	if isNotNullOrUknown(data.For) {
		spec.For = util.Ptr(data.For.ValueString())
	}
	if isNotNullOrUknown(data.KeepFiringFor) {
		spec.KeepFiringFor = util.Ptr(data.KeepFiringFor.ValueString())
	}
	if isNotNullOrUknown(data.MissingSeriesEvalsToResolve) {
		spec.MissingSeriesEvalsToResolve = util.Ptr(data.MissingSeriesEvalsToResolve.ValueInt64())
	}

	if isNotNullOrUknown(data.NotificationSettings) {
		settings, diags := parseNotificationSettingsV1Beta1(ctx, data.NotificationSettings)
		if diags.HasError() {
			return diags
		}
		spec.NotificationSettings = settings
	}

	if isNotNullOrUknown(data.Annotations) {
		annotations, diags := parseTemplateStringMap[v1beta1.AlertRuleTemplateString](ctx, data.Annotations)
		if diags.HasError() {
			return diags
		}
		spec.Annotations = annotations
	}

	if isNotNullOrUknown(data.Labels) {
		labels, diags := parseTemplateStringMap[v1beta1.AlertRuleTemplateString](ctx, data.Labels)
		if diags.HasError() {
			return diags
		}
		spec.Labels = labels
	}

	if isNotNullOrUknown(data.PanelRef) {
		panelRef, diags := parsePanelRefV1Beta1(data.PanelRef)
		if diags.HasError() {
			return diags
		}
		spec.PanelRef = &panelRef
	}

	if isNotNullOrUknown(data.Expressions) {
		expressions, diags := parseAlertRuleExpressionsV1Beta1(ctx, data.Expressions)
		if diags.HasError() {
			return diags
		}
		spec.Expressions = expressions
	}

	if err := dst.SetSpec(spec); err != nil {
		return diag.Diagnostics{diag.NewErrorDiagnostic("failed to set spec", err.Error())}
	}
	return nil
}

func parseNotificationSettingsV1Beta1(ctx context.Context, src types.Object) (*v1beta1.AlertRuleNotificationSettings, diag.Diagnostics) {
	var data notificationSettingsV1Beta1Model
	if diags := src.As(ctx, &data, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	}); diags.HasError() {
		return nil, diags
	}

	switch {
	case isNotNullOrUknown(data.SimplifiedRouting) && !isNotNullOrUknown(data.NamedRoutingTree):
		routing, diags := parseSimplifiedRoutingV1Beta1(ctx, data.SimplifiedRouting)
		if diags.HasError() {
			return nil, diags
		}
		return &v1beta1.AlertRuleNotificationSettings{SimplifiedRouting: routing}, nil
	case isNotNullOrUknown(data.NamedRoutingTree) && !isNotNullOrUknown(data.SimplifiedRouting):
		var tree NamedRoutingTreeModel
		if diags := data.NamedRoutingTree.As(ctx, &tree, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty:    true,
			UnhandledUnknownAsEmpty: true,
		}); diags.HasError() {
			return nil, diags
		}
		return &v1beta1.AlertRuleNotificationSettings{
			NamedRoutingTree: &v1beta1.AlertRuleNamedRoutingTree{
				Type:        v1beta1.AlertRuleNotificationSettingsTypeNamedRoutingTree,
				RoutingTree: tree.RoutingTree.ValueString(),
			},
		}, nil
	default:
		return nil, diag.Diagnostics{diag.NewErrorDiagnostic(
			"Invalid notification_settings",
			"configure exactly one of simplified_routing or named_routing_tree",
		)}
	}
}

func parseSimplifiedRoutingV1Beta1(ctx context.Context, src types.Object) (*v1beta1.AlertRuleSimplifiedRouting, diag.Diagnostics) {
	var data SimplifiedRoutingModel
	if diags := src.As(ctx, &data, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	}); diags.HasError() {
		return nil, diags
	}

	result := &v1beta1.AlertRuleSimplifiedRouting{
		Type:     v1beta1.AlertRuleNotificationSettingsTypeSimplifiedRouting,
		Receiver: data.ContactPoint.ValueString(),
	}

	if isNotNullOrUknown(data.GroupBy) {
		if diags := data.GroupBy.ElementsAs(ctx, &result.GroupBy, false); diags.HasError() {
			return nil, diags
		}
	}
	if isNotNullOrUknown(data.MuteTimings) {
		if diags := data.MuteTimings.ElementsAs(ctx, &result.MuteTimeIntervals, false); diags.HasError() {
			return nil, diags
		}
	}
	if isNotNullOrUknown(data.ActiveTimings) {
		if diags := data.ActiveTimings.ElementsAs(ctx, &result.ActiveTimeIntervals, false); diags.HasError() {
			return nil, diags
		}
	}
	if isNotNullOrUknown(data.GroupWait) {
		result.GroupWait = util.Ptr(v1beta1.AlertRulePromDuration(data.GroupWait.ValueString()))
	}
	if isNotNullOrUknown(data.GroupInterval) {
		result.GroupInterval = util.Ptr(v1beta1.AlertRulePromDuration(data.GroupInterval.ValueString()))
	}
	if isNotNullOrUknown(data.RepeatInterval) {
		result.RepeatInterval = util.Ptr(v1beta1.AlertRulePromDuration(data.RepeatInterval.ValueString()))
	}

	return result, nil
}

func parsePanelRefV1Beta1(src types.Map) (v1beta1.AlertRulePanelRef, diag.Diagnostics) {
	attrs := src.Elements()
	dashboardUID, ok := attrs["dashboard_uid"].(types.String)
	if !ok {
		return v1beta1.AlertRulePanelRef{}, diag.Diagnostics{
			diag.NewErrorDiagnostic("Invalid panel_ref.dashboard_uid", "dashboard_uid must be a string"),
		}
	}

	panelIDStr, ok := attrs["panel_id"].(types.String)
	if !ok {
		return v1beta1.AlertRulePanelRef{}, diag.Diagnostics{
			diag.NewErrorDiagnostic("Invalid panel_ref.panel_id", "panel_id must be a number"),
		}
	}
	panelID, err := strconv.ParseInt(panelIDStr.ValueString(), 10, 64)
	if err != nil {
		return v1beta1.AlertRulePanelRef{}, diag.Diagnostics{
			diag.NewErrorDiagnostic("Invalid panel_ref.panel_id", "panel_id must be a number"),
		}
	}

	return v1beta1.AlertRulePanelRef{
		DashboardUID: dashboardUID.ValueString(),
		PanelID:      panelID,
	}, nil
}

func parseAlertRuleExpressionsV1Beta1(ctx context.Context, src types.Map) (v1beta1.AlertRuleExpressionMap, diag.Diagnostics) {
	expressions := make(v1beta1.AlertRuleExpressionMap)
	for ref, val := range src.Elements() {
		exprJSON, ok, diags := parseExpressionJSON(val)
		if diags.HasError() {
			return nil, diags
		}
		if !ok {
			continue
		}

		expr := v1beta1.AlertRuleExpression{}
		if model, ok := exprJSON["model"].(map[string]any); ok {
			expr.Model = model
		}
		if queryType, ok := exprJSON["query_type"].(string); ok && queryType != "" {
			expr.QueryType = util.Ptr(queryType)
		}
		if datasourceUID, ok := exprJSON["datasource_uid"].(string); ok && datasourceUID != "" {
			expr.DatasourceUID = util.Ptr(v1beta1.AlertRuleDatasourceUID(datasourceUID))
		}
		if source, ok := exprJSON["source"].(bool); ok {
			expr.Source = util.Ptr(source)
		}
		if relTimeRange, ok := exprJSON["relative_time_range"].(map[string]any); ok {
			from, _ := relTimeRange["from"].(string)
			to, _ := relTimeRange["to"].(string)
			if from != "" || to != "" {
				expr.RelativeTimeRange = &v1beta1.AlertRuleRelativeTimeRange{
					From: v1beta1.AlertRulePromDurationWMillis(from),
					To:   v1beta1.AlertRulePromDurationWMillis(to),
				}
			}
		}
		expressions[ref] = expr
	}
	return expressions, nil
}

func saveAlertRuleV1Beta1Spec(ctx context.Context, src *v1beta1.AlertRule, dst *ResourceModel) diag.Diagnostics {
	values := map[string]attr.Value{
		"title":                           types.StringValue(src.Spec.Title),
		"paused":                          types.BoolPointerValue(src.Spec.Paused),
		"no_data_state":                   types.StringValue(string(src.Spec.NoDataState)),
		"exec_err_state":                  types.StringValue(string(src.Spec.ExecErrState)),
		"for":                             types.StringPointerValue(src.Spec.For),
		"keep_firing_for":                 types.StringPointerValue(src.Spec.KeepFiringFor),
		"missing_series_evals_to_resolve": types.Int64PointerValue(src.Spec.MissingSeriesEvalsToResolve),
	}

	trigger, diags := types.ObjectValueFrom(ctx, ruleTriggerType.AttrTypes, RuleTriggerModel{
		Interval: types.StringValue(string(src.Spec.Trigger.Interval)),
	})
	if diags.HasError() {
		return diags
	}
	values["trigger"] = trigger

	settings, diags := saveNotificationSettingsV1Beta1(ctx, src.Spec.NotificationSettings)
	if diags.HasError() {
		return diags
	}
	values["notification_settings"] = settings

	annotations, diags := saveTemplateStringMap(ctx, src.Spec.Annotations)
	if diags.HasError() {
		return diags
	}
	values["annotations"] = annotations

	labels, diags := saveTemplateStringMap(ctx, src.Spec.Labels)
	if diags.HasError() {
		return diags
	}
	values["labels"] = labels

	values["panel_ref"] = types.MapNull(types.StringType)
	if src.Spec.PanelRef != nil {
		panelRef, diags := types.MapValue(types.StringType, map[string]attr.Value{
			"dashboard_uid": types.StringValue(src.Spec.PanelRef.DashboardUID),
			"panel_id":      types.StringValue(strconv.FormatInt(src.Spec.PanelRef.PanelID, 10)),
		})
		if diags.HasError() {
			return diags
		}
		values["panel_ref"] = panelRef
	}

	expressions, diags := saveExpressionsJSON(src.Spec.Expressions)
	if diags.HasError() {
		return diags
	}
	values["expressions"] = expressions

	spec, diags := types.ObjectValue(alertRuleSpecV1Beta1Type.AttrTypes, values)
	if diags.HasError() {
		return diags
	}
	dst.Spec = spec
	return nil
}

func saveNotificationSettingsV1Beta1(ctx context.Context, src *v1beta1.AlertRuleNotificationSettings) (types.Object, diag.Diagnostics) {
	if src == nil {
		return types.ObjectNull(notificationSettingsV1Beta1Type.AttrTypes), nil
	}

	values := map[string]attr.Value{
		"simplified_routing": types.ObjectNull(simplifiedRoutingType.AttrTypes),
		"named_routing_tree": types.ObjectNull(namedRoutingTreeType.AttrTypes),
	}

	switch {
	case src.SimplifiedRouting != nil:
		routing, diags := saveSimplifiedRoutingV1Beta1(ctx, src.SimplifiedRouting)
		if diags.HasError() {
			return types.ObjectNull(notificationSettingsV1Beta1Type.AttrTypes), diags
		}
		values["simplified_routing"] = routing
	case src.NamedRoutingTree != nil:
		tree, diags := types.ObjectValueFrom(ctx, namedRoutingTreeType.AttrTypes, NamedRoutingTreeModel{
			RoutingTree: types.StringValue(src.NamedRoutingTree.RoutingTree),
		})
		if diags.HasError() {
			return types.ObjectNull(notificationSettingsV1Beta1Type.AttrTypes), diags
		}
		values["named_routing_tree"] = tree
	default:
		return types.ObjectNull(notificationSettingsV1Beta1Type.AttrTypes), nil
	}

	return types.ObjectValue(notificationSettingsV1Beta1Type.AttrTypes, values)
}

func saveSimplifiedRoutingV1Beta1(ctx context.Context, src *v1beta1.AlertRuleSimplifiedRouting) (types.Object, diag.Diagnostics) {
	values := map[string]attr.Value{
		"contact_point":   types.StringValue(src.Receiver),
		"group_by":        types.ListNull(types.StringType),
		"mute_timings":    types.ListNull(types.StringType),
		"active_timings":  types.ListNull(types.StringType),
		"group_wait":      types.StringNull(),
		"group_interval":  types.StringNull(),
		"repeat_interval": types.StringNull(),
	}

	for name, list := range map[string][]string{
		"group_by":       src.GroupBy,
		"mute_timings":   stringSlice(src.MuteTimeIntervals),
		"active_timings": stringSlice(src.ActiveTimeIntervals),
	} {
		if len(list) == 0 {
			continue
		}
		value, diags := types.ListValueFrom(ctx, types.StringType, list)
		if diags.HasError() {
			return types.ObjectNull(simplifiedRoutingType.AttrTypes), diags
		}
		values[name] = value
	}

	if src.GroupWait != nil {
		values["group_wait"] = types.StringValue(string(*src.GroupWait))
	}
	if src.GroupInterval != nil {
		values["group_interval"] = types.StringValue(string(*src.GroupInterval))
	}
	if src.RepeatInterval != nil {
		values["repeat_interval"] = types.StringValue(string(*src.RepeatInterval))
	}

	return types.ObjectValue(simplifiedRoutingType.AttrTypes, values)
}
