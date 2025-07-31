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
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
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
)

func resourceReport() *common.Resource {
	schema := &schema.Resource{
		Description: `
**Note:** This resource is available only with Grafana Enterprise 7.+.

* [Official documentation](https://grafana.com/docs/grafana/latest/dashboards/create-reports/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/reporting/)
`,
		CreateContext: CreateReport,
		UpdateContext: UpdateReport,
		ReadContext:   ReadReport,
		DeleteContext: DeleteReport,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"org_id": orgIDAttribute(),
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Generated identifier of the report.",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the report.",
			},
			"recipients": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "List of recipients of the report.",
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringMatch(common.EmailRegexp, "must be an email address"),
				},
				MinItems: 1,
			},
			"reply_to": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "Reply-to email address of the report.",
				ValidateFunc: validation.StringMatch(common.EmailRegexp, "must be an email address"),
			},
			"message": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Message to be sent in the report.",
			},
			"include_dashboard_link": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Whether to include a link to the dashboard in the report.",
			},
			"include_table_csv": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether to include a CSV file of table panel data.",
			},
			"layout": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  common.AllowedValuesDescription("Layout of the report", reportLayouts),
				Default:      reportLayoutGrid,
				ValidateFunc: validation.StringInSlice(reportLayouts, false),
			},
			"orientation": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  common.AllowedValuesDescription("Orientation of the report", reportOrientations),
				Default:      reportOrientationLandscape,
				ValidateFunc: validation.StringInSlice(reportOrientations, false),
			},
			"formats": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: common.AllowedValuesDescription("Specifies what kind of attachment to generate for the report", reportFormats),
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringInSlice(reportFormats, false),
				},
			},
			"schedule": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "Schedule of the report.",
				MinItems:    1,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"frequency": {
							Type:         schema.TypeString,
							Required:     true,
							Description:  common.AllowedValuesDescription("Frequency of the report", reportFrequencies),
							ValidateFunc: validation.StringInSlice(reportFrequencies, false),
						},
						"start_time": {
							Type:             schema.TypeString,
							Optional:         true,
							Description:      fmt.Sprintf("Start time of the report. If empty, the start date will be set to the creation time. Note that times will be saved as UTC in Grafana. Use %s format if you want to set a custom timezone", timeDateShortFormat),
							ValidateDiagFunc: validateDate,
							DiffSuppressFunc: checkStartTimeDiff,
						},
						"end_time": {
							Type:             schema.TypeString,
							Optional:         true,
							Description:      fmt.Sprintf("End time of the report. If empty, the report will be sent indefinitely (according to frequency). Note that times will be saved as UTC in Grafana. Use %s format if you want to set a custom timezone", timeDateShortFormat),
							ValidateDiagFunc: validateDate,
							DiffSuppressFunc: checkEndTimeDiff,
						},
						"workdays_only": {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "Whether to send the report only on work days.",
							Default:     false,
							DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
								return !reportWorkdaysOnlyConfigAllowed(d.Get("schedule.0.frequency").(string))
							},
						},
						"custom_interval": {
							Type:     schema.TypeString,
							Optional: true,
							Description: "Custom interval of the report.\n" +
								"**Note:** This field is only available when frequency is set to `custom`.",
							ValidateDiagFunc: func(i interface{}, p cty.Path) diag.Diagnostics {
								_, _, err := parseCustomReportInterval(i)
								return diag.FromErr(err)
							},
						},
						"last_day_of_month": {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "Send the report on the last day of the month",
							Default:     false,
						},
						"timezone": {
							Type:             schema.TypeString,
							Optional:         true,
							Description:      "Set the report time zone.",
							Default:          "GMT",
							ValidateDiagFunc: validateTimezone,
						},
					},
				},
			},
			"dashboards": {
				Type:        schema.TypeList,
				Description: "List of dashboards to render into the report",
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"uid": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Dashboard uid.",
						},
						"time_range": {
							Type:        schema.TypeList,
							MinItems:    1,
							MaxItems:    1,
							Optional:    true,
							Description: "Time range of the report.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"from": {
										Type:        schema.TypeString,
										Optional:    true,
										Description: "Start of the time range.",
									},
									"to": {
										Type:        schema.TypeString,
										Optional:    true,
										Description: "End of the time range.",
									},
								},
							},
							DiffSuppressFunc: func(k, oldValue, newValue string, d *schema.ResourceData) bool {
								return oldValue == "1" && newValue == "0"
							},
						},
						"report_variables": {
							Type:             schema.TypeMap,
							Description:      "Add report variables to the dashboard. Values should be separated by commas.",
							Optional:         true,
							Elem:             schema.TypeString,
							ValidateDiagFunc: validateReportVariables,
						},
					},
				},
			},
		},
	}

	return common.NewLegacySDKResource(
		common.CategoryGrafanaEnterprise,
		"grafana_report",
		orgResourceIDInt("id"),
		schema,
	).
		WithLister(listerFunctionOrgResource(listReports)).
		WithPreferredResourceNameField("name")
}

func listReports(ctx context.Context, client *goapi.GrafanaHTTPAPI, orgID int64) ([]string, error) {
	var ids []string
	resp, err := client.Reports.GetReports()
	if err != nil && common.IsNotFoundError(err) {
		return nil, nil // Reports are not available in the current Grafana version (Probably OSS)
	}
	if err != nil {
		return nil, err
	}

	for _, report := range resp.Payload {
		ids = append(ids, MakeOrgResourceID(orgID, report.ID))
	}

	return ids, nil
}

func CreateReport(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID := OAPIClientFromNewOrgResource(meta, d)

	report, err := schemaToReport(d)
	if err != nil {
		return diag.FromErr(err)
	}

	res, err := client.Reports.CreateReport(&report)
	if err != nil {
		data, _ := json.Marshal(report)
		return diag.Errorf("error creating the following report:\n%s\n%v", string(data), err)
	}

	d.SetId(MakeOrgResourceID(orgID, res.Payload.ID))
	return ReadReport(ctx, d, meta)
}

func ReadReport(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, idStr := OAPIClientFromExistingOrgResource(meta, d.Id())
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	r, err := client.Reports.GetReport(id)
	if err, shouldReturn := common.CheckReadError("report", d, err); shouldReturn {
		return err
	}

	d.SetId(MakeOrgResourceID(r.Payload.OrgID, id))
	d.Set("name", r.Payload.Name)
	d.Set("recipients", strings.Split(r.Payload.Recipients, ","))
	d.Set("reply_to", r.Payload.ReplyTo)
	d.Set("message", r.Payload.Message)
	d.Set("include_dashboard_link", r.Payload.EnableDashboardURL)
	d.Set("include_table_csv", r.Payload.EnableCSV)
	d.Set("layout", r.Payload.Options.Layout)
	d.Set("orientation", r.Payload.Options.Orientation)
	d.Set("org_id", strconv.FormatInt(r.Payload.OrgID, 10))

	if _, ok := d.GetOk("formats"); ok {
		formats := make([]string, len(r.Payload.Formats))
		for i, format := range r.Payload.Formats {
			formats[i] = string(format)
		}
		d.Set("formats", common.StringSliceToSet(formats))
	}

	schedule := map[string]interface{}{
		"frequency":     r.Payload.Schedule.Frequency,
		"workdays_only": r.Payload.Schedule.WorkdaysOnly,
		"timezone":      r.Payload.Schedule.TimeZone,
	}
	if r.Payload.Schedule.IntervalAmount != 0 && r.Payload.Schedule.IntervalFrequency != "" {
		schedule["custom_interval"] = fmt.Sprintf("%d %s", r.Payload.Schedule.IntervalAmount, r.Payload.Schedule.IntervalFrequency)
	}

	if r.Payload.Schedule.StartDate != nil {
		strfmt.MarshalFormat = time.RFC3339
		schedule["start_time"] = r.Payload.Schedule.StartDate.String()
	}
	if r.Payload.Schedule.EndDate != nil {
		strfmt.MarshalFormat = time.RFC3339
		schedule["end_time"] = r.Payload.Schedule.EndDate.String()
	}
	if r.Payload.Schedule.DayOfMonth == "last" {
		schedule["last_day_of_month"] = true
	}

	d.Set("schedule", []interface{}{schedule})

	dashboards := make([]interface{}, len(r.Payload.Dashboards))
	for i, dashboard := range r.Payload.Dashboards {
		dashboards[i] = map[string]interface{}{
			"uid": dashboard.Dashboard.UID,
			"time_range": []interface{}{
				map[string]interface{}{
					"to":   dashboard.TimeRange.To,
					"from": dashboard.TimeRange.From,
				},
			},
			"report_variables": parseReportVariablesResponse(dashboard.ReportVariables),
		}
	}

	d.Set("dashboards", dashboards)

	return nil
}

func UpdateReport(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, idStr := OAPIClientFromExistingOrgResource(meta, d.Id())
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	report, err := schemaToReport(d)
	if err != nil {
		return diag.FromErr(err)
	}

	if _, err := client.Reports.UpdateReport(id, &report); err != nil {
		data, _ := json.Marshal(report)
		return diag.Errorf("error updating the following report:\n%s\n%v", string(data), err)
	}
	return ReadReport(ctx, d, meta)
}

func DeleteReport(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, idStr := OAPIClientFromExistingOrgResource(meta, d.Id())
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = client.Reports.DeleteReport(id)
	diag, _ := common.CheckReadError("report", d, err)
	return diag
}

func schemaToReport(d *schema.ResourceData) (models.CreateOrUpdateReport, error) {
	frequency := d.Get("schedule.0.frequency").(string)
	timezone := d.Get("schedule.0.timezone").(string)
	report := models.CreateOrUpdateReport{
		Name:               d.Get("name").(string),
		Recipients:         strings.Join(common.ListToStringSlice(d.Get("recipients").([]interface{})), ","),
		ReplyTo:            d.Get("reply_to").(string),
		Message:            d.Get("message").(string),
		EnableDashboardURL: d.Get("include_dashboard_link").(bool),
		EnableCSV:          d.Get("include_table_csv").(bool),
		Options: &models.ReportOptions{
			Layout:      d.Get("layout").(string),
			Orientation: d.Get("orientation").(string),
		},
		Schedule: &models.ReportSchedule{
			Frequency: frequency,
			TimeZone:  timezone,
		},
		Formats: []models.Type{reportFormatPDF},
	}

	report = setDashboards(report, d)

	if v, ok := d.GetOk("formats"); ok && v != nil {
		report.Formats = []models.Type{}
		formats := common.SetToStringSlice(v.(*schema.Set))
		for _, format := range formats {
			report.Formats = append(report.Formats, models.Type(format))
		}
	}

	location, err := time.LoadLocation(timezone)
	if err != nil {
		return models.CreateOrUpdateReport{}, err
	}

	// Set schedule start time
	if frequency != reportFrequencyNever {
		if startTimeStr := d.Get("schedule.0.start_time").(string); startTimeStr != "" {
			date, err := formatDate(startTimeStr, location)
			if err != nil {
				return models.CreateOrUpdateReport{}, err
			}
			report.Schedule.StartDate = date
		}
	}

	// Set schedule end time
	if frequency != reportFrequencyOnce && frequency != reportFrequencyNever {
		if endTimeStr := d.Get("schedule.0.end_time").(string); endTimeStr != "" {
			date, err := formatDate(endTimeStr, location)
			if err != nil {
				return models.CreateOrUpdateReport{}, err
			}
			report.Schedule.EndDate = date
		}
	}

	if frequency == reportFrequencyMonthly {
		if lastDayOfMonth := d.Get("schedule.0.last_day_of_month").(bool); lastDayOfMonth {
			report.Schedule.DayOfMonth = "last"
		}
	}

	if reportWorkdaysOnlyConfigAllowed(frequency) {
		report.Schedule.WorkdaysOnly = d.Get("schedule.0.workdays_only").(bool)
	}
	if frequency == reportFrequencyCustom {
		customInterval := d.Get("schedule.0.custom_interval").(string)
		amount, unit, err := parseCustomReportInterval(customInterval)
		if err != nil {
			return models.CreateOrUpdateReport{}, err
		}
		report.Schedule.IntervalAmount = int64(amount)
		report.Schedule.IntervalFrequency = unit
	}

	return report, nil
}

func setDashboards(report models.CreateOrUpdateReport, d *schema.ResourceData) models.CreateOrUpdateReport {
	dashboards := d.Get("dashboards").([]interface{})
	for _, dashboard := range dashboards {
		dash := dashboard.(map[string]interface{})
		timeRange := dash["time_range"].([]interface{})
		tr := &models.ReportTimeRange{}
		if len(timeRange) > 0 {
			timeRange := timeRange[0].(map[string]interface{})
			tr = &models.ReportTimeRange{From: timeRange["from"].(string), To: timeRange["to"].(string)}
		}

		report.Dashboards = append(report.Dashboards, &models.ReportDashboard{
			Dashboard: &models.ReportDashboardID{
				UID: dash["uid"].(string),
			},
			TimeRange:       tr,
			ReportVariables: parseReportVariablesRequest(dash["report_variables"]),
		})
	}
	return report
}

func reportWorkdaysOnlyConfigAllowed(frequency string) bool {
	return frequency == reportFrequencyHourly || frequency == reportFrequencyDaily || frequency == reportFrequencyCustom
}

func parseCustomReportInterval(i interface{}) (int, string, error) {
	parseErr := errors.New("custom_interval must be in format `<number> <unit>` where unit is one of `hours`, `days`, `weeks`, `months`")

	v := i.(string)
	split := strings.Split(v, " ")
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

func validateTimezone(i interface{}, path cty.Path) diag.Diagnostics {
	timezone := i.(string)
	_, err := time.LoadLocation(timezone)
	return diag.FromErr(err)
}

func validateReportVariables(i interface{}, path cty.Path) diag.Diagnostics {
	m, ok := i.(map[string]interface{})
	if !ok {
		return diag.FromErr(errors.New("report_variables schema should be a map of strings separated by commas"))
	}

	for _, v := range m {
		if _, ok := v.(string); !ok {
			return diag.FromErr(fmt.Errorf("value %#v isn't a string", v))
		}
	}

	return nil
}

func parseReportVariablesRequest(reportVariables interface{}) map[string][]string {
	if reportVariables == nil {
		return nil
	}
	rvMap := reportVariables.(map[string]interface{})
	newMap := make(map[string][]string, len(rvMap))
	for k, rv := range rvMap {
		newMap[k] = strings.Split(rv.(string), ",")
	}

	return newMap
}

func parseReportVariablesResponse(reportVariables interface{}) map[string]interface{} {
	if reportVariables == nil {
		return nil
	}
	rvMap := reportVariables.(map[string]interface{})
	newMap := make(map[string]interface{}, len(rvMap))
	for k, rv := range rvMap {
		rvType := rv.([]interface{})
		values := make([]string, len(rvType))
		for i, v := range rvType {
			values[i] = v.(string)
		}
		newMap[k] = strings.Join(values, ",")
	}

	return newMap
}

func validateDate(i interface{}, _ cty.Path) diag.Diagnostics {
	v, ok := i.(string)
	if !ok {
		return diag.FromErr(fmt.Errorf("time should be a string"))
	}

	_, timezoneFormat := time.Parse(time.RFC3339, v)
	_, noTimezoneFormat := time.Parse(timeDateShortFormat, v)

	if timezoneFormat != nil && noTimezoneFormat != nil {
		return diag.FromErr(fmt.Errorf("time format should be %s or %s", time.RFC3339, timeDateShortFormat))
	}

	return nil
}

func formatDate(date string, timezone *time.Location) (*strfmt.DateTime, error) {
	parsedDate, err := time.Parse(timeDateShortFormat, date)
	if err != nil {
		return CheckTimezoneFormatDate(date, timezone)
	}

	dateTime := strfmt.DateTime(parsedDate.In(timezone))
	return &dateTime, nil
}

// CheckTimezoneFormatDate is exported for testing purposes
func CheckTimezoneFormatDate(date string, timezone *time.Location) (*strfmt.DateTime, error) {
	parsedDate, err := time.Parse(time.RFC3339, date)
	if err != nil {
		return nil, err
	}

	// If the date is already in RFC3339 format (contains timezone info),
	// just convert it to the target timezone instead of rejecting it.
	// This handles the case where dates from the API (in state) are being re-processed.
	dateTime := strfmt.DateTime(parsedDate.In(timezone))
	return &dateTime, nil
}

func checkStartTimeDiff(_, old, new string, _ *schema.ResourceData) bool {
	oldParsed, newParsed, shouldSkip := checkDateTime(old, new)
	if shouldSkip {
		return true
	}

	// If empty, the start date will be set to the current time (at the time of creation)
	if new == "" && oldParsed.Before(time.Now()) {
		return true
	}

	return oldParsed.Equal(newParsed)
}

func checkEndTimeDiff(_, old, new string, _ *schema.ResourceData) bool {
	oldParsed, newParsed, shouldSkip := checkDateTime(old, new)
	if shouldSkip {
		return true
	}

	return oldParsed.Equal(newParsed)
}

func checkDateTime(old, new string) (time.Time, time.Time, bool) {
	oldParsed, oldErr := time.Parse(time.RFC3339, old)
	newParsed, newErr := time.Parse(time.RFC3339, new)

	if oldErr != nil && newErr != nil {
		oldParsed, _ = time.Parse(timeDateShortFormat, old)
		newParsed, _ = time.Parse(timeDateShortFormat, new)
	} else if newErr != nil {
		if _, err := time.Parse(timeDateShortFormat, new); err == nil {
			return time.Time{}, time.Time{}, true
		}
	}

	return oldParsed, newParsed, false
}
