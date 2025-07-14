package oncall

import (
	"context"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceShift() *common.DataSource {
	schema := &schema.Resource{
		Description: `
* [HTTP API](https://grafana.com/docs/oncall/latest/oncall-api-reference/on_call_shifts/)
`,
		ReadContext: withClient[schema.ReadContextFunc](dataSourceShiftRead),
		Schema: common.CloneResourceSchemaForDatasource(resourceOnCallShift().Schema, map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The shift's name.",
			},
		}),
	}
	return common.NewLegacySDKDataSource(common.CategoryOnCall, "grafana_oncall_on_call_shift", schema)
}

func dataSourceShiftRead(ctx context.Context, d *schema.ResourceData, client *onCallAPI.Client) diag.Diagnostics {
	options := &onCallAPI.ListOnCallShiftOptions{}
	nameData := d.Get("name").(string)

	options.Name = nameData

	shiftsResponse, _, err := client.OnCallShifts.ListOnCallShifts(options)
	if err != nil {
		return diag.FromErr(err)
	}

	if len(shiftsResponse.OnCallShifts) == 0 {
		return diag.Errorf("couldn't find a shift matching: %s", options.Name)
	} else if len(shiftsResponse.OnCallShifts) != 1 {
		return diag.Errorf("more than one shift found matching: %s", options.Name)
	}

	shift := shiftsResponse.OnCallShifts[0]

	d.SetId(shift.ID)
	d.Set("team_id", shift.TeamId)
	d.Set("name", shift.Name)
	d.Set("type", shift.Type)
	d.Set("level", shift.Level)
	d.Set("start", shift.Start)
	d.Set("until", shift.Until)
	d.Set("duration", shift.Duration)
	d.Set("frequency", shift.Frequency)
	d.Set("week_start", shift.WeekStart)
	d.Set("interval", shift.Interval)
	d.Set("users", shift.Users)
	d.Set("rolling_users", shift.RollingUsers)
	d.Set("by_day", shift.ByDay)
	d.Set("by_month", shift.ByMonth)
	d.Set("by_monthday", shift.ByMonthday)
	d.Set("time_zone", shift.TimeZone)
	d.Set("start_rotation_from_user_index", shift.StartRotationFromUserIndex)

	return nil
}
