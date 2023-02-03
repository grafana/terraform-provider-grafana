package provider

import (
	"context"
	"log"
	"net/http"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/grafana/terraform-provider-grafana/provider/common"
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
				Description: "The ID of the OnCall team. To get one, create a team in Grafana, and navigate to the OnCall plugin (to sync the team with OnCall). You can then get the ID using the `grafana_oncall_team` datasource.",
			},
		},
	}
}

func ResourceOnCallEscalationChainCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client).OnCallClient

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
	client := m.(*common.Client).OnCallClient

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
	client := m.(*common.Client).OnCallClient

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
	client := m.(*common.Client).OnCallClient

	_, err := client.EscalationChains.DeleteEscalationChain(d.Id(), &onCallAPI.DeleteEscalationChainOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	return nil
}
