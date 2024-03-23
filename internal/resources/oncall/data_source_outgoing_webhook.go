package oncall

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	onCallAPI "github.com/grafana/amixr-api-go-client"
)

func dataSourceOutgoingWebhook() *schema.Resource {
	return &schema.Resource{
		Description: `
* [HTTP API](https://grafana.com/docs/oncall/latest/oncall-api-reference/outgoing_webhooks/)
`,
		ReadContext: withClient[schema.ReadContextFunc](dataSourceOutgoingWebhookRead),
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The outgoing webhook name.",
			},
		},
	}
}

func dataSourceOutgoingWebhookRead(ctx context.Context, d *schema.ResourceData, client *onCallAPI.Client) diag.Diagnostics {
	options := &onCallAPI.ListWebhookOptions{}
	name := d.Get("name").(string)

	options.Name = name

	outgoingWebhookResponse, _, err := client.Webhooks.ListWebhooks(options)
	if err != nil {
		return diag.FromErr(err)
	}

	if len(outgoingWebhookResponse.Webhooks) == 0 {
		return diag.Errorf("couldn't find an outgoing webhook matching: %s", options.Name)
	} else if len(outgoingWebhookResponse.Webhooks) != 1 {
		return diag.Errorf("more than one outgoing webhook found matching: %s", options.Name)
	}

	outgoingWebhook := outgoingWebhookResponse.Webhooks[0]

	d.SetId(outgoingWebhook.ID)
	d.Set("name", outgoingWebhook.Name)

	return nil
}
