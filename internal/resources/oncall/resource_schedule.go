package oncall

import (
	"context"
	"log"
	"net/http"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

var scheduleTypeOptions = []string{
	"ical",
	"calendar",
}

func ResourceSchedule() *schema.Resource {
	return &schema.Resource{
		Description: `
* [HTTP API](https://grafana.com/docs/oncall/latest/oncall-api-reference/schedules/)
`,
		CreateContext: resourceScheduleCreate,
		ReadContext:   resourceScheduleRead,
		UpdateContext: resourceScheduleUpdate,
		DeleteContext: resourceScheduleDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
				Description:  "The schedule's name.",
			},
			"team_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The ID of the OnCall team. To get one, create a team in Grafana, and navigate to the OnCall plugin (to sync the team with OnCall). You can then get the ID using the `grafana_oncall_team` datasource.",
			},
			"type": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice(scheduleTypeOptions, false),
				Description:  "The schedule's type.",
			},
			"time_zone": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The schedule's time zone.",
			},
			"ical_url_primary": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The URL of the external calendar iCal file.",
			},
			"ical_url_overrides": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The URL of external iCal calendar which override primary events.",
			},
			"enable_web_overrides": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Enable overrides via web UI (it will ignore ical_url_overrides).",
			},
			"slack": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"channel_id": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Slack channel id. Reminder about schedule shifts will be directed to this channel in Slack.",
						},
						"user_group_id": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: " Slack user group id. Members of user group will be updated when on-call users change.",
						},
					},
				},
				MaxItems:    1,
				Description: "The Slack-specific settings for a schedule.",
			},
			"shifts": {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional:    true,
				Description: "The list of ID's of on-call shifts.",
			},
		},
	}
}

func resourceScheduleCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client).OnCallClient

	nameData := d.Get("name").(string)
	teamIDData := d.Get("team_id").(string)
	typeData := d.Get("type").(string)
	slackData := d.Get("slack").([]interface{})

	createOptions := &onCallAPI.CreateScheduleOptions{
		TeamId: teamIDData,
		Name:   nameData,
		Type:   typeData,
		Slack:  expandScheduleSlack(slackData),
	}

	iCalURLPrimaryData, iCalURLPrimaryOk := d.GetOk("ical_url_primary")
	if iCalURLPrimaryOk {
		if typeData == "ical" {
			url := iCalURLPrimaryData.(string)
			createOptions.ICalUrlPrimary = &url
		} else {
			return diag.Errorf("ical_url_primary can not be set with type: %s", typeData)
		}
	}

	iCalURLOverridesData, iCalURLOverridesOk := d.GetOk("ical_url_overrides")
	if iCalURLOverridesOk {
		url := iCalURLOverridesData.(string)
		createOptions.ICalUrlOverrides = &url
	}

	enableWebOverridesData, enableWebOverridesOk := d.GetOk("enable_web_overrides")
	if enableWebOverridesOk {
		enable := enableWebOverridesData.(bool)
		createOptions.EnableWebOverrides = enable
	}

	shiftsData, shiftsOk := d.GetOk("shifts")
	if shiftsOk {
		if typeData == "calendar" {
			shiftsDataSlice := common.SetToStringSlice(shiftsData.(*schema.Set))
			createOptions.Shifts = &shiftsDataSlice
		} else {
			return diag.Errorf("shifts can not be set with type: %s", typeData)
		}
	}

	timeZoneData, timeZoneOk := d.GetOk("time_zone")
	if timeZoneOk {
		if typeData == "calendar" {
			createOptions.TimeZone = timeZoneData.(string)
		} else {
			return diag.Errorf("time_zone can not be set with type: %s", typeData)
		}
	}

	schedule, _, err := client.Schedules.CreateSchedule(createOptions)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(schedule.ID)

	return resourceScheduleRead(ctx, d, m)
}

func resourceScheduleUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client).OnCallClient

	nameData := d.Get("name").(string)
	teamIDData := d.Get("team_id").(string)
	slackData := d.Get("slack").([]interface{})
	typeData := d.Get("type").(string)

	updateOptions := &onCallAPI.UpdateScheduleOptions{
		Name:   nameData,
		TeamId: teamIDData,
		Slack:  expandScheduleSlack(slackData),
	}

	iCalURLPrimaryData, iCalURLPrimaryOk := d.GetOk("ical_url_primary")
	if iCalURLPrimaryOk {
		if typeData == "ical" {
			url := iCalURLPrimaryData.(string)
			updateOptions.ICalUrlPrimary = &url
		} else {
			return diag.Errorf("ical_url_primary can not be set with type: %s", typeData)
		}
	}

	iCalURLOverridesData, iCalURLOverridesOk := d.GetOk("ical_url_overrides")
	if iCalURLOverridesOk {
		url := iCalURLOverridesData.(string)
		updateOptions.ICalUrlOverrides = &url
	}

	enableWebOverridesData, enableWebOverridesOk := d.GetOk("enable_web_overrides")
	if enableWebOverridesOk {
		enable := enableWebOverridesData.(bool)
		updateOptions.EnableWebOverrides = enable
	}

	timeZoneData, timeZoneOk := d.GetOk("time_zone")
	if timeZoneOk {
		if typeData == "calendar" {
			updateOptions.TimeZone = timeZoneData.(string)
		} else {
			return diag.Errorf("time_zone can not be set with type: %s", typeData)
		}
	}

	shiftsData, shiftsOk := d.GetOk("shifts")
	if shiftsOk {
		if typeData == "calendar" {
			shiftsDataSlice := common.SetToStringSlice(shiftsData.(*schema.Set))
			updateOptions.Shifts = &shiftsDataSlice
		} else {
			return diag.Errorf("shifts can not be set with type: %s", typeData)
		}
	}

	schedule, _, err := client.Schedules.UpdateSchedule(d.Id(), updateOptions)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(schedule.ID)

	return resourceScheduleRead(ctx, d, m)
}

func resourceScheduleRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client).OnCallClient
	options := &onCallAPI.GetScheduleOptions{}
	schedule, r, err := client.Schedules.GetSchedule(d.Id(), options)
	if err != nil {
		if r != nil && r.StatusCode == http.StatusNotFound {
			log.Printf("[WARN] removing schedule %s from state because it no longer exists", d.Get("name").(string))
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	d.Set("name", schedule.Name)
	d.Set("team_id", schedule.TeamId)
	d.Set("type", schedule.Type)
	d.Set("ical_url_primary", schedule.ICalUrlPrimary)
	d.Set("ical_url_overrides", schedule.ICalUrlOverrides)
	d.Set("enable_web_overrides", schedule.EnableWebOverrides)
	d.Set("time_zone", schedule.TimeZone)
	d.Set("slack", flattenScheduleSlack(schedule.Slack))
	d.Set("shifts", schedule.Shifts)

	return nil
}

func resourceScheduleDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client).OnCallClient
	options := &onCallAPI.DeleteScheduleOptions{}
	_, err := client.Schedules.DeleteSchedule(d.Id(), options)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	return nil
}

func flattenScheduleSlack(in *onCallAPI.SlackSchedule) []map[string]interface{} {
	slack := make([]map[string]interface{}, 0, 1)

	out := make(map[string]interface{})

	if in.ChannelId != nil {
		out["channel_id"] = in.ChannelId
	}

	if in.UserGroupId != nil {
		out["user_group_id"] = in.UserGroupId
	}

	if in.ChannelId != nil || in.UserGroupId != nil {
		slack = append(slack, out)
	}
	return slack
}

func expandScheduleSlack(in []interface{}) *onCallAPI.SlackSchedule {
	slackSchedule := onCallAPI.SlackSchedule{}

	for _, r := range in {
		inputMap := r.(map[string]interface{})
		if inputMap["channel_id"] != "" {
			channelID := inputMap["channel_id"].(string)
			slackSchedule.ChannelId = &channelID
		}
		if inputMap["user_group_id"] != "" {
			userGroupID := inputMap["user_group_id"].(string)
			slackSchedule.UserGroupId = &userGroupID
		}
	}

	return &slackSchedule
}
