package oncall

import (
	"context"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// All on-call resources have a single string ID format
var resourceID = common.NewResourceID(common.StringIDField("id"))

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

var DataSources = []*common.DataSource{
	dataSourceUser(),
	dataSourceEscalationChain(),
	dataSourceSchedule(),
	dataSourceSlackChannel(),
	dataSourceOutgoingWebhook(),
	dataSourceUserGroup(),
	dataSourceTeam(),
	dataSourceIntegration(),
}

var Resources = []*common.Resource{
	resourceIntegration(),
	resourceRoute(),
	resourceEscalationChain(),
	resourceEscalation(),
	resourceOnCallShift(),
	resourceSchedule(),
	resourceOutgoingWebhook(),
}
