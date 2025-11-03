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

var escalationOptions = []string{
	"wait",
	"notify_persons",
	"notify_person_next_each_time",
	"notify_on_call_from_schedule",
	"trigger_webhook",
	"notify_user_group",
	"resolve",
	"notify_whole_channel",
	"notify_if_time_from_to",
	"notify_if_num_alerts_in_window",
	"repeat_escalation",
	"notify_team_members",
	"declare_incident",
}

var escalationOptionsVerbal = strings.Join(escalationOptions, ", ")

func resourceEscalation() *common.Resource {
	schema := &schema.Resource{
		Description: `
* [Official documentation](https://grafana.com/docs/oncall/latest/configure/escalation-chains-and-routes/)
* [HTTP API](https://grafana.com/docs/oncall/latest/oncall-api-reference/escalation_policies/)
`,
		CreateContext: withClient[schema.CreateContextFunc](resourceEscalationCreate),
		ReadContext:   withClient[schema.ReadContextFunc](resourceEscalationRead),
		UpdateContext: withClient[schema.UpdateContextFunc](resourceEscalationUpdate),
		DeleteContext: withClient[schema.DeleteContextFunc](resourceEscalationDelete),
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"escalation_chain_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The ID of the escalation chain.",
			},
			"position": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "The position of the escalation step (starts from 0).",
			},
			"type": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice(escalationOptions, false),
				Description:  fmt.Sprintf("The type of escalation policy. Can be %s", escalationOptionsVerbal),
			},
			"important": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Will activate \"important\" personal notification rules. Actual for steps: notify_persons, notify_person_next_each_time, notify_on_call_from_schedule, notify_user_group and notify_team_members",
			},
			"duration": {
				Type:     schema.TypeInt,
				Optional: true,
				ConflictsWith: []string{
					"notify_on_call_from_schedule",
					"persons_to_notify",
					"persons_to_notify_next_each_time",
					"notify_to_team_members",
					"action_to_trigger",
					"group_to_notify",
					"notify_if_time_from",
					"notify_if_time_to",
					"num_alerts_in_window",
					"num_minutes_in_window",
				},
				ValidateFunc: validation.IntBetween(60, 86400),
				Description:  "The duration of delay for wait type step. (60-86400) seconds",
			},
			"notify_on_call_from_schedule": {
				Type:     schema.TypeString,
				Optional: true,
				ConflictsWith: []string{
					"duration",
					"persons_to_notify",
					"persons_to_notify_next_each_time",
					"notify_to_team_members",
					"action_to_trigger",
					"group_to_notify",
					"notify_if_time_from",
					"notify_if_time_to",
					"num_alerts_in_window",
					"num_minutes_in_window",
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
					"notify_to_team_members",
					"action_to_trigger",
					"group_to_notify",
					"notify_if_time_from",
					"notify_if_time_to",
					"num_alerts_in_window",
					"num_minutes_in_window",
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
					"notify_to_team_members",
					"action_to_trigger",
					"group_to_notify",
					"notify_if_time_from",
					"notify_if_time_to",
				},
				Description: "The list of ID's of users for notify_person_next_each_time type step.",
			},
			"notify_to_team_members": {
				Type:     schema.TypeString,
				Optional: true,
				ConflictsWith: []string{
					"duration",
					"notify_on_call_from_schedule",
					"persons_to_notify",
					"persons_to_notify_next_each_time",
					"action_to_trigger",
					"group_to_notify",
					"notify_if_time_from",
					"notify_if_time_to",
				},
				Description: "The ID of a Team for a notify_team_members type step.",
			},
			"action_to_trigger": {
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
				Description: "The ID of an Action for trigger_webhook type step.",
			},
			"group_to_notify": {
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
			"notify_if_time_from": {
				Type:     schema.TypeString,
				Optional: true,
				ConflictsWith: []string{
					"duration",
					"notify_on_call_from_schedule",
					"persons_to_notify",
					"persons_to_notify_next_each_time",
					"notify_to_team_members",
					"action_to_trigger",
				},
				RequiredWith: []string{
					"notify_if_time_to",
				},
				Description: "The beginning of the time interval for notify_if_time_from_to type step in UTC (for example 08:00:00Z).",
			},
			"notify_if_time_to": {
				Type:     schema.TypeString,
				Optional: true,
				ConflictsWith: []string{
					"duration",
					"notify_on_call_from_schedule",
					"persons_to_notify",
					"persons_to_notify_next_each_time",
					"notify_to_team_members",
					"action_to_trigger",
				},
				RequiredWith: []string{
					"notify_if_time_from",
				},
				Description: "The end of the time interval for notify_if_time_from_to type step in UTC (for example 18:00:00Z).",
			},
			"num_alerts_in_window": {
				Type:     schema.TypeInt,
				Optional: true,
				ConflictsWith: []string{
					"duration",
					"notify_on_call_from_schedule",
					"persons_to_notify",
					"persons_to_notify_next_each_time",
					"notify_to_team_members",
					"action_to_trigger",
					"group_to_notify",
				},
				RequiredWith: []string{
					"num_minutes_in_window",
				},
				Description: "Number of alerts that must occur within the time window to continue escalation for notify_if_num_alerts_in_window type step.",
			},
			"num_minutes_in_window": {
				Type:     schema.TypeInt,
				Optional: true,
				ConflictsWith: []string{
					"duration",
					"notify_on_call_from_schedule",
					"persons_to_notify",
					"persons_to_notify_next_each_time",
					"notify_to_team_members",
					"action_to_trigger",
					"group_to_notify",
				},
				RequiredWith: []string{
					"num_alerts_in_window",
				},
				Description: "Time window in minutes to count alerts for notify_if_num_alerts_in_window type step.",
			},
			"severity": {
				Type:     schema.TypeString,
				Optional: true,
				ConflictsWith: []string{
					"duration",
					"notify_on_call_from_schedule",
					"persons_to_notify",
					"persons_to_notify_next_each_time",
					"notify_to_team_members",
					"notify_if_time_from",
					"notify_if_time_to",
					"action_to_trigger",
				},
				Description: "The severity of the incident for declare_incident type step.",
			},
		},
	}

	return common.NewLegacySDKResource(
		common.CategoryOnCall,
		"grafana_oncall_escalation",
		resourceID,
		schema,
	).
		WithLister(oncallListerFunction(listEscalations))
}

func listEscalations(client *onCallAPI.Client, listOptions onCallAPI.ListOptions) (ids []string, nextPage *string, err error) {
	resp, _, err := client.Escalations.ListEscalations(&onCallAPI.ListEscalationOptions{ListOptions: listOptions})
	if err != nil {
		return nil, nil, err
	}
	for _, i := range resp.Escalations {
		ids = append(ids, i.ID)
	}
	return ids, resp.Next, nil
}

func resourceEscalationCreate(ctx context.Context, d *schema.ResourceData, client *onCallAPI.Client) diag.Diagnostics {
	escalationChainIDData := d.Get("escalation_chain_id").(string)

	createOptions := &onCallAPI.CreateEscalationOptions{
		EscalationChainId: escalationChainIDData,
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
			return diag.Errorf("duration can not be set with type: %s", typeData)
		}
	}

	personsToNotifyData, personsToNotifyDataOk := d.GetOk("persons_to_notify")
	if personsToNotifyDataOk {
		if typeData == "notify_persons" {
			personsToNotifyDataSlice := common.SetToStringSlice(personsToNotifyData.(*schema.Set))
			createOptions.PersonsToNotify = &personsToNotifyDataSlice
		} else {
			return diag.Errorf("persons_to_notify can not be set with type: %s", typeData)
		}
	}

	teamToNotifyData, teamToNotifyDataOk := d.GetOk("notify_to_team_members")
	if teamToNotifyDataOk {
		if typeData == "notify_team_members" {
			createOptions.TeamToNotify = teamToNotifyData.(string)
		} else {
			return diag.Errorf("notify_to_team_members can not be set with type: %s", typeData)
		}
	}

	severityData, severityDataOk := d.GetOk("severity")
	if severityDataOk {
		if typeData == "declare_incident" {
			createOptions.Severity = severityData.(string)
		} else {
			return diag.Errorf("severity can not be set with type: %s", typeData)
		}
	}

	notifyOnCallFromScheduleData, notifyOnCallFromScheduleDataOk := d.GetOk("notify_on_call_from_schedule")
	if notifyOnCallFromScheduleDataOk {
		if typeData == "notify_on_call_from_schedule" {
			createOptions.NotifyOnCallFromSchedule = notifyOnCallFromScheduleData.(string)
		} else {
			return diag.Errorf("notify_on_call_from_schedule can not be set with type: %s", typeData)
		}
	}

	personsToNotifyNextEachTimeData, personsToNotifyNextEachTimeDataOk := d.GetOk("persons_to_notify_next_each_time")
	if personsToNotifyNextEachTimeDataOk {
		if typeData == "notify_person_next_each_time" {
			personsToNotifyNextEachTimeDataSlice := common.SetToStringSlice(personsToNotifyNextEachTimeData.(*schema.Set))
			createOptions.PersonsToNotify = &personsToNotifyNextEachTimeDataSlice
		} else {
			return diag.Errorf("persons_to_notify_next_each_time can not be set with type: %s", typeData)
		}
	}

	notifyToGroupData, notifyToGroupDataOk := d.GetOk("group_to_notify")
	if notifyToGroupDataOk {
		if typeData == "notify_user_group" {
			createOptions.GroupToNotify = notifyToGroupData.(string)
		} else {
			return diag.Errorf("notify_to_group can not be set with type: %s", typeData)
		}
	}

	actionToTriggerData, actionToTriggerDataOk := d.GetOk("action_to_trigger")
	if actionToTriggerDataOk {
		if typeData == "trigger_webhook" {
			createOptions.ActionToTrigger = actionToTriggerData.(string)
		} else {
			return diag.Errorf("action to trigger can not be set with type: %s", typeData)
		}
	}

	notifyIfTimeFromData, notifyIfTimeFromDataOk := d.GetOk("notify_if_time_from")
	if notifyIfTimeFromDataOk {
		if typeData == "notify_if_time_from_to" {
			createOptions.NotifyIfTimeFrom = notifyIfTimeFromData.(string)
		} else {
			return diag.Errorf("notify_if_time_from can not be set with type: %s", typeData)
		}
	}

	notifyIfTimeToData, notifyIfTimeToDataOk := d.GetOk("notify_if_time_to")
	if notifyIfTimeToDataOk {
		if typeData == "notify_if_time_from_to" {
			createOptions.NotifyIfTimeTo = notifyIfTimeToData.(string)
		} else {
			return diag.Errorf("notify_if_time_to can not be set with type: %s", typeData)
		}
	}

	numAlertsInWindowData, numAlertsInWindowDataOk := d.GetOk("num_alerts_in_window")
	if numAlertsInWindowDataOk {
		if typeData == "notify_if_num_alerts_in_window" {
			createOptions.NumAlertsInWindow = numAlertsInWindowData.(int)
		} else {
			return diag.Errorf("num_alerts_in_window can not be set with type: %s", typeData)
		}
	}

	numMinutesInWindowData, numMinutesInWindowDataOk := d.GetOk("num_minutes_in_window")
	if numMinutesInWindowDataOk {
		if typeData == "notify_if_num_alerts_in_window" {
			createOptions.NumMinutesInWindow = numMinutesInWindowData.(int)
		} else {
			return diag.Errorf("num_minutes_in_window can not be set with type: %s", typeData)
		}
	}

	importanceData := d.Get("important").(bool)
	createOptions.Important = &importanceData

	positionData := d.Get("position").(int)
	createOptions.Position = &positionData

	escalation, _, err := client.Escalations.CreateEscalation(createOptions)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(escalation.ID)

	return resourceEscalationRead(ctx, d, client)
}

func resourceEscalationRead(ctx context.Context, d *schema.ResourceData, client *onCallAPI.Client) diag.Diagnostics {
	escalation, r, err := client.Escalations.GetEscalation(d.Id(), &onCallAPI.GetEscalationOptions{})
	if err != nil {
		if r != nil && r.StatusCode == http.StatusNotFound {
			return common.WarnMissing("escalation", d)
		}
		return diag.FromErr(err)
	}

	d.Set("escalation_chain_id", escalation.EscalationChainId)
	d.Set("position", escalation.Position)
	d.Set("type", escalation.Type)
	if escalation.Duration != nil {
		d.Set("duration", escalation.Duration)
	}
	if escalation.NotifyOnCallFromSchedule != nil {
		d.Set("notify_on_call_from_schedule", escalation.NotifyOnCallFromSchedule)
	}
	if escalation.PersonsToNotify != nil {
		d.Set("persons_to_notify", escalation.PersonsToNotify)
	}
	if escalation.PersonsToNotifyEachTime != nil {
		d.Set("persons_to_notify_next_each_time", escalation.PersonsToNotifyEachTime)
	}
	if escalation.TeamToNotify != nil {
		d.Set("notify_to_team_members", escalation.TeamToNotify)
	}
	if escalation.Severity != nil {
		d.Set("severity", escalation.Severity)
	}
	if escalation.GroupToNotify != nil {
		d.Set("group_to_notify", escalation.GroupToNotify)
	}
	if escalation.ActionToTrigger != nil {
		d.Set("action_to_trigger", escalation.ActionToTrigger)
	}
	if escalation.Important != nil {
		d.Set("important", escalation.Important)
	}
	if escalation.NotifyIfTimeFrom != nil {
		d.Set("notify_if_time_from", escalation.NotifyIfTimeFrom)
	}
	if escalation.NotifyIfTimeTo != nil {
		d.Set("notify_if_time_to", escalation.NotifyIfTimeTo)
	}
	if escalation.NumAlertsInWindow != nil {
		d.Set("num_alerts_in_window", escalation.NumAlertsInWindow)
	}
	if escalation.NumMinutesInWindow != nil {
		d.Set("num_minutes_in_window", escalation.NumMinutesInWindow)
	}

	return nil
}

func resourceEscalationUpdate(ctx context.Context, d *schema.ResourceData, client *onCallAPI.Client) diag.Diagnostics {
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
			personsToNotifyDataSlice := common.SetToStringSlice(personsToNotifyData.(*schema.Set))
			updateOptions.PersonsToNotify = &personsToNotifyDataSlice
		}
	}

	teamToNotifyData, teamToNotifyDataOk := d.GetOk("notify_to_team_members")
	if teamToNotifyDataOk {
		if typeData == "notify_team_members" {
			updateOptions.TeamToNotify = teamToNotifyData.(string)
		}
	}

	severityData, severityDataOk := d.GetOk("severity")
	if severityDataOk {
		if typeData == "declare_incident" {
			updateOptions.Severity = severityData.(string)
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
			personsToNotifyNextEachTimeDataSlice := common.SetToStringSlice(personsToNotifyNextEachTimeData.(*schema.Set))
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
		if typeData == "trigger_webhook" {
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

	numAlertsInWindowData, numAlertsInWindowDataOk := d.GetOk("num_alerts_in_window")
	if numAlertsInWindowDataOk {
		if typeData == "notify_if_num_alerts_in_window" {
			updateOptions.NumAlertsInWindow = numAlertsInWindowData.(int)
		}
	}

	numMinutesInWindowData, numMinutesInWindowDataOk := d.GetOk("num_minutes_in_window")
	if numMinutesInWindowDataOk {
		if typeData == "notify_if_num_alerts_in_window" {
			updateOptions.NumMinutesInWindow = numMinutesInWindowData.(int)
		}
	}

	positionData := d.Get("position").(int)
	updateOptions.Position = &positionData

	importanceData := d.Get("important").(bool)
	updateOptions.Important = &importanceData

	escalation, _, err := client.Escalations.UpdateEscalation(d.Id(), updateOptions)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(escalation.ID)
	return resourceEscalationRead(ctx, d, client)
}

func resourceEscalationDelete(ctx context.Context, d *schema.ResourceData, client *onCallAPI.Client) diag.Diagnostics {
	_, err := client.Escalations.DeleteEscalation(d.Id(), &onCallAPI.DeleteEscalationOptions{})
	return diag.FromErr(err)
}
