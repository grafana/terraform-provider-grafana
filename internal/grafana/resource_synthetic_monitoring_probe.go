package grafana

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	sm "github.com/grafana/synthetic-monitoring-agent/pkg/pb/synthetic_monitoring"
	"github.com/grafana/terraform-provider-grafana/internal/common"
)

func ResourceSyntheticMonitoringProbe() *schema.Resource {
	return &schema.Resource{

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
			StateContext: ImportProbeStateWithToken,
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
			"auth_token": {
				Description: "The probe authentication token. Your probe must use this to authenticate with Grafana Cloud.",
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
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
				DiffSuppressFunc: common.SchemaDiffFloat32,
			},
			"longitude": {
				Description:      "Longitude coordinates.",
				Type:             schema.TypeFloat,
				Required:         true,
				DiffSuppressFunc: common.SchemaDiffFloat32,
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
				ValidateDiagFunc: func(i interface{}, p cty.Path) diag.Diagnostics {
					for k, vInt := range i.(map[string]interface{}) {
						v := vInt.(string)
						lbl := sm.Label{Name: k, Value: v}
						if err := lbl.Validate(); err != nil {
							return diag.Errorf(`invalid label "%s=%s": %s`, k, v, err)
						}
					}
					return nil
				},
			},
			"public": {
				Description: "Public probes are run by Grafana Labs and can be used by all users. Only Grafana Labs managed public probes will be set to `true`.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
		},
	}
}

func resourceSyntheticMonitoringProbeCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*common.Client).SMAPI
	p := makeProbe(d)
	res, token, err := c.AddProbe(ctx, *p)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(strconv.FormatInt(res.Id, 10))
	d.Set("tenant_id", res.TenantId)
	d.Set("auth_token", base64.StdEncoding.EncodeToString(token))
	return resourceSyntheticMonitoringProbeRead(ctx, d, meta)
}

func resourceSyntheticMonitoringProbeRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*common.Client).SMAPI
	id, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}
	prb, err := c.GetProbe(ctx, id)
	if err != nil {
		if strings.Contains(err.Error(), "404 Not Found") {
			log.Printf("[WARN] removing probe %s from state because it no longer exists", d.Id())
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	d.Set("tenant_id", prb.TenantId)
	d.Set("name", prb.Name)
	d.Set("latitude", prb.Latitude)
	d.Set("longitude", prb.Longitude)
	d.Set("region", prb.Region)
	d.Set("public", prb.Public)

	if len(prb.Labels) > 0 {
		// Convert []sm.Label into a map before set.
		labels := make(map[string]string, len(prb.Labels))
		for _, l := range prb.Labels {
			labels[l.Name] = l.Value
		}
		d.Set("labels", labels)
	}

	return nil
}

func resourceSyntheticMonitoringProbeUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*common.Client).SMAPI
	p := makeProbe(d)
	_, err := c.UpdateProbe(ctx, *p)
	if err != nil {
		return diag.FromErr(err)
	}
	return resourceSyntheticMonitoringProbeRead(ctx, d, meta)
}

func resourceSyntheticMonitoringProbeDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*common.Client).SMAPI
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

// ImportProbeStateWithToken is an implementation of StateContextFunc
// that can be used to pass the ID of the probe and the existing
// auth_token.
func ImportProbeStateWithToken(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	parts := strings.SplitN(d.Id(), ":", 2)

	// the auth_token is optional
	if len(parts) == 2 {
		if parts[0] == "" || parts[1] == "" {
			return nil, fmt.Errorf("invalid id %q, expected format 'probe_id:auth_token'", d.Id())
		}

		if _, err := base64.StdEncoding.DecodeString(parts[1]); err != nil {
			return nil, fmt.Errorf("invalid auth_token %q, expecting a base64-encoded string", parts[1])
		}

		if err := d.Set("auth_token", parts[1]); err != nil {
			return nil, fmt.Errorf("failed to set auth_token: %s", err)
		}
	}

	d.SetId(parts[0])

	return []*schema.ResourceData{d}, nil
}
