package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common/assistantapi"
	"github.com/grafana/terraform-provider-grafana/v4/internal/util"
)

const resourceWatcherName = "grafana_assistant_watcher"

var resourceWatcherID = common.NewResourceID(common.StringIDField("id"))

type watcherResource struct {
	client *assistantapi.Client
}

type watcherSlackTargetModel struct {
	Type      types.String `tfsdk:"type"`
	ChannelID types.String `tfsdk:"channel_id"`
	UserID    types.String `tfsdk:"user_id"`
	TeamID    types.String `tfsdk:"team_id"`
}

type watcherSlackModel struct {
	Enabled types.Bool               `tfsdk:"enabled"`
	Target  *watcherSlackTargetModel `tfsdk:"target"`
}

type watcherInvestigationModel struct {
	Enabled   types.Bool `tfsdk:"enabled"`
	TeamNames types.List `tfsdk:"team_names"`
}

type watcherActionsModel struct {
	Slack         *watcherSlackModel         `tfsdk:"slack"`
	Investigation *watcherInvestigationModel `tfsdk:"investigation"`
}

type watcherModel struct {
	ReviewURL              types.String         `tfsdk:"review_url"`
	RequestID              types.String         `tfsdk:"request_id"`
	Labels                 types.Map            `tfsdk:"labels"`
	TeamUID                types.String         `tfsdk:"team_uid"`
	PauseAt                types.String         `tfsdk:"pause_at"`
	ArchiveOnStop          types.Bool           `tfsdk:"archive_on_stop"`
	DefinitionVersion      types.Int64          `tfsdk:"definition_version"`
	ID                     types.String         `tfsdk:"id"`
	Name                   types.String         `tfsdk:"name"`
	Description            types.String         `tfsdk:"description"`
	Prompt                 types.String         `tfsdk:"prompt"`
	DatasourceUIDs         types.List           `tfsdk:"datasource_uids"`
	Sensitivity            types.String         `tfsdk:"sensitivity"`
	TriggerIntervalSeconds types.Int64          `tfsdk:"trigger_interval_seconds"`
	DisableDecisionSkip    types.Bool           `tfsdk:"disable_decision_skip"`
	CalibrationContext     types.String         `tfsdk:"calibration_context"`
	Status                 types.String         `tfsdk:"status"`
	CalibratedAt           types.String         `tfsdk:"calibrated_at"`
	Actions                *watcherActionsModel `tfsdk:"actions"`
}

func makeResourceWatcher() *common.Resource {
	return common.NewResource(
		common.CategoryGrafanaAssistant,
		resourceWatcherName,
		resourceWatcherID,
		&watcherResource{},
	)
}

func (r *watcherResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceWatcherName
}

func (r *watcherResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Manages operator configuration for a Grafana Assistant Watcher. Assistant owns generated checks, calibration guidance and learned state. Use grafana_assistant_watcher_calibration to perform unattended setup and explicitly opt into activation. Changes to the monitoring goal or data-source scope replace the Watcher; metadata and delivery updates preserve it. Watchers require the Grafana Assistant app and credentials with access to the monitored data sources.`,
		Attributes: map[string]schema.Attribute{
			"id":         idAttribute(),
			"review_url": schema.StringAttribute{Description: "Assistant URL for reviewing generated checks and calibration.", Computed: true},
			"request_id": schema.StringAttribute{Description: "UUID identifying this creation request. Generated when omitted. Supply a stable UUID to recover creation across process/state loss on Assistant versions supporting requestId. Use a new UUID when replacing the Watcher; reusing a key with different inputs is rejected. This is separate from the server-assigned Watcher ID.", Optional: true, Computed: true,
				Validators:    []validator.String{stringvalidator.RegexMatches(regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`), "must be a canonical lowercase UUID")},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured()}},
			"name": schema.StringAttribute{
				Description: "Human-friendly watcher name.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Optional description.",
				Optional:    true,
			},
			"prompt": schema.StringAttribute{
				Description:   "Monitoring goal. Changing the goal replaces the Watcher so stale checks cannot silently monitor different intent.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Required:      true,
			},
			"datasource_uids": schema.ListAttribute{
				Description:   "Allowed data-source UIDs. Changing scope replaces the Watcher and requires new calibration.",
				PlanModifiers: []planmodifier.List{listplanmodifier.RequiresReplace()},
				Optional:      true,
				ElementType:   types.StringType,
			},
			"sensitivity": schema.StringAttribute{
				Description: "How readily the watcher raises concern. One of `sensitive`, `balanced`, or `relaxed`.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("balanced"),
				Validators: []validator.String{
					stringvalidator.OneOf("sensitive", "balanced", "relaxed"),
				},
			},
			"trigger_interval_seconds": schema.Int64Attribute{
				Description: "Evaluation interval in seconds: 300, 360, 600, 720, 900, 1200, 1800, 3600, 7200 or 10800. Assistant adjusts generated range selectors when this changes; interval changes can reset affected learning.",
				Optional:    true, Computed: true, Default: int64default.StaticInt64(600),
				Validators: []validator.Int64{int64validator.OneOf(300, 360, 600, 720, 900, 1200, 1800, 3600, 7200, 10800)},
			},
			"disable_decision_skip": schema.BoolAttribute{
				Description: "Send every run to the decision model, even when its telemetry is deterministically clean. Defaults to letting provably uneventful runs skip the model review.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"calibration_context": schema.StringAttribute{Description: "Read-only guidance produced by Assistant calibration.", Computed: true},
			"definition_version":  schema.Int64Attribute{Description: "Current server definition revision, used to reject concurrent updates.", Computed: true},
			"labels":              schema.MapAttribute{Description: "Operator labels propagated to notifications.", Optional: true, ElementType: types.StringType},
			"team_uid":            schema.StringAttribute{Description: "Grafana team UID. The caller must belong to the team. Removing the attribute clears team assignment.", Optional: true},
			"pause_at":            schema.StringAttribute{Description: "RFC3339 deadline for automatic pause. Removing it cancels the deadline.", Optional: true},
			"archive_on_stop":     schema.BoolAttribute{Description: "Archive at pause_at. Requires a deadline. Defaults to false.", Optional: true, Computed: true, Default: booldefault.StaticBool(false)},
			"status": schema.StringAttribute{
				Description: "Lifecycle state reported by the API: `draft`, `calibrating`, `ready`, `running`, `paused`, or `error`.",
				Computed:    true,
			},
			"calibrated_at": schema.StringAttribute{
				Description: "When the watcher was last calibrated.",
				Computed:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"actions": schema.SingleNestedBlock{
				Description: "What the watcher does after a run that raises concern.",
				Blocks: map[string]schema.Block{
					"slack": schema.SingleNestedBlock{
						Description: "Slack delivery. Requires the Slack workspace to already be connected to the stack, which is a one-time manual step in Assistant settings.",
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								Description: "Whether the watcher may post to Slack.",
								Optional:    true,
								Computed:    true,
								Default:     booldefault.StaticBool(false),
							},
						},
						Blocks: map[string]schema.Block{
							"target": schema.SingleNestedBlock{
								Description: "Where the watcher posts.",
								Attributes: map[string]schema.Attribute{
									// Optional for the same reason as thresholds.comparator:
									// a Required attribute here is enforced even when
									// the enclosing block is omitted.
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
										Description: "Slack user ID. Required for a `dm` target.",
										Optional:    true,
									},
									"team_id": schema.StringAttribute{
										Description: "Slack workspace (team) ID.",
										Optional:    true,
									},
								},
							},
						},
					},
					"investigation": schema.SingleNestedBlock{
						Description: "Assistant Investigation launched when the watcher escalates a critical finding.",
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								Description: "Whether the watcher may launch an investigation.",
								Optional:    true,
								Computed:    true,
								Default:     booldefault.StaticBool(false),
							},
							"team_names": schema.ListAttribute{
								Description: "Grafana team names granted access to launched investigations. Empty keeps them private to the creator.",
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

func (r *watcherResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil || r.client != nil {
		return
	}
	client, err := withClientForResource(req, resp)
	if err != nil {
		return
	}
	r.client = client
}

func (r *watcherResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan watcherModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	ds, dsDiags := listValueToStrings(ctx, plan.DatasourceUIDs)
	resp.Diagnostics.Append(dsDiags...)
	actions, actionDiags := watcherActionsFromModel(ctx, plan.Actions)
	resp.Diagnostics.Append(actionDiags...)
	labels := map[string]string{}
	if !plan.Labels.IsNull() {
		resp.Diagnostics.Append(plan.Labels.ElementsAs(ctx, &labels, false)...)
	}
	if resp.Diagnostics.HasError() {
		return
	}
	if plan.RequestID.IsNull() || plan.RequestID.IsUnknown() {
		plan.RequestID = types.StringValue(uuid.NewString())
	}
	created, err := r.client.CreateWatcher(ctx, assistantapi.WatcherCreate{
		RequestID: plan.RequestID.ValueString(),
		Name:      plan.Name.ValueString(), Description: plan.Description.ValueString(), Prompt: plan.Prompt.ValueString(),
		DatasourceUIDs: ds, Sensitivity: plan.Sensitivity.ValueString(), TriggerIntervalSeconds: plan.TriggerIntervalSeconds.ValueInt64(),
		DisableDecisionSkip: boolPtrOrNil(plan.DisableDecisionSkip), Labels: labels, TeamUID: plan.TeamUID.ValueString(),
		AutoStop: &assistantapi.WatcherAutoStop{PauseAt: plan.PauseAt.ValueString(), Archive: plan.ArchiveOnStop.ValueBool()}, Actions: actions,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create assistant watcher", err.Error()+". Creation is not automatically retried. If the server accepted the request but its response was lost, locate the Watcher in Assistant and import its ID before applying again.")
		return
	}
	model, diags := r.watcherToModel(ctx, created, plan)
	resp.Diagnostics.Append(diags...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	}
}

func (r *watcherResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state watcherModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	watcher, err := r.client.GetWatcher(ctx, state.ID.ValueString())
	if errors.Is(err, assistantapi.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Failed to read assistant watcher", err.Error())
		return
	}
	model, diags := r.watcherToModel(ctx, watcher, state)
	resp.Diagnostics.Append(diags...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	}
}

func (r *watcherResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state watcherModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if plan.RequestID.IsUnknown() {
		plan.RequestID = state.RequestID
	}
	live, err := r.client.GetWatcher(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read assistant watcher before update", err.Error())
		return
	}
	// Use the revision observed by Terraform's refresh, not the just-fetched
	// revision: otherwise a concurrent definition change would be silently adopted.
	patch, diags := watcherPatch(ctx, plan, state, live)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	updated, err := r.client.UpdateWatcher(ctx, state.ID.ValueString(), patch)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update assistant watcher", err.Error())
		return
	}
	model, diags := r.watcherToModel(ctx, updated, plan)
	resp.Diagnostics.Append(diags...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	}
}

func watcherPatch(ctx context.Context, plan, state watcherModel, live assistantapi.Watcher) (assistantapi.WatcherUpdate, diag.Diagnostics) {
	var diags diag.Diagnostics
	patch := assistantapi.WatcherUpdate{ExpectedDefinitionVersion: util.Ptr(state.DefinitionVersion.ValueInt64())}
	if !plan.Name.Equal(state.Name) {
		patch.Name = util.Ptr(plan.Name.ValueString())
	}
	if !plan.Description.Equal(state.Description) {
		patch.Description = util.Ptr(plan.Description.ValueString())
	}
	if !plan.Sensitivity.Equal(state.Sensitivity) {
		patch.Sensitivity = util.Ptr(plan.Sensitivity.ValueString())
	}
	if !plan.TriggerIntervalSeconds.Equal(state.TriggerIntervalSeconds) {
		patch.TriggerIntervalSeconds = util.Ptr(plan.TriggerIntervalSeconds.ValueInt64())
	}
	if !plan.DisableDecisionSkip.Equal(state.DisableDecisionSkip) {
		patch.DisableDecisionSkip = util.Ptr(plan.DisableDecisionSkip.ValueBool())
	}
	if !plan.TeamUID.Equal(state.TeamUID) {
		patch.TeamUID = util.Ptr(plan.TeamUID.ValueString())
	}
	if !plan.PauseAt.Equal(state.PauseAt) || !plan.ArchiveOnStop.Equal(state.ArchiveOnStop) {
		patch.AutoStop = &assistantapi.WatcherAutoStop{PauseAt: plan.PauseAt.ValueString(), Archive: plan.ArchiveOnStop.ValueBool()}
	}
	if !plan.Labels.Equal(state.Labels) {
		labels := map[string]string{}
		if !plan.Labels.IsNull() {
			diags.Append(plan.Labels.ElementsAs(ctx, &labels, false)...)
		}
		patch.Labels = &labels
	}
	if !reflect.DeepEqual(plan.Actions, state.Actions) {
		wanted, actionDiags := watcherActionsFromModel(ctx, plan.Actions)
		diags.Append(actionDiags...)
		patch.Actions = mergeWatcherActions(wanted, live.Actions)
	}
	return patch, diags
}

// The backend replaces Slack/Teams/investigation but preserves omitted webhook,
// Alerting and IRM. Retain Teams and server-owned Slack fields when managing
// Slack/investigation; never echo write-only secret markers into requests.
func mergeWatcherActions(wanted, live *assistantapi.WatcherActions) *assistantapi.WatcherActions {
	if wanted == nil {
		wanted = &assistantapi.WatcherActions{}
	}
	if live == nil {
		return wanted
	}
	wanted.MSTeams = append(json.RawMessage(nil), live.MSTeams...)
	if wanted.Slack != nil && live.Slack != nil {
		wanted.Slack.MinSeverity = live.Slack.MinSeverity
	}
	if wanted.Slack != nil && wanted.Slack.Target != nil && live.Slack != nil && live.Slack.Target != nil &&
		wanted.Slack.Target.ChannelID == live.Slack.Target.ChannelID {
		wanted.Slack.Target.ThreadTS = live.Slack.Target.ThreadTS
	}
	return wanted
}

func (r *watcherResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state watcherModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.client.DeleteWatcher(ctx, state.ID.ValueString())
	if err != nil && !errors.Is(err, assistantapi.ErrNotFound) {
		resp.Diagnostics.AddError("Failed to delete assistant watcher", err.Error())
	}
}

func (r *watcherResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	watcher, err := r.client.GetWatcher(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import assistant watcher", err.Error())
		return
	}
	model, diags := r.watcherToModel(ctx, watcher, watcherModel{})
	resp.Diagnostics.Append(diags...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	}
}

func watcherActionsFromModel(ctx context.Context, actions *watcherActionsModel) (*assistantapi.WatcherActions, diag.Diagnostics) {
	var diags diag.Diagnostics
	if actions == nil {
		return nil, diags
	}

	result := &assistantapi.WatcherActions{}
	if actions.Slack != nil {
		slack := &assistantapi.WatcherSlackAction{Enabled: actions.Slack.Enabled.ValueBool()}
		if t := actions.Slack.Target; t != nil {
			slack.Target = &assistantapi.WatcherSlackTarget{
				Type:      t.Type.ValueString(),
				ChannelID: t.ChannelID.ValueString(),
				UserID:    t.UserID.ValueString(),
				TeamID:    t.TeamID.ValueString(),
			}
		}
		result.Slack = slack
	}
	if actions.Investigation != nil {
		teamNames, teamDiags := listValueToStrings(ctx, actions.Investigation.TeamNames)
		diags.Append(teamDiags...)
		result.Investigation = &assistantapi.WatcherInvestigationAction{
			Enabled:   actions.Investigation.Enabled.ValueBool(),
			TeamNames: teamNames,
		}
	}
	return result, diags
}

// watcherToModel maps an API watcher onto Terraform state. `cfg` carries the
// plan or prior state so that blocks the user never configured stay absent
// instead of materializing as empty blocks on every plan.
func (r *watcherResource) watcherToModel(ctx context.Context, watcher assistantapi.Watcher, cfg watcherModel) (watcherModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	datasourceUIDs, dsDiags := stringsToListValuePreserving(ctx, watcher.DatasourceUIDs, cfg.DatasourceUIDs)
	diags.Append(dsDiags...)

	actions, actionDiags := watcherActionsToModel(ctx, watcher.Actions, cfg.Actions)
	diags.Append(actionDiags...)

	calibratedAt := types.StringNull()
	if watcher.CalibratedAt != nil {
		calibratedAt = types.StringValue(*watcher.CalibratedAt)
	}

	labels := types.MapNull(types.StringType)
	if len(watcher.Labels) > 0 || (!cfg.Labels.IsNull() && !cfg.Labels.IsUnknown()) {
		var labelDiags diag.Diagnostics
		values := watcher.Labels
		if values == nil {
			values = map[string]string{}
		}
		labels, labelDiags = types.MapValueFrom(ctx, types.StringType, values)
		diags.Append(labelDiags...)
	}
	team := watcherStringValue("", cfg.TeamUID)
	if watcher.Team != nil {
		team = watcherStringValue(watcher.Team.UID, cfg.TeamUID)
	}
	pauseAt := watcherStringValue("", cfg.PauseAt)
	archive := false
	if watcher.AutoStop != nil {
		pauseAt = watcherTimestampValue(watcher.AutoStop.PauseAt, cfg.PauseAt)
		archive = watcher.AutoStop.Archive
	}

	requestID := cfg.RequestID
	if watcher.RequestID != "" {
		requestID = types.StringValue(watcher.RequestID)
	}
	return watcherModel{
		ReviewURL:              types.StringValue(r.client.WatcherURL(watcher.ID)),
		RequestID:              requestID,
		ID:                     types.StringValue(watcher.ID),
		Name:                   types.StringValue(watcher.Name),
		Description:            watcherStringValue(watcher.Description, cfg.Description),
		Prompt:                 types.StringValue(watcher.Prompt),
		DatasourceUIDs:         datasourceUIDs,
		Sensitivity:            types.StringValue(watcher.Sensitivity),
		TriggerIntervalSeconds: types.Int64Value(watcher.TriggerIntervalSeconds),
		DisableDecisionSkip:    boolValueOrNull(watcher.DisableDecisionSkip),
		CalibrationContext:     stringValueOrNull(watcher.CalibrationContext),
		DefinitionVersion:      types.Int64Value(watcher.DefinitionVersion),
		Labels:                 labels, TeamUID: team, PauseAt: pauseAt, ArchiveOnStop: types.BoolValue(archive),
		Status:       types.StringValue(watcher.Status),
		CalibratedAt: calibratedAt,
		Actions:      actions,
	}, diags
}

func watcherActionsToModel(ctx context.Context, actions *assistantapi.WatcherActions, cfg *watcherActionsModel) (*watcherActionsModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	if actions == nil {
		return nil, diags
	}
	// The API always echoes an action policy, including disabled defaults for
	// channels the user never configured. Mirroring that back into state would
	// show a permanent diff against a config with no `actions` block at all.
	if cfg == nil && !watcherActionsConfigured(actions) {
		return nil, diags
	}

	result := &watcherActionsModel{}
	if actions.Slack != nil && (cfg == nil || cfg.Slack != nil || actions.Slack.Enabled) {
		slack := &watcherSlackModel{Enabled: types.BoolValue(actions.Slack.Enabled)}
		if t := actions.Slack.Target; t != nil {
			slack.Target = &watcherSlackTargetModel{
				Type:      stringValueOrNull(t.Type),
				ChannelID: stringValueOrNull(t.ChannelID),
				UserID:    stringValueOrNull(t.UserID),
				TeamID:    stringValueOrNull(t.TeamID),
			}
		}
		result.Slack = slack
	}
	if actions.Investigation != nil && (cfg == nil || cfg.Investigation != nil || actions.Investigation.Enabled) {
		var cfgTeamNames types.List
		if cfg != nil && cfg.Investigation != nil {
			cfgTeamNames = cfg.Investigation.TeamNames
		}
		teamNames, teamDiags := stringsToListValuePreserving(ctx, actions.Investigation.TeamNames, cfgTeamNames)
		diags.Append(teamDiags...)
		result.Investigation = &watcherInvestigationModel{
			Enabled:   types.BoolValue(actions.Investigation.Enabled),
			TeamNames: teamNames,
		}
	}
	if result.Slack == nil && result.Investigation == nil {
		if cfg != nil {
			return result, diags
		}
		return nil, diags
	}
	return result, diags
}

func watcherActionsConfigured(actions *assistantapi.WatcherActions) bool {
	if actions == nil {
		return false
	}
	if actions.Slack != nil && (actions.Slack.Enabled || actions.Slack.Target != nil) {
		return true
	}
	if actions.Investigation != nil && (actions.Investigation.Enabled || len(actions.Investigation.TeamNames) > 0) {
		return true
	}
	return false
}

func (r *watcherResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var pause types.String
	var archive types.Bool
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("pause_at"), &pause)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("archive_on_stop"), &archive)...)
	if resp.Diagnostics.HasError() || pause.IsUnknown() {
		return
	}
	if archive.ValueBool() && pause.IsNull() {
		resp.Diagnostics.AddAttributeError(path.Root("archive_on_stop"), "Missing stop deadline", "archive_on_stop requires pause_at.")
	}
	if !pause.IsNull() {
		if _, err := time.Parse(time.RFC3339, pause.ValueString()); err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("pause_at"), "Invalid stop deadline", "pause_at must be an RFC3339 timestamp.")
		}
	}
}

func watcherStringValue(value string, configured types.String) types.String {
	if value == "" && !configured.IsNull() && !configured.IsUnknown() && configured.ValueString() == "" {
		return configured
	}
	return stringValueOrNull(value)
}
func watcherTimestampValue(value string, configured types.String) types.String {
	if !configured.IsNull() && !configured.IsUnknown() {
		a, ae := time.Parse(time.RFC3339, value)
		b, be := time.Parse(time.RFC3339, configured.ValueString())
		if ae == nil && be == nil && a.Equal(b) {
			return configured
		}
	}
	return watcherStringValue(value, configured)
}
