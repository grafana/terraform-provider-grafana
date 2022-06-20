package grafana

import (
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	onCallAPI "github.com/grafana/amixr-api-go-client"
)

func DataSourceOnCallAction() *schema.Resource {
	return &schema.Resource{
		Description: `
**Note:** This data source is going to be deprecated, please use outgoing webhook data source instead.
* [HTTP API](https://grafana.com/docs/grafana-cloud/oncall/oncall-api-reference/outgoing_webhooks/)
`,
		Read:               dataSourceOnCallActionRead,
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

func dataSourceOnCallActionRead(d *schema.ResourceData, m interface{}) error {
	client := m.(*client).onCallAPI
	if client == nil {
		return errors.New("grafana OnCall api client is not configured")
	}
	options := &onCallAPI.ListCustomActionOptions{}
	nameData := d.Get("name").(string)

	options.Name = nameData

	customActionsResponse, _, err := client.CustomActions.ListCustomActions(options)
	if err != nil {
		return err
	}

	if len(customActionsResponse.CustomActions) == 0 {
		return fmt.Errorf("couldn't find an action matching: %s", options.Name)
	} else if len(customActionsResponse.CustomActions) != 1 {
		return fmt.Errorf("more than one action found matching: %s", options.Name)
	}

	custom_action := customActionsResponse.CustomActions[0]

	d.SetId(custom_action.ID)
	d.Set("name", custom_action.Name)

	return nil
}
