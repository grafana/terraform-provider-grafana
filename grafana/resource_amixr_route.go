package grafana

import (
	"errors"
	"log"
	"net/http"

	amixrAPI "github.com/grafana/amixr-api-go-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceAmixrRoute() *schema.Resource {
	return &schema.Resource{
		Description: `
* [Official documentation](https://grafana.com/docs/grafana-cloud/oncall/routes/)
* [HTTP API](https://grafana.com/docs/grafana-cloud/oncall/oncall-api-reference/routes/)
`,
		Create: resourceAmixrRouteCreate,
		Read:   resourceAmixrRouteRead,
		Update: resourceAmixrRouteUpdate,
		Delete: resourceAmixrRouteDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"integration_id": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The ID of the integration.",
			},
			"escalation_chain_id": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The ID of the escalation chain.",
			},
			"position": &schema.Schema{
				Type:        schema.TypeInt,
				Required:    true,
				Description: "The position of the route (starts from 0).",
			},
			"routing_regex": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Python Regex query. Route is chosen for an alert if there is a match inside the alert payload.",
			},
			"slack": &schema.Schema{
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

func resourceAmixrRouteCreate(d *schema.ResourceData, m interface{}) error {
	client := m.(*client).amixrAPI
	if client == nil {
		err := errors.New("amixr api client is not configured")
		return err
	}

	integrationIdData := d.Get("integration_id").(string)
	escalationChainIdData := d.Get("escalation_chain_id").(string)
	routingRegexData := d.Get("routing_regex").(string)
	positionData := d.Get("position").(int)
	slackData := d.Get("slack").([]interface{})

	createOptions := &amixrAPI.CreateRouteOptions{
		IntegrationId:     integrationIdData,
		EscalationChainId: escalationChainIdData,
		RoutingRegex:      routingRegexData,
		Position:          &positionData,
		ManualOrder:       true,
		Slack:             expandRouteSlack(slackData),
	}

	route, _, err := client.Routes.CreateRoute(createOptions)
	if err != nil {
		return err
	}

	d.SetId(route.ID)

	return resourceAmixrRouteRead(d, m)
}

func resourceAmixrRouteRead(d *schema.ResourceData, m interface{}) error {
	client := m.(*client).amixrAPI
	if client == nil {
		err := errors.New("amixr api client is not configured")
		return err
	}

	route, r, err := client.Routes.GetRoute(d.Id(), &amixrAPI.GetRouteOptions{})
	if err != nil {
		if r != nil && r.StatusCode == http.StatusNotFound {
			log.Printf("[WARN] removing route %s from state because it no longer exists", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("integration_id", route.IntegrationId)
	d.Set("escalation_chain_id", route.EscalationChainId)
	d.Set("routing_regex", route.RoutingRegex)
	d.Set("position", route.Position)
	d.Set("slack", flattenRouteSlack(route.SlackRoute))

	return nil
}

func resourceAmixrRouteUpdate(d *schema.ResourceData, m interface{}) error {
	client := m.(*client).amixrAPI
	if client == nil {
		err := errors.New("amixr api client is not configured")
		return err
	}

	escalationChainIdData := d.Get("escalation_chain_id").(string)
	routingRegexData := d.Get("routing_regex").(string)
	positionData := d.Get("position").(int)
	slackData := d.Get("slack").([]interface{})

	updateOptions := &amixrAPI.UpdateRouteOptions{
		EscalationChainId: escalationChainIdData,
		RoutingRegex:      routingRegexData,
		Position:          &positionData,
		ManualOrder:       true,
		Slack:             expandRouteSlack(slackData),
	}

	route, _, err := client.Routes.UpdateRoute(d.Id(), updateOptions)
	if err != nil {
		return err
	}

	d.SetId(route.ID)
	return resourceAmixrRouteRead(d, m)
}

func resourceAmixrRouteDelete(d *schema.ResourceData, m interface{}) error {
	client := m.(*client).amixrAPI
	if client == nil {
		err := errors.New("amixr api client is not configured")
		return err
	}

	_, err := client.Routes.DeleteRoute(d.Id(), &amixrAPI.DeleteRouteOptions{})
	if err != nil {
		return err
	}

	d.SetId("")

	return nil
}
