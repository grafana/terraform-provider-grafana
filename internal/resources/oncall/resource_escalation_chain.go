package oncall

import (
	"context"
	"log"
	"net/http"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceEscalationChain() *common.Resource {
	schema := &schema.Resource{
		Description: `
* [HTTP API](https://grafana.com/docs/oncall/latest/oncall-api-reference/escalation_chains/)
`,
		CreateContext: withClient[schema.CreateContextFunc](resourceEscalationChainCreate),
		ReadContext:   withClient[schema.ReadContextFunc](resourceEscalationChainRead),
		UpdateContext: withClient[schema.UpdateContextFunc](resourceEscalationChainUpdate),
		DeleteContext: withClient[schema.DeleteContextFunc](resourceEscalationChainDelete),
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

	return common.NewLegacySDKResource(
		common.CategoryOnCall,
		"grafana_oncall_escalation_chain",
		resourceID,
		schema,
	)
}

func resourceEscalationChainCreate(ctx context.Context, d *schema.ResourceData, client *onCallAPI.Client) diag.Diagnostics {
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

	return resourceEscalationChainRead(ctx, d, client)
}

func resourceEscalationChainRead(ctx context.Context, d *schema.ResourceData, client *onCallAPI.Client) diag.Diagnostics {
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

func resourceEscalationChainUpdate(ctx context.Context, d *schema.ResourceData, client *onCallAPI.Client) diag.Diagnostics {
	nameData := d.Get("name").(string)
	teamIDData := d.Get("team_id").(string)

	updateOptions := &onCallAPI.UpdateEscalationChainOptions{
		Name:   nameData,
		TeamId: teamIDData,
	}

	escalationChain, _, err := client.EscalationChains.UpdateEscalationChain(d.Id(), updateOptions)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(escalationChain.ID)
	return resourceEscalationChainRead(ctx, d, client)
}

func resourceEscalationChainDelete(ctx context.Context, d *schema.ResourceData, client *onCallAPI.Client) diag.Diagnostics {
	_, err := client.EscalationChains.DeleteEscalationChain(d.Id(), &onCallAPI.DeleteEscalationChainOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	return nil
}
