package oncall

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var routeTypeOptions = []string{
	"jinja2",
	"regex",
}

var routeTypeOptionsVerbal = strings.Join(routeTypeOptions, ", ")

func resourceRoute() *schema.Resource {
	return &schema.Resource{
		Description: `
* [HTTP API](https://grafana.com/docs/oncall/latest/oncall-api-reference/routes/)
`,
		CreateContext: withClient[schema.CreateContextFunc](resourceRouteCreate),
		ReadContext:   withClient[schema.ReadContextFunc](resourceRouteRead),
		UpdateContext: withClient[schema.UpdateContextFunc](resourceRouteUpdate),
		DeleteContext: withClient[schema.DeleteContextFunc](resourceRouteDelete),
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
			"routing_type": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice(routeTypeOptions, false),
				Default:      "regex",
				Description:  fmt.Sprintf("The type of route. Can be %s", routeTypeOptionsVerbal),
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
							Optional:    true,
							Description: "Slack channel id. Alerts will be directed to this channel in Slack.",
						},
						"enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "Enable notification in Slack.",
							Default:     true,
						},
					},
				},
				MaxItems:    1,
				Description: "Slack-specific settings for a route.",
			},
			"telegram": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Telegram channel id. Alerts will be directed to this channel in Telegram.",
						},
						"enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "Enable notification in Telegram.",
							Default:     true,
						},
					},
				},
				MaxItems:    1,
				Description: "Telegram-specific settings for a route.",
			},
			"msteams": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "MS teams channel id. Alerts will be directed to this channel in Microsoft teams.",
						},
						"enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "Enable notification in MS teams.",
							Default:     true,
						},
					},
				},
				MaxItems:    1,
				Description: "MS teams-specific settings for a route.",
			},
		},
	}
}

func resourceRouteCreate(ctx context.Context, d *schema.ResourceData, client *onCallAPI.Client) diag.Diagnostics {
	integrationID := d.Get("integration_id").(string)
	escalationChainID := d.Get("escalation_chain_id").(string)
	routingType := d.Get("routing_type").(string)
	routingRegex := d.Get("routing_regex").(string)
	position := d.Get("position").(int)
	slack := d.Get("slack").([]interface{})
	telegram := d.Get("telegram").([]interface{})
	msTeams := d.Get("msteams").([]interface{})

	createOptions := &onCallAPI.CreateRouteOptions{
		IntegrationId:     integrationID,
		EscalationChainId: escalationChainID,
		RoutingType:       routingType,
		RoutingRegex:      routingRegex,
		Position:          &position,
		ManualOrder:       true,
		Slack:             expandRouteSlack(slack),
		Telegram:          expandRouteTelegram(telegram),
		MSTeams:           expandRouteMSTeams(msTeams),
	}

	route, _, err := client.Routes.CreateRoute(createOptions)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(route.ID)

	return resourceRouteRead(ctx, d, client)
}

func resourceRouteRead(ctx context.Context, d *schema.ResourceData, client *onCallAPI.Client) diag.Diagnostics {
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
	d.Set("routing_type", route.RoutingType)
	d.Set("routing_regex", route.RoutingRegex)
	d.Set("position", route.Position)

	// Set messengers data only if related fields are presented
	_, slackOk := d.GetOk("slack")
	if slackOk {
		d.Set("slack", flattenRouteSlack(route.SlackRoute))
	}
	_, telegramOk := d.GetOk("telegram")
	if telegramOk {
		d.Set("telegram", flattenRouteTelegram(route.TelegramRoute))
	}
	_, msteamsOk := d.GetOk("msteams")
	if msteamsOk {
		d.Set("msteams", flattenRouteMSTeams(route.MSTeamsRoute))
	}

	return nil
}

func resourceRouteUpdate(ctx context.Context, d *schema.ResourceData, client *onCallAPI.Client) diag.Diagnostics {
	escalationChainID := d.Get("escalation_chain_id").(string)
	routingType := d.Get("routing_type").(string)
	routingRegex := d.Get("routing_regex").(string)
	position := d.Get("position").(int)
	slack := d.Get("slack").([]interface{})
	telegram := d.Get("telegram").([]interface{})
	msTeams := d.Get("msteams").([]interface{})

	updateOptions := &onCallAPI.UpdateRouteOptions{
		EscalationChainId: escalationChainID,
		RoutingType:       routingType,
		RoutingRegex:      routingRegex,
		Position:          &position,
		ManualOrder:       true,
		Slack:             expandRouteSlack(slack),
		Telegram:          expandRouteTelegram(telegram),
		MSTeams:           expandRouteMSTeams(msTeams),
	}

	route, _, err := client.Routes.UpdateRoute(d.Id(), updateOptions)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(route.ID)
	return resourceRouteRead(ctx, d, client)
}

func resourceRouteDelete(ctx context.Context, d *schema.ResourceData, client *onCallAPI.Client) diag.Diagnostics {
	_, err := client.Routes.DeleteRoute(d.Id(), &onCallAPI.DeleteRouteOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	return nil
}
