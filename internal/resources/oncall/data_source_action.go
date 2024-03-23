package oncall

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	onCallAPI "github.com/grafana/amixr-api-go-client"
)

func dataSourceAction() *schema.Resource {
	return &schema.Resource{
		Description: `
**Note:** This data source is going to be deprecated, please use outgoing webhook data source instead.
* [HTTP API](https://grafana.com/docs/oncall/latest/oncall-api-reference/outgoing_webhooks/)

!> Deprecated: Use the ` + "`grafana_oncall_outgoing_webhook`" + ` data source instead.
`,
		ReadContext:        withClient[schema.ReadContextFunc](dataSourceActionRead),
		DeprecationMessage: "This data source is going to be deprecated, please use outgoing webhook data source instead.",
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The action name.",
			},
		},
	}
}

func dataSourceActionRead(ctx context.Context, d *schema.ResourceData, client *onCallAPI.Client) diag.Diagnostics {
	options := &onCallAPI.ListCustomActionOptions{}
	nameData := d.Get("name").(string)

	options.Name = nameData

	customActionsResponse, _, err := client.CustomActions.ListCustomActions(options)
	if err != nil {
		return diag.FromErr(err)
	}

	if len(customActionsResponse.CustomActions) == 0 {
		return diag.Errorf("couldn't find an action matching: %s", options.Name)
	} else if len(customActionsResponse.CustomActions) != 1 {
		return diag.Errorf("more than one action found matching: %s", options.Name)
	}

	customAction := customActionsResponse.CustomActions[0]

	d.SetId(customAction.ID)
	d.Set("name", customAction.Name)

	return nil
}
