package grafana

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	sm "github.com/grafana/synthetic-monitoring-agent/pkg/pb/synthetic_monitoring"
)

var (
	syntheticMonitoringProbe = &schema.Resource{

		Description: `
Besides the public probes run by Grafana Labs, you can also install your
own private probes. These are only accessible to you and only write data to
your Grafana Cloud account. Private probes are instances of the open source
Grafana Synthetic Monitoring Agent.

* [Official documentation](https://grafana.com/docs/grafana-cloud/synthetic-monitoring/private-probes/)
`,

		CreateContext: resourceSyntheticMonitoringProbeCreate,
		ReadContext:   resourceSyntheticMonitoringProbeRead,
		UpdateContext: resourceSyntheticMonitoringProbeUpdate,
		DeleteContext: resourceSyntheticMonitoringProbeDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"id": {
				Description: "The ID of the probe.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"tenant_id": {
				Description: "The tenant ID of the probe.",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"name": {
				Description: "Name of the probe.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"latitude": {
				Description:      "Latitude coordinates.",
				Type:             schema.TypeFloat,
				Required:         true,
				DiffSuppressFunc: schemaDiffFloat32,
			},
			"longitude": {
				Description:      "Longitude coordinates.",
				Type:             schema.TypeFloat,
				Required:         true,
				DiffSuppressFunc: schemaDiffFloat32,
			},
			"region": {
				Description: "Region of the probe.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"labels": {
				Description: "Custom labels to be included with collected metrics and logs.",
				Type:        schema.TypeMap,
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"public": {
				Description: "Public probes are run by Grafana Labs and can be used by all users. You must be an admin to set this to `true`.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
		},
	}
)

func resourceSyntheticMonitoringProbe() *schema.Resource {
	return syntheticMonitoringProbe
}

func resourceSyntheticMonitoringProbeCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*client).smapi
	var diags diag.Diagnostics
	p := makeProbe(d)
	res, _, err := c.AddProbe(ctx, *p)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(strconv.FormatInt(res.Id, 10))
	d.Set("tenant_id", res.TenantId)
	return diags
}

func resourceSyntheticMonitoringProbeRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*client).smapi
	var diags diag.Diagnostics
	prbs, err := c.ListProbes(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	var prb sm.Probe
	for _, p := range prbs {
		if strconv.FormatInt(p.Id, 10) == d.Id() {
			prb = p
			break
		}
	}

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

func resourceSyntheticMonitoringProbeUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*client).smapi
	var diags diag.Diagnostics
	p := makeProbe(d)
	_, err := c.UpdateProbe(ctx, *p)
	if err != nil {
		return diag.FromErr(err)
	}
	return diags
}

func resourceSyntheticMonitoringProbeDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*client).smapi
	var diags diag.Diagnostics
	id, _ := strconv.ParseInt(d.Id(), 10, 64)
	err := c.DeleteProbe(ctx, id)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")
	return diags
}

// makeProbe populates an instance of sm.Probe. We need this for create and
// update calls with the SM API client.
func makeProbe(d *schema.ResourceData) *sm.Probe {

	var id int64
	if d.Id() != "" {
		id, _ = strconv.ParseInt(d.Id(), 10, 64)
	}

	var labels []sm.Label
	for name, value := range d.Get("labels").(map[string]interface{}) {
		labels = append(labels, sm.Label{
			Name:  name,
			Value: value.(string),
		})
	}

	return &sm.Probe{
		Id:        id,
		TenantId:  int64(d.Get("tenant_id").(int)),
		Name:      d.Get("name").(string),
		Latitude:  float32(d.Get("latitude").(float64)),
		Longitude: float32(d.Get("longitude").(float64)),
		Region:    d.Get("region").(string),
		Labels:    labels,
		Public:    d.Get("public").(bool),
	}
}
