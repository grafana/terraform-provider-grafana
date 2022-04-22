package grafana

import (
	"errors"
	"fmt"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceOnCallEscalationChain() *schema.Resource {
	return &schema.Resource{
		Description: `
* [HTTP API](https://grafana.com/docs/grafana-cloud/oncall/oncall-api-reference/escalation_chains/)
`,
		Read: dataSourceEscalationChainRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The escalation chain name.",
			},
		},
	}
}

func dataSourceEscalationChainRead(d *schema.ResourceData, m interface{}) error {
	client := m.(*client).onCallAPI
	if client == nil {
		return errors.New("Grafana OnCall api client is not configured")
	}
	options := &onCallAPI.ListEscalationChainOptions{}
	nameData := d.Get("name").(string)

	options.Name = nameData

	escalationChainsResponse, _, err := client.EscalationChains.ListEscalationChains(options)
	if err != nil {
		return err
	}

	if len(escalationChainsResponse.EscalationChains) == 0 {
		return fmt.Errorf("couldn't find an escalation chain matching: %s", options.Name)
	} else if len(escalationChainsResponse.EscalationChains) != 1 {
		return fmt.Errorf("more than one escalation chain found matching: %s", options.Name)
	}

	escalationChain := escalationChainsResponse.EscalationChains[0]

	d.Set("name", escalationChain.Name)

	d.SetId(escalationChain.ID)

	return nil
}
