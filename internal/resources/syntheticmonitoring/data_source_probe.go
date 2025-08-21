package syntheticmonitoring

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	sm "github.com/grafana/synthetic-monitoring-agent/pkg/pb/synthetic_monitoring"
	smapi "github.com/grafana/synthetic-monitoring-api-go-client"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

func dataSourceProbe() *common.DataSource {
	schema := &schema.Resource{
		Description: "Data source for retrieving a single probe by name.",
		ReadContext: withClient[schema.ReadContextFunc](dataSourceProbeRead),
		Schema: common.CloneResourceSchemaForDatasource(resourceProbe().Schema, map[string]*schema.Schema{
			"name": {
				Description: "Name of the probe.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"auth_token": nil,
		}),
	}
	return common.NewLegacySDKDataSource(common.CategorySyntheticMonitoring, "grafana_synthetic_monitoring_probe", schema)
}

func dataSourceProbeRead(ctx context.Context, d *schema.ResourceData, c *smapi.Client) diag.Diagnostics {
	var diags diag.Diagnostics
	prbs, err := c.ListProbes(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	var prb sm.Probe
	for _, p := range prbs {
		if p.Name == d.Get("name") {
			prb = p
			break
		}
	}

	if prb.Id == 0 {
		return diag.Errorf("Probe with name %s not found", d.Get("name"))
	}

	d.SetId(strconv.FormatInt(prb.Id, 10))
	d.Set("tenant_id", prb.TenantId)
	d.Set("name", prb.Name)
	d.Set("latitude", prb.Latitude)
	d.Set("longitude", prb.Longitude)
	d.Set("region", prb.Region)
	d.Set("public", prb.Public)

	// Convert []sm.Label into a map before set.
	labels := make(map[string]string, len(prb.Labels))
	for _, l := range prb.Labels {
		labels[l.Name] = l.Value
	}
	d.Set("labels", labels)

	return diags
}
