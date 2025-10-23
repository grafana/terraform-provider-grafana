package k6

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/grafana/k6-cloud-openapi-client-go/k6"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common/k6providerapi"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.ResourceWithConfigure   = (*scheduleResource)(nil)
	_ resource.ResourceWithImportState = (*scheduleResource)(nil)
)

var (
	resourceScheduleName = "grafana_k6_schedule"
	resourceScheduleID   = common.NewResourceID(common.IntIDField("load_test_id"))
)

func resourceSchedule() *common.Resource {
	return common.NewResource(
		common.CategoryK6,
		resourceScheduleName,
		resourceScheduleID,
		&scheduleResource{},
	).WithLister(k6ListerFunction(listSchedules))
}

// recurrenceRuleModel maps the recurrence rule schema data.
type recurrenceRuleModel struct {
	Frequency types.String   `tfsdk:"frequency"`
	Interval  types.Int32    `tfsdk:"interval"`
	Count     types.Int32    `tfsdk:"count"`
	Until     types.String   `tfsdk:"until"`
	Byday     []types.String `tfsdk:"byday"`
}

// cronScheduleModel maps the cron schedule schema data.
type cronScheduleModel struct {
	Schedule types.String `tfsdk:"schedule"`
	Timezone types.String `tfsdk:"timezone"`
}

// scheduleResourceModel maps the resource schema data.
type scheduleResourceModel struct {
	ID             types.String         `tfsdk:"id"`
	LoadTestID     types.String         `tfsdk:"load_test_id"`
	Starts         types.String         `tfsdk:"starts"`
	RecurrenceRule *recurrenceRuleModel `tfsdk:"recurrence_rule"`
	Cron           *cronScheduleModel   `tfsdk:"cron"`
	Deactivated    types.Bool           `tfsdk:"deactivated"`
	NextRun        types.String         `tfsdk:"next_run"`
	CreatedBy      types.String         `tfsdk:"created_by"`
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"starts": schema.StringAttribute{
				Description: "The start time for the schedule (RFC3339 format).",
				Required:    true,
			},
			"deactivated": schema.BoolAttribute{
				Description: "Whether the schedule is deactivated.",
				Computed:    true,
				Default:     booldefault.StaticBool(false),
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
		Blocks: map[string]schema.Block{
			"recurrence_rule": schema.SingleNestedBlock{
				Description: "The schedule recurrence settings. If not specified, the test will run only once on the 'starts' date. Only one of `recurrence_rule` and `cron` can be set.",
				Attributes: map[string]schema.Attribute{
					"frequency": schema.StringAttribute{
						Description: "The frequency of the schedule (HOURLY, DAILY, WEEKLY, MONTHLY, YEARLY).",
						Optional:    true,
						Validators: []validator.String{
							stringvalidator.OneOf("HOURLY", "DAILY", "WEEKLY", "MONTHLY", "YEARLY"),
						},
					},
					"interval": schema.Int32Attribute{
						Description: "The interval between each frequency iteration (e.g., 2 = every 2 hours for HOURLY). Defaults to 1.",
						Optional:    true,
						Computed:    true,
						Default:     int32default.StaticInt32(1),
					},
					"count": schema.Int32Attribute{
						Description: "How many times the recurrence will repeat.",
						Optional:    true,
					},
					"until": schema.StringAttribute{
						Description: "The end time for the recurrence (RFC3339 format).",
						Optional:    true,
					},
					"byday": schema.ListAttribute{
						Description: "The weekdays when the 'WEEKLY' recurrence will be applied (e.g., ['MO', 'WE', 'FR']). Cannot be set for other frequencies.",
						Optional:    true,
						ElementType: types.StringType,
					},
				},
				Validators: []validator.Object{
					objectvalidator.AlsoRequires(
						path.MatchRelative().AtName("frequency"),
					),
				},
			},
			"cron": schema.SingleNestedBlock{
				Description: "The cron schedule to trigger the test periodically. If not specified, the test will run only once on the 'starts' date. Only one of `recurrence_rule` and `cron` can be set.",
				Attributes: map[string]schema.Attribute{
					"schedule": schema.StringAttribute{
						Description: "A cron expression with exactly 5 entries, or an alias. The allowed aliases are: @yearly, @annually, @monthly, @weekly, @daily, @hourly.",
						Optional:    true,
					},
					"timezone": schema.StringAttribute{
						Description: "The timezone of the cron expression. For example, 'UTC' or 'Europe/London'.",
						Optional:    true,
					},
				},
				Validators: []validator.Object{
					objectvalidator.AlsoRequires(
						path.MatchRelative().AtName("schedule"),
						path.MatchRelative().AtName("timezone"),
					),
					objectvalidator.ConflictsWith(
						path.MatchRoot("recurrence_rule"),
					),
				},
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

	// Parse starts time
	startsTime, err := time.Parse(time.RFC3339, plan.Starts.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing starts time",
			"Could not parse starts time '"+plan.Starts.ValueString()+"'. Expected RFC3339 format: "+err.Error(),
		)
		return
	}

	scheduleRequest := k6.NewCreateScheduleRequest(startsTime)

	// Parse recurrence rule
	if plan.RecurrenceRule != nil {
		// Parse frequency
		frequency, err := k6.NewFrequencyFromValue(plan.RecurrenceRule.Frequency.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error parsing frequency",
				"Invalid frequency '"+plan.RecurrenceRule.Frequency.ValueString()+"'. Valid values are: HOURLY, DAILY, WEEKLY, MONTHLY, YEARLY",
			)
			return
		}

		recurrenceRule := k6.NewScheduleRecurrenceRule(*frequency)
		if !plan.RecurrenceRule.Interval.IsNull() {
			recurrenceRule.SetInterval(plan.RecurrenceRule.Interval.ValueInt32())
		}
		if !plan.RecurrenceRule.Count.IsNull() {
			recurrenceRule.SetCount(plan.RecurrenceRule.Count.ValueInt32())
		}
		if !plan.RecurrenceRule.Until.IsNull() {
			untilTime, err := time.Parse(time.RFC3339, plan.RecurrenceRule.Until.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(
					"Error parsing recurrence rule until time",
					"Could not parse recurrence rule until time '"+plan.RecurrenceRule.Until.ValueString()+"'. Expected RFC3339 format: "+err.Error(),
				)
				return
			}
			recurrenceRule.SetUntil(untilTime)
		}
		if len(plan.RecurrenceRule.Byday) > 0 {
			byday := make([]k6.Weekday, 0, len(plan.RecurrenceRule.Byday))
			for _, v := range plan.RecurrenceRule.Byday {
				byday = append(byday, k6.Weekday(v.ValueString()))
			}
			recurrenceRule.SetByday(byday)
		}

		scheduleRequest.SetRecurrenceRule(*recurrenceRule)
	}

	if plan.Cron != nil {
		cronSchedule := k6.NewScheduleCron(plan.Cron.Schedule.ValueString(), plan.Cron.Timezone.ValueString())
		scheduleRequest.SetCron(*cronSchedule)
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
				loadTestID, existingSchedule.GetId(), loadTestID),
		)
		return
	}

	// Create new schedule
	ctx = context.WithValue(ctx, k6.ContextAccessToken, r.config.Token)
	createReq := r.client.SchedulesAPI.LoadTestsScheduleCreate(ctx, int32(loadTestID)).
		CreateScheduleRequest(scheduleRequest).
		XStackId(r.config.StackID)

	schedule, httpResp, err := createReq.Execute()
	if err != nil {
		var apiErrMsg string
		if httpResp != nil {
			bodyBytes, _ := io.ReadAll(httpResp.Body)
			apiErrMsg = string(bodyBytes)
		}
		resp.Diagnostics.AddError(
			"Error creating schedule",
			"Could not create schedule, unexpected error: "+err.Error()+"\nAPI response: "+apiErrMsg,
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

	// If the LoadTestID is empty, we cannot read the resource.
	if state.LoadTestID.ValueString() == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	intID, err := strconv.ParseInt(state.LoadTestID.ValueString(), 10, 32)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing load test ID",
			"Could not parse load test ID '"+state.LoadTestID.ValueString()+"': "+err.Error(),
		)
		return
	}
	loadTestID := int32(intID)

	// Retrieve the schedule by load test ID
	ctx = context.WithValue(ctx, k6.ContextAccessToken, r.config.Token)
	readReq := r.client.SchedulesAPI.LoadTestsScheduleRetrieve(ctx, loadTestID).
		XStackId(r.config.StackID)

	schedule, httpResp, err := readReq.Execute()

	if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	} else if err != nil {
		resp.Diagnostics.AddError(
			"Error reading k6 schedule",
			"Could not read k6 schedule for load test "+state.LoadTestID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Overwrite items with refreshed state
	r.populateModelFromAPI(schedule, &state)

	if state.Deactivated.ValueBool() {
		resp.Diagnostics.AddWarning(
			"Found a deactivated schedule.",
			"The schedule has been deactivated on remote and will be reactivated on the next apply.",
		)
	}

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

	scheduleRequest := k6.NewCreateScheduleRequest(startsTime)

	// Parse recurrence rule
	if plan.RecurrenceRule != nil {
		// Parse frequency
		frequency, err := k6.NewFrequencyFromValue(plan.RecurrenceRule.Frequency.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error parsing frequency",
				"Invalid frequency '"+plan.RecurrenceRule.Frequency.ValueString()+"'. Valid values are: HOURLY, DAILY, WEEKLY, MONTHLY, YEARLY",
			)
			return
		}

		recurrenceRule := k6.NewScheduleRecurrenceRule(*frequency)
		if !plan.RecurrenceRule.Interval.IsNull() {
			recurrenceRule.SetInterval(plan.RecurrenceRule.Interval.ValueInt32())
		}
		if !plan.RecurrenceRule.Count.IsNull() {
			recurrenceRule.SetCount(plan.RecurrenceRule.Count.ValueInt32())
		}
		if !plan.RecurrenceRule.Until.IsNull() {
			untilTime, err := time.Parse(time.RFC3339, plan.RecurrenceRule.Until.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(
					"Error parsing recurrence rule until time",
					"Could not parse recurrence rule until time '"+plan.RecurrenceRule.Until.ValueString()+"'. Expected RFC3339 format: "+err.Error(),
				)
				return
			}
			recurrenceRule.SetUntil(untilTime)
		}
		if len(plan.RecurrenceRule.Byday) > 0 {
			byday := make([]k6.Weekday, 0, len(plan.RecurrenceRule.Byday))
			for _, v := range plan.RecurrenceRule.Byday {
				byday = append(byday, k6.Weekday(v.ValueString()))
			}
			recurrenceRule.SetByday(byday)
		}

		scheduleRequest.SetRecurrenceRule(*recurrenceRule)
	}

	if plan.Cron != nil {
		cronSchedule := k6.NewScheduleCron(plan.Cron.Schedule.ValueString(), plan.Cron.Timezone.ValueString())
		scheduleRequest.SetCron(*cronSchedule)
	}

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

	// Parse load test ID to first retrieve the schedule
	loadTestID, err := strconv.ParseInt(state.LoadTestID.ValueString(), 10, 32)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing load test ID",
			"Could not parse load test ID '"+state.LoadTestID.ValueString()+"': "+err.Error(),
		)
		return
	}

	// First retrieve the schedule to get its ID
	ctx = context.WithValue(ctx, k6.ContextAccessToken, r.config.Token)
	schedule, httpResp, err := r.client.SchedulesAPI.LoadTestsScheduleRetrieve(ctx, int32(loadTestID)).
		XStackId(r.config.StackID).
		Execute()

	if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
		// Schedule already doesn't exist, nothing to delete
		return
	} else if err != nil {
		resp.Diagnostics.AddError(
			"Error retrieving k6 schedule for deletion",
			"Could not retrieve k6 schedule for load test "+state.LoadTestID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Delete existing schedule using its ID
	deleteReq := r.client.SchedulesAPI.SchedulesDestroy(ctx, schedule.GetId()).
		XStackId(r.config.StackID)

	_, err = deleteReq.Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting k6 schedule",
			"Could not delete k6 schedule with id "+strconv.Itoa(int(schedule.GetId()))+": "+err.Error(),
		)
	}
}

func (r *scheduleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("load_test_id"), req, resp)
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
	if recurrenceRule, ok := schedule.GetRecurrenceRuleOk(); ok && recurrenceRule != nil {
		model.RecurrenceRule = &recurrenceRuleModel{
			Frequency: types.StringValue(string(recurrenceRule.GetFrequency())),
		}

		if interval, ok := recurrenceRule.GetIntervalOk(); ok && interval != nil {
			model.RecurrenceRule.Interval = types.Int32Value(*interval)
		} else {
			model.RecurrenceRule.Interval = types.Int32Null()
		}

		if count, ok := recurrenceRule.GetCountOk(); ok && count != nil {
			model.RecurrenceRule.Count = types.Int32Value(*count)
		} else {
			model.RecurrenceRule.Count = types.Int32Null()
		}

		if until, ok := recurrenceRule.GetUntilOk(); ok && until != nil {
			model.RecurrenceRule.Until = types.StringValue(until.Format(time.RFC3339))
		} else {
			model.RecurrenceRule.Until = types.StringNull()
		}

		if byday, ok := recurrenceRule.GetBydayOk(); ok && byday != nil && len(byday) > 0 {
			model.RecurrenceRule.Byday = make([]types.String, 0, len(byday))
			for _, v := range byday {
				model.RecurrenceRule.Byday = append(model.RecurrenceRule.Byday, types.StringValue(string(v)))
			}
		} else {
			model.RecurrenceRule.Byday = nil
		}
	} else {
		model.RecurrenceRule = nil
	}

	if cronSchedule, ok := schedule.GetCronOk(); ok && cronSchedule != nil {
		model.Cron = &cronScheduleModel{
			Schedule: types.StringValue(cronSchedule.GetSchedule()),
			Timezone: types.StringValue(cronSchedule.GetTimeZone()),
		}
	} else {
		model.Cron = nil
	}
}

// listSchedules retrieves the list ids of all the existing schedules.
func listSchedules(ctx context.Context, client *k6.APIClient, config *k6providerapi.K6APIConfig) ([]string, error) {
	ctx = context.WithValue(ctx, k6.ContextAccessToken, config.Token)
	resp, _, err := client.SchedulesAPI.SchedulesList(ctx).
		XStackId(config.StackID).
		Execute()
	if err != nil {
		return nil, err
	}

	var ids []string
	for _, schedule := range resp.Value {
		ids = append(ids, strconv.Itoa(int(schedule.GetId())))
	}
	return ids, nil
}
