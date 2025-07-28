package oncall

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

var rollingUsers = "rolling_users"
var recurrentEvent = "recurrent_event"
var singleEvent = "single_event"

var onCallShiftTypeOptions = []string{
	rollingUsers,
	recurrentEvent,
	singleEvent,
}

var onCallShiftTypeOptionsVerbal = strings.Join(onCallShiftTypeOptions, ", ")

var onCallShiftFrequencyOptions = []string{
	"hourly",
	"daily",
	"weekly",
	"monthly",
}

var onCallShiftFrequencyOptionsVerbal = strings.Join(onCallShiftFrequencyOptions, ", ")

var onCallShiftWeekDayOptions = []string{
	"MO",
	"TU",
	"WE",
	"TH",
	"FR",
	"SA",
	"SU",
}

var onCallShiftWeekDayOptionsVerbal = strings.Join(onCallShiftWeekDayOptions, ", ")

var sourceTerraform = 3

func resourceOnCallShift() *common.Resource {
	schema := &schema.Resource{
		Description: `
* [HTTP API](https://grafana.com/docs/oncall/latest/oncall-api-reference/on_call_shifts/)
`,
		CreateContext: withClient[schema.CreateContextFunc](resourceOnCallShiftCreate),
		ReadContext:   withClient[schema.ReadContextFunc](resourceOnCallShiftRead),
		UpdateContext: withClient[schema.UpdateContextFunc](resourceOnCallShiftUpdate),
		DeleteContext: withClient[schema.DeleteContextFunc](resourceOnCallShiftDelete),
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"team_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The ID of the OnCall team (using the `grafana_oncall_team` datasource).",
			},
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
				Description:  "The shift's name.",
			},
			"type": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice(onCallShiftTypeOptions, false),
				Description:  fmt.Sprintf("The shift's type. Can be %s", onCallShiftTypeOptionsVerbal),
			},
			"level": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The priority level. The higher the value, the higher the priority.",
			},
			"start": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
				Description:  "The start time of the on-call shift. This parameter takes a date format as yyyy-MM-dd'T'HH:mm:ss (for example \"2020-09-05T08:00:00\")",
			},
			"duration": {
				Type:         schema.TypeInt,
				Required:     true,
				ValidateFunc: validation.IntAtLeast(0),
				Description:  "The duration of the event.",
			},
			"frequency": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice(onCallShiftFrequencyOptions, false),
				Description:  fmt.Sprintf("The frequency of the event. Can be %s", onCallShiftFrequencyOptionsVerbal),
				RequiredWith: []string{
					"interval",
				},
			},
			"until": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringIsNotEmpty,
				Description:  "The end time of recurrent on-call shifts (endless if null). This parameter takes a date format as yyyy-MM-dd'T'HH:mm:ss (for example \"2020-09-05T08:00:00\")",
			},
			"users": {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional:    true,
				Description: "The list of on-call users (for single_event and recurrent_event event type).	",
			},
			"rolling_users": {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeSet,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
				},
				Optional:    true,
				Description: "The list of lists with on-call users (for rolling_users event type)",
			},
			"interval": {
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntAtLeast(1),
				Description:  "The positive integer representing at which intervals the recurrence rule repeats.",
				RequiredWith: []string{
					"frequency",
				},
			},
			"week_start": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice(onCallShiftWeekDayOptions, false),
				Description:  fmt.Sprintf("Start day of the week in iCal format. Can be %s", onCallShiftWeekDayOptionsVerbal),
			},
			"by_day": {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringInSlice(onCallShiftWeekDayOptions, false),
				},
				Optional:    true,
				Description: fmt.Sprintf("This parameter takes a list of days in iCal format. Can be %s", onCallShiftWeekDayOptionsVerbal),
			},
			"by_month": {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type:         schema.TypeInt,
					ValidateFunc: validation.IntBetween(1, 12),
				},
				Optional:    true,
				Description: "This parameter takes a list of months. Valid values are 1 to 12",
			},
			"by_monthday": {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type:         schema.TypeInt,
					ValidateFunc: validation.IntBetween(-31, 31),
				},
				Optional:    true,
				Description: "This parameter takes a list of days of the month.  Valid values are 1 to 31 or -31 to -1",
			},
			"time_zone": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringIsNotEmpty,
				Description:  "The shift's timezone.  Overrides schedule's timezone.",
			},
			"start_rotation_from_user_index": {
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntAtLeast(0),
				Description:  "The index of the list of users in rolling_users, from which on-call rotation starts.",
			},
		},
	}

	return common.NewLegacySDKResource(
		common.CategoryOnCall,
		"grafana_oncall_on_call_shift",
		resourceID,
		schema,
	).
		WithLister(oncallListerFunction(listShifts)).
		WithPreferredResourceNameField("name")
}

func listShifts(client *onCallAPI.Client, listOptions onCallAPI.ListOptions) (ids []string, nextPage *string, err error) {
	resp, _, err := client.OnCallShifts.ListOnCallShifts(&onCallAPI.ListOnCallShiftOptions{ListOptions: listOptions})
	if err != nil {
		return nil, nil, err
	}
	for _, i := range resp.OnCallShifts {
		ids = append(ids, i.ID)
	}
	return ids, resp.Next, nil
}

func resourceOnCallShiftCreate(ctx context.Context, d *schema.ResourceData, client *onCallAPI.Client) diag.Diagnostics {
	teamIDData := d.Get("team_id").(string)
	typeData := d.Get("type").(string)
	nameData := d.Get("name").(string)
	startData := d.Get("start").(string)
	durationData := d.Get("duration").(int)

	createOptions := &onCallAPI.CreateOnCallShiftOptions{
		TeamId:   teamIDData,
		Type:     typeData,
		Name:     nameData,
		Start:    startData,
		Duration: durationData,
		Source:   sourceTerraform,
	}

	levelData := d.Get("level").(int)
	createOptions.Level = &levelData

	frequencyData, frequencyOk := d.GetOk("frequency")
	if frequencyOk {
		if typeData != singleEvent {
			f := frequencyData.(string)
			createOptions.Frequency = &f
		} else {
			return diag.Errorf("frequency can not be set with type: %s", typeData)
		}
	}

	usersData, usersDataOk := d.GetOk("users")
	if usersDataOk {
		if typeData != rollingUsers {
			usersDataSlice := common.SetToStringSlice(usersData.(*schema.Set))
			createOptions.Users = &usersDataSlice
		} else {
			return diag.Errorf("`users` can not be set with type: %s, use `rolling_users` field instead", typeData)
		}
	}

	intervalData, intervalOk := d.GetOk("interval")
	if intervalOk {
		if typeData != singleEvent {
			i := intervalData.(int)
			createOptions.Interval = &i
		} else {
			return diag.Errorf("interval can not be set with type: %s", typeData)
		}
	}

	weekStartData, weekStartOk := d.GetOk("week_start")
	if weekStartOk {
		if typeData != singleEvent {
			w := weekStartData.(string)
			createOptions.WeekStart = &w
		} else {
			return diag.Errorf("week_start can not be set with type: %s", typeData)
		}
	}

	byDayData, byDayOk := d.GetOk("by_day")
	if byDayOk {
		if typeData != singleEvent {
			byDayDataSlice := common.SetToStringSlice(byDayData.(*schema.Set))
			createOptions.ByDay = &byDayDataSlice
		} else {
			return diag.Errorf("by_day can not be set with type: %s", typeData)
		}
	}

	byMonthData, byMonthOk := d.GetOk("by_month")
	if byMonthOk {
		if typeData != singleEvent {
			byMonthDataSlice := common.SetToIntSlice[int](byMonthData.(*schema.Set))
			createOptions.ByMonth = &byMonthDataSlice
		} else {
			return diag.Errorf("by_month can not be set with type: %s", typeData)
		}
	}

	byMonthdayData, byMonthdayOk := d.GetOk("by_monthday")
	if byMonthdayOk {
		if typeData != singleEvent {
			byMonthdayDataSlice := common.SetToIntSlice[int](byMonthdayData.(*schema.Set))
			createOptions.ByMonthday = &byMonthdayDataSlice
		} else {
			return diag.Errorf("by_monthday can not be set with type: %s", typeData)
		}
	}

	rollingUsersData, rollingUsersOk := d.GetOk(rollingUsers)
	if rollingUsersOk {
		if typeData == rollingUsers {
			listSet := rollingUsersData.([]interface{})
			for _, set := range listSet {
				if set == nil {
					return diag.Errorf("`rolling_users` can not include an empty group")
				}
			}
			rollingUsersDataSlice := common.ListOfSetsToStringSlice(rollingUsersData.([]interface{}))
			createOptions.RollingUsers = &rollingUsersDataSlice
		} else {
			return diag.Errorf("`rolling_users` can not be set with type: %s, use `users` field instead", typeData)
		}
	}

	untilData, untilOk := d.GetOk("until")
	if untilOk {
		if typeData == singleEvent {
			return diag.Errorf("`until` can not be set with type: %s", typeData)
		} else {
			u := untilData.(string)
			createOptions.Until = &u
		}
	}

	timeZoneData, timeZoneOk := d.GetOk("time_zone")
	if timeZoneOk {
		tz := timeZoneData.(string)
		createOptions.TimeZone = &tz
	}

	if typeData == rollingUsers {
		startRotationFromUserIndexData := d.Get("start_rotation_from_user_index")
		i := startRotationFromUserIndexData.(int)
		createOptions.StartRotationFromUserIndex = &i
	} // todo: add validation for start_rotation_from_user_index

	onCallShift, _, err := client.OnCallShifts.CreateOnCallShift(createOptions)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(onCallShift.ID)

	return resourceOnCallShiftRead(ctx, d, client)
}

func resourceOnCallShiftUpdate(ctx context.Context, d *schema.ResourceData, client *onCallAPI.Client) diag.Diagnostics {
	typeData := d.Get("type").(string)
	nameData := d.Get("name").(string)
	teamIDData := d.Get("team_id").(string)
	startData := d.Get("start").(string)
	durationData := d.Get("duration").(int)

	updateOptions := &onCallAPI.UpdateOnCallShiftOptions{
		Type:     typeData,
		Name:     nameData,
		TeamId:   teamIDData,
		Start:    startData,
		Duration: durationData,
		Source:   sourceTerraform,
	}

	levelData := d.Get("level").(int)
	updateOptions.Level = &levelData

	frequencyData, frequencyOk := d.GetOk("frequency")
	if frequencyOk {
		if typeData != singleEvent {
			f := frequencyData.(string)
			updateOptions.Frequency = &f
		} else {
			return diag.Errorf("frequency can not be set with type: %s", typeData)
		}
	}

	usersData, usersDataOk := d.GetOk("users")
	if usersDataOk {
		if typeData != rollingUsers {
			usersDataSlice := common.SetToStringSlice(usersData.(*schema.Set))
			updateOptions.Users = &usersDataSlice
		} else {
			return diag.Errorf("`users` can not be set with type: %s, use `rolling_users` field instead", typeData)
		}
	}

	intervalData, intervalOk := d.GetOk("interval")
	if intervalOk {
		if typeData != singleEvent {
			i := intervalData.(int)
			updateOptions.Interval = &i
		} else {
			return diag.Errorf("interval can not be set with type: %s", typeData)
		}
	}

	weekStartData, weekStartOk := d.GetOk("week_start")
	if weekStartOk {
		if typeData != singleEvent {
			w := weekStartData.(string)
			updateOptions.WeekStart = &w
		} else {
			return diag.Errorf("week_start can not be set with type: %s", typeData)
		}
	}

	byDayData, byDayOk := d.GetOk("by_day")
	if byDayOk {
		if typeData != singleEvent {
			byDayDataSlice := common.SetToStringSlice(byDayData.(*schema.Set))
			updateOptions.ByDay = &byDayDataSlice
		} else {
			return diag.Errorf("by_day can not be set with type: %s", typeData)
		}
	}

	byMonthData, byMonthOk := d.GetOk("by_month")
	if byMonthOk {
		if typeData != singleEvent {
			byMonthDataSlice := common.SetToIntSlice[int](byMonthData.(*schema.Set))
			updateOptions.ByMonth = &byMonthDataSlice
		} else {
			return diag.Errorf("by_month can not be set with type: %s", typeData)
		}
	}

	byMonthDayData, byMonthDayOk := d.GetOk("by_monthday")
	if byMonthDayOk {
		if typeData != singleEvent {
			byMonthDayData := common.SetToIntSlice[int](byMonthDayData.(*schema.Set))
			updateOptions.ByMonthday = &byMonthDayData
		} else {
			return diag.Errorf("by_monthday can not be set with type: %s", typeData)
		}
	}

	untilData, untilOk := d.GetOk("until")
	if untilOk {
		if typeData == singleEvent {
			return diag.Errorf("`until` can not be set with type: %s", typeData)
		} else {
			u := untilData.(string)
			updateOptions.Until = &u
		}
	}

	timeZoneData, timeZoneOk := d.GetOk("time_zone")
	if timeZoneOk {
		tz := timeZoneData.(string)
		updateOptions.TimeZone = &tz
	}

	rollingUsersData, rollingUsersOk := d.GetOk(rollingUsers)
	if rollingUsersOk {
		if typeData == rollingUsers {
			listSet := rollingUsersData.([]interface{})
			for _, set := range listSet {
				if set == nil {
					return diag.Errorf("`rolling_users` can not include an empty group")
				}
			}
			rollingUsersDataSlice := common.ListOfSetsToStringSlice(rollingUsersData.([]interface{}))
			updateOptions.RollingUsers = &rollingUsersDataSlice
		} else {
			return diag.Errorf("`rolling_users` can not be set with type: %s, use `users` field instead", typeData)
		}
	}

	if typeData == rollingUsers {
		startRotationFromUserIndexData := d.Get("start_rotation_from_user_index")
		i := startRotationFromUserIndexData.(int)
		updateOptions.StartRotationFromUserIndex = &i
	} // todo: add validation for start_rotation_from_user_index

	onCallShift, _, err := client.OnCallShifts.UpdateOnCallShift(d.Id(), updateOptions)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(onCallShift.ID)

	return resourceOnCallShiftRead(ctx, d, client)
}

func resourceOnCallShiftRead(ctx context.Context, d *schema.ResourceData, client *onCallAPI.Client) diag.Diagnostics {
	options := &onCallAPI.GetOnCallShiftOptions{}
	onCallShift, r, err := client.OnCallShifts.GetOnCallShift(d.Id(), options)
	if err != nil {
		if r != nil && r.StatusCode == http.StatusNotFound {
			return common.WarnMissing("on-call shift", d)
		}
		return diag.FromErr(err)
	}

	d.Set("team_id", onCallShift.TeamId)
	d.Set("name", onCallShift.Name)
	d.Set("type", onCallShift.Type)
	d.Set("level", onCallShift.Level)
	d.Set("start", onCallShift.Start)
	d.Set("until", onCallShift.Until)
	d.Set("duration", onCallShift.Duration)
	d.Set("frequency", onCallShift.Frequency)
	d.Set("week_start", onCallShift.WeekStart)
	d.Set("interval", onCallShift.Interval)
	d.Set("users", onCallShift.Users)
	d.Set("rolling_users", onCallShift.RollingUsers)
	d.Set("by_day", onCallShift.ByDay)
	d.Set("by_month", onCallShift.ByMonth)
	d.Set("by_monthday", onCallShift.ByMonthday)
	d.Set("time_zone", onCallShift.TimeZone)
	d.Set("start_rotation_from_user_index", onCallShift.StartRotationFromUserIndex)

	return nil
}

func resourceOnCallShiftDelete(ctx context.Context, d *schema.ResourceData, client *onCallAPI.Client) diag.Diagnostics {
	options := &onCallAPI.DeleteOnCallShiftOptions{}
	_, err := client.OnCallShifts.DeleteOnCallShift(d.Id(), options)
	return diag.FromErr(err)
}
