package oncall

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &teamHoursResource{}
	_ resource.ResourceWithConfigure   = &teamHoursResource{}
	_ resource.ResourceWithImportState = &teamHoursResource{}
)

const maxTeamHoursIntervals = 3

var timeOfDayRegexp = regexp.MustCompile(`^([01][0-9]|2[0-3]):[0-5][0-9](:[0-5][0-9])?$`)

func resourceTeamHours() *common.Resource {
	return common.NewResource(
		common.CategoryOnCall,
		"grafana_oncall_team_hours",
		resourceID,
		&teamHoursResource{},
	)
}

type teamHoursRangeModel struct {
	Start types.String `tfsdk:"start"`
	End   types.String `tfsdk:"end"`
}

type teamHoursIntervalModel struct {
	Name      types.String          `tfsdk:"name"`
	Monday    []teamHoursRangeModel `tfsdk:"monday"`
	Tuesday   []teamHoursRangeModel `tfsdk:"tuesday"`
	Wednesday []teamHoursRangeModel `tfsdk:"wednesday"`
	Thursday  []teamHoursRangeModel `tfsdk:"thursday"`
	Friday    []teamHoursRangeModel `tfsdk:"friday"`
	Saturday  []teamHoursRangeModel `tfsdk:"saturday"`
	Sunday    []teamHoursRangeModel `tfsdk:"sunday"`
}

type teamHoursResourceModel struct {
	ID        types.String             `tfsdk:"id"`
	TeamID    types.String             `tfsdk:"team_id"`
	Timezone  types.String             `tfsdk:"timezone"`
	Intervals []teamHoursIntervalModel `tfsdk:"interval"`
}

// days returns pointers to each weekday's ranges in Monday..Sunday order.
func (m *teamHoursIntervalModel) days() [7]*[]teamHoursRangeModel {
	return [7]*[]teamHoursRangeModel{&m.Monday, &m.Tuesday, &m.Wednesday, &m.Thursday, &m.Friday, &m.Saturday, &m.Sunday}
}

// normalizeTimeOfDay pads HH:MM to HH:MM:SS, the form the API returns.
func normalizeTimeOfDay(s string) string {
	if len(s) == 5 {
		return s + ":00"
	}
	return s
}

func rangesToAPI(in []teamHoursRangeModel) []onCallAPI.TimeRange {
	out := make([]onCallAPI.TimeRange, 0, len(in))
	for _, r := range in {
		out = append(out, onCallAPI.TimeRange{Start: r.Start.ValueString(), End: r.End.ValueString()})
	}
	return out
}

func intervalsToAPI(in []teamHoursIntervalModel) []onCallAPI.TeamHoursInterval {
	out := make([]onCallAPI.TeamHoursInterval, 0, len(in))
	for i := range in {
		d := in[i].days()
		out = append(out, onCallAPI.TeamHoursInterval{
			Name: in[i].Name.ValueString(),
			Hours: onCallAPI.WeekHours{
				Monday:    rangesToAPI(*d[0]),
				Tuesday:   rangesToAPI(*d[1]),
				Wednesday: rangesToAPI(*d[2]),
				Thursday:  rangesToAPI(*d[3]),
				Friday:    rangesToAPI(*d[4]),
				Saturday:  rangesToAPI(*d[5]),
				Sunday:    rangesToAPI(*d[6]),
			},
		})
	}
	return out
}

// mergeRanges converts API ranges to the model, keeping the prior (configured) value
// when it means the same thing, so "09:00" in config doesn't drift to "09:00:00".
func mergeRanges(api []onCallAPI.TimeRange, prior []teamHoursRangeModel) []teamHoursRangeModel {
	if len(api) == 0 {
		return []teamHoursRangeModel{}
	}
	out := make([]teamHoursRangeModel, len(api))
	for i, r := range api {
		out[i] = teamHoursRangeModel{Start: types.StringValue(r.Start), End: types.StringValue(r.End)}
		if i < len(prior) &&
			normalizeTimeOfDay(prior[i].Start.ValueString()) == r.Start &&
			normalizeTimeOfDay(prior[i].End.ValueString()) == r.End {
			out[i] = prior[i]
		}
	}
	return out
}

func intervalsFromAPI(api []onCallAPI.TeamHoursInterval, prior []teamHoursIntervalModel) []teamHoursIntervalModel {
	out := make([]teamHoursIntervalModel, len(api))
	for i, in := range api {
		var p teamHoursIntervalModel
		if i < len(prior) {
			p = prior[i]
		}
		pd := p.days()
		w := in.Hours
		out[i] = teamHoursIntervalModel{Name: types.StringValue(in.Name)}
		od := out[i].days()
		*od[0] = mergeRanges(w.Monday, *pd[0])
		*od[1] = mergeRanges(w.Tuesday, *pd[1])
		*od[2] = mergeRanges(w.Wednesday, *pd[2])
		*od[3] = mergeRanges(w.Thursday, *pd[3])
		*od[4] = mergeRanges(w.Friday, *pd[4])
		*od[5] = mergeRanges(w.Saturday, *pd[5])
		*od[6] = mergeRanges(w.Sunday, *pd[6])
	}
	return out
}

type teamHoursResource struct {
	basePluginFrameworkResource
}

func (r *teamHoursResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "grafana_oncall_team_hours"
}

func dayBlock(day string) schema.ListNestedBlock {
	return schema.ListNestedBlock{
		MarkdownDescription: fmt.Sprintf("A time range on %s. Up to three, in start-time order and not overlapping. Omit the block to make the day a day off.", day),
		Validators:          []validator.List{listvalidator.SizeAtMost(3)},
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"start": schema.StringAttribute{
					Required:            true,
					MarkdownDescription: "Start time as `HH:MM` or `HH:MM:SS`, in the team's timezone.",
					Validators:          []validator.String{stringvalidator.RegexMatches(timeOfDayRegexp, "must be HH:MM or HH:MM:SS")},
				},
				"end": schema.StringAttribute{
					Required:            true,
					MarkdownDescription: "End time as `HH:MM` or `HH:MM:SS`, in the team's timezone. Must be after `start`; to end at midnight use `23:59:59`.",
					Validators:          []validator.String{stringvalidator.RegexMatches(timeOfDayRegexp, "must be HH:MM or HH:MM:SS")},
				},
			},
		},
	}
}

func (r *teamHoursResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	dayBlocks := map[string]schema.Block{}
	intervalAttrs := map[string]schema.Attribute{
		"name": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Name of the interval, up to 100 characters.",
			Validators:          []validator.String{stringvalidator.LengthBetween(1, 100)},
		},
	}
	for _, d := range []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"} {
		dayBlocks[d] = dayBlock(strings.ToUpper(d[:1]) + d[1:])
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: `Manages the team hours of a Grafana IRM (OnCall) team.

A team has up to three named intervals, each with up to three time ranges per weekday. The team is within its hours when the current time falls in any range of any interval. Use them with the ` + "`pause_until_team_hours`" + ` step of the ` + "`grafana_oncall_escalation`" + ` resource.

Times are read in the team's timezone, which cannot be set with this resource: it is set in **IRM** > **Settings** > **Teams** and defaults to UTC.

Destroying this resource removes all of the team's intervals, so IRM treats the team as always within its hours.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"team_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the OnCall team, for example from the `grafana_oncall_team` data source.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"timezone": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The team's timezone, as set in IRM. Empty when unset, in which case UTC is used.",
			},
		},
		Blocks: map[string]schema.Block{
			"interval": schema.ListNestedBlock{
				MarkdownDescription: "A named set of weekly hours. Up to three. Each needs at least one time range.",
				Validators: []validator.List{
					listvalidator.IsRequired(),
					listvalidator.SizeBetween(1, maxTeamHoursIntervals),
				},
				NestedObject: schema.NestedBlockObject{Attributes: intervalAttrs, Blocks: dayBlocks},
			},
		},
	}
}

func (r *teamHoursResource) apply(plan *teamHoursResourceModel, intervals []onCallAPI.TeamHoursInterval) error {
	out, _, err := r.client.TeamHours.SetTeamHours(&onCallAPI.SetTeamHoursOptions{
		TeamID:    plan.TeamID.ValueString(),
		Intervals: intervals,
	})
	if err != nil {
		return err
	}
	plan.ID = plan.TeamID
	plan.Timezone = types.StringPointerValue(out.Timezone)
	if plan.Timezone.IsNull() {
		plan.Timezone = types.StringValue("")
	}
	return nil
}

func (r *teamHoursResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan teamHoursResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.apply(&plan, intervalsToAPI(plan.Intervals)); err != nil {
		resp.Diagnostics.AddError("Failed to set team hours", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *teamHoursResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan teamHoursResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.apply(&plan, intervalsToAPI(plan.Intervals)); err != nil {
		resp.Diagnostics.AddError("Failed to set team hours", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *teamHoursResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state teamHoursResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hours, _, err := r.client.TeamHours.GetTeamHours(state.TeamID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read team hours", err.Error())
		return
	}
	if hours == nil || len(hours.Intervals) == 0 {
		resp.State.RemoveResource(ctx)
		return
	}

	state.ID = state.TeamID
	state.Timezone = types.StringPointerValue(hours.Timezone)
	if state.Timezone.IsNull() {
		state.Timezone = types.StringValue("")
	}
	state.Intervals = intervalsFromAPI(hours.Intervals, state.Intervals)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *teamHoursResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state teamHoursResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// If the team is gone (or already has no hours) there is nothing to clear. Check first:
	// clearing POSTs team_id, and the backend answers an unknown team with a 400, not a 404.
	hours, _, err := r.client.TeamHours.GetTeamHours(state.TeamID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read team hours before clearing", err.Error())
		return
	}
	if hours == nil || len(hours.Intervals) == 0 {
		return
	}
	// There is no delete endpoint: replacing the intervals with none clears them.
	if err := r.apply(&state, []onCallAPI.TeamHoursInterval{}); err != nil {
		resp.Diagnostics.AddError("Failed to clear team hours", err.Error())
	}
}

func (r *teamHoursResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("team_id"), req.ID)...)
}
