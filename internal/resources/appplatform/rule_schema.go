package appplatform

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// This file holds the Terraform schemas and models that every API version of the rules kinds
// shares. Nothing here uses the API's Go types, so each version parses and saves its spec with
// its own types. Fork a piece into the version's file when that version's schema diverges.

type AlertRuleSpecModel struct {
	Title                       types.String `tfsdk:"title"`
	Expressions                 types.Map    `tfsdk:"expressions"`
	Paused                      types.Bool   `tfsdk:"paused"`
	Trigger                     types.Object `tfsdk:"trigger"`
	NoDataState                 types.String `tfsdk:"no_data_state"`
	ExecErrState                types.String `tfsdk:"exec_err_state"`
	For                         types.String `tfsdk:"for"`
	KeepFiringFor               types.String `tfsdk:"keep_firing_for"`
	MissingSeriesEvalsToResolve types.Int64  `tfsdk:"missing_series_evals_to_resolve"`
	NotificationSettings        types.Object `tfsdk:"notification_settings"`
	Annotations                 types.Map    `tfsdk:"annotations"`
	Labels                      types.Map    `tfsdk:"labels"`
	PanelRef                    types.Map    `tfsdk:"panel_ref"`
}

var simplifiedRoutingType = types.ObjectType{
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

var namedRoutingTreeType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"routing_tree": types.StringType,
	},
}

type SimplifiedRoutingModel struct {
	ContactPoint   types.String `tfsdk:"contact_point"`
	GroupBy        types.List   `tfsdk:"group_by"`
	MuteTimings    types.List   `tfsdk:"mute_timings"`
	ActiveTimings  types.List   `tfsdk:"active_timings"`
	GroupWait      types.String `tfsdk:"group_wait"`
	GroupInterval  types.String `tfsdk:"group_interval"`
	RepeatInterval types.String `tfsdk:"repeat_interval"`
}

type NamedRoutingTreeModel struct {
	RoutingTree types.String `tfsdk:"routing_tree"`
}

// alertRuleSpecSchema returns the alert rule spec schema shared by every API version.
// Versions differ only in the notification_settings block and the descriptions.
func alertRuleSpecSchema(notificationSettings schema.Block) ResourceSpecSchema {
	return ResourceSpecSchema{
		SpecAttributes: map[string]schema.Attribute{
			"title": schema.StringAttribute{
				Required:    true,
				Description: "The title of the alert rule.",
			},
			"expressions": schema.MapAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "A sequence of stages that describe the contents of the rule. Each value is a JSON string representing an expression object.",
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
			"panel_ref": schema.MapAttribute{
				ElementType: types.StringType,
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
			"notification_settings": notificationSettings,
		},
	}
}

// notificationRoutingBlocks returns the simplified_routing and named_routing_tree blocks
// shared by the notification_settings block of every API version.
func notificationRoutingBlocks() map[string]schema.Block {
	return map[string]schema.Block{
		"simplified_routing": schema.SingleNestedBlock{
			Description: "Simplified routing to a contact point with optional grouping and timing overrides.",
			Validators:  []validator.Object{requireAttrsWhenPresent("contact_point")},
			Attributes: map[string]schema.Attribute{
				"contact_point": schema.StringAttribute{
					Optional:    true,
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
		},
		"named_routing_tree": schema.SingleNestedBlock{
			Description: "Route notifications to a specific routing tree.",
			Validators:  []validator.Object{requireAttrsWhenPresent("routing_tree")},
			Attributes: map[string]schema.Attribute{
				"routing_tree": schema.StringAttribute{
					Optional:    true,
					Description: "The name of the routing tree to use.",
				},
			},
		},
	}
}

type NullableObject interface {
	IsNull() bool
	IsUnknown() bool
}

func isNotNullOrUknown[T NullableObject](v T) bool {
	return !v.IsNull() && !v.IsUnknown()
}

var recordingRuleSpecType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"title":                 types.StringType,
		"expressions":           types.MapType{ElemType: types.StringType},
		"paused":                types.BoolType,
		"trigger":               ruleTriggerType,
		"metric":                types.StringType,
		"labels":                types.MapType{ElemType: types.StringType},
		"target_datasource_uid": types.StringType,
	},
}

type RecordingRuleSpecModel struct {
	Title               types.String `tfsdk:"title"`
	Expressions         types.Map    `tfsdk:"expressions"`
	Paused              types.Bool   `tfsdk:"paused"`
	Trigger             types.Object `tfsdk:"trigger"`
	Metric              types.String `tfsdk:"metric"`
	Labels              types.Map    `tfsdk:"labels"`
	TargetDatasourceUID types.String `tfsdk:"target_datasource_uid"`
}

// recordingRuleSpecSchema returns the recording rule spec schema shared by every API version.
func recordingRuleSpecSchema() ResourceSpecSchema {
	return ResourceSpecSchema{
		SpecAttributes: map[string]schema.Attribute{
			"title": schema.StringAttribute{
				Required:    true,
				Description: "The title of the recording rule.",
			},
			"expressions": schema.MapAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "A sequence of stages that describe the contents of the rule. Each value is a JSON string representing an expression object.",
			},
			"paused": schema.BoolAttribute{
				Optional:    true,
				Description: "Sets whether the recording rule should be paused or not.",
			},
			"metric": schema.StringAttribute{
				Required:    true,
				Description: "The name of the metric to write to.",
			},
			"labels": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Key-value pairs to attach to the recorded metric.",
			},
			"target_datasource_uid": schema.StringAttribute{
				Required:    true,
				Description: "The UID of the datasource to write the metric to.",
			},
		},
		SpecBlocks: map[string]schema.Block{
			"trigger": schema.SingleNestedBlock{
				Description: "The trigger configuration for the recording rule.",
				Attributes: map[string]schema.Attribute{
					"interval": schema.StringAttribute{
						Required:    true,
						Description: "The interval at which the recording rule should be evaluated.",
						Validators: []validator.String{
							PrometheusDurationValidator{},
						},
					},
				},
			},
		},
	}
}

var ruleSequenceRuleRefType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"name": types.StringType,
	},
}

type ruleSequenceRuleRefModel struct {
	Name types.String `tfsdk:"name"`
}

// ruleSequenceSpecSchema returns the rule sequence spec schema shared by every API version.
// alertRulesAttr names the alert rules attribute, which v1beta1 renamed from alerting_rules to alert_rules.
func ruleSequenceSpecSchema(alertRulesAttr string) ResourceSpecSchema {
	return ResourceSpecSchema{
		SpecAttributes: map[string]schema.Attribute{
			"recording_rules": schema.ListAttribute{
				Required:    true,
				ElementType: ruleSequenceRuleRefType,
				Description: "The recording rules that belong to this sequence, evaluated in the order listed. At least one recording rule is required. Each entry references a recording rule by its `name` (the rule's UID).",
			},
			alertRulesAttr: schema.ListAttribute{
				Optional:    true,
				ElementType: ruleSequenceRuleRefType,
				Description: "The alert rules that belong to this sequence, evaluated in the order listed. Each entry references an alert rule by its `name` (the rule's UID).",
			},
		},
		SpecBlocks: map[string]schema.Block{
			"trigger": schema.SingleNestedBlock{
				Description: "The trigger configuration shared by every rule in the sequence.",
				Attributes: map[string]schema.Attribute{
					"interval": schema.StringAttribute{
						Required:    true,
						Description: "The interval at which the rules in the sequence should be evaluated.",
						Validators: []validator.String{
							PrometheusDurationValidator{},
						},
					},
				},
			},
		},
	}
}
