package grafana

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceSyntheticMonitoringProbes() *schema.Resource {
	return &schema.Resource{
		Description: "Data source for retrieving all probes.",
		ReadContext: dataSourceSyntheticMonitoringProbesRead,
		Schema: map[string]*schema.Schema{
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
	c := meta.(*client).smapi
	var diags diag.Diagnostics
	prbs, err := c.ListProbes(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	probes := make(map[string]interface{}, len(prbs))
	for _, p := range prbs {
		probes[p.Name] = p.Id
	}

	d.SetId("probes")
	d.Set("probes", probes)

	return diags
}
