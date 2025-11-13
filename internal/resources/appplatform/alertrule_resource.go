package appplatform

import (
	"context"
	"encoding/json"

	"github.com/grafana/grafana/apps/alerting/rules/pkg/apis/alerting/v0alpha1"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/util"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var alertRuleSpecType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"title":                           types.StringType,
		"expressions":                     types.DynamicType,
		"paused":                          types.BoolType,
		"trigger":                         ruleTriggerType,
		"no_data_state":                   types.StringType,
		"exec_err_state":                  types.StringType,
		"for":                             types.StringType,
		"keep_firing_for":                 types.StringType,
		"missing_series_evals_to_resolve": types.Int64Type,
		"notification_settings":           notificationSettingsType,
		"annotations":                     types.MapType{ElemType: types.StringType},
		"labels":                          types.MapType{ElemType: types.StringType},
		"panel_ref":                       types.DynamicType,
	},
}

type AlertRuleSpecModel struct {
	Title                       types.String  `tfsdk:"title"`
	Expressions                 types.Dynamic `tfsdk:"expressions"`
	Paused                      types.Bool    `tfsdk:"paused"`
	Trigger                     types.Object  `tfsdk:"trigger"`
	NoDataState                 types.String  `tfsdk:"no_data_state"`
	ExecErrState                types.String  `tfsdk:"exec_err_state"`
	For                         types.String  `tfsdk:"for"`
	KeepFiringFor               types.String  `tfsdk:"keep_firing_for"`
	MissingSeriesEvalsToResolve types.Int64   `tfsdk:"missing_series_evals_to_resolve"`
	NotificationSettings        types.Object  `tfsdk:"notification_settings"`
	Annotations                 types.Map     `tfsdk:"annotations"`
	Labels                      types.Map     `tfsdk:"labels"`
	PanelRef                    types.Dynamic `tfsdk:"panel_ref"`
}

var notificationSettingsType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"contact_point":   types.StringType,
		"group_by":        types.ListType{ElemType: types.StringType},
		"mute_timings":    types.ListType{ElemType: types.StringType},
		"active_timings":  types.ListType{ElemType: types.StringType},
		"group_wait":      types.StringType,
		"group_interval":  types.StringType,
		"repeat_interval": types.StringType,
	},
}

type NotificationSettingsModel struct {
	ContactPoint   types.String `tfsdk:"contact_point"`
	GroupBy        types.List   `tfsdk:"group_by"`
	MuteTimings    types.List   `tfsdk:"mute_timings"`
	ActiveTimings  types.List   `tfsdk:"active_timings"`
	GroupWait      types.String `tfsdk:"group_wait"`
	GroupInterval  types.String `tfsdk:"group_interval"`
	RepeatInterval types.String `tfsdk:"repeat_interval"`
}

var panelRefType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"dashboard_uid": types.StringType,
		"panel_id":      types.Int64Type,
	},
}

type PanelRefModel struct {
	DashboardUID types.String `tfsdk:"dashboard_uid"`
	PanelID      types.Int64  `tfsdk:"panel_id"`
}

func AlertRule() NamedResource {
	return NewNamedResource[*v0alpha1.AlertRule, *v0alpha1.AlertRuleList](
		common.CategoryAlerting,
		ResourceConfig[*v0alpha1.AlertRule]{
			Kind: v0alpha1.AlertRuleKind(),
			Schema: ResourceSpecSchema{
				Description: "Manages Grafana Alert Rules.",
				MarkdownDescription: `
Manages Grafana Alert Rules.
`,
				SpecAttributes: map[string]schema.Attribute{
					"title": schema.StringAttribute{
						Required:    true,
						Description: "The title of the alert rule.",
					},
					"expressions": schema.DynamicAttribute{
						Required:    true,
						Description: "A sequence of stages that describe the contents of the rule.",
						Validators: []validator.Dynamic{
							ExpressionsDynamicValidator{},
						},
					},
					"paused": schema.BoolAttribute{
						Optional:    true,
						Description: "Sets whether the rule should be paused or not.",
					},
					"no_data_state": schema.StringAttribute{
						Required:    true,
						Description: "Describes what state to enter when the rule's query returns No Data. Options are OK, NoData, KeepLast, and Alerting.",
					},
					"exec_err_state": schema.StringAttribute{
						Required:    true,
						Description: "Describes what state to enter when the rule's query is invalid and the rule cannot be executed. Options are OK, Error, KeepLast, and Alerting.",
					},
					"for": schema.StringAttribute{
						Optional:    true,
						Description: "The amount of time for which the rule must be breached for the rule to be considered to be Firing. Before this time has elapsed, the rule is only considered to be Pending.",
						Validators: []validator.String{
							PrometheusDurationValidator{},
						},
					},
					"keep_firing_for": schema.StringAttribute{
						Optional:    true,
						Description: "The amount of time for which the rule will considered to be Recovering after initially Firing. Before this time has elapsed, the rule will continue to fire once it's been triggered.",
						Validators: []validator.String{
							PrometheusDurationValidator{},
						},
					},
					"missing_series_evals_to_resolve": schema.Int64Attribute{
						Optional:    true,
						Description: "The number of missing series evaluations that must occur before the rule is considered to be resolved.",
					},
					"annotations": schema.MapAttribute{
						Optional:    true,
						ElementType: types.StringType,
						Description: "Key-value pairs of metadata to attach to the alert rule. They add additional information, such as a `summary` or `runbook_url`, to help identify and investigate alerts.",
					},
					"labels": schema.MapAttribute{
						Optional:    true,
						ElementType: types.StringType,
						Description: "Key-value pairs to attach to the alert rule that can be used in matching, grouping, and routing.",
					},
					"panel_ref": schema.DynamicAttribute{
						Optional:    true,
						Description: "Reference to a panel that this alert rule is associated with. Should be an object with 'dashboard_uid' (string) and 'panel_id' (number) fields.",
					},
				},
				SpecBlocks: map[string]schema.Block{
					"trigger": schema.SingleNestedBlock{
						Description: "The trigger configuration for the alert rule.",
						Attributes: map[string]schema.Attribute{
							"interval": schema.StringAttribute{
								Required:    true,
								Description: "The interval at which the alert rule should be evaluated.",
								Validators: []validator.String{
									PrometheusDurationValidator{},
								},
							},
						},
					},
					"notification_settings": nfSettingsBlock(),
				},
			},
			SpecParser: parseAlertRuleSpec,
			SpecSaver:  saveAlertRuleSpec,
		})
}

func nfSettingsBlock() schema.Block {
	return schema.SingleNestedBlock{
		Description: "Notification settings for the rule. If specified, it overrides the notification policies.",
		Attributes: map[string]schema.Attribute{
			"contact_point": schema.StringAttribute{
				Required:    true,
				Description: "The contact point to route notifications that match this rule to.",
			},
			"group_by": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "A list of alert labels to group alerts into notifications by.",
			},
			"mute_timings": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "A list of mute timing names to apply to alerts that match this policy.",
			},
			"active_timings": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "A list of time interval names to apply to alerts that match this policy to suppress them unless they are sent at the specified time.",
			},
			"group_wait": schema.StringAttribute{
				Optional:    true,
				Description: "Time to wait to buffer alerts of the same group before sending a notification.",
				Validators: []validator.String{
					PrometheusDurationValidator{},
				},
			},
			"group_interval": schema.StringAttribute{
				Optional:    true,
				Description: "Minimum time interval between two notifications for the same group.",
				Validators: []validator.String{
					PrometheusDurationValidator{},
				},
			},
			"repeat_interval": schema.StringAttribute{
				Optional:    true,
				Description: "Minimum time interval for re-sending a notification if an alert is still firing.",
				Validators: []validator.String{
					PrometheusDurationValidator{},
				},
			},
		},
	}
}

func parseAlertRuleBasicFields(data *AlertRuleSpecModel, spec *v0alpha1.AlertRuleSpec) {
	if !data.Paused.IsNull() && !data.Paused.IsUnknown() {
		spec.Paused = util.Ptr(data.Paused.ValueBool())
	}

	if !data.NoDataState.IsNull() && !data.NoDataState.IsUnknown() {
		spec.NoDataState = data.NoDataState.ValueString()
	}

	if !data.ExecErrState.IsNull() && !data.ExecErrState.IsUnknown() {
		spec.ExecErrState = data.ExecErrState.ValueString()
	}

	if !data.For.IsNull() && !data.For.IsUnknown() {
		spec.For = util.Ptr(data.For.ValueString())
	}

	if !data.KeepFiringFor.IsNull() && !data.KeepFiringFor.IsUnknown() {
		spec.KeepFiringFor = util.Ptr(data.KeepFiringFor.ValueString())
	}

	if !data.MissingSeriesEvalsToResolve.IsNull() && !data.MissingSeriesEvalsToResolve.IsUnknown() {
		spec.MissingSeriesEvalsToResolve = util.Ptr(data.MissingSeriesEvalsToResolve.ValueInt64())
	}
}

func parseAlertRuleAnnotations(ctx context.Context, data *AlertRuleSpecModel, spec *v0alpha1.AlertRuleSpec) diag.Diagnostics {
	if data.Annotations.IsNull() || data.Annotations.IsUnknown() {
		return nil
	}

	annotations := make(map[string]string)
	if diags := data.Annotations.ElementsAs(ctx, &annotations, false); diags.HasError() {
		return diags
	}
	spec.Annotations = make(map[string]v0alpha1.AlertRuleTemplateString)
	for k, v := range annotations {
		spec.Annotations[k] = v0alpha1.AlertRuleTemplateString(v)
	}
	return nil
}

func parseAlertRuleLabels(ctx context.Context, data *AlertRuleSpecModel, spec *v0alpha1.AlertRuleSpec) diag.Diagnostics {
	if data.Labels.IsNull() || data.Labels.IsUnknown() {
		return nil
	}

	labels := make(map[string]string)
	if diags := data.Labels.ElementsAs(ctx, &labels, false); diags.HasError() {
		return diags
	}
	spec.Labels = make(map[string]v0alpha1.AlertRuleTemplateString)
	for k, v := range labels {
		spec.Labels[k] = v0alpha1.AlertRuleTemplateString(v)
	}
	return nil
}

func parseAlertRuleExpressions(ctx context.Context, data *AlertRuleSpecModel, spec *v0alpha1.AlertRuleSpec) diag.Diagnostics {
	if data.Expressions.IsNull() || data.Expressions.IsUnknown() {
		return nil
	}

	expressionsMap, diags := ParseExpressionsFromDynamic(ctx, data.Expressions)
	if diags.HasError() {
		return diags
	}

	spec.Expressions = make(map[string]v0alpha1.AlertRuleExpression)
	for ref, obj := range expressionsMap {
		exprData, diags := parseAlertRuleExpressionModel(ctx, obj)
		if diags.HasError() {
			return diags
		}
		spec.Expressions[ref] = exprData
	}
	return nil
}

func parseAlertRuleSpec(ctx context.Context, src types.Object, dst *v0alpha1.AlertRule) diag.Diagnostics {
	var data AlertRuleSpecModel
	if diag := src.As(ctx, &data, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	}); diag.HasError() {
		return diag
	}

	spec := v0alpha1.AlertRuleSpec{
		Title: data.Title.ValueString(),
	}

	if !data.Trigger.IsNull() && !data.Trigger.IsUnknown() {
		trigger, diags := parseAlertRuleTrigger(ctx, data.Trigger)
		if diags.HasError() {
			return diags
		}
		spec.Trigger = trigger
	}

	parseAlertRuleBasicFields(&data, &spec)

	if !data.NotificationSettings.IsNull() && !data.NotificationSettings.IsUnknown() {
		notificationSettings, diag := parseNotificationSettings(ctx, data.NotificationSettings)
		if diag.HasError() {
			return diag
		}
		spec.NotificationSettings = &notificationSettings
	}

	if diags := parseAlertRuleAnnotations(ctx, &data, &spec); diags.HasError() {
		return diags
	}

	if diags := parseAlertRuleLabels(ctx, &data, &spec); diags.HasError() {
		return diags
	}

	if !data.PanelRef.IsNull() && !data.PanelRef.IsUnknown() {
		panelRef, diags := parsePanelRef(ctx, data.PanelRef)
		if diags.HasError() {
			return diags
		}
		spec.PanelRef = &panelRef
	}

	if diags := parseAlertRuleExpressions(ctx, &data, &spec); diags.HasError() {
		return diags
	}

	if err := dst.SetSpec(spec); err != nil {
		return diag.Diagnostics{
			diag.NewErrorDiagnostic("failed to set spec", err.Error()),
		}
	}

	return diag.Diagnostics{}
}

func saveAlertRuleSpec(ctx context.Context, src *v0alpha1.AlertRule, dst *ResourceModel) diag.Diagnostics {
	values := make(map[string]attr.Value)

	values["title"] = types.StringValue(src.Spec.Title)
	trigger, d := types.ObjectValueFrom(ctx, ruleTriggerType.AttrTypes, src.Spec.Trigger)
	if d.HasError() {
		return d
	}
	values["trigger"] = trigger
	if src.Spec.Paused != nil {
		values["paused"] = types.BoolValue(*src.Spec.Paused)
	} else {
		values["paused"] = types.BoolNull()
	}
	values["no_data_state"] = types.StringValue(src.Spec.NoDataState)
	values["exec_err_state"] = types.StringValue(src.Spec.ExecErrState)
	if src.Spec.For != nil {
		values["for"] = types.StringValue(*src.Spec.For)
	} else {
		values["for"] = types.StringNull()
	}
	if src.Spec.KeepFiringFor != nil {
		values["keep_firing_for"] = types.StringValue(*src.Spec.KeepFiringFor)
	} else {
		values["keep_firing_for"] = types.StringNull()
	}
	if src.Spec.MissingSeriesEvalsToResolve != nil {
		values["missing_series_evals_to_resolve"] = types.Int64Value(*src.Spec.MissingSeriesEvalsToResolve)
	} else {
		values["missing_series_evals_to_resolve"] = types.Int64Null()
	}
	nfSettings, d := types.ObjectValueFrom(ctx, notificationSettingsType.AttrTypes, src.Spec.NotificationSettings)
	if d.HasError() {
		return d
	}
	values["notification_settings"] = nfSettings
	if src.Spec.Annotations != nil {
		annotations, d := types.MapValueFrom(ctx, types.StringType, src.Spec.Annotations)
		if d.HasError() {
			return d
		}
		values["annotations"] = annotations
	} else {
		values["annotations"] = types.MapNull(types.StringType)
	}
	if src.Spec.Labels != nil {
		labels, d := types.MapValueFrom(ctx, types.StringType, src.Spec.Labels)
		if d.HasError() {
			return d
		}
		values["labels"] = labels
	} else {
		values["labels"] = types.MapNull(types.StringType)
	}
	if src.Spec.PanelRef != nil {
		panelRef, d := types.ObjectValueFrom(ctx, panelRefType.AttrTypes, src.Spec.PanelRef)
		if d.HasError() {
			return d
		}
		// Convert to dynamic
		dynamicValue := types.DynamicValue(panelRef)
		values["panel_ref"] = dynamicValue
	} else {
		values["panel_ref"] = types.DynamicNull()
	}
	if len(src.Spec.Expressions) > 0 {
		// Convert expressions to a map of objects for the dynamic type
		expressionsMap := make(map[string]attr.Value)
		for ref, expr := range src.Spec.Expressions {
			// Use the conversion function to parse JSON strings back to HCL objects
			exprObj, d := ConvertAPIExpressionToTerraform(ctx, expr, ruleExpressionType.AttrTypes)
			if d.HasError() {
				return d
			}
			expressionsMap[ref] = exprObj
		}
		// Use shared conversion function
		dynamicValue, d := ConvertExpressionsMapToDynamic(ctx, expressionsMap)
		if d.HasError() {
			return d
		}
		values["expressions"] = dynamicValue
	} else {
		// Set to null if no expressions
		values["expressions"] = types.DynamicNull()
	}

	spec, d := types.ObjectValue(alertRuleSpecType.AttrTypes, values)
	if d.HasError() {
		return d
	}
	dst.Spec = spec

	return diag.Diagnostics{}
}

// Parser helpers

func parseNotificationSettings(ctx context.Context, src types.Object) (v0alpha1.AlertRuleV0alpha1SpecNotificationSettings, diag.Diagnostics) {
	var data NotificationSettingsModel
	if diag := src.As(ctx, &data, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	}); diag.HasError() {
		return v0alpha1.AlertRuleV0alpha1SpecNotificationSettings{}, diag
	}

	result := v0alpha1.AlertRuleV0alpha1SpecNotificationSettings{
		Receiver: data.ContactPoint.ValueString(),
	}

	if !data.GroupBy.IsNull() && !data.GroupBy.IsUnknown() {
		var groupBy []string
		if diag := data.GroupBy.ElementsAs(ctx, &groupBy, false); diag.HasError() {
			return v0alpha1.AlertRuleV0alpha1SpecNotificationSettings{}, diag
		}
		result.GroupBy = groupBy
	}

	if !data.MuteTimings.IsNull() && !data.MuteTimings.IsUnknown() {
		var muteTimings []string
		if diag := data.MuteTimings.ElementsAs(ctx, &muteTimings, false); diag.HasError() {
			return v0alpha1.AlertRuleV0alpha1SpecNotificationSettings{}, diag
		}
		result.MuteTimeIntervals = make([]v0alpha1.AlertRuleTimeIntervalRef, len(muteTimings))
		for i, muteTiming := range muteTimings {
			result.MuteTimeIntervals[i] = v0alpha1.AlertRuleTimeIntervalRef(muteTiming)
		}
	}

	if !data.ActiveTimings.IsNull() && !data.ActiveTimings.IsUnknown() {
		var activeTimings []string
		if diag := data.ActiveTimings.ElementsAs(ctx, &activeTimings, false); diag.HasError() {
			return v0alpha1.AlertRuleV0alpha1SpecNotificationSettings{}, diag
		}
		result.ActiveTimeIntervals = make([]v0alpha1.AlertRuleTimeIntervalRef, len(activeTimings))
		for i, activeTiming := range activeTimings {
			result.ActiveTimeIntervals[i] = v0alpha1.AlertRuleTimeIntervalRef(activeTiming)
		}
	}

	if !data.GroupWait.IsNull() && !data.GroupWait.IsUnknown() {
		result.GroupWait = util.Ptr(v0alpha1.AlertRulePromDuration(data.GroupWait.ValueString()))
	}

	if !data.GroupInterval.IsNull() && !data.GroupInterval.IsUnknown() {
		result.GroupInterval = util.Ptr(v0alpha1.AlertRulePromDuration(data.GroupInterval.ValueString()))
	}

	if !data.RepeatInterval.IsNull() && !data.RepeatInterval.IsUnknown() {
		result.RepeatInterval = util.Ptr(v0alpha1.AlertRulePromDuration(data.RepeatInterval.ValueString()))
	}

	return result, diag.Diagnostics{}
}

func parseAlertRuleTrigger(ctx context.Context, src types.Object) (v0alpha1.AlertRuleIntervalTrigger, diag.Diagnostics) {
	var data RuleTriggerModel
	if diag := src.As(ctx, &data, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	}); diag.HasError() {
		return v0alpha1.AlertRuleIntervalTrigger{}, diag
	}
	return v0alpha1.AlertRuleIntervalTrigger{
		Interval: v0alpha1.AlertRulePromDuration(data.Interval.ValueString()),
	}, diag.Diagnostics{}
}

func parsePanelRef(ctx context.Context, src types.Dynamic) (v0alpha1.AlertRuleV0alpha1SpecPanelRef, diag.Diagnostics) {
	panelRefObj, ok := src.UnderlyingValue().(types.Object)
	if !ok {
		return v0alpha1.AlertRuleV0alpha1SpecPanelRef{}, diag.Diagnostics{
			diag.NewErrorDiagnostic("Invalid panel_ref type", "panel_ref must be an object with dashboard_uid and panel_id fields"),
		}
	}

	attrs := panelRefObj.Attributes()
	dashboardUID, ok := attrs["dashboard_uid"].(types.String)
	if !ok {
		return v0alpha1.AlertRuleV0alpha1SpecPanelRef{}, diag.Diagnostics{
			diag.NewErrorDiagnostic("Invalid panel_ref.dashboard_uid", "dashboard_uid must be a string"),
		}
	}

	panelID, ok := attrs["panel_id"].(types.Number)
	if !ok {
		return v0alpha1.AlertRuleV0alpha1SpecPanelRef{}, diag.Diagnostics{
			diag.NewErrorDiagnostic("Invalid panel_ref.panel_id", "panel_id must be a number"),
		}
	}

	panelIDBigFloat := panelID.ValueBigFloat()
	panelIDInt64, _ := panelIDBigFloat.Int64()

	return v0alpha1.AlertRuleV0alpha1SpecPanelRef{
		DashboardUID: dashboardUID.ValueString(),
		PanelID:      panelIDInt64,
	}, diag.Diagnostics{}
}

func parseAlertRuleRelativeTimeRange(ctx context.Context, src types.Object) (v0alpha1.AlertRuleRelativeTimeRange, diag.Diagnostics) {
	var data RelativeTimeRangeModel
	if diag := src.As(ctx, &data, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	}); diag.HasError() {
		return v0alpha1.AlertRuleRelativeTimeRange{}, diag
	}

	return v0alpha1.AlertRuleRelativeTimeRange{
		From: v0alpha1.AlertRulePromDurationWMillis(data.From.ValueString()),
		To:   v0alpha1.AlertRulePromDurationWMillis(data.To.ValueString()),
	}, diag.Diagnostics{}
}

func parseAlertRuleExpressionModel(ctx context.Context, src types.Object) (v0alpha1.AlertRuleExpression, diag.Diagnostics) {
	var srcExpr RuleExpressionModel
	if diag := src.As(ctx, &srcExpr, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	}); diag.HasError() {
		return v0alpha1.AlertRuleExpression{}, diag
	}

	dstExpr := v0alpha1.AlertRuleExpression{}

	// Model should be a map/object for the API, not a JSON string
	// Parse the JSON string back to a map
	if !srcExpr.Model.IsNull() && !srcExpr.Model.IsUnknown() {
		modelStr := srcExpr.Model.ValueString()
		var modelMap map[string]interface{}
		if err := json.Unmarshal([]byte(modelStr), &modelMap); err != nil {
			return v0alpha1.AlertRuleExpression{}, diag.Diagnostics{
				diag.NewErrorDiagnostic("Failed to parse model JSON", err.Error()),
			}
		}
		dstExpr.Model = modelMap
	}

	// Handle relative time range if present
	if !srcExpr.RelativeTimeRange.IsNull() && !srcExpr.RelativeTimeRange.IsUnknown() {
		relativeTimeRange, diags := parseAlertRuleRelativeTimeRange(ctx, srcExpr.RelativeTimeRange)
		if diags.HasError() {
			return v0alpha1.AlertRuleExpression{}, diags
		}
		dstExpr.RelativeTimeRange = &v0alpha1.AlertRuleRelativeTimeRange{
			From: relativeTimeRange.From,
			To:   relativeTimeRange.To,
		}
	}

	if srcExpr.QueryType.ValueString() != "" {
		dstExpr.QueryType = util.Ptr(srcExpr.QueryType.ValueString())
	}
	if srcExpr.DatasourceUID.ValueString() != "" {
		dstExpr.DatasourceUID = util.Ptr(v0alpha1.AlertRuleDatasourceUID(srcExpr.DatasourceUID.ValueString()))
	}
	if !srcExpr.Source.IsNull() && !srcExpr.Source.IsUnknown() {
		dstExpr.Source = util.Ptr(srcExpr.Source.ValueBool())
	}

	return dstExpr, diag.Diagnostics{}
}
