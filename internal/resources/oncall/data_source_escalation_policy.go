package oncall

import (
	"context"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceEscalationPolicy() *common.DataSource {
	schema := &schema.Resource{
		Description: `
* [HTTP API](https://grafana.com/docs/oncall/latest/oncall-api-reference/escalation_policies/)
`,
		ReadContext: withClient[schema.ReadContextFunc](dataSourceEscalationPolicyRead),
		Schema: map[string]*schema.Schema{
			"escalation_chain_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The ID of the escalation chain.",
			},
			"position": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "The position of the escalation step (starts from 0).",
			},
			"type": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The type of escalation policy.",
			},
			"important": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the step uses important notification rules.",
			},
			"duration": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The duration of delay for wait type step.",
			},
			"notify_on_call_from_schedule": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "ID of a Schedule for notify_on_call_from_schedule type step.",
			},
			"persons_to_notify": {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed:    true,
				Description: "The list of ID's of users for notify_persons type step.",
			},
			"persons_to_notify_next_each_time": {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed:    true,
				Description: "The list of ID's of users for notify_person_next_each_time type step.",
			},
			"notify_to_team_members": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of a Team for a notify_team_members type step.",
			},
			"action_to_trigger": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of an Action for trigger_webhook type step.",
			},
			"group_to_notify": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of a User Group for notify_user_group type step.",
			},
			"notify_if_time_from": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The beginning of the time interval for notify_if_time_from_to type step.",
			},
			"notify_if_time_to": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The end of the time interval for notify_if_time_from_to type step.",
			},
			"num_alerts_in_window": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Number of alerts for notify_if_num_alerts_in_window type step.",
			},
			"num_minutes_in_window": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Time window in minutes for notify_if_num_alerts_in_window type step.",
			},
			"severity": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The severity of the incident for declare_incident type step.",
			},
		},
	}
	return common.NewLegacySDKDataSource(common.CategoryOnCall, "grafana_oncall_escalation_policy", schema)
}

func dataSourceEscalationPolicyRead(ctx context.Context, d *schema.ResourceData, client *onCallAPI.Client) diag.Diagnostics {
	escalationChainID := d.Get("escalation_chain_id").(string)
	position := d.Get("position").(int)

	options := &onCallAPI.ListEscalationOptions{}

	escalationsResponse, _, err := client.Escalations.ListEscalations(options)
	if err != nil {
		return diag.FromErr(err)
	}

	for _, escalation := range escalationsResponse.Escalations {
		if escalation.EscalationChainId == escalationChainID && escalation.Position == position {
			d.SetId(escalation.ID)
			d.Set("type", escalation.Type)
			if escalation.Important != nil {
				d.Set("important", escalation.Important)
			}
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
			if escalation.ActionToTrigger != nil {
				d.Set("action_to_trigger", escalation.ActionToTrigger)
			}
			if escalation.GroupToNotify != nil {
				d.Set("group_to_notify", escalation.GroupToNotify)
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
			if escalation.Severity != nil {
				d.Set("severity", escalation.Severity)
			}
			return nil
		}
	}

	return diag.Errorf("couldn't find an escalation policy matching: escalation_chain_id=%s, position=%d", escalationChainID, position)
}
