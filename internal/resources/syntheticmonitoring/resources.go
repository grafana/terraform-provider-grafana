package syntheticmonitoring

import (
	"context"

	smapi "github.com/grafana/synthetic-monitoring-api-go-client"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type crudWithClientFunc func(ctx context.Context, d *schema.ResourceData, client *smapi.Client) diag.Diagnostics

func withClient[T schema.CreateContextFunc | schema.UpdateContextFunc | schema.ReadContextFunc | schema.DeleteContextFunc](f crudWithClientFunc) T {
	return func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
		client := meta.(*common.Client).SMAPI
		if client == nil {
			return diag.Errorf("the SM client is required for this resource. Set the sm_access_token provider attribute")
		}
		return f(ctx, d, client)
	}
}

var DataSources = []*common.DataSource{
	dataSourceProbe(),
	dataSourceProbes(),
}

var Resources = []*common.Resource{
	resourceCheck(),
	resourceProbe(),
	resourceCheckAlerts(),
}
