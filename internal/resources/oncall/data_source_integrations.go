package oncall

import (
	"context"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceIntegration() *schema.Resource {
	return &schema.Resource{
		Description: `
* [HTTP API](https://grafana.com/docs/oncall/latest/oncall-api-reference/integrations/)
`,
		ReadContext: withClient[schema.ReadContextFunc](dataSourceIntegrationRead),
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The integration ID.",
			},
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The integration name.",
			},
		},
	}
}

func dataSourceIntegrationRead(ctx context.Context, d *schema.ResourceData, client *onCallAPI.Client) diag.Diagnostics {
	options := &onCallAPI.GetIntegrationOptions{}
	integrationID := d.Get("id").(string)

	integrationResponse, _, err := client.Integrations.GetIntegration(integrationID, options)
	if err != nil {
		if strings.HasPrefix(err.Error(), "status: 404") {
			return diag.Errorf("no integration exists with ID %q", integrationID)
		}
		return diag.FromErr(err)
	}

	d.SetId(integrationResponse.id)
	d.Set("id", integrationResponse.id)
	d.Set("name", integrationResponse.name)

	return nil
}