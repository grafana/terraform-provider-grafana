package assistant

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common/assistantapi"
)

const resourceAutomationName = "grafana_assistant_automation"

var resourceAutomationID = common.NewResourceID(common.StringIDField("id"))

type automationResource struct {
	client *assistantapi.Client
}

type automationSlackTargetModel struct {
	Type      types.String `tfsdk:"type"`
	ChannelID types.String `tfsdk:"channel_id"`
	UserID    types.String `tfsdk:"user_id"`
	TeamID    types.String `tfsdk:"team_id"`
}

type automationSlackModel struct {
	Enabled  types.Bool                  `tfsdk:"enabled"`
	NotifyOn types.List                  `tfsdk:"notify_on"`
	Target   *automationSlackTargetModel `tfsdk:"target"`
}

type automationEmailModel struct {
	Enabled   types.Bool `tfsdk:"enabled"`
	NotifyOn  types.List `tfsdk:"notify_on"`
	Addresses types.List `tfsdk:"addresses"`
}

type automationNotificationsModel struct {
	Slack *automationSlackModel `tfsdk:"slack"`
	Email *automationEmailModel `tfsdk:"email"`
}

type automationModel struct {
	ID               types.String                  `tfsdk:"id"`
	Name             types.String                  `tfsdk:"name"`
	Description      types.String                  `tfsdk:"description"`
	Prompt           types.String                  `tfsdk:"prompt"`
	Enabled          types.Bool                    `tfsdk:"enabled"`
	ExecutionMode    types.String                  `tfsdk:"execution_mode"`
	Triggers         types.Map                     `tfsdk:"triggers"`
	Labels           types.Map                     `tfsdk:"labels"`
	ScheduleEndsAt   types.String                  `tfsdk:"schedule_ends_at"`
	Scope            types.String                  `tfsdk:"scope"`
	ScheduleCron     types.String                  `tfsdk:"schedule_cron"`
	ScheduleTimezone types.String                  `tfsdk:"schedule_timezone"`
	NextRunAt        types.String                  `tfsdk:"next_run_at"`
	ContextItems     types.String                  `tfsdk:"context_items"`
	Notifications    *automationNotificationsModel `tfsdk:"notifications"`
}

func makeResourceAutomation() *common.Resource {
	return common.NewResource(
		common.CategoryGrafanaAssistant,
		resourceAutomationName,
		resourceAutomationID,
		&automationResource{},
	)
}

func (r *automationResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceAutomationName
}

func (r *automationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Manages a Grafana Assistant Automation: a saved Assistant prompt that runs on
demand or on a cron schedule. Each run creates its own Assistant conversation.

Automations are a Grafana Cloud-only feature and require the Grafana Assistant app.

execution_mode selects the identity used for runs. The default user mode uses the creator's
Grafana permissions. service_account mode requires tenant scope and organization Admin
permission to configure. Authenticating Terraform with a service-account token does not
automatically select service_account execution. Runtime memory and run history remain backend-owned.`,
		Attributes: map[string]schema.Attribute{
			"id": idAttribute(),
			"name": schema.StringAttribute{
				Description: "Human-friendly automation name.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Optional description.",
				Optional:    true,
			},
			"prompt": schema.StringAttribute{
				Description: "The prompt sent to Assistant on each run. Prefix with `/<skill-name>` to start from a skill.",
				Required:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the automation runs on its schedule. Disabled automations can still be run manually.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"scope": schema.StringAttribute{
				Description: "Visibility. `user` keeps the automation private to its creator; `tenant` shares it with everybody in the org.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("user"),
				Validators: []validator.String{
					stringvalidator.OneOf("user", "tenant"),
				},
			},
			"execution_mode": schema.StringAttribute{
				Description: "Run identity: user (default) or service_account. service_account requires scope = tenant and Grafana organization Admin permission, including updates to an existing service-account automation.",
				Optional:    true, Computed: true, Default: stringdefault.StaticString("user"),
				Validators: []validator.String{stringvalidator.OneOf("user", "service_account")},
			},
			// A map of object values is supported by the provider's v5 mux.
			// MapNestedAttribute requires protocol v6 and cannot be served here.
			"triggers": schema.MapAttribute{
				Description: "Event triggers keyed by stable, lowercase UUIDs. Each value requires type (backend trigger type) and config (a JSON object string, normally jsonencode). Keep UUIDs unchanged when editing. Omit or use an empty map to remove all triggers. GitHub and Watcher triggers require service_account execution; Slack triggers run as the mapped event author. Backend defaults and equivalent JSON formatting do not cause drift.",
				Optional:    true,
				ElementType: automationTriggerObjectType,
			},
			"labels": schema.MapAttribute{
				Description: "Operator labels. Omit or use an empty map to remove managed labels.", Optional: true, ElementType: types.StringType,
			},
			"schedule_ends_at": schema.StringAttribute{
				Description: "Inclusive schedule deadline in RFC3339 format with second precision. Requires schedule_cron and backend support for bounded schedules. Omit to remove the deadline. New or changed deadlines must be in the future.", Optional: true,
			},
			"schedule_cron": schema.StringAttribute{
				Description: "Standard 5-field cron expression, or multiple expressions separated by semicolons. For example `0 9 * * 1-5; 0 10 * * 0`. Omit for a manual-only automation. Combined firings must be at least 15 minutes apart.",
				Optional:    true,
			},
			"schedule_timezone": schema.StringAttribute{
				Description: "IANA timezone the schedule is interpreted in, for example `Europe/Paris`. Defaults to UTC when schedule_cron is set; absent for manual-only automations.",
				Optional:    true,
				Computed:    true,
			},
			"next_run_at": schema.StringAttribute{
				Description: "When the automation is next scheduled to fire.",
				Computed:    true,
			},
			"context_items": schema.StringAttribute{
				Description: "Context items attached to the prompt, as a JSON array string.",
				Optional:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"notifications": schema.SingleNestedBlock{
				Description: "Where to send the result of each run. Webhook notifications are not supported by this resource because their URL and credentials are write-only secure settings the API never returns.",
				Blocks: map[string]schema.Block{
					"slack": schema.SingleNestedBlock{
						Description: "Slack delivery. Requires the Slack workspace to already be connected to the stack, which is a one-time manual step in Assistant settings.",
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								Description: "Whether Slack notifications are sent.",
								Optional:    true,
								Computed:    true,
								Default:     booldefault.StaticBool(false),
							},
							"notify_on": schema.ListAttribute{
								Description: "Run outcomes that trigger a notification. Any of `completed`, `failed`, `needs_approval`.",
								Validators:  []validator.List{listvalidator.ValueStringsAre(stringvalidator.OneOf("completed", "failed", "needs_approval"))},
								Optional:    true,
								ElementType: types.StringType,
							},
						},
						Blocks: map[string]schema.Block{
							"target": schema.SingleNestedBlock{
								Description: "Where the notification is posted.",
								Attributes: map[string]schema.Attribute{
									// Optional, not Required: Terraform enforces Required
									// attributes inside a SingleNestedBlock even when
									// the block is omitted, which would break every
									// automation that configures no notifications.
									"type": schema.StringAttribute{
										Description: "Target type. One of `channel` or `dm`. Required when a `target` block is present.",
										Optional:    true,
										Validators: []validator.String{
											stringvalidator.OneOf("channel", "dm"),
										},
									},
									"channel_id": schema.StringAttribute{
										Description: "Slack channel ID. Required for a `channel` target.",
										Optional:    true,
									},
									"user_id": schema.StringAttribute{
										Description: "Slack user ID for a `dm` target. Omit to use the creator’s verified Slack mapping.",
										Optional:    true,
									},
									"team_id": schema.StringAttribute{
										Description: "Slack workspace (team) ID. Omit to let Assistant resolve it; automatically resolved IDs are not managed unless configured.",
										Optional:    true,
									},
								},
							},
						},
					},
					"email": schema.SingleNestedBlock{
						Description: "Email delivery. Email notifications must be enabled for the stack.",
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								Description: "Whether email notifications are sent.",
								Optional:    true,
								Computed:    true,
								Default:     booldefault.StaticBool(false),
							},
							"notify_on": schema.ListAttribute{
								Description: "Run outcomes that trigger a notification. Any of `completed`, `failed`, `needs_approval`.",
								Validators:  []validator.List{listvalidator.ValueStringsAre(stringvalidator.OneOf("completed", "failed", "needs_approval"))},
								Optional:    true,
								ElementType: types.StringType,
							},
							"addresses": schema.ListAttribute{
								Description: "Recipient email addresses, up to 25.",
								Optional:    true,
								ElementType: types.StringType,
							},
						},
					},
				},
			},
		},
	}
}

func (r *automationResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var cron, timezone types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("schedule_cron"), &cron)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("schedule_timezone"), &timezone)...)
	if !timezone.IsNull() && cron.IsNull() {
		resp.Diagnostics.AddAttributeError(path.Root("schedule_timezone"), "Schedule required",
			"schedule_timezone requires schedule_cron. Omit both for a manual-only automation.")
	}
	if !cron.IsNull() && !cron.IsUnknown() && strings.TrimSpace(cron.ValueString()) == "" {
		resp.Diagnostics.AddAttributeError(path.Root("schedule_cron"), "Empty schedule", "Omit schedule_cron instead of setting an empty string.")
	}
	var mode, scope, deadline types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("execution_mode"), &mode)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("scope"), &scope)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("schedule_ends_at"), &deadline)...)
	if mode.ValueString() == "service_account" && !scope.IsUnknown() && scope.ValueString() != "tenant" {
		resp.Diagnostics.AddAttributeError(path.Root("execution_mode"), "Tenant scope required", "execution_mode = service_account requires scope = tenant.")
	}
	if !deadline.IsNull() {
		if cron.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("schedule_ends_at"), "Schedule required", "schedule_ends_at requires schedule_cron.")
		}
		if !deadline.IsUnknown() {
			parsed, err := time.Parse(time.RFC3339, deadline.ValueString())
			if err != nil || parsed.Nanosecond() != 0 {
				resp.Diagnostics.AddAttributeError(path.Root("schedule_ends_at"), "Invalid schedule deadline", "Use an RFC3339 timestamp with second precision, for example 2030-01-01T00:00:00Z.")
			}
		}
	}

	var triggers types.Map
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("triggers"), &triggers)...)
	resp.Diagnostics.Append(validateAutomationTriggers(ctx, triggers, mode)...)
}

// Timezone is a default of a schedule, not of an automation. Reading config
// rather than prior state also clears the old timezone when a schedule is removed.
func (r *automationResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	var cron, timezone types.String
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("schedule_cron"), &cron)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("schedule_timezone"), &timezone)...)
	if resp.Diagnostics.HasError() || !timezone.IsNull() {
		return
	}
	switch {
	case cron.IsUnknown():
		timezone = types.StringUnknown()
	case cron.IsNull():
		timezone = types.StringNull()
	default:
		timezone = types.StringValue("UTC")
	}
	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("schedule_timezone"), timezone)...)
}

func (r *automationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil || r.client != nil {
		return
	}
	client, err := withClientForResource(req, resp)
	if err != nil {
		return
	}
	r.client = client
}

func (r *automationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan automationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	contextItems, diags := rawJSONFromString(ctx, plan.ContextItems)
	resp.Diagnostics.Append(diags...)
	notifications, notifDiags := automationNotificationsFromModel(ctx, plan.Notifications)
	resp.Diagnostics.Append(notifDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	labels := map[string]string{}
	resp.Diagnostics.Append(plan.Labels.ElementsAs(ctx, &labels, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	triggers, triggerDiags := automationTriggersFromModel(ctx, plan.Triggers)
	resp.Diagnostics.Append(triggerDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateAutomation(ctx, assistantapi.AutomationCreate{
		Name:             plan.Name.ValueString(),
		Description:      plan.Description.ValueString(),
		Prompt:           plan.Prompt.ValueString(),
		Enabled:          plan.Enabled.ValueBool(),
		Scope:            plan.Scope.ValueString(),
		ExecutionMode:    plan.ExecutionMode.ValueString(),
		Labels:           labels,
		Triggers:         triggers,
		ScheduleEndsAt:   plan.ScheduleEndsAt.ValueString(),
		ScheduleCron:     plan.ScheduleCron.ValueString(),
		ScheduleTimezone: plan.ScheduleTimezone.ValueString(),
		ContextItems:     contextItems,
		Notifications:    notifications,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create assistant automation", err.Error())
		return
	}

	state, stateDiags := automationToModel(ctx, created, plan)
	resp.Diagnostics.Append(stateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *automationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state automationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	automation, err := r.client.GetAutomation(ctx, state.ID.ValueString())
	if err != nil {
		if errors.Is(err, assistantapi.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read assistant automation", err.Error())
		return
	}

	model, diags := automationToModel(ctx, automation, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *automationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state automationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	update, diags := automationUpdateFromModels(ctx, plan, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	updated, err := r.client.UpdateAutomation(ctx, state.ID.ValueString(), state.Scope.ValueString(), update)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update assistant automation", err.Error())
		return
	}

	model, stateDiags := automationToModel(ctx, updated, plan)
	resp.Diagnostics.Append(stateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *automationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state automationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteAutomation(ctx, state.ID.ValueString(), state.Scope.ValueString()); err != nil {
		if errors.Is(err, assistantapi.ErrNotFound) {
			return
		}
		resp.Diagnostics.AddError("Failed to delete assistant automation", err.Error())
	}
}

func (r *automationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	automation, err := r.client.GetAutomation(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import assistant automation", err.Error())
		return
	}
	model, diags := automationToModel(ctx, automation, automationModel{})
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

// automationNotificationsFromModel always returns a non-nil provider set. The
// API merges notification patches per provider — a null value deletes, an
// absent key is left alone — so every provider Terraform does not configure has
// to be sent as an explicit null for a removed block to actually clear.
func automationNotificationsFromModel(ctx context.Context, notifications *automationNotificationsModel) (*assistantapi.AutomationNotifications, diag.Diagnostics) {
	var diags diag.Diagnostics
	if notifications == nil {
		return &assistantapi.AutomationNotifications{}, diags
	}

	result := &assistantapi.AutomationNotifications{}
	if notifications.Slack != nil {
		notifyOn, notifyDiags := listValueToStrings(ctx, notifications.Slack.NotifyOn)
		diags.Append(notifyDiags...)
		slack := &assistantapi.AutomationSlackNotification{
			Enabled:  notifications.Slack.Enabled.ValueBool(),
			NotifyOn: notifyOn,
		}
		if t := notifications.Slack.Target; t != nil {
			slack.Target = &assistantapi.AutomationSlackTarget{
				Type:      t.Type.ValueString(),
				ChannelID: t.ChannelID.ValueString(),
				UserID:    t.UserID.ValueString(),
				TeamID:    t.TeamID.ValueString(),
			}
		}
		result.Slack = slack
	}
	if notifications.Email != nil {
		notifyOn, notifyDiags := listValueToStrings(ctx, notifications.Email.NotifyOn)
		diags.Append(notifyDiags...)
		addresses, addrDiags := listValueToStrings(ctx, notifications.Email.Addresses)
		diags.Append(addrDiags...)
		result.Email = &assistantapi.AutomationEmailNotification{
			Enabled:   notifications.Email.Enabled.ValueBool(),
			NotifyOn:  notifyOn,
			Addresses: addresses,
		}
	}
	return result, diags
}

// automationToModel maps an API automation onto Terraform state. `cfg` carries
// the plan or prior state so notification blocks the user never configured stay
// absent rather than surfacing as empty blocks on every plan.
func automationToModel(ctx context.Context, automation assistantapi.Automation, cfg automationModel) (automationModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	cron := types.StringNull()
	timezone := types.StringNull()
	nextRunAt := types.StringNull()
	deadline := types.StringNull()
	if automation.Schedule != nil {
		cron = stringValueOrNull(automation.Schedule.Cron)
		if !cfg.ScheduleCron.IsNull() && !cfg.ScheduleCron.IsUnknown() && canonicalAutomationCron(cfg.ScheduleCron.ValueString()) == canonicalAutomationCron(automation.Schedule.Cron) {
			cron = cfg.ScheduleCron
		}
		timezone = types.StringValue(automation.Schedule.Timezone)
		if automation.Schedule.Timezone == "" {
			timezone = types.StringValue("UTC")
		}
		deadline = automationDeadlineToModel(automation.Schedule.EndsAt, cfg.ScheduleEndsAt)
		nextRunAt = stringValueOrNull(automation.Schedule.NextRunAt)
	}

	notifications, notifDiags := automationNotificationsToModel(ctx, automation.Notifications, cfg.Notifications)
	diags.Append(notifDiags...)

	labels, labelDiags := types.MapValueFrom(ctx, types.StringType, automation.Labels)
	diags.Append(labelDiags...)
	if len(automation.Labels) == 0 {
		labels = types.MapNull(types.StringType)
		if !cfg.Labels.IsNull() && !cfg.Labels.IsUnknown() && len(cfg.Labels.Elements()) == 0 {
			labels = cfg.Labels
		}
	}
	mode := automation.ExecutionMode
	if mode == "" {
		mode = "user"
	}

	triggers, triggerDiags := automationTriggersToModel(ctx, automation.Triggers, cfg.Triggers)
	diags.Append(triggerDiags...)

	return automationModel{
		ID:               types.StringValue(automation.ID),
		Name:             types.StringValue(automation.Name),
		Description:      stringValueOrNull(automation.Description),
		Prompt:           types.StringValue(automation.Prompt),
		Enabled:          types.BoolValue(automation.Enabled),
		Scope:            types.StringValue(automation.Scope),
		ExecutionMode:    types.StringValue(mode),
		Labels:           labels,
		Triggers:         triggers,
		ScheduleEndsAt:   deadline,
		ScheduleCron:     cron,
		ScheduleTimezone: timezone,
		NextRunAt:        nextRunAt,
		ContextItems:     automationContextToModel(automation.ContextItems, cfg.ContextItems),
		Notifications:    notifications,
	}, diags
}

func automationNotificationsToModel(ctx context.Context, notifications *assistantapi.AutomationNotifications, cfg *automationNotificationsModel) (*automationNotificationsModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	if notifications == nil {
		return nil, diags
	}
	// The API echoes a provider set on every response, including disabled
	// defaults nobody configured. Reflecting that back would give state a
	// notifications block that a config without one can never match.
	if cfg == nil && !automationNotificationsConfigured(notifications) {
		return nil, diags
	}

	result := &automationNotificationsModel{}
	if n := notifications.Slack; n != nil && (cfg == nil || cfg.Slack != nil || n.Enabled) {
		var cfgNotifyOn types.List
		if cfg != nil && cfg.Slack != nil {
			cfgNotifyOn = cfg.Slack.NotifyOn
		}
		notifyOn, notifyDiags := automationNotificationListToModel(ctx, n.NotifyOn, cfgNotifyOn, false)
		diags.Append(notifyDiags...)
		slack := &automationSlackModel{
			Enabled:  types.BoolValue(n.Enabled),
			NotifyOn: notifyOn,
		}
		if t := n.Target; t != nil {
			var configuredUserID, configuredTeamID *types.String
			if cfg != nil && cfg.Slack != nil && cfg.Slack.Target != nil {
				configuredUserID = &cfg.Slack.Target.UserID
				configuredTeamID = &cfg.Slack.Target.TeamID
			}
			slack.Target = &automationSlackTargetModel{
				Type:      stringValueOrNull(t.Type),
				ChannelID: stringValueOrNull(t.ChannelID),
				UserID:    automationResolvedSlackID(t.UserID, configuredUserID),
				TeamID:    automationResolvedSlackID(t.TeamID, configuredTeamID),
			}
		}
		result.Slack = slack
	}
	if n := notifications.Email; n != nil && (cfg == nil || cfg.Email != nil || n.Enabled) {
		var cfgNotifyOn, cfgAddresses types.List
		if cfg != nil && cfg.Email != nil {
			cfgNotifyOn = cfg.Email.NotifyOn
			cfgAddresses = cfg.Email.Addresses
		}
		notifyOn, notifyDiags := automationNotificationListToModel(ctx, n.NotifyOn, cfgNotifyOn, false)
		diags.Append(notifyDiags...)
		addresses, addrDiags := automationNotificationListToModel(ctx, n.Addresses, cfgAddresses, true)
		diags.Append(addrDiags...)
		result.Email = &automationEmailModel{
			Enabled:   types.BoolValue(n.Enabled),
			NotifyOn:  notifyOn,
			Addresses: addresses,
		}
	}
	if result.Slack == nil && result.Email == nil {
		return nil, diags
	}
	return result, diags
}

// automationNotificationsConfigured reports whether a provider set carries
// anything a user actually asked for, as opposed to the disabled defaults the
// API returns for every automation.
func automationNotificationsConfigured(notifications *assistantapi.AutomationNotifications) bool {
	if notifications == nil {
		return false
	}
	if n := notifications.Slack; n != nil && (n.Enabled || n.Target != nil || len(n.NotifyOn) > 0) {
		return true
	}
	if n := notifications.Email; n != nil && (n.Enabled || len(n.Addresses) > 0 || len(n.NotifyOn) > 0) {
		return true
	}
	return false
}
