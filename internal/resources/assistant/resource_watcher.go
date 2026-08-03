package assistant

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
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

type watcherThresholdsModel struct {
	Comparator types.String  `tfsdk:"comparator"`
	Warning    types.Float64 `tfsdk:"warning"`
	Critical   types.Float64 `tfsdk:"critical"`
	Source     types.String  `tfsdk:"source"`
}

type watcherQueryModel struct {
	Type          types.String            `tfsdk:"type"`
	Expr          types.String            `tfsdk:"expr"`
	DatasourceUID types.String            `tfsdk:"datasource_uid"`
	Comment       types.String            `tfsdk:"comment"`
	Enabled       types.Bool              `tfsdk:"enabled"`
	Role          types.String            `tfsdk:"role"`
	GoodWhen      types.String            `tfsdk:"good_when"`
	Thresholds    *watcherThresholdsModel `tfsdk:"thresholds"`
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
	ID                     types.String         `tfsdk:"id"`
	Name                   types.String         `tfsdk:"name"`
	Description            types.String         `tfsdk:"description"`
	Prompt                 types.String         `tfsdk:"prompt"`
	DatasourceUIDs         types.List           `tfsdk:"datasource_uids"`
	Sensitivity            types.String         `tfsdk:"sensitivity"`
	TriggerIntervalSeconds types.Int64          `tfsdk:"trigger_interval_seconds"`
	DisableDecisionSkip    types.Bool           `tfsdk:"disable_decision_skip"`
	CalibrationContext     types.String         `tfsdk:"calibration_context"`
	Started                types.Bool           `tfsdk:"started"`
	Status                 types.String         `tfsdk:"status"`
	CalibratedAt           types.String         `tfsdk:"calibrated_at"`
	Queries                []watcherQueryModel  `tfsdk:"query"`
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
		Description: `Manages a Grafana Assistant Watcher: an always-on agent that evaluates a
calibrated set of checks against your telemetry on a schedule and raises concern when a run
finds something notable.

Watchers are a Grafana Cloud-only public preview feature and require the Grafana Assistant app.

Terraform supplies the calibrated checks directly instead of running Assistant's interactive
calibration. Calibrate a watcher once in the UI, then copy the resulting queries and
` + "`calibration_context`" + ` into this resource to manage it as code.

The watcher runs with the identity that created it, so the credentials Terraform uses must have
access to every data source the watcher queries.`,
		Attributes: map[string]schema.Attribute{
			"id": idAttribute(),
			"name": schema.StringAttribute{
				Description: "Human-friendly watcher name.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Optional description.",
				Optional:    true,
			},
			"prompt": schema.StringAttribute{
				Description: "What the watcher should monitor. Include service names, expected rollout behavior, known noise, and acceptable tolerances.",
				Required:    true,
			},
			"datasource_uids": schema.ListAttribute{
				Description: "UIDs of the Prometheus and Loki data sources the watcher may query.",
				Optional:    true,
				ElementType: types.StringType,
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
				Description: "How often the watcher evaluates its checks, in seconds. Between 900 (15 minutes) and 10800 (3 hours). " +
					"Update the range selectors in `query.expr` in the same change, so consecutive runs tile the timeline exactly once: " +
					"when the interval changes, the API rewrites every range selector that matched the previous interval " +
					"(`long_window_budget` checks are left alone). Because `expr` is a required, non-computed attribute, an interval " +
					"change that leaves the old selectors in place makes the API return expressions that differ from the plan, and " +
					"Terraform fails the apply. Moving from 900 to 1800 means rewriting `[15m]` to `[30m]` at the same time.",
				Optional: true,
				Computed: true,
				Default:  int64default.StaticInt64(900),
				Validators: []validator.Int64{
					int64validator.Between(900, 10800),
				},
			},
			"disable_decision_skip": schema.BoolAttribute{
				Description: "Send every run to the decision model, even when its telemetry is deterministically clean. Defaults to letting provably uneventful runs skip the model review.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"calibration_context": schema.StringAttribute{
				Description: "Decision-making guidance the watcher applies on every run: the calibration baseline, normal ranges, known benign patterns, and acknowledged conditions.",
				Optional:    true,
			},
			"started": schema.BoolAttribute{
				Description: "Whether scheduled runs are active. Set to `false` to keep the watcher calibrated but paused.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
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
			"query": schema.ListNestedBlock{
				Description: "A calibrated check the watcher evaluates on every run. At least one enabled query is required before the watcher can start.",
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Description: "Check language. One of `promql`, `logql`, or `alerts`.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("promql", "logql", "alerts"),
							},
						},
						"expr": schema.StringAttribute{
							Description: "PromQL or LogQL expression, or Alertmanager label matchers for an `alerts` check.",
							Required:    true,
						},
						"datasource_uid": schema.StringAttribute{
							Description: "Target data source UID. Omitted for `alerts` checks, which read Grafana-managed Alertmanager.",
							Optional:    true,
						},
						"comment": schema.StringAttribute{
							Description: "Human-readable explanation of what this check covers.",
							Optional:    true,
						},
						"enabled": schema.BoolAttribute{
							Description: "Whether this check runs.",
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(true),
						},
						"role": schema.StringAttribute{
							Description: "Role this evidence plays in the incident lifecycle. One of `fast_incident`, `current_health`, `long_window_budget`, or `context`.",
							Optional:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("fast_incident", "current_health", "long_window_budget", "context"),
							},
						},
						"good_when": schema.StringAttribute{
							Description: "Direction that represents improvement. One of `low`, `high`, or `absent`.",
							Optional:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("low", "high", "absent"),
							},
						},
					},
					Blocks: map[string]schema.Block{
						"thresholds": schema.SingleNestedBlock{
							Description: "Authoritative numeric warning and critical boundaries for this check. Omit for `alerts` checks, where a firing alert is itself the signal.",
							Attributes: map[string]schema.Attribute{
								// Mandatory whenever the block is present, but declared
								// Optional: Terraform enforces Required attributes
								// inside a SingleNestedBlock even when the block is
								// omitted, which would break every check that has no
								// thresholds at all. The block is sent as configured
								// and the API rejects a missing comparator.
								"comparator": schema.StringAttribute{
									Description: "Comparison used to enter the warning or critical state. One of `gt`, `gte`, `lt`, or `lte`. Required when a `thresholds` block is present.",
									Optional:    true,
									Validators: []validator.String{
										stringvalidator.OneOf("gt", "gte", "lt", "lte"),
									},
								},
								"warning": schema.Float64Attribute{
									Description: "Warning boundary.",
									Optional:    true,
								},
								"critical": schema.Float64Attribute{
									Description: "Critical boundary.",
									Optional:    true,
								},
								"source": schema.StringAttribute{
									Description: "Where the boundary came from: an alert rule name, an SLO UUID, or `derived`.",
									Optional:    true,
								},
							},
						},
					},
				},
			},
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

	datasourceUIDs, diags := listValueToStrings(ctx, plan.DatasourceUIDs)
	resp.Diagnostics.Append(diags...)
	actions, actionDiags := watcherActionsFromModel(ctx, plan.Actions)
	resp.Diagnostics.Append(actionDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateWatcher(ctx, assistantapi.WatcherCreate{
		Name:                   plan.Name.ValueString(),
		Description:            plan.Description.ValueString(),
		Prompt:                 plan.Prompt.ValueString(),
		Sensitivity:            plan.Sensitivity.ValueString(),
		TriggerIntervalSeconds: plan.TriggerIntervalSeconds.ValueInt64(),
		DisableDecisionSkip:    boolPtrOrNil(plan.DisableDecisionSkip),
		DatasourceUIDs:         datasourceUIDs,
		Actions:                actions,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create assistant watcher", err.Error())
		return
	}

	// A newly created watcher is a draft. Supplying the calibrated checks with
	// finalizeCalibration marks it calibrated and moves it to ready, which is
	// what makes a declaratively-managed watcher startable without an
	// interactive calibration session.
	watcher, err := r.client.AddWatcherQueries(ctx, created.ID, assistantapi.WatcherAddQueries{
		Queries:             watcherQueriesFromModel(plan.Queries),
		CalibrationContext:  plan.CalibrationContext.ValueString(),
		FinalizeCalibration: util.Ptr(true),
	})
	if err != nil {
		// The draft watcher already exists remotely. Persist it so Terraform
		// owns (and can clean up) the tainted resource instead of leaking an
		// uncalibrated watcher that nothing tracks.
		resp.Diagnostics.AddError("Failed to calibrate assistant watcher", err.Error())
		if draft, draftDiags := watcherToModel(ctx, created, plan); !draftDiags.HasError() {
			resp.Diagnostics.Append(resp.State.Set(ctx, draft)...)
		}
		return
	}

	if plan.Started.ValueBool() {
		// Keep the calibrated watcher on a separate variable: StartWatcher
		// returns a zero value on failure, and losing it here would strand a
		// fully-created watcher that Terraform no longer tracks.
		started, startErr := r.client.StartWatcher(ctx, watcher.ID)
		if startErr != nil {
			resp.Diagnostics.AddError("Failed to start assistant watcher", startErr.Error())
			if partial, partialDiags := watcherToModel(ctx, watcher, plan); !partialDiags.HasError() {
				resp.Diagnostics.Append(resp.State.Set(ctx, partial)...)
			}
			return
		}
		watcher = started
	}

	state, stateDiags := watcherToModel(ctx, watcher, plan)
	resp.Diagnostics.Append(stateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *watcherResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state watcherModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	watcher, err := r.client.GetWatcher(ctx, state.ID.ValueString())
	if err != nil {
		if errors.Is(err, assistantapi.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read assistant watcher", err.Error())
		return
	}

	model, diags := watcherToModel(ctx, watcher, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *watcherResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state watcherModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	datasourceUIDs, diags := listValueToStrings(ctx, plan.DatasourceUIDs)
	resp.Diagnostics.Append(diags...)
	actions, actionDiags := watcherActionsFromModel(ctx, plan.Actions)
	resp.Diagnostics.Append(actionDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	queries := watcherQueriesFromModel(plan.Queries)
	interval := plan.TriggerIntervalSeconds.ValueInt64()
	watcher, err := r.client.UpdateWatcher(ctx, state.ID.ValueString(), assistantapi.WatcherUpdate{
		Name:                   util.Ptr(plan.Name.ValueString()),
		Description:            util.Ptr(plan.Description.ValueString()),
		Prompt:                 util.Ptr(plan.Prompt.ValueString()),
		Sensitivity:            util.Ptr(plan.Sensitivity.ValueString()),
		TriggerIntervalSeconds: &interval,
		DisableDecisionSkip:    util.Ptr(plan.DisableDecisionSkip.ValueBool()),
		DatasourceUIDs:         &datasourceUIDs,
		CalibrationContext:     util.Ptr(plan.CalibrationContext.ValueString()),
		Queries:                &queries,
		// Always sent, even when the block is absent: the field is omitted from
		// the request when nil, which would leave a previously configured
		// action policy in place after the user deleted the block.
		Actions: actions,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to update assistant watcher", err.Error())
		return
	}

	// An update replaces the query list but never finalizes calibration, so a
	// watcher left in draft (a create whose finalize failed) would stay
	// unstartable forever. Re-finalize here so a subsequent apply repairs it.
	switch watcher.Status {
	case assistantapi.WatcherStatusDraft, assistantapi.WatcherStatusCalibrating:
		calibrated, calErr := r.client.AddWatcherQueries(ctx, watcher.ID, assistantapi.WatcherAddQueries{
			CalibrationContext:  plan.CalibrationContext.ValueString(),
			FinalizeCalibration: util.Ptr(true),
		})
		if calErr != nil {
			resp.Diagnostics.AddError("Failed to calibrate assistant watcher", calErr.Error())
			r.setStateBestEffort(ctx, &resp.State, &resp.Diagnostics, watcher, plan)
			return
		}
		watcher = calibrated
	}

	reconciled, err := r.reconcileStarted(ctx, watcher, plan.Started.ValueBool())
	if err != nil {
		// The update itself landed remotely. Persist it so state does not fall
		// back to pre-update values that no longer exist on the server.
		resp.Diagnostics.AddError("Failed to update assistant watcher lifecycle", err.Error())
		r.setStateBestEffort(ctx, &resp.State, &resp.Diagnostics, watcher, plan)
		return
	}

	model, stateDiags := watcherToModel(ctx, reconciled, plan)
	resp.Diagnostics.Append(stateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

// setStateBestEffort persists what did land remotely alongside an error, so a
// partially applied change stays under Terraform's management.
func (r *watcherResource) setStateBestEffort(ctx context.Context, state *tfsdk.State, diags *diag.Diagnostics, watcher assistantapi.Watcher, cfg watcherModel) {
	model, modelDiags := watcherToModel(ctx, watcher, cfg)
	if modelDiags.HasError() {
		return
	}
	diags.Append(state.Set(ctx, model)...)
}

func (r *watcherResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state watcherModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteWatcher(ctx, state.ID.ValueString()); err != nil {
		if errors.Is(err, assistantapi.ErrNotFound) {
			return
		}
		resp.Diagnostics.AddError("Failed to delete assistant watcher", err.Error())
	}
}

func (r *watcherResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	watcher, err := r.client.GetWatcher(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import assistant watcher", err.Error())
		return
	}
	model, diags := watcherToModel(ctx, watcher, watcherModel{})
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

// reconcileStarted drives the watcher's lifecycle to match the configured
// `started` value. Start and pause are separate endpoints rather than an
// updatable field, so the desired state is applied after the update lands.
func (r *watcherResource) reconcileStarted(ctx context.Context, watcher assistantapi.Watcher, started bool) (assistantapi.Watcher, error) {
	running := watcher.Status == assistantapi.WatcherStatusRunning
	switch {
	case started && !running:
		return r.client.StartWatcher(ctx, watcher.ID)
	case !started && running:
		return r.client.PauseWatcher(ctx, watcher.ID)
	}
	return watcher, nil
}

func watcherQueriesFromModel(queries []watcherQueryModel) []assistantapi.WatcherQuery {
	result := make([]assistantapi.WatcherQuery, 0, len(queries))
	for _, q := range queries {
		query := assistantapi.WatcherQuery{
			Type:          q.Type.ValueString(),
			Expr:          q.Expr.ValueString(),
			DatasourceUID: q.DatasourceUID.ValueString(),
			Comment:       q.Comment.ValueString(),
			Enabled:       util.Ptr(q.Enabled.ValueBool()),
			Role:          q.Role.ValueString(),
			GoodWhen:      q.GoodWhen.ValueString(),
		}
		// Sent whenever the block is present, even without a comparator: the
		// API rejects that with a clear error, whereas dropping the block here
		// would silently discard configured thresholds and leave state without
		// a block the config declares.
		if q.Thresholds != nil {
			query.Thresholds = &assistantapi.WatcherQueryThresholds{
				Comparator: q.Thresholds.Comparator.ValueString(),
				Warning:    float64PtrOrNil(q.Thresholds.Warning),
				Critical:   float64PtrOrNil(q.Thresholds.Critical),
				Source:     q.Thresholds.Source.ValueString(),
			}
		}
		result = append(result, query)
	}
	return result
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
func watcherToModel(ctx context.Context, watcher assistantapi.Watcher, cfg watcherModel) (watcherModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	datasourceUIDs, dsDiags := stringsToListValuePreserving(ctx, watcher.DatasourceUIDs, cfg.DatasourceUIDs)
	diags.Append(dsDiags...)

	queries := make([]watcherQueryModel, 0, len(watcher.Queries))
	for _, q := range watcher.Queries {
		enabled := true
		if q.Enabled != nil {
			enabled = *q.Enabled
		}
		model := watcherQueryModel{
			Type:          types.StringValue(q.Type),
			Expr:          types.StringValue(q.Expr),
			DatasourceUID: stringValueOrNull(q.DatasourceUID),
			Comment:       stringValueOrNull(q.Comment),
			Enabled:       types.BoolValue(enabled),
			Role:          stringValueOrNull(q.Role),
			GoodWhen:      stringValueOrNull(q.GoodWhen),
		}
		if q.Thresholds != nil {
			model.Thresholds = &watcherThresholdsModel{
				Comparator: stringValueOrNull(q.Thresholds.Comparator),
				Warning:    float64ValueOrNull(q.Thresholds.Warning),
				Critical:   float64ValueOrNull(q.Thresholds.Critical),
				Source:     stringValueOrNull(q.Thresholds.Source),
			}
		}
		queries = append(queries, model)
	}

	actions, actionDiags := watcherActionsToModel(ctx, watcher.Actions, cfg.Actions)
	diags.Append(actionDiags...)

	calibratedAt := types.StringNull()
	if watcher.CalibratedAt != nil {
		calibratedAt = types.StringValue(*watcher.CalibratedAt)
	}

	return watcherModel{
		ID:                     types.StringValue(watcher.ID),
		Name:                   types.StringValue(watcher.Name),
		Description:            stringValueOrNull(watcher.Description),
		Prompt:                 types.StringValue(watcher.Prompt),
		DatasourceUIDs:         datasourceUIDs,
		Sensitivity:            types.StringValue(watcher.Sensitivity),
		TriggerIntervalSeconds: types.Int64Value(watcher.TriggerIntervalSeconds),
		DisableDecisionSkip:    boolValueOrNull(watcher.DisableDecisionSkip),
		CalibrationContext:     stringValueOrNull(watcher.CalibrationContext),
		Started:                types.BoolValue(watcher.Status == assistantapi.WatcherStatusRunning),
		Status:                 types.StringValue(watcher.Status),
		CalibratedAt:           calibratedAt,
		Queries:                queries,
		Actions:                actions,
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
