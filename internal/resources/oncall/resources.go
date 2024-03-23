package oncall

import (
	"context"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/grafana/terraform-provider-grafana/v2/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type crudWithClientFunc func(ctx context.Context, d *schema.ResourceData, client *onCallAPI.Client) diag.Diagnostics

func withClient[T schema.CreateContextFunc | schema.UpdateContextFunc | schema.ReadContextFunc | schema.DeleteContextFunc](f crudWithClientFunc) T {
	return func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
		client := meta.(*common.Client).OnCallClient
		if client == nil {
			return diag.Errorf("the OnCall client is required for this resource. Set the oncall_access_token provider attribute")
		}
		return f(ctx, d, client)
	}
}

var DatasourcesMap = map[string]*schema.Resource{
	"grafana_oncall_user":             dataSourceUser(),
	"grafana_oncall_escalation_chain": dataSourceEscalationChain(),
	"grafana_oncall_schedule":         dataSourceSchedule(),
	"grafana_oncall_slack_channel":    dataSourceSlackChannel(),
	"grafana_oncall_action":           dataSourceAction(), // deprecated
	"grafana_oncall_outgoing_webhook": dataSourceOutgoingWebhook(),
	"grafana_oncall_user_group":       dataSourceUserGroup(),
	"grafana_oncall_team":             dataSourceTeam(),
}

var ResourcesMap = map[string]*schema.Resource{
	"grafana_oncall_integration":      resourceIntegration(),
	"grafana_oncall_route":            resourceRoute(),
	"grafana_oncall_escalation_chain": resourceEscalationChain(),
	"grafana_oncall_escalation":       resourceEscalation(),
	"grafana_oncall_on_call_shift":    resourceOnCallShift(),
	"grafana_oncall_schedule":         resourceSchedule(),
	"grafana_oncall_outgoing_webhook": resourceOutgoingWebhook(),
}
