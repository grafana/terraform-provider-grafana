package k6

import (
	"context"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/grafana/k6-cloud-openapi-client-go/k6"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSourceWithConfigure = (*scheduleDataSource)(nil)
)

var (
	dataSourceScheduleName = "grafana_k6_schedule"
)

func dataSourceSchedule() *common.DataSource {
	return common.NewDataSource(
		common.CategoryK6,
		dataSourceScheduleName,
		&scheduleDataSource{},
	)
}

// scheduleDataSourceModel maps the data source schema data.
type scheduleDataSourceModel struct {
	ID             types.String         `tfsdk:"id"`
	LoadTestID     types.String         `tfsdk:"load_test_id"`
	Starts         types.String         `tfsdk:"starts"`
	RecurrenceRule *recurrenceRuleModel `tfsdk:"recurrence_rule"`
	Deactivated    types.Bool           `tfsdk:"deactivated"`
	NextRun        types.String         `tfsdk:"next_run"`
	CreatedBy      types.String         `tfsdk:"created_by"`
}

// scheduleDataSource is the data source implementation.
type scheduleDataSource struct {
	basePluginFrameworkDataSource
}

// Metadata returns the data source type name.
func (d *scheduleDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = dataSourceScheduleName
}

// Schema defines the schema for the data source.
func (d *scheduleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves a k6 schedule.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Numeric identifier of the schedule.",
				Required:    true,
			},
			"load_test_id": schema.StringAttribute{
				Description: "The identifier of the load test to schedule.",
				Computed:    true,
			},
			"starts": schema.StringAttribute{
				Description: "The start time for the schedule (RFC3339 format).",
				Computed:    true,
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
		Blocks: map[string]schema.Block{
			"recurrence_rule": schema.SingleNestedBlock{
				Description: "The schedule recurrence settings. If null, the test will run only once on the starts date.",
				Attributes: map[string]schema.Attribute{
					"frequency": schema.StringAttribute{
						Description: "The frequency of the schedule (HOURLY, DAILY, WEEKLY, MONTHLY).",
						Computed:    true,
					},
					"interval": schema.Int32Attribute{
						Description: "The interval between each frequency iteration (e.g., 2 = every 2 hours for HOURLY).",
						Computed:    true,
					},
					"count": schema.Int32Attribute{
						Description: "How many times the recurrence will repeat.",
						Computed:    true,
					},
					"until": schema.StringAttribute{
						Description: "The end time for the recurrence (RFC3339 format).",
						Computed:    true,
					},
					"byday": schema.ListAttribute{
						Description: "The weekdays when the 'WEEKLY' recurrence will be applied (e.g., ['MO', 'WE', 'FR']). Cannot be set for other frequencies.",
						Computed:    true,
						ElementType: types.StringType,
					},
				},
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *scheduleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state scheduleDataSourceModel
	diags := req.Config.Get(ctx, &state)
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

	// Retrieve the schedule
	ctx = context.WithValue(ctx, k6.ContextAccessToken, d.config.Token)
	k6Req := d.client.SchedulesAPI.SchedulesRetrieve(ctx, scheduleID).
		XStackId(d.config.StackID)

	schedule, _, err := k6Req.Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading k6 schedule",
			"Could not read k6 schedule with id "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Populate the data source model from the API response
	populateScheduleDataSourceModel(schedule, &state)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// populateScheduleDataSourceModel populates the data source model from the k6 API response
func populateScheduleDataSourceModel(schedule *k6.ScheduleApiModel, model *scheduleDataSourceModel) {
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
}
