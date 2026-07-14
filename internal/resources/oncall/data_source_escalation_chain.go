package oncall

import (
	"context"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceEscalationChain() *common.DataSource {
	schema := &schema.Resource{
		Description: `
* [HTTP API](https://grafana.com/docs/oncall/latest/oncall-api-reference/escalation_chains/)
`,
		ReadContext: withClient[schema.ReadContextFunc](dataSourceEscalationChainRead),
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The escalation chain name.",
			},
		},
	}
	return common.NewLegacySDKDataSource(common.CategoryOnCall, "grafana_oncall_escalation_chain", schema)
}

func dataSourceEscalationChainRead(ctx context.Context, d *schema.ResourceData, client *onCallAPI.Client) diag.Diagnostics {
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
