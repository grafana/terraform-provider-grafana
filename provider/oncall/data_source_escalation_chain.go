package oncall

import (
	"context"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/grafana/terraform-provider-grafana/provider/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceEscalationChain() *schema.Resource {
	return &schema.Resource{
		Description: `
* [HTTP API](https://grafana.com/docs/grafana-cloud/oncall/oncall-api-reference/escalation_chains/)
`,
		ReadContext: dataSourceEscalationChainRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The escalation chain name.",
			},
		},
	}
}

func dataSourceEscalationChainRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client).OnCallClient
	options := &onCallAPI.ListEscalationChainOptions{}
	nameData := d.Get("name").(string)

	options.Name = nameData

	escalationChainsResponse, _, err := client.EscalationChains.ListEscalationChains(options)
	if err != nil {
		return diag.FromErr(err)
	}

	if len(escalationChainsResponse.EscalationChains) == 0 {
		return diag.Errorf("couldn't find an escalation chain matching: %s", options.Name)
	} else if len(escalationChainsResponse.EscalationChains) != 1 {
		return diag.Errorf("more than one escalation chain found matching: %s", options.Name)
	}

	escalationChain := escalationChainsResponse.EscalationChains[0]

	d.Set("name", escalationChain.Name)

	d.SetId(escalationChain.ID)

	return nil
}
