package grafana

import (
	"context"
	"log"
	"net/http"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceOnCallEscalationChain() *schema.Resource {
	return &schema.Resource{
		Description: `
* [HTTP API](https://grafana.com/docs/grafana-cloud/oncall/oncall-api-reference/escalation_chains/)
`,
		CreateContext: ResourceOnCallEscalationChainCreate,
		ReadContext:   ResourceOnCallEscalationChainRead,
		UpdateContext: ResourceOnCallEscalationChainUpdate,
		DeleteContext: ResourceOnCallEscalationChainDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the escalation chain.",
			},
			"team_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The ID of the team.",
			},
		},
	}
}

func ResourceOnCallEscalationChainCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*client).onCallAPI
	if client == nil {
		return diag.Errorf("grafana OnCall api client is not configured")
	}

	nameData := d.Get("name").(string)
	teamIDData := d.Get("team_id").(string)

	createOptions := &onCallAPI.CreateEscalationChainOptions{
		Name:   nameData,
		TeamId: teamIDData,
	}

	escalationChain, _, err := client.EscalationChains.CreateEscalationChain(createOptions)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(escalationChain.ID)

	return ResourceOnCallEscalationChainRead(ctx, d, m)
}

func ResourceOnCallEscalationChainRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*client).onCallAPI
	if client == nil {
		return diag.Errorf("grafana OnCall api client is not configured")
	}

	escalationChain, r, err := client.EscalationChains.GetEscalationChain(d.Id(), &onCallAPI.GetEscalationChainOptions{})
	if err != nil {
		if r != nil && r.StatusCode == http.StatusNotFound {
			log.Printf("[WARN] removing escalation chain %s from state because it no longer exists", d.Get("name").(string))
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	d.Set("name", escalationChain.Name)
	d.Set("team_id", escalationChain.TeamId)

	return nil
}

func ResourceOnCallEscalationChainUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*client).onCallAPI
	if client == nil {
		return diag.Errorf("grafana OnCall api client is not configured")
	}

	nameData := d.Get("name").(string)

	updateOptions := &onCallAPI.UpdateEscalationChainOptions{
		Name: nameData,
	}

	escalationChain, _, err := client.EscalationChains.UpdateEscalationChain(d.Id(), updateOptions)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(escalationChain.ID)
	return ResourceOnCallEscalationChainRead(ctx, d, m)
}

func ResourceOnCallEscalationChainDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*client).onCallAPI
	if client == nil {
		return diag.Errorf("grafana OnCall api client is not configured")
	}

	_, err := client.EscalationChains.DeleteEscalationChain(d.Id(), &onCallAPI.DeleteEscalationChainOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	return nil
}
