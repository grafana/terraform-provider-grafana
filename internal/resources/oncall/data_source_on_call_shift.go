package oncall

import (
	"context"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceOnCallShift() *common.DataSource {
	schema := &schema.Resource{
		Description: `
* [HTTP API](https://grafana.com/docs/oncall/latest/oncall-api-reference/on_call_shifts/)
`,
		ReadContext: withClient[schema.ReadContextFunc](dataSourceOnCallShiftRead),
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The shift's name.",
			},
			"type": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The shift's type.",
			},
			"team_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the OnCall team.",
			},
			"level": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The priority level.",
			},
			"start": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The start time of the on-call shift.",
			},
			"duration": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The duration of the event.",
			},
			"frequency": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The frequency of the event.",
			},
			"users": {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed:    true,
				Description: "The list of on-call users.",
			},
			"rolling_users": {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeSet,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
				},
				Computed:    true,
				Description: "The list of lists with on-call users (for rolling_users event type).",
			},
			"interval": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The positive integer representing at which intervals the recurrence rule repeats.",
			},
			"week_start": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Start day of the week in iCal format.",
			},
			"by_day": {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed:    true,
				Description: "The list of days in iCal format.",
			},
			"by_month": {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeInt,
				},
				Computed:    true,
				Description: "The list of months.",
			},
			"by_monthday": {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeInt,
				},
				Computed:    true,
				Description: "The list of days of the month.",
			},
			"time_zone": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The shift's timezone.",
			},
		},
	}
	return common.NewLegacySDKDataSource(common.CategoryOnCall, "grafana_oncall_on_call_shift", schema)
}

func dataSourceOnCallShiftRead(ctx context.Context, d *schema.ResourceData, client *onCallAPI.Client) diag.Diagnostics {
	options := &onCallAPI.ListOnCallShiftOptions{}
	nameData := d.Get("name").(string)

	options.Name = nameData

	shiftsResponse, _, err := client.OnCallShifts.ListOnCallShifts(options)
	if err != nil {
		return diag.FromErr(err)
	}

	if len(shiftsResponse.OnCallShifts) == 0 {
		return diag.Errorf("couldn't find an on-call shift matching: %s", options.Name)
	} else if len(shiftsResponse.OnCallShifts) != 1 {
		return diag.Errorf("more than one on-call shift found matching: %s", options.Name)
	}

	shift := shiftsResponse.OnCallShifts[0]

	d.SetId(shift.ID)
	d.Set("name", shift.Name)
	d.Set("type", shift.Type)
	d.Set("team_id", shift.TeamId)
	d.Set("level", shift.Level)
	d.Set("start", shift.Start)
	d.Set("duration", shift.Duration)
	d.Set("frequency", shift.Frequency)
	d.Set("users", shift.Users)
	d.Set("rolling_users", shift.RollingUsers)
	d.Set("interval", shift.Interval)
	d.Set("week_start", shift.WeekStart)
	d.Set("by_day", shift.ByDay)
	d.Set("by_month", shift.ByMonth)
	d.Set("by_monthday", shift.ByMonthday)
	d.Set("time_zone", shift.TimeZone)

	return nil
}
