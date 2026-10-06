package oncall

import (
	"context"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceRoute() *common.DataSource {
	schema := &schema.Resource{
		Description: `
* [HTTP API](https://grafana.com/docs/oncall/latest/oncall-api-reference/routes/)
`,
		ReadContext: withClient[schema.ReadContextFunc](dataSourceRouteRead),
		Schema: map[string]*schema.Schema{
			"integration_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The ID of the integration.",
			},
			"routing_regex": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The routing regex to match.",
			},
			"escalation_chain_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the escalation chain.",
			},
			"position": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The position of the route (starts from 0).",
			},
			"routing_type": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The type of route (regex or jinja2).",
			},
		},
	}
	return common.NewLegacySDKDataSource(common.CategoryOnCall, "grafana_oncall_route", schema)
}

func dataSourceRouteRead(ctx context.Context, d *schema.ResourceData, client *onCallAPI.Client) diag.Diagnostics {
	integrationID := d.Get("integration_id").(string)
	routingRegex := d.Get("routing_regex").(string)

	page := 1
	for {
		options := &onCallAPI.ListRouteOptions{
			ListOptions: onCallAPI.ListOptions{
				Page: page,
			},
			IntegrationId: integrationID,
			RoutingRegex:  routingRegex,
		}

		routesResponse, _, err := client.Routes.ListRoutes(options)
		if err != nil {
			return diag.FromErr(err)
		}

		for _, route := range routesResponse.Routes {
			if route.RoutingRegex == routingRegex {
				d.SetId(route.ID)
				d.Set("escalation_chain_id", route.EscalationChainId)
				d.Set("position", route.Position)
				d.Set("routing_type", route.RoutingType)
				return nil
			}
		}

		if routesResponse.Next == nil {
			break
		}
		page++
	}

	return diag.Errorf("couldn't find a route matching: integration_id=%s, routing_regex=%s", integrationID, routingRegex)
}
