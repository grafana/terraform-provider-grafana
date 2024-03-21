package machinelearning

import (
	"context"

	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func checkClient(f func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics) func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
		client := meta.(*common.Client).MLAPI
		if client == nil {
			return diag.Errorf("the ML API client is required for this resource. Set the url and auth provider attributes")
		}
		return f(ctx, d, meta)
	}
}

var DatasourcesMap = map[string]*schema.Resource{}

var Resources = []*common.Resource{
	resourceJob(),
	resourceHoliday(),
	resourceOutlierDetector(),
}
