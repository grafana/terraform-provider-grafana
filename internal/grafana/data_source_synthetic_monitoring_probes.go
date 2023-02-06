package grafana

import (
	"context"

	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DatasourceSyntheticMonitoringProbes() *schema.Resource {
	return &schema.Resource{
		Description: "Data source for retrieving all probes.",
		ReadContext: dataSourceSyntheticMonitoringProbesRead,
		Schema: map[string]*schema.Schema{
			"filter_deprecated": {
				Type:        schema.TypeBool,
				Description: "If true, only probes that are not deprecated will be returned.",
				Optional:    true,
				Default:     true,
			},
			"probes": {
				Description: "Map of probes with their names as keys and IDs as values.",
				Type:        schema.TypeMap,
				Computed:    true,
				Elem: &schema.Schema{
					Type: schema.TypeInt,
				},
			},
		},
	}
}

func dataSourceSyntheticMonitoringProbesRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*common.Client).SMAPI
	var diags diag.Diagnostics
	prbs, err := c.ListProbes(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	probes := make(map[string]interface{}, len(prbs))
	for _, p := range prbs {
		if !p.Deprecated || !d.Get("filter_deprecated").(bool) {
			probes[p.Name] = p.Id
		}
	}

	d.SetId("probes")
	d.Set("probes", probes)

	return diags
}
