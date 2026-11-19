package assistant

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common/assistantapi"
)

const resourceWatcherCalibrationName = "grafana_assistant_watcher_calibration"

type watcherCalibrationResource struct {
	client       *assistantapi.Client
	pollInterval time.Duration
}
type watcherCalibrationModel struct {
	ID             types.String `tfsdk:"id"`
	WatcherID      types.String `tfsdk:"watcher_id"`
	Started        types.Bool   `tfsdk:"started"`
	TimeoutSeconds types.Int64  `tfsdk:"timeout_seconds"`
	Status         types.String `tfsdk:"status"`
	FailureReason  types.String `tfsdk:"failure_reason"`
	Summary        types.String `tfsdk:"summary"`
}

func makeResourceWatcherCalibration() *common.Resource {
	return common.NewResource(common.CategoryGrafanaAssistant, resourceWatcherCalibrationName, common.NewResourceID(common.StringIDField("id")), &watcherCalibrationResource{})
}
func (r *watcherCalibrationResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceWatcherCalibrationName
}
func (r *watcherCalibrationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Ensures a Watcher has completed initial calibration without interactive questions. Separate setup state means a calibration failure or timeout cannot taint or replace the Watcher configuration resource. Reapplying resumes an existing attempt or accepts completed calibration, including calibration performed in Assistant. This resource does not enforce generated checks or rerun calibration after UI edits. Changing the monitoring goal or datasource scope replaces the parent Watcher. Removing this resource forgets setup state; it does not delete or pause the Watcher.`,
		Attributes: map[string]schema.Attribute{
			"id":              idAttribute(),
			"watcher_id":      schema.StringAttribute{Description: "ID of the Watcher to calibrate. Import uses the same ID.", Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"started":         schema.BoolAttribute{Description: "Explicit activation intent. Defaults to false. A false-to-true edit starts monitoring; a true-to-false edit pauses it. External pauses are respected: refresh and unrelated edits never resume monitoring. To resume intentionally, apply false then true.", Optional: true, Computed: true, Default: booldefault.StaticBool(false)},
			"timeout_seconds": schema.Int64Attribute{Description: "Maximum time to wait for initial calibration. A timeout leaves the Watcher and its attempt intact; reapply to resume observation.", Optional: true, Computed: true, Default: int64default.StaticInt64(1200), Validators: []validator.Int64{int64validator.Between(1, 7200)}},
			"status":          schema.StringAttribute{Description: "Most recently observed calibration status.", Computed: true},
			"failure_reason":  schema.StringAttribute{Description: "Machine-readable calibration failure reason, when available.", Computed: true},
			"summary":         schema.StringAttribute{Description: "Requester-visible failure summary. May be empty when the agent did not produce a closing message.", Computed: true},
		},
	}
}
func (r *watcherCalibrationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil || r.client != nil {
		return
	}
	client, err := withClientForResource(req, resp)
	if err == nil {
		r.client = client
	}
}
func (r *watcherCalibrationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan watcherCalibrationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = plan.WatcherID
	status, err := r.ensureCalibration(ctx, plan)
	setCalibrationStatus(&plan, status)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	if err != nil {
		resp.Diagnostics.AddError("Watcher setup did not complete", err.Error()+". The Watcher configuration is retained. Reapply to recover setup; do not replace the parent Watcher.")
	}
}
func (r *watcherCalibrationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state watcherCalibrationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	status, err := r.client.GetWatcherCalibration(ctx, state.WatcherID.ValueString())
	if errors.Is(err, assistantapi.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Failed to read Watcher setup", err.Error())
		return
	}
	setCalibrationStatus(&state, status)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}
func (r *watcherCalibrationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state watcherCalibrationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !plan.Started.Equal(state.Started) {
		var err error
		if plan.Started.ValueBool() {
			_, err = r.client.StartWatcher(ctx, plan.WatcherID.ValueString())
		} else {
			_, err = r.client.PauseWatcher(ctx, plan.WatcherID.ValueString())
		}
		if err != nil {
			resp.Diagnostics.AddError("Failed to change Watcher activation", err.Error())
			return
		}
	}
	status, err := r.client.GetWatcherCalibration(ctx, plan.WatcherID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read Watcher setup", err.Error())
		return
	}
	setCalibrationStatus(&plan, status)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}
func (r *watcherCalibrationResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// Setup owns no independent backend object. In particular, replacing tainted
	// setup after a timeout must never delete, cancel or pause the parent Watcher.
}
func (r *watcherCalibrationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	watcher, err := r.client.GetWatcher(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import Watcher setup", err.Error())
		return
	}
	status, err := r.client.GetWatcherCalibration(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import Watcher setup", err.Error())
		return
	}
	state := watcherCalibrationModel{ID: types.StringValue(req.ID), WatcherID: types.StringValue(req.ID), Started: types.BoolValue(watcher.Status == assistantapi.WatcherStatusRunning), TimeoutSeconds: types.Int64Value(1200)}
	setCalibrationStatus(&state, status)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}
func setCalibrationStatus(model *watcherCalibrationModel, status assistantapi.WatcherCalibration) {
	model.Status = types.StringValue(status.Status)
	model.FailureReason = stringValueOrNull(status.FailureReason)
	model.Summary = stringValueOrNull(status.Summary)
}

// ensureCalibration is an initial-setup convergence operation, not a request to
// rerun calibration. Completed UI/backend calibration satisfies it. That makes
// setup recovery safe without assuming the public POST has a replay identity.
func (r *watcherCalibrationResource) ensureCalibration(ctx context.Context, plan watcherCalibrationModel) (assistantapi.WatcherCalibration, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(plan.TimeoutSeconds.ValueInt64())*time.Second)
	defer cancel()
	id := plan.WatcherID.ValueString()
	watcher, err := r.client.GetWatcher(ctx, id)
	if err != nil {
		return assistantapi.WatcherCalibration{}, err
	}
	status, err := r.client.GetWatcherCalibration(ctx, id)
	if err != nil {
		return status, err
	}
	if watcher.CalibratedAt != nil && watcher.Status != assistantapi.WatcherStatusCalibrating {
		// Explicit setup creation may opt into starting an already calibrated
		// imported Watcher. Ordinary Read never reaches this path.
		if plan.Started.ValueBool() && watcher.Status != assistantapi.WatcherStatusRunning {
			_, err = r.client.StartWatcher(ctx, id)
		}
		return status, err
	}
	switch status.Status {
	case "queued", "running":
		if !status.Unattended || status.StartWhenReady != plan.Started.ValueBool() {
			return status, fmt.Errorf("existing calibration has different unattended or activation settings; finish or cancel it in Assistant before applying")
		}
	case "needs_input":
		return status, fmt.Errorf("existing calibration requires input; resolve or cancel it in Assistant before applying")
	case "idle", "failed", "cancelled":
		status, err = r.client.CalibrateWatcher(ctx, id, assistantapi.WatcherCalibrationRequest{Unattended: true, StartWhenReady: plan.Started.ValueBool()})
		if err != nil {
			if !assistantapi.IsAmbiguousWriteError(err) {
				return status, err
			}
			// A lost response can hide successful acceptance. Observe once instead of
			// replaying POST, which could create a new attempt after fast completion.
			recovered, readErr := r.client.GetWatcherCalibration(ctx, id)
			if readErr != nil || !recovered.Unattended || recovered.StartWhenReady != plan.Started.ValueBool() || (recovered.Status != "queued" && recovered.Status != "running" && recovered.Status != "completed") {
				return recovered, err
			}
			status = recovered
		}
	case "completed":
		return status, fmt.Errorf("calibration reports completion but the Watcher has no calibrated definition")
	default:
		return status, fmt.Errorf("unsupported calibration status %q", status.Status)
	}
	return r.waitForCalibration(ctx, id, status)
}

func (r *watcherCalibrationResource) waitForCalibration(ctx context.Context, id string, status assistantapi.WatcherCalibration) (assistantapi.WatcherCalibration, error) {
	var err error
	startedAt := status.StartedAt
	interval := r.pollInterval
	if interval <= 0 {
		interval = 2 * time.Second
	}
	for {
		if startedAt != "" && status.StartedAt != startedAt {
			return status, fmt.Errorf("calibration was superseded by another attempt; inspect the Watcher before retrying")
		}
		switch status.Status {
		case "completed":
			return status, nil
		case "failed", "cancelled", "needs_input":
			return status, fmt.Errorf("calibration %s (%s): %s %s", status.Status, status.FailureReason, status.Message, status.Summary)
		case "queued", "running":
		default:
			return status, fmt.Errorf("unexpected calibration status %q", status.Status)
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return status, ctx.Err()
		case <-timer.C:
		}
		status, err = r.client.GetWatcherCalibration(ctx, id)
		if err != nil {
			return status, err
		}
	}
}
