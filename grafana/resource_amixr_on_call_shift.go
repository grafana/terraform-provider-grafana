package grafana

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	amixrAPI "github.com/grafana/amixr-api-go-client"
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

func ResourceAmixrOnCallShift() *schema.Resource {
	return &schema.Resource{
		Description: `
* [HTTP API](https://grafana.com/docs/grafana-cloud/oncall/oncall-api-reference/on_call_shifts/)
`,
		Create: resourceAmixrOnCallShiftCreate,
		Read:   resourceAmixrOnCallShiftRead,
		Update: resourceAmixrOnCallShiftUpdate,
		Delete: resourceAmixrOnCallShiftDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"team_id": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The ID of the team.",
			},
			"name": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
				Description:  "The shift's name.",
			},
			"type": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice(onCallShiftTypeOptions, false),
				Description:  fmt.Sprintf("The shift's type. Can be %s", onCallShiftTypeOptionsVerbal),
			},
			"level": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The priority level. The higher the value, the higher the priority.",
			},
			"start": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
				Description:  "The start time of the on-call shift. This parameter takes a date format as yyyy-MM-dd'T'HH:mm:ss (for example \"2020-09-05T08:00:00\")",
			},
			"duration": &schema.Schema{
				Type:         schema.TypeInt,
				Required:     true,
				ValidateFunc: validation.IntAtLeast(0),
				Description:  "The duration of the event.",
			},
			"frequency": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice(onCallShiftFrequencyOptions, false),
				Description:  fmt.Sprintf("The frequency of the event. Can be %s", onCallShiftFrequencyOptionsVerbal),
			},
			"users": &schema.Schema{
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional: true,
				Description: "The list of on-call users (for single_event and recurrent_event event type).	",
			},
			"rolling_users": &schema.Schema{
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
			"interval": &schema.Schema{
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntAtLeast(1),
				Description:  "The positive integer representing at which intervals the recurrence rule repeats.",
			},
			"week_start": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice(onCallShiftWeekDayOptions, false),
				Description:  fmt.Sprintf("Start day of the week in iCal format. Can be %s", onCallShiftWeekDayOptionsVerbal),
			},
			"by_day": &schema.Schema{
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringInSlice(onCallShiftWeekDayOptions, false),
				},
				Optional:    true,
				Description: fmt.Sprintf("This parameter takes a list of days in iCal format. Can be %s", onCallShiftWeekDayOptionsVerbal),
			},
			"by_month": &schema.Schema{
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type:         schema.TypeInt,
					ValidateFunc: validation.IntBetween(1, 12),
				},
				Optional:    true,
				Description: "This parameter takes a list of months. Valid values are 1 to 12",
			},
			"by_monthday": &schema.Schema{
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type:         schema.TypeInt,
					ValidateFunc: validation.IntBetween(-31, 31),
				},
				Optional:    true,
				Description: "This parameter takes a list of days of the month.  Valid values are 1 to 31 or -31 to -1",
			},
			"time_zone": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringIsNotEmpty,
				Description:  "The shift's timezone.  Overrides schedule's timezone.",
			},
			"start_rotation_from_user_index": &schema.Schema{
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntAtLeast(0),
				Description:  "The index of the list of users in rolling_users, from which on-call rotation starts.",
			},
		},
	}
}

func resourceAmixrOnCallShiftCreate(d *schema.ResourceData, m interface{}) error {
	client := m.(*client).amixrAPI
	if client == nil {
		err := errors.New("amixr api client is not configured")
		return err
	}

	teamIdData := d.Get("team_id").(string)
	typeData := d.Get("type").(string)
	nameData := d.Get("name").(string)
	startData := d.Get("start").(string)
	durationData := d.Get("duration").(int)

	createOptions := &amixrAPI.CreateOnCallShiftOptions{
		TeamId:   teamIdData,
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
			return fmt.Errorf("frequency can not be set with type: %s", typeData)
		}
	}

	usersData, usersDataOk := d.GetOk("users")
	if usersDataOk {
		if typeData != rollingUsers {
			usersDataSlice := setToStringSlice(usersData.(*schema.Set))
			createOptions.Users = &usersDataSlice
		} else {
			return fmt.Errorf("`users` can not be set with type: %s, use `rolling_users` field instead", typeData)
		}
	}

	intervalData, intervalOk := d.GetOk("interval")
	if intervalOk {
		if typeData != singleEvent {
			i := intervalData.(int)
			createOptions.Interval = &i
		} else {
			return fmt.Errorf("interval can not be set with type: %s", typeData)
		}
	}

	weekStartData, weekStartOk := d.GetOk("week_start")
	if weekStartOk {
		if typeData != singleEvent {
			w := weekStartData.(string)
			createOptions.WeekStart = &w
		} else {
			return fmt.Errorf("week_start can not be set with type: %s", typeData)
		}
	}

	byDayData, byDayOk := d.GetOk("by_day")
	if byDayOk {
		if typeData != singleEvent {
			byDayDataSlice := setToStringSlice(byDayData.(*schema.Set))
			createOptions.ByDay = &byDayDataSlice
		} else {
			return fmt.Errorf("by_day can not be set with type: %s", typeData)
		}
	}

	byMonthData, byMonthOk := d.GetOk("by_month")
	if byMonthOk {
		if typeData != singleEvent {
			byMonthDataSlice := setToIntSlice(byMonthData.(*schema.Set))
			createOptions.ByMonth = &byMonthDataSlice
		} else {
			return fmt.Errorf("by_month can not be set with type: %s", typeData)
		}
	}

	byMonthdayData, byMonthdayOk := d.GetOk("by_monthday")
	if byMonthdayOk {
		if typeData != singleEvent {
			byMonthdayDataSlice := setToIntSlice(byMonthdayData.(*schema.Set))
			createOptions.ByMonthday = &byMonthdayDataSlice
		} else {
			return fmt.Errorf("by_monthday can not be set with type: %s", typeData)
		}
	}

	rollingUsersData, rollingUsersOk := d.GetOk(rollingUsers)
	if rollingUsersOk {
		if typeData == rollingUsers {
			rollingUsersDataSlice := listOfSetsToStringSlice(rollingUsersData.([]interface{}))
			createOptions.RollingUsers = &rollingUsersDataSlice
		} else {
			return fmt.Errorf("`rolling_users` can not be set with type: %s, use `users` field instead", typeData)
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
		return err
	}

	d.SetId(onCallShift.ID)

	return resourceAmixrOnCallShiftRead(d, m)
}

func resourceAmixrOnCallShiftUpdate(d *schema.ResourceData, m interface{}) error {
	client := m.(*client).amixrAPI
	if client == nil {
		err := errors.New("amixr api client is not configured")
		return err
	}

	typeData := d.Get("type").(string)
	nameData := d.Get("name").(string)
	startData := d.Get("start").(string)
	durationData := d.Get("duration").(int)

	updateOptions := &amixrAPI.UpdateOnCallShiftOptions{
		Type:     typeData,
		Name:     nameData,
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
			return fmt.Errorf("frequency can not be set with type: %s", typeData)
		}
	}

	usersData, usersDataOk := d.GetOk("users")
	if usersDataOk {
		if typeData != rollingUsers {
			usersDataSlice := setToStringSlice(usersData.(*schema.Set))
			updateOptions.Users = &usersDataSlice
		} else {
			return fmt.Errorf("`users` can not be set with type: %s, use `rolling_users` field instead", typeData)
		}
	}

	intervalData, intervalOk := d.GetOk("interval")
	if intervalOk {
		if typeData != singleEvent {
			i := intervalData.(int)
			updateOptions.Interval = &i
		} else {
			return fmt.Errorf("interval can not be set with type: %s", typeData)
		}
	}

	weekStartData, weekStartOk := d.GetOk("week_start")
	if weekStartOk {
		if typeData != singleEvent {
			w := weekStartData.(string)
			updateOptions.WeekStart = &w
		} else {
			return fmt.Errorf("week_start can not be set with type: %s", typeData)
		}
	}

	byDayData, byDayOk := d.GetOk("by_day")
	if byDayOk {
		if typeData != singleEvent {
			byDayDataSlice := setToStringSlice(byDayData.(*schema.Set))
			updateOptions.ByDay = &byDayDataSlice
		} else {
			return fmt.Errorf("by_day can not be set with type: %s", typeData)
		}
	}

	byMonthData, byMonthOk := d.GetOk("by_month")
	if byMonthOk {
		if typeData != singleEvent {
			byMonthDataSlice := setToIntSlice(byMonthData.(*schema.Set))
			updateOptions.ByMonth = &byMonthDataSlice
		} else {
			return fmt.Errorf("by_month can not be set with type: %s", typeData)
		}
	}

	byMonthDayData, byMonthDayOk := d.GetOk("by_monthday")
	if byMonthDayOk {
		if typeData != singleEvent {
			byMonthDayData := setToIntSlice(byMonthDayData.(*schema.Set))
			updateOptions.ByMonthday = &byMonthDayData
		} else {
			return fmt.Errorf("by_monthday can not be set with type: %s", typeData)
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
			rollingUsersDataSlice := listOfSetsToStringSlice(rollingUsersData.([]interface{}))
			updateOptions.RollingUsers = &rollingUsersDataSlice
		} else {
			return fmt.Errorf("`rolling_users` can not be set with type: %s, use `users` field instead", typeData)
		}
	}

	if typeData == rollingUsers {
		startRotationFromUserIndexData := d.Get("start_rotation_from_user_index")
		i := startRotationFromUserIndexData.(int)
		updateOptions.StartRotationFromUserIndex = &i
	} // todo: add validation for start_rotation_from_user_index

	onCallShift, _, err := client.OnCallShifts.UpdateOnCallShift(d.Id(), updateOptions)
	if err != nil {
		return err
	}

	d.SetId(onCallShift.ID)

	return resourceAmixrOnCallShiftRead(d, m)
}

func resourceAmixrOnCallShiftRead(d *schema.ResourceData, m interface{}) error {
	client := m.(*client).amixrAPI
	if client == nil {
		err := errors.New("amixr api client is not configured")
		return err
	}
	options := &amixrAPI.GetOnCallShiftOptions{}
	onCallShift, r, err := client.OnCallShifts.GetOnCallShift(d.Id(), options)

	if err != nil {
		if r != nil && r.StatusCode == http.StatusNotFound {
			log.Printf("[WARN] removing on-call shift %s from state because it no longer exists", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("team_id", onCallShift.TeamId)
	d.Set("name", onCallShift.Name)
	d.Set("type", onCallShift.Type)
	d.Set("level", onCallShift.Level)
	d.Set("start", onCallShift.Start)
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

func resourceAmixrOnCallShiftDelete(d *schema.ResourceData, m interface{}) error {
	client := m.(*client).amixrAPI
	if client == nil {
		err := errors.New("amixr api client is not configured")
		return err
	}
	options := &amixrAPI.DeleteOnCallShiftOptions{}
	_, err := client.OnCallShifts.DeleteOnCallShift(d.Id(), options)
	if err != nil {
		return err
	}

	d.SetId("")

	return nil
}
