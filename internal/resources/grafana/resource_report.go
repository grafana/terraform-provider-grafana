package grafana

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
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
)

var (
	reportLayouts      = []string{reportLayoutSimple, reportLayoutGrid}
	reportOrientations = []string{reportOrientationLandscape, reportOrientationPortrait}
	reportFrequencies  = []string{reportFrequencyNever, reportFrequencyOnce, reportFrequencyHourly, reportFrequencyDaily, reportFrequencyWeekly, reportFrequencyMonthly, reportFrequencyCustom}
	reportFormats      = []string{reportFormatPDF, reportFormatCSV, reportFormatImage}
)

func ResourceReport() *schema.Resource {
	return &schema.Resource{
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
			"dashboard_id": {
				Type:         schema.TypeInt,
				ExactlyOneOf: []string{"dashboard_id", "dashboard_uid"},
				Computed:     true,
				Optional:     true,
				Deprecated:   "Use dashboard_uid instead",
				Description:  "Dashboard to be sent in the report. This field is deprecated, use `dashboard_uid` instead.",
			},
			"dashboard_uid": {
				Type:         schema.TypeString,
				ExactlyOneOf: []string{"dashboard_id", "dashboard_uid"},
				Computed:     true,
				Optional:     true,
				Description:  "Dashboard to be sent in the report.",
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
			"time_range": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Time range of the report.",
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"from": {
							Type:         schema.TypeString,
							Optional:     true,
							Description:  "Start of the time range.",
							RequiredWith: []string{"time_range.0.to"},
						},
						"to": {
							Type:         schema.TypeString,
							Optional:     true,
							Description:  "End of the time range.",
							RequiredWith: []string{"time_range.0.from"},
						},
					},
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
							Type:         schema.TypeString,
							Optional:     true,
							Description:  "Start time of the report. If empty, the start date will be set to the creation time. Note that times will be saved as UTC in Grafana.",
							ValidateFunc: validation.IsRFC3339Time,
							DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
								oldParsed, _ := time.Parse(time.RFC3339, old)
								newParsed, _ := time.Parse(time.RFC3339, new)

								// If empty, the start date will be set to the current time (at the time of creation)
								if new == "" && oldParsed.Before(time.Now()) {
									return true
								}

								return oldParsed.Equal(newParsed)
							},
						},
						"end_time": {
							Type:         schema.TypeString,
							Optional:     true,
							Description:  "End time of the report. If empty, the report will be sent indefinitely (according to frequency). Note that times will be saved as UTC in Grafana.",
							ValidateFunc: validation.IsRFC3339Time,
							DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
								oldParsed, _ := time.Parse(time.RFC3339, old)
								newParsed, _ := time.Parse(time.RFC3339, new)
								return oldParsed.Equal(newParsed)
							},
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
					},
				},
			},
		},
	}
}

func CreateReport(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID := DeprecatedClientFromNewOrgResource(meta, d)

	report, err := schemaToReport(client, d)
	if err != nil {
		return diag.FromErr(err)
	}

	id, err := client.NewReport(report)
	if err != nil {
		data, _ := json.Marshal(report)
		return diag.Errorf("error creating the following report:\n%s\n%v", string(data), err)
	}
	d.SetId(MakeOrgResourceID(orgID, id))
	return ReadReport(ctx, d, meta)
}

func ReadReport(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, idStr := DeprecatedClientFromExistingOrgResource(meta, d.Id())
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}
	r, err := client.Report(id)
	if err, shouldReturn := common.CheckReadError("report", d, err); shouldReturn {
		return err
	}

	d.SetId(MakeOrgResourceID(r.OrgID, id))
	d.Set("dashboard_id", r.Dashboards[0].Dashboard.ID)
	d.Set("dashboard_uid", r.Dashboards[0].Dashboard.UID)
	d.Set("name", r.Name)
	d.Set("recipients", strings.Split(r.Recipients, ","))
	d.Set("reply_to", r.ReplyTo)
	d.Set("message", r.Message)
	d.Set("include_dashboard_link", r.EnableDashboardURL)
	d.Set("include_table_csv", r.EnableCSV)
	d.Set("layout", r.Options.Layout)
	d.Set("orientation", r.Options.Orientation)
	d.Set("org_id", strconv.FormatInt(r.OrgID, 10))

	if _, ok := d.GetOk("formats"); ok {
		d.Set("formats", common.StringSliceToSet(r.Formats))
	}

	timeRange := r.Dashboards[0].TimeRange
	if timeRange.From != "" {
		d.Set("time_range", []interface{}{
			map[string]interface{}{
				"from": timeRange.From,
				"to":   timeRange.To,
			},
		})
	}

	schedule := map[string]interface{}{
		"frequency":     r.Schedule.Frequency,
		"workdays_only": r.Schedule.WorkdaysOnly,
	}
	if r.Schedule.IntervalAmount != 0 && r.Schedule.IntervalFrequency != "" {
		schedule["custom_interval"] = fmt.Sprintf("%d %s", r.Schedule.IntervalAmount, r.Schedule.IntervalFrequency)
	}
	if r.Schedule.StartDate != nil {
		schedule["start_time"] = r.Schedule.StartDate.Format(time.RFC3339)
	}
	if r.Schedule.EndDate != nil {
		schedule["end_time"] = r.Schedule.EndDate.Format(time.RFC3339)
	}
	if r.Schedule.DayOfMonth == "last" {
		schedule["last_day_of_month"] = true
	}

	d.Set("schedule", []interface{}{schedule})

	return nil
}

func UpdateReport(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, idStr := DeprecatedClientFromExistingOrgResource(meta, d.Id())
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	report, err := schemaToReport(client, d)
	if err != nil {
		return diag.FromErr(err)
	}
	report.ID = id

	if err := client.UpdateReport(report); err != nil {
		data, _ := json.Marshal(report)
		return diag.Errorf("error updating the following report:\n%s\n%v", string(data), err)
	}
	return ReadReport(ctx, d, meta)
}

func DeleteReport(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, idStr := DeprecatedClientFromExistingOrgResource(meta, d.Id())
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	err = client.DeleteReport(id)
	diag, _ := common.CheckReadError("report", d, err)
	return diag
}

func schemaToReport(client *gapi.Client, d *schema.ResourceData) (gapi.Report, error) {
	id := int64(d.Get("dashboard_id").(int))
	uid := d.Get("dashboard_uid").(string)
	if uid == "" {
		dashboards, err := client.DashboardsByIDs([]int64{id})
		if err != nil {
			return gapi.Report{}, fmt.Errorf("error getting dashboard %d: %v", id, err)
		}
		for _, dashboard := range dashboards {
			if int64(dashboard.ID) == id {
				uid = dashboard.UID
				break
			}
		}
		if uid == "" {
			return gapi.Report{}, fmt.Errorf("dashboard %d not found", id)
		}
	}

	frequency := d.Get("schedule.0.frequency").(string)
	report := gapi.Report{
		Dashboards: []gapi.ReportDashboard{
			{
				Dashboard: gapi.ReportDashboardIdentifier{
					UID: uid,
				},
			},
		},
		Name:               d.Get("name").(string),
		Recipients:         strings.Join(common.ListToStringSlice(d.Get("recipients").([]interface{})), ","),
		ReplyTo:            d.Get("reply_to").(string),
		Message:            d.Get("message").(string),
		EnableDashboardURL: d.Get("include_dashboard_link").(bool),
		EnableCSV:          d.Get("include_table_csv").(bool),
		Options: gapi.ReportOptions{
			Layout:      d.Get("layout").(string),
			Orientation: d.Get("orientation").(string),
		},
		Schedule: gapi.ReportSchedule{
			Frequency: frequency,
			TimeZone:  "GMT",
		},
		Formats: []string{reportFormatPDF},
	}

	if v, ok := d.GetOk("formats"); ok && v != nil {
		report.Formats = common.SetToStringSlice(v.(*schema.Set))
	}

	// Set dashboard time range
	timeRange := d.Get("time_range").([]interface{})
	if len(timeRange) > 0 {
		timeRange := timeRange[0].(map[string]interface{})
		report.Dashboards[0].TimeRange = gapi.ReportDashboardTimeRange{From: timeRange["from"].(string), To: timeRange["to"].(string)}
	}

	// Set schedule start time
	if frequency != reportFrequencyNever {
		if startTimeStr := d.Get("schedule.0.start_time").(string); startTimeStr != "" {
			startDate, err := time.Parse(time.RFC3339, startTimeStr)
			if err != nil {
				return gapi.Report{}, err
			}
			startDate = startDate.UTC()
			report.Schedule.StartDate = &startDate
		}
	}

	// Set schedule end time
	if frequency != reportFrequencyOnce && frequency != reportFrequencyNever {
		if endTimeStr := d.Get("schedule.0.end_time").(string); endTimeStr != "" {
			endDate, err := time.Parse(time.RFC3339, endTimeStr)
			if err != nil {
				return gapi.Report{}, err
			}
			endDate = endDate.UTC()
			report.Schedule.EndDate = &endDate
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
			return gapi.Report{}, err
		}
		report.Schedule.IntervalAmount = int64(amount)
		report.Schedule.IntervalFrequency = unit
	}

	return report, nil
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
