package syntheticmonitoring

import (
	"context"

	smapi "github.com/grafana/synthetic-monitoring-api-go-client"
	"github.com/grafana/terraform-provider-grafana/v2/internal/common"
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

var DatasourcesMap = map[string]*schema.Resource{
	"grafana_synthetic_monitoring_probe":  dataSourceProbe(),
	"grafana_synthetic_monitoring_probes": dataSourceProbes(),
}

var Resources = []*common.Resource{
	resourceCheck(),
	resourceProbe(),
}
