package k6

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/grafana/k6-cloud-openapi-client-go/k6"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSourceWithConfigure = (*schedulesDataSource)(nil)
)

var (
	dataSourceSchedulesName = "grafana_k6_schedules"
)

func dataSourceSchedules() *common.DataSource {
	return common.NewDataSource(
		common.CategoryK6,
		dataSourceSchedulesName,
		&schedulesDataSource{},
	)
}

// schedulesDataSourceModel maps the data source schema data.
type schedulesDataSourceModel struct {
	ID        types.String `tfsdk:"id"`
	Schedules types.List   `tfsdk:"schedules"`
}

// schedulesDataSource is the data source implementation.
type schedulesDataSource struct {
	basePluginFrameworkDataSource
}

// Metadata returns the data source type name.
func (d *schedulesDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = dataSourceSchedulesName
}

// Schema defines the schema for the data source.
func (d *schedulesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves all k6 schedules.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The identifier for this data source.",
				Computed:    true,
			},
			"schedules": schema.ListAttribute{
				Description: "List of k6 schedules.",
				Computed:    true,
				ElementType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"id":           types.StringType,
						"load_test_id": types.StringType,
						"starts":       types.StringType,
						"recurrence_rule": types.ObjectType{
							AttrTypes: map[string]attr.Type{
								"frequency": types.StringType,
								"interval":  types.Int32Type,
								"count":     types.Int32Type,
								"until":     types.StringType,
								"byday":     types.ListType{ElemType: types.StringType},
							},
						},
						"deactivated": types.BoolType,
						"next_run":    types.StringType,
						"created_by":  types.StringType,
					},
				},
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *schedulesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state schedulesDataSourceModel
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Retrieve all schedules
	ctx = context.WithValue(ctx, k6.ContextAccessToken, d.config.Token)
	k6Req := d.client.SchedulesAPI.SchedulesList(ctx).
		XStackId(d.config.StackID)

	schedulesList, _, err := k6Req.Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading k6 schedules",
			"Could not read k6 schedules: "+err.Error(),
		)
		return
	}

	state.ID = types.StringValue("k6-schedules")

	// Create schedule objects
	scheduleObjects := make([]attr.Value, 0, len(schedulesList.Value))

	// Add all schedules
	for _, schedule := range schedulesList.Value {
		scheduleModel := scheduleDataSourceModel{}
		populateScheduleDataSourceModel(&schedule, &scheduleModel)

		// Create recurrence rule object
		var recurrenceRuleObj attr.Value
		if scheduleModel.RecurrenceRule != nil {
			bydayList, _ := types.ListValue(types.StringType, make([]attr.Value, 0))
			if scheduleModel.RecurrenceRule.Byday != nil {
				bydayValues := make([]attr.Value, len(scheduleModel.RecurrenceRule.Byday))
				for i, day := range scheduleModel.RecurrenceRule.Byday {
					bydayValues[i] = day
				}
				bydayList, _ = types.ListValue(types.StringType, bydayValues)
			}

			recurrenceRuleObj, _ = types.ObjectValue(
				map[string]attr.Type{
					"frequency": types.StringType,
					"interval":  types.Int32Type,
					"count":     types.Int32Type,
					"until":     types.StringType,
					"byday":     types.ListType{ElemType: types.StringType},
				},
				map[string]attr.Value{
					"frequency": scheduleModel.RecurrenceRule.Frequency,
					"interval":  scheduleModel.RecurrenceRule.Interval,
					"count":     scheduleModel.RecurrenceRule.Count,
					"until":     scheduleModel.RecurrenceRule.Until,
					"byday":     bydayList,
				},
			)
		} else {
			recurrenceRuleObj = types.ObjectNull(map[string]attr.Type{
				"frequency": types.StringType,
				"interval":  types.Int32Type,
				"count":     types.Int32Type,
				"until":     types.StringType,
				"byday":     types.ListType{ElemType: types.StringType},
			})
		}

		// Create schedule object
		scheduleObj, _ := types.ObjectValue(
			map[string]attr.Type{
				"id":           types.StringType,
				"load_test_id": types.StringType,
				"starts":       types.StringType,
				"recurrence_rule": types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"frequency": types.StringType,
						"interval":  types.Int32Type,
						"count":     types.Int32Type,
						"until":     types.StringType,
						"byday":     types.ListType{ElemType: types.StringType},
					},
				},
				"deactivated": types.BoolType,
				"next_run":    types.StringType,
				"created_by":  types.StringType,
			},
			map[string]attr.Value{
				"id":              scheduleModel.ID,
				"load_test_id":    scheduleModel.LoadTestID,
				"starts":          scheduleModel.Starts,
				"recurrence_rule": recurrenceRuleObj,
				"deactivated":     scheduleModel.Deactivated,
				"next_run":        scheduleModel.NextRun,
				"created_by":      scheduleModel.CreatedBy,
			},
		)

		scheduleObjects = append(scheduleObjects, scheduleObj)
	}

	// Create the list
	state.Schedules, _ = types.ListValue(
		types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"id":           types.StringType,
				"load_test_id": types.StringType,
				"starts":       types.StringType,
				"recurrence_rule": types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"frequency": types.StringType,
						"interval":  types.Int32Type,
						"count":     types.Int32Type,
						"until":     types.StringType,
						"byday":     types.ListType{ElemType: types.StringType},
					},
				},
				"deactivated": types.BoolType,
				"next_run":    types.StringType,
				"created_by":  types.StringType,
			},
		},
		scheduleObjects,
	)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
