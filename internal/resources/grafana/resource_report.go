package grafana

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/go-openapi/strfmt"
	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	reportFrequencyHourly  = "hourly"
	reportFrequencyDaily   = "daily"
	reportFrequencyWeekly  = "weekly"
	reportFrequencyMonthly = "monthly"
	reportFrequencyCustom  = "custom"
	reportFrequencyOnce    = "once"
	reportFrequencyNever   = "never"

	reportOrientationPortrait  = "portrait"
	reportOrientationLandscape = "landscape"

	reportLayoutGrid   = "grid"
	reportLayoutSimple = "simple"

	reportFormatPDF   = "pdf"
	reportFormatCSV   = "csv"
	reportFormatImage = "image"

	timeDateShortFormat = "2006-01-02T15:04:05"
)

var (
	reportLayouts      = []string{reportLayoutSimple, reportLayoutGrid}
	reportOrientations = []string{reportOrientationLandscape, reportOrientationPortrait}
	reportFrequencies  = []string{reportFrequencyNever, reportFrequencyOnce, reportFrequencyHourly, reportFrequencyDaily, reportFrequencyWeekly, reportFrequencyMonthly, reportFrequencyCustom}
	reportFormats      = []string{reportFormatPDF, reportFormatCSV, reportFormatImage}

	_ resource.Resource                = &reportResource{}
	_ resource.ResourceWithConfigure   = &reportResource{}
	_ resource.ResourceWithImportState = &reportResource{}
	_ resource.ResourceWithModifyPlan  = &reportResource{}
)

type resourceReportTimeRangeModel struct {
	From types.String `tfsdk:"from"`
	To   types.String `tfsdk:"to"`
}

type resourceReportDashboardModel struct {
	UID             types.String                   `tfsdk:"uid"`
	TimeRange       []resourceReportTimeRangeModel `tfsdk:"time_range"`
	ReportVariables types.Map                      `tfsdk:"report_variables"`
}

type resourceReportScheduleModel struct {
	Frequency      types.String `tfsdk:"frequency"`
	StartTime      types.String `tfsdk:"start_time"`
	EndTime        types.String `tfsdk:"end_time"`
	WorkdaysOnly   types.Bool   `tfsdk:"workdays_only"`
	CustomInterval types.String `tfsdk:"custom_interval"`
	LastDayOfMonth types.Bool   `tfsdk:"last_day_of_month"`
	Timezone       types.String `tfsdk:"timezone"`
}

type resourceReportModel struct {
	ID                   types.String                   `tfsdk:"id"`
	OrgID                types.String                   `tfsdk:"org_id"`
	Name                 types.String                   `tfsdk:"name"`
	Recipients           types.List                     `tfsdk:"recipients"`
	ReplyTo              types.String                   `tfsdk:"reply_to"`
	Message              types.String                   `tfsdk:"message"`
	IncludeDashboardLink types.Bool                     `tfsdk:"include_dashboard_link"`
	IncludeTableCSV      types.Bool                     `tfsdk:"include_table_csv"`
	Layout               types.String                   `tfsdk:"layout"`
	Orientation          types.String                   `tfsdk:"orientation"`
	Formats              types.Set                      `tfsdk:"formats"`
	Schedule             []resourceReportScheduleModel  `tfsdk:"schedule"`
	Dashboards           []resourceReportDashboardModel `tfsdk:"dashboards"`
}

type reportResource struct {
	basePluginFrameworkResource
}

func makeResourceReport() *common.Resource {
	return common.NewResource(
		common.CategoryGrafanaEnterprise,
		"grafana_report",
		orgResourceIDInt("id"),
		&reportResource{},
	).
		WithLister(listerFunctionOrgResource(listReports)).
		WithPreferredResourceNameField("name")
}

func listReports(ctx context.Context, client *goapi.GrafanaHTTPAPI, orgID int64) ([]string, error) {
	var ids []string
	resp, err := client.Reports.GetReports()
	if err != nil && common.IsNotFoundError(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	for _, report := range resp.Payload {
		ids = append(ids, MakeOrgResourceID(orgID, report.ID))
	}
	return ids, nil
}

func (r *reportResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "grafana_report"
}

func (r *reportResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
**Note:** This resource is available only with Grafana Enterprise 7.+.

* [Official documentation](https://grafana.com/docs/grafana/latest/dashboards/create-reports/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/reporting/)
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Generated identifier of the report.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"org_id": pluginFrameworkOrgIDAttribute(),
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the report.",
			},
			"recipients": schema.ListAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "List of recipients of the report.",
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					listvalidator.ValueStringsAre(
						stringvalidator.RegexMatches(common.EmailRegexp, "must be an email address"),
					),
				},
			},
			"reply_to": schema.StringAttribute{
				Optional:    true,
				Description: "Reply-to email address of the report.",
				Validators: []validator.String{
					stringvalidator.RegexMatches(common.EmailRegexp, "must be an email address"),
				},
			},
			"message": schema.StringAttribute{
				Optional:    true,
				Description: "Message to be sent in the report.",
			},
			"include_dashboard_link": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether to include a link to the dashboard in the report.",
			},
			"include_table_csv": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether to include a CSV file of table panel data.",
			},
			"layout": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(reportLayoutGrid),
				Description: common.AllowedValuesDescription("Layout of the report", reportLayouts),
				Validators: []validator.String{
					stringvalidator.OneOf(reportLayouts...),
				},
			},
			"orientation": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(reportOrientationLandscape),
				Description: common.AllowedValuesDescription("Orientation of the report", reportOrientations),
				Validators: []validator.String{
					stringvalidator.OneOf(reportOrientations...),
				},
			},
			"formats": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: common.AllowedValuesDescription("Specifies what kind of attachment to generate for the report", reportFormats),
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(
						stringvalidator.OneOf(reportFormats...),
					),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"schedule": schema.ListNestedBlock{
				Description: "(Required) Schedule of the report.",
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					listvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"frequency": schema.StringAttribute{
							Required:    true,
							Description: common.AllowedValuesDescription("Frequency of the report", reportFrequencies),
							Validators: []validator.String{
								stringvalidator.OneOf(reportFrequencies...),
							},
						},
						"start_time": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: fmt.Sprintf("Start time of the report. If empty, the start date will be set to the creation time. Note that times will be saved as UTC in Grafana. Use %s format if you want to set a custom timezone", timeDateShortFormat),
							Validators:  []validator.String{dateStringValidator{}},
							PlanModifiers: []planmodifier.String{
								startTimePlanModifier{},
							},
						},
						"end_time": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: fmt.Sprintf("End time of the report. If empty, the report will be sent indefinitely (according to frequency). Note that times will be saved as UTC in Grafana. Use %s format if you want to set a custom timezone", timeDateShortFormat),
							Validators:  []validator.String{dateStringValidator{}},
							PlanModifiers: []planmodifier.String{
								endTimePlanModifier{},
							},
						},
						"workdays_only": schema.BoolAttribute{
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(false),
							Description: "Whether to send the report only on work days.",
						},
						"custom_interval": schema.StringAttribute{
							Optional: true,
							Description: "Custom interval of the report.\n" +
								"**Note:** This field is only available when frequency is set to `custom`.",
							Validators: []validator.String{customIntervalValidator{}},
						},
						"last_day_of_month": schema.BoolAttribute{
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(false),
							Description: "Send the report on the last day of the month",
						},
						"timezone": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString("GMT"),
							Description: "Set the report time zone.",
							Validators:  []validator.String{timezoneValidator{}},
						},
					},
				},
			},
			"dashboards": schema.ListNestedBlock{
				Description: "List of dashboards to render into the report",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"uid": schema.StringAttribute{
							Required:    true,
							Description: "Dashboard uid.",
						},
						"report_variables": schema.MapAttribute{
							Optional:    true,
							ElementType: types.StringType,
							Description: "Add report variables to the dashboard. Values should be separated by commas.",
						},
					},
					Blocks: map[string]schema.Block{
						"time_range": schema.ListNestedBlock{
							Description: "Time range of the report.",
							Validators: []validator.List{
								listvalidator.SizeAtMost(1),
							},
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"from": schema.StringAttribute{
										Optional:    true,
										Description: "Start of the time range.",
									},
									"to": schema.StringAttribute{
										Optional:    true,
										Description: "End of the time range.",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *reportResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	var plan resourceReportModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || len(plan.Schedule) == 0 {
		return
	}

	modified := false
	schedule := &plan.Schedule[0]
	frequency := schedule.Frequency.ValueString()

	if !reportWorkdaysOnlyConfigAllowed(frequency) {
		schedule.WorkdaysOnly = types.BoolValue(false)
		modified = true
	}

	if modified {
		resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
	}
}

func (r *reportResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	data, diags := r.read(ctx, req.ID, true)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if data == nil {
		resp.Diagnostics.AddError("Resource not found", fmt.Sprintf("report %q not found", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *reportResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resourceReportModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, orgID, err := r.clientFromNewOrgResource(data.OrgID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get client", err.Error())
		return
	}

	report, diags := modelToReport(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	res, err := client.Reports.CreateReport(&report)
	if err != nil {
		payload, _ := json.Marshal(report)
		resp.Diagnostics.AddError("Failed to create report", fmt.Sprintf("error creating the following report:\n%s\n%v", string(payload), err))
		return
	}

	data.ID = types.StringValue(MakeOrgResourceID(orgID, res.Payload.ID))
	readData, diags := r.read(ctx, data.ID.ValueString(), !data.Formats.IsNull())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	preserveScheduleTimes(readData, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *reportResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resourceReportModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readData, diags := r.read(ctx, data.ID.ValueString(), !data.Formats.IsNull())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if readData == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	// Preserve the user-provided time format from the existing state only when the API
	// value represents the same instant. The v5 mux requires plan values for
	// Optional+Computed attributes in blocks to match config exactly (no plan modifier
	// allowed), so we avoid format-only diffs while still surfacing genuine time changes.
	preserveScheduleTimesIfSemanticEqual(readData, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *reportResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resourceReportModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, orgID, split, err := r.clientFromExistingOrgResource(orgResourceIDInt("id"), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse resource ID", err.Error())
		return
	}
	reportID := split[0].(int64)

	report, diags := modelToReport(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := client.Reports.UpdateReport(reportID, &report); err != nil {
		payload, _ := json.Marshal(report)
		resp.Diagnostics.AddError("Failed to update report", fmt.Sprintf("error updating the following report:\n%s\n%v", string(payload), err))
		return
	}

	data.ID = types.StringValue(MakeOrgResourceID(orgID, reportID))
	readData, diags := r.read(ctx, data.ID.ValueString(), !data.Formats.IsNull())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	preserveScheduleTimes(readData, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *reportResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resourceReportModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, _, split, err := r.clientFromExistingOrgResource(orgResourceIDInt("id"), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse resource ID", err.Error())
		return
	}
	reportID := split[0].(int64)

	_, err = client.Reports.DeleteReport(reportID)
	if err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Failed to delete report", err.Error())
	}
}

func (r *reportResource) read(ctx context.Context, id string, preserveFormats bool) (*resourceReportModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	client, orgID, split, err := r.clientFromExistingOrgResource(orgResourceIDInt("id"), id)
	if err != nil {
		diags.AddError("Failed to parse resource ID", err.Error())
		return nil, diags
	}
	reportID := split[0].(int64)

	resp, err := client.Reports.GetReport(reportID)
	if err != nil {
		if common.IsNotFoundError(err) {
			return nil, diags
		}
		diags.AddError("Failed to read report", err.Error())
		return nil, diags
	}
	p := resp.Payload

	recipients, recipientDiags := types.ListValueFrom(ctx, types.StringType, strings.Split(p.Recipients, ","))
	diags.Append(recipientDiags...)
	if diags.HasError() {
		return nil, diags
	}

	schedule := resourceReportScheduleModel{
		Frequency:    types.StringValue(p.Schedule.Frequency),
		WorkdaysOnly: types.BoolValue(p.Schedule.WorkdaysOnly),
		Timezone:     types.StringValue(p.Schedule.TimeZone),
		StartTime:    types.StringValue(""),
		EndTime:      types.StringValue(""),
	}
	if p.Schedule.IntervalAmount != 0 && p.Schedule.IntervalFrequency != "" {
		schedule.CustomInterval = types.StringValue(fmt.Sprintf("%d %s", p.Schedule.IntervalAmount, p.Schedule.IntervalFrequency))
	} else {
		schedule.CustomInterval = types.StringNull()
	}
	if p.Schedule.StartDate != nil {
		schedule.StartTime = types.StringValue(time.Time(*p.Schedule.StartDate).Format(time.RFC3339))
	}
	if p.Schedule.EndDate != nil {
		schedule.EndTime = types.StringValue(time.Time(*p.Schedule.EndDate).Format(time.RFC3339))
	}
	schedule.LastDayOfMonth = types.BoolValue(p.Schedule.DayOfMonth == "last")

	dashboards := make([]resourceReportDashboardModel, len(p.Dashboards))
	for i, d := range p.Dashboards {
		var timeRange []resourceReportTimeRangeModel
		if d.TimeRange != nil && (d.TimeRange.From != "" || d.TimeRange.To != "") {
			timeRange = []resourceReportTimeRangeModel{{
				From: types.StringValue(d.TimeRange.From),
				To:   types.StringValue(d.TimeRange.To),
			}}
		}

		rvMap := make(map[string]string)
		if rvRaw, ok := d.ReportVariables.(map[string]any); ok {
			for k, v := range rvRaw {
				if vals, ok := v.([]any); ok {
					strs := make([]string, len(vals))
					for j, val := range vals {
						strs[j] = fmt.Sprint(val)
					}
					rvMap[k] = strings.Join(strs, ",")
				}
			}
		}
		var rvTypes types.Map
		if len(rvMap) > 0 {
			var rvDiags diag.Diagnostics
			rvTypes, rvDiags = types.MapValueFrom(ctx, types.StringType, rvMap)
			diags.Append(rvDiags...)
			if diags.HasError() {
				return nil, diags
			}
		} else {
			rvTypes = types.MapNull(types.StringType)
		}

		dashboards[i] = resourceReportDashboardModel{
			UID:             types.StringValue(d.Dashboard.UID),
			TimeRange:       timeRange,
			ReportVariables: rvTypes,
		}
	}

	data := &resourceReportModel{
		ID:                   types.StringValue(MakeOrgResourceID(orgID, reportID)),
		OrgID:                types.StringValue(strconv.FormatInt(orgID, 10)),
		Name:                 types.StringValue(p.Name),
		Recipients:           recipients,
		ReplyTo:              nullableString(p.ReplyTo),
		Message:              nullableString(p.Message),
		IncludeDashboardLink: types.BoolValue(p.EnableDashboardURL),
		IncludeTableCSV:      types.BoolValue(p.EnableCSV),
		Layout:               types.StringValue(p.Options.Layout),
		Orientation:          types.StringValue(p.Options.Orientation),
		Schedule:             []resourceReportScheduleModel{schedule},
		Dashboards:           dashboards,
	}

	if preserveFormats {
		formatStrs := make([]string, len(p.Formats))
		for i, f := range p.Formats {
			formatStrs[i] = string(f)
		}
		formatsVal, formatDiags := types.SetValueFrom(ctx, types.StringType, formatStrs)
		diags.Append(formatDiags...)
		if diags.HasError() {
			return nil, diags
		}
		data.Formats = formatsVal
	} else {
		data.Formats = types.SetNull(types.StringType)
	}

	return data, diags
}

func modelToReport(ctx context.Context, data *resourceReportModel) (models.CreateOrUpdateReport, diag.Diagnostics) {
	var diags diag.Diagnostics

	var recipients []string
	diags.Append(data.Recipients.ElementsAs(ctx, &recipients, false)...)
	if diags.HasError() {
		return models.CreateOrUpdateReport{}, diags
	}

	if len(data.Schedule) == 0 {
		diags.AddError("Missing schedule", "report must have exactly one schedule block")
		return models.CreateOrUpdateReport{}, diags
	}
	schedule := data.Schedule[0]
	frequency := schedule.Frequency.ValueString()
	timezone := schedule.Timezone.ValueString()

	report := models.CreateOrUpdateReport{
		Name:               data.Name.ValueString(),
		Recipients:         strings.Join(recipients, ","),
		ReplyTo:            data.ReplyTo.ValueString(),
		Message:            data.Message.ValueString(),
		EnableDashboardURL: data.IncludeDashboardLink.ValueBool(),
		EnableCSV:          data.IncludeTableCSV.ValueBool(),
		Options: &models.ReportOptions{
			Layout:      data.Layout.ValueString(),
			Orientation: data.Orientation.ValueString(),
		},
		Schedule: &models.ReportSchedule{
			Frequency: frequency,
			TimeZone:  timezone,
		},
		Formats: []models.Type{reportFormatPDF},
	}

	if !data.Formats.IsNull() && len(data.Formats.Elements()) > 0 {
		var formatStrs []string
		diags.Append(data.Formats.ElementsAs(ctx, &formatStrs, false)...)
		if diags.HasError() {
			return models.CreateOrUpdateReport{}, diags
		}
		report.Formats = []models.Type{}
		for _, f := range formatStrs {
			report.Formats = append(report.Formats, models.Type(f))
		}
	}

	for _, d := range data.Dashboards {
		tr := &models.ReportTimeRange{}
		if len(d.TimeRange) > 0 {
			tr = &models.ReportTimeRange{
				From: d.TimeRange[0].From.ValueString(),
				To:   d.TimeRange[0].To.ValueString(),
			}
		}

		var rvMap map[string]string
		if !d.ReportVariables.IsNull() {
			diags.Append(d.ReportVariables.ElementsAs(ctx, &rvMap, false)...)
			if diags.HasError() {
				return models.CreateOrUpdateReport{}, diags
			}
		}
		rvForAPI := make(map[string][]string, len(rvMap))
		for k, v := range rvMap {
			rvForAPI[k] = strings.Split(v, ",")
		}

		report.Dashboards = append(report.Dashboards, &models.ReportDashboard{
			Dashboard:       &models.ReportDashboardID{UID: d.UID.ValueString()},
			TimeRange:       tr,
			ReportVariables: rvForAPI,
		})
	}

	location, err := time.LoadLocation(timezone)
	if err != nil {
		diags.AddError("Invalid timezone", err.Error())
		return models.CreateOrUpdateReport{}, diags
	}

	if frequency != reportFrequencyNever {
		if s := schedule.StartTime.ValueString(); s != "" {
			date, err := formatDate(s, location)
			if err != nil {
				diags.AddError("Invalid start_time", err.Error())
				return models.CreateOrUpdateReport{}, diags
			}
			report.Schedule.StartDate = date
		}
	}

	if frequency != reportFrequencyOnce && frequency != reportFrequencyNever {
		if s := schedule.EndTime.ValueString(); s != "" {
			date, err := formatDate(s, location)
			if err != nil {
				diags.AddError("Invalid end_time", err.Error())
				return models.CreateOrUpdateReport{}, diags
			}
			report.Schedule.EndDate = date
		}
	}

	if frequency == reportFrequencyMonthly && schedule.LastDayOfMonth.ValueBool() {
		report.Schedule.DayOfMonth = "last"
	}

	if reportWorkdaysOnlyConfigAllowed(frequency) {
		report.Schedule.WorkdaysOnly = schedule.WorkdaysOnly.ValueBool()
	}

	if frequency == reportFrequencyCustom {
		amount, unit, err := parseCustomReportInterval(schedule.CustomInterval.ValueString())
		if err != nil {
			diags.AddError("Invalid custom_interval", err.Error())
			return models.CreateOrUpdateReport{}, diags
		}
		report.Schedule.IntervalAmount = int64(amount)
		report.Schedule.IntervalFrequency = unit
	}

	return report, diags
}

func reportWorkdaysOnlyConfigAllowed(frequency string) bool {
	return frequency == reportFrequencyHourly || frequency == reportFrequencyDaily || frequency == reportFrequencyCustom
}

func parseCustomReportInterval(s string) (int, string, error) {
	parseErr := errors.New("custom_interval must be in format `<number> <unit>` where unit is one of `hours`, `days`, `weeks`, `months`")
	split := strings.Split(s, " ")
	if len(split) != 2 {
		return 0, "", parseErr
	}
	number, err := strconv.Atoi(split[0])
	if err != nil {
		return 0, "", parseErr
	}
	unit := split[1]
	if unit != "hours" && unit != "days" && unit != "weeks" && unit != "months" {
		return 0, "", parseErr
	}
	return number, unit, nil
}

func formatDate(date string, timezone *time.Location) (*strfmt.DateTime, error) {
	parsedDate, err := time.ParseInLocation(timeDateShortFormat, date, timezone)
	if err != nil {
		return CheckTimezoneFormatDate(date, timezone)
	}
	dateTime := strfmt.DateTime(parsedDate)
	return &dateTime, nil
}

// CheckTimezoneFormatDate is exported for testing purposes.
func CheckTimezoneFormatDate(date string, timezone *time.Location) (*strfmt.DateTime, error) {
	parsedDate, err := time.Parse(time.RFC3339, date)
	if err != nil {
		return nil, err
	}
	dateTime := strfmt.DateTime(parsedDate.In(timezone))
	return &dateTime, nil
}

// preserveScheduleTimes copies start_time and end_time from src (plan/config) into dst
// (API-read data) when src has a non-empty value. Used after Create and Update to keep the
// user-provided format in state.
//
// The Terraform v5 mux requires that plan values for Optional+Computed attributes inside
// ListNestedBlocks match the config value byte-for-byte (no plan modifier can change them).
// Because of this, we cannot normalize times in the plan the way SDKv2's DiffSuppressFunc did.
// Instead, we store whatever the user wrote in config and avoid surfacing the API-normalized
// form into state. When config omits a time (empty string), we keep the API value so that
// provider-assigned times (e.g. "start now" for a new report) are still reflected.
func preserveScheduleTimes(dst, src *resourceReportModel) {
	if dst == nil || src == nil || len(dst.Schedule) == 0 || len(src.Schedule) == 0 {
		return
	}
	if v := src.Schedule[0].StartTime.ValueString(); v != "" {
		dst.Schedule[0].StartTime = src.Schedule[0].StartTime
	}
	if v := src.Schedule[0].EndTime.ValueString(); v != "" {
		dst.Schedule[0].EndTime = src.Schedule[0].EndTime
	}
}

// preserveScheduleTimesIfSemanticEqual is like preserveScheduleTimes but only copies a time
// from src (prior state) into dst (fresh API data) when both values represent the same instant.
// Used during Read so that format-only differences do not produce spurious diffs, while genuine
// out-of-band schedule changes made directly in Grafana are still surfaced.
func preserveScheduleTimesIfSemanticEqual(dst, src *resourceReportModel) {
	if dst == nil || src == nil || len(dst.Schedule) == 0 || len(src.Schedule) == 0 {
		return
	}
	loc, err := time.LoadLocation(dst.Schedule[0].Timezone.ValueString())
	if err != nil {
		loc = time.UTC
	}
	if v := src.Schedule[0].StartTime.ValueString(); v != "" && scheduleTimeSemanticEqual(v, dst.Schedule[0].StartTime.ValueString(), loc) {
		dst.Schedule[0].StartTime = src.Schedule[0].StartTime
	}
	if v := src.Schedule[0].EndTime.ValueString(); v != "" && scheduleTimeSemanticEqual(v, dst.Schedule[0].EndTime.ValueString(), loc) {
		dst.Schedule[0].EndTime = src.Schedule[0].EndTime
	}
}

// scheduleTimeSemanticEqual reports whether two time strings represent the same instant.
// RFC3339 strings carry their own offset and are parsed as-is. Short-format strings
// (timeDateShortFormat, no timezone indicator) are interpreted in loc, matching the
// behaviour of formatDate which also uses ParseInLocation for short-format input.
func scheduleTimeSemanticEqual(a, b string, loc *time.Location) bool {
	if a == b {
		return true
	}
	parseTime := func(s string) (time.Time, bool) {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			return t, true
		}
		if t, err := time.ParseInLocation(timeDateShortFormat, s, loc); err == nil {
			return t, true
		}
		return time.Time{}, false
	}
	ta, ok := parseTime(a)
	if !ok {
		return false
	}
	tb, ok := parseTime(b)
	if !ok {
		return false
	}
	return ta.Equal(tb)
}

// nullableString returns types.StringNull() for empty strings, types.StringValue(s) otherwise.
// Used for Optional-only fields where the API returns "" but Terraform expects null.
func nullableString(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}

// Plan modifiers

type startTimePlanModifier struct{}

func (m startTimePlanModifier) Description(_ context.Context) string {
	return "Suppresses diffs when start times are semantically equal or when the old time is in the past and no new time is set."
}
func (m startTimePlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}
func (m startTimePlanModifier) PlanModifyString(_ context.Context, _ planmodifier.StringRequest, _ *planmodifier.StringResponse) {
	// The v5 mux requires that plan values for Optional+Computed attributes inside blocks
	// exactly match the config value — no modification (not even to Unknown) is permitted.
	// Time normalization is handled by preserveScheduleTimes in Create/Update/Read instead.
}

type endTimePlanModifier struct{}

func (m endTimePlanModifier) Description(_ context.Context) string {
	return "Suppresses diffs when end times are semantically equal."
}
func (m endTimePlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}
func (m endTimePlanModifier) PlanModifyString(_ context.Context, _ planmodifier.StringRequest, _ *planmodifier.StringResponse) {
	// See startTimePlanModifier for explanation.
}

// Validators

type dateStringValidator struct{}

func (v dateStringValidator) Description(_ context.Context) string {
	return fmt.Sprintf("value must be in %s or %s format", time.RFC3339, timeDateShortFormat)
}
func (v dateStringValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}
func (v dateStringValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	s := req.ConfigValue.ValueString()
	if s == "" {
		return
	}
	_, errRFC := time.Parse(time.RFC3339, s)
	_, errShort := time.Parse(timeDateShortFormat, s)
	if errRFC != nil && errShort != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid date format", v.Description(context.Background()))
	}
}

type timezoneValidator struct{}

func (v timezoneValidator) Description(_ context.Context) string {
	return "value must be a valid timezone"
}
func (v timezoneValidator) MarkdownDescription(ctx context.Context) string { return v.Description(ctx) }
func (v timezoneValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if _, err := time.LoadLocation(req.ConfigValue.ValueString()); err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid timezone", err.Error())
	}
}

type customIntervalValidator struct{}

func (v customIntervalValidator) Description(_ context.Context) string {
	return "value must be in format `<number> <unit>` where unit is one of `hours`, `days`, `weeks`, `months`"
}
func (v customIntervalValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}
func (v customIntervalValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if _, _, err := parseCustomReportInterval(req.ConfigValue.ValueString()); err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid custom_interval", err.Error())
	}
}
