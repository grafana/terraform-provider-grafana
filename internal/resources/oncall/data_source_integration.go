package oncall

import (
	"context"
	"strings"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceIntegration() *common.DataSource {
	schema := &schema.Resource{
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
			"link": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The link for the integration.",
			},
			"inbound_email": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The inbound email for the integration. Only available for integration type `inbound_email`.",
			},
		},
	}
	return common.NewLegacySDKDataSource(common.CategoryOnCall, "grafana_oncall_integration", schema)
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

	d.SetId(integrationResponse.ID)
	d.Set("id", integrationResponse.ID)
	d.Set("name", integrationResponse.Name)
	d.Set("link", integrationResponse.Link)
	d.Set("inbound_email", integrationResponse.InboundEmail)
	return nil
}
