package grafana

import (
	"context"
	"log"
	"net/http"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceOnCallRoute() *schema.Resource {
	return &schema.Resource{
		Description: `
* [HTTP API](https://grafana.com/docs/grafana-cloud/oncall/oncall-api-reference/routes/)
`,
		CreateContext: ResourceOnCallRouteCreate,
		ReadContext:   ResourceOnCallRouteRead,
		UpdateContext: ResourceOnCallRouteUpdate,
		DeleteContext: ResourceOnCallRouteDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"integration_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The ID of the integration.",
			},
			"escalation_chain_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The ID of the escalation chain.",
			},
			"position": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "The position of the route (starts from 0).",
			},
			"routing_regex": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Python Regex query. Route is chosen for an alert if there is a match inside the alert payload.",
			},
			"slack": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"channel_id": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Slack channel id. Alerts will be directed to this channel in Slack.",
						},
					},
				},
				MaxItems:    1,
				Description: "Slack-specific settings for a route.",
			},
		},
	}
}

func ResourceOnCallRouteCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*client).onCallAPI

	integrationIDData := d.Get("integration_id").(string)
	escalationChainIDData := d.Get("escalation_chain_id").(string)
	routingRegexData := d.Get("routing_regex").(string)
	positionData := d.Get("position").(int)
	slackData := d.Get("slack").([]interface{})

	createOptions := &onCallAPI.CreateRouteOptions{
		IntegrationId:     integrationIDData,
		EscalationChainId: escalationChainIDData,
		RoutingRegex:      routingRegexData,
		Position:          &positionData,
		ManualOrder:       true,
		Slack:             expandRouteSlack(slackData),
	}

	route, _, err := client.Routes.CreateRoute(createOptions)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(route.ID)

	return ResourceOnCallRouteRead(ctx, d, m)
}

func ResourceOnCallRouteRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*client).onCallAPI

	route, r, err := client.Routes.GetRoute(d.Id(), &onCallAPI.GetRouteOptions{})
	if err != nil {
		if r != nil && r.StatusCode == http.StatusNotFound {
			log.Printf("[WARN] removing route %s from state because it no longer exists", d.Id())
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	d.Set("integration_id", route.IntegrationId)
	d.Set("escalation_chain_id", route.EscalationChainId)
	d.Set("routing_regex", route.RoutingRegex)
	d.Set("position", route.Position)
	d.Set("slack", flattenRouteSlack(route.SlackRoute))

	return nil
}

func ResourceOnCallRouteUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*client).onCallAPI

	escalationChainIDData := d.Get("escalation_chain_id").(string)
	routingRegexData := d.Get("routing_regex").(string)
	positionData := d.Get("position").(int)
	slackData := d.Get("slack").([]interface{})

	updateOptions := &onCallAPI.UpdateRouteOptions{
		EscalationChainId: escalationChainIDData,
		RoutingRegex:      routingRegexData,
		Position:          &positionData,
		ManualOrder:       true,
		Slack:             expandRouteSlack(slackData),
	}

	route, _, err := client.Routes.UpdateRoute(d.Id(), updateOptions)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(route.ID)
	return ResourceOnCallRouteRead(ctx, d, m)
}

func ResourceOnCallRouteDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*client).onCallAPI

	_, err := client.Routes.DeleteRoute(d.Id(), &onCallAPI.DeleteRouteOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	return nil
}
