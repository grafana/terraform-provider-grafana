package grafana

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

var escalationOptions = []string{
	"wait",
	"notify_persons",
	"notify_person_next_each_time",
	"notify_on_call_from_schedule",
	"trigger_action",
	"notify_user_group",
	"resolve",
	"notify_whole_channel",
	"notify_if_time_from_to",
}

var escalationOptionsVerbal = strings.Join(escalationOptions, ", ")

var durationOptions = []int{
	60,
	300,
	900,
	1800,
	3600,
}

func ResourceOnCallEscalation() *schema.Resource {
	return &schema.Resource{
		Description: `
* [Official documentation](https://grafana.com/docs/grafana-cloud/oncall/escalation-policies/)
* [HTTP API](https://grafana.com/docs/grafana-cloud/oncall/oncall-api-reference/escalation_policies/)
`,
		Create: resourceEscalationCreate,
		Read:   resourceEscalationRead,
		Update: resourceEscalationUpdate,
		Delete: resourceEscalationDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"escalation_chain_id": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The ID of the escalation chain.",
			},
			"position": &schema.Schema{
				Type:        schema.TypeInt,
				Required:    true,
				Description: "The position of the escalation step (starts from 0).",
			},
			"type": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice(escalationOptions, false),
				Description:  fmt.Sprintf("The type of escalation policy. Can be %s", escalationOptionsVerbal),
			},
			"important": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Will activate \"important\" personal notification rules. Actual for steps: notify_persons, notify_on_call_from_schedule and notify_user_group",
			},
			"duration": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ConflictsWith: []string{
					"notify_on_call_from_schedule",
					"persons_to_notify",
					"persons_to_notify_next_each_time",
					"action_to_trigger",
					"group_to_notify",
					"notify_if_time_from",
					"notify_if_time_to",
				},
				ValidateFunc: validation.IntInSlice(durationOptions),
				Description:  "The duration of delay for wait type step.",
			},
			"notify_on_call_from_schedule": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ConflictsWith: []string{
					"duration",
					"persons_to_notify",
					"persons_to_notify_next_each_time",
					"action_to_trigger",
					"group_to_notify",
					"notify_if_time_from",
					"notify_if_time_to",
				},
				Description: "ID of a Schedule for notify_on_call_from_schedule type step.",
			},
			"persons_to_notify": {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional: true,
				ConflictsWith: []string{
					"duration",
					"notify_on_call_from_schedule",
					"persons_to_notify_next_each_time",
					"action_to_trigger",
					"group_to_notify",
					"notify_if_time_from",
					"notify_if_time_to",
				},
				Description: "The list of ID's of users for notify_persons type step.",
			},
			"persons_to_notify_next_each_time": {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional: true,
				ConflictsWith: []string{
					"duration",
					"notify_on_call_from_schedule",
					"persons_to_notify",
					"action_to_trigger",
					"group_to_notify",
					"notify_if_time_from",
					"notify_if_time_to",
				},
				Description: "The list of ID's of users for notify_person_next_each_time type step.",
			},
			"action_to_trigger": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ConflictsWith: []string{
					"duration",
					"notify_on_call_from_schedule",
					"persons_to_notify",
					"persons_to_notify_next_each_time",
					"group_to_notify",
					"notify_if_time_from",
					"notify_if_time_to",
				},
				Description: "The ID of an Action for trigger_action type step.",
			},
			"group_to_notify": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ConflictsWith: []string{
					"duration",
					"notify_on_call_from_schedule",
					"persons_to_notify",
					"persons_to_notify_next_each_time",
					"action_to_trigger",
					"notify_if_time_from",
					"notify_if_time_to",
				},
				Description: "The ID of a User Group for notify_user_group type step.",
			},
			"notify_if_time_from": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ConflictsWith: []string{
					"duration",
					"notify_on_call_from_schedule",
					"persons_to_notify",
					"persons_to_notify_next_each_time",
					"action_to_trigger",
				},
				RequiredWith: []string{
					"notify_if_time_to",
				},
				Description: "The beginning of the time interval for notify_if_time_from_to type step in UTC (for example 08:00:00Z).",
			},
			"notify_if_time_to": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ConflictsWith: []string{
					"duration",
					"notify_on_call_from_schedule",
					"persons_to_notify",
					"persons_to_notify_next_each_time",
					"action_to_trigger",
				},
				RequiredWith: []string{
					"notify_if_time_from",
				},
				Description: "The end of the time interval for notify_if_time_from_to type step in UTC (for example 18:00:00Z).",
			},
		},
	}
}

func resourceEscalationCreate(d *schema.ResourceData, m interface{}) error {
	client := m.(*client).onCallAPI
	if client == nil {
		return errors.New("grafana OnCall api client is not configured")
	}

	escalationChainIdData := d.Get("escalation_chain_id").(string)

	createOptions := &onCallAPI.CreateEscalationOptions{
		EscalationChainId: escalationChainIdData,
		ManualOrder:       true,
	}

	typeData, typeOk := d.GetOk("type")
	if typeOk {
		t := typeData.(string)
		createOptions.Type = &t
	}

	durationData, durationOk := d.GetOk("duration")
	if durationOk {
		if typeData == "wait" {
			createOptions.Duration = durationData.(int)
		} else {
			return fmt.Errorf("duration can not be set with type: %s", typeData)
		}
	}

	personsToNotifyData, personsToNotifyDataOk := d.GetOk("persons_to_notify")
	if personsToNotifyDataOk {
		if typeData == "notify_persons" {
			personsToNotifyDataSlice := setToStringSlice(personsToNotifyData.(*schema.Set))
			createOptions.PersonsToNotify = &personsToNotifyDataSlice
		} else {
			return fmt.Errorf("persons_to_notify can not be set with type: %s", typeData)
		}
	}

	notifyOnCallFromScheduleData, notifyOnCallFromScheduleDataOk := d.GetOk("notify_on_call_from_schedule")
	if notifyOnCallFromScheduleDataOk {
		if typeData == "notify_on_call_from_schedule" {
			createOptions.NotifyOnCallFromSchedule = notifyOnCallFromScheduleData.(string)
		} else {
			return fmt.Errorf("notify_on_call_from_schedule can not be set with type: %s", typeData)
		}
	}

	personsToNotifyNextEachTimeData, personsToNotifyNextEachTimeDataOk := d.GetOk("persons_to_notify_next_each_time")
	if personsToNotifyNextEachTimeDataOk {
		if typeData == "notify_person_next_each_time" {
			personsToNotifyNextEachTimeDataSlice := setToStringSlice(personsToNotifyNextEachTimeData.(*schema.Set))
			createOptions.PersonsToNotify = &personsToNotifyNextEachTimeDataSlice
		} else {
			return fmt.Errorf("persons_to_notify_next_each_time can not be set with type: %s", typeData)
		}
	}

	notifyToGroupData, notifyToGroupDataOk := d.GetOk("group_to_notify")
	if notifyToGroupDataOk {
		if typeData == "notify_user_group" {
			createOptions.GroupToNotify = notifyToGroupData.(string)
		} else {
			return fmt.Errorf("notify_to_group can not be set with type: %s", typeData)
		}
	}

	actionToTriggerData, actionToTriggerDataOk := d.GetOk("action_to_trigger")
	if actionToTriggerDataOk {
		if typeData == "trigger_action" {
			createOptions.ActionToTrigger = actionToTriggerData.(string)
		} else {
			return fmt.Errorf("action to trigger can not be set with type: %s", typeData)
		}
	}

	notifyIfTimeFromData, notifyIfTimeFromDataOk := d.GetOk("notify_if_time_from")
	if notifyIfTimeFromDataOk {
		if typeData == "notify_if_time_from_to" {
			createOptions.NotifyIfTimeFrom = notifyIfTimeFromData.(string)
		} else {
			return fmt.Errorf("notify_if_time_from can not be set with type: %s", typeData)
		}
	}

	notifyIfTimeToData, notifyIfTimeToDataOk := d.GetOk("notify_if_time_to")
	if notifyIfTimeToDataOk {
		if typeData == "notify_if_time_from_to" {
			createOptions.NotifyIfTimeTo = notifyIfTimeToData.(string)
		} else {
			return fmt.Errorf("notify_if_time_to can not be set with type: %s", typeData)
		}
	}

	importanceData := d.Get("important").(bool)
	createOptions.Important = &importanceData

	positionData := d.Get("position").(int)
	createOptions.Position = &positionData

	escalation, _, err := client.Escalations.CreateEscalation(createOptions)
	if err != nil {
		return err
	}

	d.SetId(escalation.ID)

	return resourceEscalationRead(d, m)
}

func resourceEscalationRead(d *schema.ResourceData, m interface{}) error {
	client := m.(*client).onCallAPI
	if client == nil {
		return errors.New("grafana OnCall api client is not configured")
	}

	escalation, r, err := client.Escalations.GetEscalation(d.Id(), &onCallAPI.GetEscalationOptions{})
	if err != nil {
		if r != nil && r.StatusCode == http.StatusNotFound {
			log.Printf("[WARN] removing escalation %s from state because it no longer exists", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("escalation_chain_id", escalation.EscalationChainId)
	d.Set("position", escalation.Position)
	d.Set("type", escalation.Type)
	d.Set("duration", escalation.Duration)
	d.Set("notify_on_call_from_schedule", escalation.NotifyOnCallFromSchedule)
	d.Set("persons_to_notify", escalation.PersonsToNotify)
	d.Set("persons_to_notify_next_each_time", escalation.PersonsToNotifyEachTime)
	d.Set("group_to_notify", escalation.GroupToNotify)
	d.Set("action_to_trigger", escalation.ActionToTrigger)
	d.Set("important", escalation.Important)
	d.Set("notify_if_time_from", escalation.NotifyIfTimeFrom)
	d.Set("notify_if_time_to", escalation.NotifyIfTimeTo)

	return nil
}

func resourceEscalationUpdate(d *schema.ResourceData, m interface{}) error {
	client := m.(*client).onCallAPI
	if client == nil {
		return errors.New("grafana OnCall api client is not configured")
	}

	updateOptions := &onCallAPI.UpdateEscalationOptions{
		ManualOrder: true,
	}

	typeData, typeOk := d.GetOk("type")
	if typeOk {
		t := typeData.(string)
		updateOptions.Type = &t
	}

	durationData, durationOk := d.GetOk("duration")
	if durationOk {
		if typeData == "wait" {
			updateOptions.Duration = durationData.(int)
		}
	}

	personsToNotifyData, personsToNotifyDataOk := d.GetOk("persons_to_notify")
	if personsToNotifyDataOk {
		if typeData == "notify_persons" {
			personsToNotifyDataSlice := setToStringSlice(personsToNotifyData.(*schema.Set))
			updateOptions.PersonsToNotify = &personsToNotifyDataSlice
		}
	}

	notifyOnCallFromScheduleData, notifyOnCallFromScheduleDataOk := d.GetOk("notify_on_call_from_schedule")
	if notifyOnCallFromScheduleDataOk {
		if typeData == "notify_on_call_from_schedule" {
			updateOptions.NotifyOnCallFromSchedule = notifyOnCallFromScheduleData.(string)
		}
	}

	personsToNotifyNextEachTimeData, personsToNotifyNextEachTimeDataOk := d.GetOk("persons_to_notify_next_each_time")
	if personsToNotifyNextEachTimeDataOk {
		if typeData == "notify_person_next_each_time" {
			personsToNotifyNextEachTimeDataSlice := setToStringSlice(personsToNotifyNextEachTimeData.(*schema.Set))
			updateOptions.PersonsToNotify = &personsToNotifyNextEachTimeDataSlice
		}
	}

	notifyToGroupData, notifyToGroupDataOk := d.GetOk("group_to_notify")
	if notifyToGroupDataOk {
		if typeData == "notify_user_group" {
			updateOptions.GroupToNotify = notifyToGroupData.(string)
		}
	}

	actionToTriggerData, actionToTriggerDataOk := d.GetOk("action_to_trigger")
	if actionToTriggerDataOk {
		if typeData == "trigger_action" {
			updateOptions.ActionToTrigger = actionToTriggerData.(string)
		}
	}

	notifyIfTimeFromData, notifyIfTimeFromDataOk := d.GetOk("notify_if_time_from")
	if notifyIfTimeFromDataOk {
		if typeData == "notify_if_time_from_to" {
			updateOptions.NotifyIfTimeFrom = notifyIfTimeFromData.(string)
		}
	}

	notifyIfTimeToData, notifyIfTimeToDataOk := d.GetOk("notify_if_time_to")
	if notifyIfTimeToDataOk {
		if typeData == "notify_if_time_from_to" {
			updateOptions.NotifyIfTimeTo = notifyIfTimeToData.(string)
		}
	}

	positionData := d.Get("position").(int)
	updateOptions.Position = &positionData

	importanceData := d.Get("important").(bool)
	updateOptions.Important = &importanceData

	escalation, _, err := client.Escalations.UpdateEscalation(d.Id(), updateOptions)
	if err != nil {
		return err
	}

	d.SetId(escalation.ID)
	return resourceEscalationRead(d, m)
}

func resourceEscalationDelete(d *schema.ResourceData, m interface{}) error {
	client := m.(*client).onCallAPI
	if client == nil {
		return errors.New("grafana OnCall api client is not configured")
	}

	_, err := client.Escalations.DeleteEscalation(d.Id(), &onCallAPI.DeleteEscalationOptions{})
	if err != nil {
		return err
	}

	d.SetId("")

	return nil
}
