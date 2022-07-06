package grafana

import (
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	onCallAPI "github.com/grafana/amixr-api-go-client"
)

func DataSourceOnCallOutgoingWebhook() *schema.Resource {
	return &schema.Resource{
		Description: `
* [HTTP API](https://grafana.com/docs/grafana-cloud/oncall/oncall-api-reference/outgoing_webhooks/)
`,
		Read: dataSourceOnCallOutgoingWebhookRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The outgoing webhook name.",
			},
		},
	}
}

func dataSourceOnCallOutgoingWebhookRead(d *schema.ResourceData, m interface{}) error {
	client := m.(*client).onCallAPI
	if client == nil {
		return errors.New("grafana OnCall api client is not configured")
	}
	options := &onCallAPI.ListCustomActionOptions{}
	name := d.Get("name").(string)

	options.Name = name

	outgoingWebhookResponse, _, err := client.CustomActions.ListCustomActions(options)
	if err != nil {
		return err
	}

	if len(outgoingWebhookResponse.CustomActions) == 0 {
		return fmt.Errorf("couldn't find an outgoing webhook matching: %s", options.Name)
	} else if len(outgoingWebhookResponse.CustomActions) != 1 {
		return fmt.Errorf("more than one outgoing webhook found matching: %s", options.Name)
	}

	outgoingWebhook := outgoingWebhookResponse.CustomActions[0]

	d.SetId(outgoingWebhook.ID)
	d.Set("name", outgoingWebhook.Name)

	return nil
}
