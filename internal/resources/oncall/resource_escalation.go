package oncall

import (
	"context"
	"fmt"
	"net/http"
	"slices"
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
	"notify_next_on_call_from_schedule",
	"trigger_webhook",
	"notify_user_group",
	"resolve",
	"notify_whole_channel",
	"notify_if_time_from_to",
	"notify_if_num_alerts_in_window",
	"repeat_escalation",
	"notify_team_members",
	"declare_incident",
	"pause_until_team_hours",
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
				Description: "Will activate \"important\" personal notification rules. Actual for steps: notify_persons, notify_person_next_each_time, notify_on_call_from_schedule, notify_next_on_call_from_schedule, notify_user_group and notify_team_members",
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
				Description: "ID of a Schedule for notify_on_call_from_schedule or notify_next_on_call_from_schedule type step.",
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
				Description: "The ID of a Team for a notify_team_members or pause_until_team_hours type step.",
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

// escalationFieldRule describes which escalation step types a field may be set with.
// label is the name used in the error message.
type escalationFieldRule struct {
	key   string
	label string
	types []string
}

var escalationFieldRules = []escalationFieldRule{
	{"duration", "duration", []string{"wait"}},
	{"persons_to_notify", "persons_to_notify", []string{"notify_persons"}},
	{"notify_to_team_members", "notify_to_team_members", []string{"notify_team_members", "pause_until_team_hours"}},
	{"severity", "severity", []string{"declare_incident"}},
	{"notify_on_call_from_schedule", "notify_on_call_from_schedule", []string{"notify_on_call_from_schedule", "notify_next_on_call_from_schedule"}},
	{"persons_to_notify_next_each_time", "persons_to_notify_next_each_time", []string{"notify_person_next_each_time"}},
	{"group_to_notify", "notify_to_group", []string{"notify_user_group"}},
	{"action_to_trigger", "action to trigger", []string{"trigger_webhook"}},
	{"notify_if_time_from", "notify_if_time_from", []string{"notify_if_time_from_to"}},
	{"notify_if_time_to", "notify_if_time_to", []string{"notify_if_time_from_to"}},
	{"num_alerts_in_window", "num_alerts_in_window", []string{"notify_if_num_alerts_in_window"}},
	{"num_minutes_in_window", "num_minutes_in_window", []string{"notify_if_num_alerts_in_window"}},
}

// validateEscalationFields checks that every configured field is allowed for the given step type.
func validateEscalationFields(d *schema.ResourceData, typeData interface{}) diag.Diagnostics {
	for _, rule := range escalationFieldRules {
		if _, ok := d.GetOk(rule.key); ok && !slices.Contains(rule.types, fmt.Sprint(typeData)) {
			return diag.Errorf("%s can not be set with type: %s", rule.label, typeData)
		}
	}
	if _, ok := d.GetOk("notify_to_team_members"); typeData == "pause_until_team_hours" && !ok {
		return diag.Errorf("notify_to_team_members is required for type: %s", typeData)
	}
	return nil
}

func resourceEscalationCreate(ctx context.Context, d *schema.ResourceData, client *onCallAPI.Client) diag.Diagnostics {
	typeData, typeOk := d.GetOk("type")
	if diags := validateEscalationFields(d, typeData); diags != nil {
		return diags
	}

	createOptions := &onCallAPI.CreateEscalationOptions{
		EscalationChainId: d.Get("escalation_chain_id").(string),
		ManualOrder:       true,
	}
	if typeOk {
		t := typeData.(string)
		createOptions.Type = &t
	}
	if v, ok := d.GetOk("duration"); ok {
		createOptions.Duration = v.(int)
	}
	if v, ok := d.GetOk("persons_to_notify"); ok {
		persons := common.SetToStringSlice(v.(*schema.Set))
		createOptions.PersonsToNotify = &persons
	}
	if v, ok := d.GetOk("notify_to_team_members"); ok {
		createOptions.TeamToNotify = v.(string)
	}
	if v, ok := d.GetOk("severity"); ok {
		createOptions.Severity = v.(string)
	}
	if v, ok := d.GetOk("notify_on_call_from_schedule"); ok {
		createOptions.NotifyOnCallFromSchedule = v.(string)
	}
	if v, ok := d.GetOk("persons_to_notify_next_each_time"); ok {
		persons := common.SetToStringSlice(v.(*schema.Set))
		createOptions.PersonsToNotify = &persons
	}
	if v, ok := d.GetOk("group_to_notify"); ok {
		createOptions.GroupToNotify = v.(string)
	}
	if v, ok := d.GetOk("action_to_trigger"); ok {
		createOptions.ActionToTrigger = v.(string)
	}
	if v, ok := d.GetOk("notify_if_time_from"); ok {
		createOptions.NotifyIfTimeFrom = v.(string)
	}
	if v, ok := d.GetOk("notify_if_time_to"); ok {
		createOptions.NotifyIfTimeTo = v.(string)
	}
	if v, ok := d.GetOk("num_alerts_in_window"); ok {
		createOptions.NumAlertsInWindow = v.(int)
	}
	if v, ok := d.GetOk("num_minutes_in_window"); ok {
		createOptions.NumMinutesInWindow = v.(int)
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
		if typeData == "notify_team_members" || typeData == "pause_until_team_hours" {
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
		if typeData == "notify_on_call_from_schedule" || typeData == "notify_next_on_call_from_schedule" {
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
