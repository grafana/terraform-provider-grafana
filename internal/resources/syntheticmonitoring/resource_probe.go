package syntheticmonitoring

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	sm "github.com/grafana/synthetic-monitoring-agent/pkg/pb/synthetic_monitoring"
	smapi "github.com/grafana/synthetic-monitoring-api-go-client"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

var resourceProbeID = common.NewResourceID(common.IntIDField("id"), common.OptionalStringIDField("authToken"))

func resourceProbe() *common.Resource {
	schema := &schema.Resource{

		Description: `
Besides the public probes run by Grafana Labs, you can also install your
own private probes. These are only accessible to you and only write data to
your Grafana Cloud account. Private probes are instances of the open source
Grafana Synthetic Monitoring Agent.

* [Official documentation](https://grafana.com/docs/grafana-cloud/testing/synthetic-monitoring/set-up/set-up-private-probes/)
`,

		CreateContext: withClient[schema.CreateContextFunc](resourceProbeCreate),
		ReadContext:   withClient[schema.ReadContextFunc](resourceProbeRead),
		UpdateContext: withClient[schema.UpdateContextFunc](resourceProbeUpdate),
		DeleteContext: withClient[schema.DeleteContextFunc](resourceProbeDelete),
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
			"disable_scripted_checks": {
				Description: "Disables scripted checks for this probe.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
			"disable_browser_checks": {
				Description: "Disables browser checks for this probe.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
		},
	}

	return common.NewLegacySDKResource(
		common.CategorySyntheticMonitoring,
		"grafana_synthetic_monitoring_probe",
		resourceProbeID,
		schema,
	).
		WithLister(listProbes).
		WithPreferredResourceNameField("name")
}

func listProbes(ctx context.Context, client *common.Client, data any) ([]string, error) {
	smClient := client.SMAPI
	if smClient == nil {
		return nil, fmt.Errorf("client not configured for SM API")
	}

	probeList, err := smClient.ListProbes(ctx)
	if err != nil {
		return nil, err
	}

	var ids []string
	for _, probe := range probeList {
		if probe.Public {
			continue
		}
		ids = append(ids, strconv.FormatInt(probe.Id, 10))
	}
	return ids, nil
}

func resourceProbeCreate(ctx context.Context, d *schema.ResourceData, c *smapi.Client) diag.Diagnostics {
	p := makeProbe(d)
	res, token, err := c.AddProbe(ctx, *p)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(strconv.FormatInt(res.Id, 10))
	d.Set("tenant_id", res.TenantId)
	d.Set("auth_token", base64.StdEncoding.EncodeToString(token))
	return resourceProbeRead(ctx, d, c)
}

func resourceProbeRead(ctx context.Context, d *schema.ResourceData, c *smapi.Client) diag.Diagnostics {
	id, err := resourceProbeID.Single(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	prb, err := c.GetProbe(ctx, id.(int64))
	if err != nil {
		if strings.Contains(err.Error(), "404 Not Found") {
			return common.WarnMissing("probe", d)
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

	if prb.Capabilities != nil {
		d.Set("disable_scripted_checks", prb.Capabilities.DisableScriptedChecks)
		d.Set("disable_browser_checks", prb.Capabilities.DisableBrowserChecks)
	} else {
		d.Set("disable_scripted_checks", false)
		d.Set("disable_browser_checks", false)
	}

	return nil
}

func resourceProbeUpdate(ctx context.Context, d *schema.ResourceData, c *smapi.Client) diag.Diagnostics {
	p := makeProbe(d)
	_, err := c.UpdateProbe(ctx, *p)
	if err != nil {
		return diag.FromErr(err)
	}
	return resourceProbeRead(ctx, d, c)
}

func resourceProbeDelete(ctx context.Context, d *schema.ResourceData, c *smapi.Client) diag.Diagnostics {
	id, _ := strconv.ParseInt(d.Id(), 10, 64)

	// Remove the probe from any checks that use it.
	checks, err := c.ListChecks(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	for _, check := range checks {
		for i, probeID := range check.Probes {
			if probeID == id {
				if len(check.Probes) == 1 {
					return diag.Errorf(`could not delete probe %d. It is the only probe for check %q.
You must also taint the check, or assign a new probe to it before deleting this probe.`, id, check.Job)
				}
				check.Probes = append(check.Probes[:i], check.Probes[i+1:]...)
				if _, err := c.UpdateCheck(ctx, check); err != nil {
					return diag.Errorf(`error while deleting probe %d, failed to remove it from check %q: %s.`, id, check.Job, err)
				}
				break
			}
		}
	}

	return diag.FromErr(c.DeleteProbe(ctx, id))
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
		Capabilities: &sm.Probe_Capabilities{
			DisableScriptedChecks: d.Get("disable_scripted_checks").(bool),
			DisableBrowserChecks:  d.Get("disable_browser_checks").(bool),
		},
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
