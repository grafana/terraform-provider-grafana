package k6

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/grafana/k6-cloud-openapi-client-go/k6"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.ResourceWithConfigure   = (*scheduleResource)(nil)
	_ resource.ResourceWithImportState = (*scheduleResource)(nil)
)

var (
	resourceScheduleName = "grafana_k6_schedule"
	resourceScheduleID   = common.NewResourceID(common.IntIDField("id"))
)

func resourceSchedule() *common.Resource {
	return common.NewResource(
		common.CategoryK6,
		resourceScheduleName,
		resourceScheduleID,
		&scheduleResource{},
	)
}

// scheduleResourceModel maps the resource schema data.
type scheduleResourceModel struct {
	ID          types.String `tfsdk:"id"`
	LoadTestID  types.String `tfsdk:"load_test_id"`
	Starts      types.String `tfsdk:"starts"`
	Frequency   types.String `tfsdk:"frequency"`
	Interval    types.Int64  `tfsdk:"interval"`
	Occurrences types.Int64  `tfsdk:"occurrences"`
	Until       types.String `tfsdk:"until"`
	Deactivated types.Bool   `tfsdk:"deactivated"`
	NextRun     types.String `tfsdk:"next_run"`
	CreatedBy   types.String `tfsdk:"created_by"`
}

// scheduleResource is the resource implementation.
type scheduleResource struct {
	basePluginFrameworkResource
}

// Metadata returns the resource type name.
func (r *scheduleResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceScheduleName
}

// Schema defines the schema for the resource.
func (r *scheduleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a k6 schedule for automated test execution.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Numeric identifier of the schedule.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"load_test_id": schema.StringAttribute{
				Description: "The identifier of the load test to schedule.",
				Required:    true,
			},
			"starts": schema.StringAttribute{
				Description: "The start time for the schedule (RFC3339 format).",
				Required:    true,
			},
			"frequency": schema.StringAttribute{
				Description: "The frequency of the schedule (HOURLY, DAILY, WEEKLY, MONTHLY).",
				Required:    true,
			},
			"interval": schema.Int64Attribute{
				Description: "The interval between each frequency iteration (e.g., 2 = every 2 hours for HOURLY).",
				Optional:    true,
			},
			"occurrences": schema.Int64Attribute{
				Description: "How many times the recurrence will repeat.",
				Optional:    true,
			},
			"until": schema.StringAttribute{
				Description: "The end time for the recurrence (RFC3339 format).",
				Optional:    true,
			},
			"deactivated": schema.BoolAttribute{
				Description: "Whether the schedule is deactivated.",
				Computed:    true,
			},
			"next_run": schema.StringAttribute{
				Description: "The next scheduled execution time.",
				Computed:    true,
			},
			"created_by": schema.StringAttribute{
				Description: "The email of the user who created the schedule.",
				Computed:    true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *scheduleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan scheduleResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Parse load test ID
	loadTestID, err := strconv.ParseInt(plan.LoadTestID.ValueString(), 10, 32)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing load test ID",
			"Could not parse load test ID '"+plan.LoadTestID.ValueString()+"': "+err.Error(),
		)
		return
	}

	// Check if a schedule already exists for this load test
	ctx = context.WithValue(ctx, k6.ContextAccessToken, r.config.Token)
	existingSchedule, _, err := r.client.SchedulesAPI.LoadTestsScheduleRetrieve(ctx, int32(loadTestID)).
		XStackId(r.config.StackID).
		Execute()
	if err == nil && existingSchedule != nil {
		resp.Diagnostics.AddError(
			"Schedule already exists for load test",
			fmt.Sprintf("Load test %d already has a schedule (ID: %d). Each load test can only have one schedule. "+
				"To replace the existing schedule, import it first: terraform import grafana_k6_schedule.resource_name %d",
				loadTestID, existingSchedule.GetId(), existingSchedule.GetId()),
		)
		return
	}

	// Parse starts time
	startsTime, err := time.Parse(time.RFC3339, plan.Starts.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing starts time",
			"Could not parse starts time '"+plan.Starts.ValueString()+"'. Expected RFC3339 format: "+err.Error(),
		)
		return
	}

	// Parse frequency
	frequency, err := k6.NewFrequencyFromValue(plan.Frequency.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing frequency",
			"Invalid frequency '"+plan.Frequency.ValueString()+"'. Valid values are: HOURLY, DAILY, WEEKLY, MONTHLY",
		)
		return
	}

	// Build recurrence rule
	recurrenceRule := k6.NewScheduleRecurrenceRule(*frequency)
	if !plan.Interval.IsNull() {
		interval := int32(plan.Interval.ValueInt64())
		recurrenceRule.SetInterval(interval)
	}
	if !plan.Occurrences.IsNull() {
		count := int32(plan.Occurrences.ValueInt64())
		recurrenceRule.SetCount(count)
	}
	if !plan.Until.IsNull() {
		untilTime, err := time.Parse(time.RFC3339, plan.Until.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error parsing until time",
				"Could not parse until time '"+plan.Until.ValueString()+"'. Expected RFC3339 format: "+err.Error(),
			)
			return
		}
		recurrenceRule.SetUntil(untilTime)
	}

	// Generate API request body from plan
	scheduleRequest := k6.NewCreateScheduleRequest(startsTime, *k6.NewNullableScheduleRecurrenceRule(recurrenceRule))

	// Create new schedule
	ctx = context.WithValue(ctx, k6.ContextAccessToken, r.config.Token)
	createReq := r.client.SchedulesAPI.LoadTestsScheduleCreate(ctx, int32(loadTestID)).
		CreateScheduleRequest(scheduleRequest).
		XStackId(r.config.StackID)

	schedule, _, err := createReq.Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating schedule",
			"Could not create schedule, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	r.populateModelFromAPI(schedule, &plan)

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read retrieves the resource information.
func (r *scheduleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state scheduleResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If the ID is empty, we cannot read the resource.
	if state.ID.ValueString() == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	intID, err := strconv.ParseInt(state.ID.ValueString(), 10, 32)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing schedule ID",
			"Could not parse schedule ID '"+state.ID.ValueString()+"': "+err.Error(),
		)
		return
	}
	scheduleID := int32(intID)

	// Retrieve the schedule
	ctx = context.WithValue(ctx, k6.ContextAccessToken, r.config.Token)
	readReq := r.client.SchedulesAPI.SchedulesRetrieve(ctx, scheduleID).
		XStackId(r.config.StackID)

	schedule, httpResp, err := readReq.Execute()

	if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	} else if err != nil {
		resp.Diagnostics.AddError(
			"Error reading k6 schedule",
			"Could not read k6 schedule with id "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Overwrite items with refreshed state
	r.populateModelFromAPI(schedule, &state)

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
// k6 schedules API replaces the schedule when creating with the same load test ID
func (r *scheduleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan
	var plan scheduleResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Parse load test ID
	loadTestID, err := strconv.ParseInt(plan.LoadTestID.ValueString(), 10, 32)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing load test ID",
			"Could not parse load test ID '"+plan.LoadTestID.ValueString()+"': "+err.Error(),
		)
		return
	}

	// Parse starts time
	startsTime, err := time.Parse(time.RFC3339, plan.Starts.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing starts time",
			"Could not parse starts time '"+plan.Starts.ValueString()+"'. Expected RFC3339 format: "+err.Error(),
		)
		return
	}

	// Parse frequency
	frequency, err := k6.NewFrequencyFromValue(plan.Frequency.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing frequency",
			"Invalid frequency '"+plan.Frequency.ValueString()+"'. Valid values are: HOURLY, DAILY, WEEKLY, MONTHLY",
		)
		return
	}

	// Build recurrence rule
	recurrenceRule := k6.NewScheduleRecurrenceRule(*frequency)
	if !plan.Interval.IsNull() {
		interval := int32(plan.Interval.ValueInt64())
		recurrenceRule.SetInterval(interval)
	}
	if !plan.Occurrences.IsNull() {
		count := int32(plan.Occurrences.ValueInt64())
		recurrenceRule.SetCount(count)
	}
	if !plan.Until.IsNull() {
		untilTime, err := time.Parse(time.RFC3339, plan.Until.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error parsing until time",
				"Could not parse until time '"+plan.Until.ValueString()+"'. Expected RFC3339 format: "+err.Error(),
			)
			return
		}
		recurrenceRule.SetUntil(untilTime)
	}

	// Generate API request body from plan
	scheduleRequest := k6.NewCreateScheduleRequest(startsTime, *k6.NewNullableScheduleRecurrenceRule(recurrenceRule))

	// Update schedule (replaces existing schedule for the load test)
	ctx = context.WithValue(ctx, k6.ContextAccessToken, r.config.Token)
	createReq := r.client.SchedulesAPI.LoadTestsScheduleCreate(ctx, int32(loadTestID)).
		CreateScheduleRequest(scheduleRequest).
		XStackId(r.config.StackID)

	schedule, _, err := createReq.Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating schedule",
			"Could not update schedule, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	r.populateModelFromAPI(schedule, &plan)

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *scheduleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state scheduleResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	intID, err := strconv.ParseInt(state.ID.ValueString(), 10, 32)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing schedule ID",
			"Could not parse schedule ID '"+state.ID.ValueString()+"': "+err.Error(),
		)
		return
	}
	scheduleID := int32(intID)

	// Delete existing schedule
	ctx = context.WithValue(ctx, k6.ContextAccessToken, r.config.Token)
	deleteReq := r.client.SchedulesAPI.SchedulesDestroy(ctx, scheduleID).
		XStackId(r.config.StackID)

	_, err = deleteReq.Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting k6 schedule",
			"Could not delete k6 schedule with id "+strconv.Itoa(int(scheduleID))+": "+err.Error(),
		)
	}
}

func (r *scheduleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// populateModelFromAPI populates the terraform model from the k6 API response
func (r *scheduleResource) populateModelFromAPI(schedule *k6.ScheduleApiModel, model *scheduleResourceModel) {
	model.ID = types.StringValue(strconv.Itoa(int(schedule.GetId())))
	model.LoadTestID = types.StringValue(strconv.Itoa(int(schedule.GetLoadTestId())))
	model.Starts = types.StringValue(schedule.GetStarts().Format(time.RFC3339))
	model.Deactivated = types.BoolValue(schedule.GetDeactivated())

	if nextRun, ok := schedule.GetNextRunOk(); ok && nextRun != nil {
		model.NextRun = types.StringValue(nextRun.Format(time.RFC3339))
	} else {
		model.NextRun = types.StringNull()
	}

	if createdBy, ok := schedule.GetCreatedByOk(); ok && createdBy != nil {
		model.CreatedBy = types.StringValue(*createdBy)
	} else {
		model.CreatedBy = types.StringNull()
	}

	// Extract recurrence rule details
	if recurrenceRule, ok := schedule.GetRecurrenceRuleOk(); ok {
		model.Frequency = types.StringValue(string(recurrenceRule.GetFrequency()))

		if interval, ok := recurrenceRule.GetIntervalOk(); ok && interval != nil {
			model.Interval = types.Int64Value(int64(*interval))
		} else {
			model.Interval = types.Int64Null()
		}

		if count, ok := recurrenceRule.GetCountOk(); ok && count != nil {
			model.Occurrences = types.Int64Value(int64(*count))
		} else {
			model.Occurrences = types.Int64Null()
		}

		if until, ok := recurrenceRule.GetUntilOk(); ok && until != nil {
			model.Until = types.StringValue(until.Format(time.RFC3339))
		} else {
			model.Until = types.StringNull()
		}
	}
}
