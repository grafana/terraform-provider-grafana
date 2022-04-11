package grafana

import (
	"log"
	"net/http"

	amixrAPI "github.com/grafana/amixr-api-go-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

var integrationTypes = []string{
	"grafana",
	"webhook",
	"alertmanager",
	"kapacitor",
	"fabric",
	"newrelic",
	"datadog",
	"pagerduty",
	"pingdom",
	"elastalert",
	"amazon_sns",
	"curler",
	"sentry",
	"formatted_webhook",
	"heartbeat",
	"demo",
	"manual",
	"stackdriver",
	"uptimerobot",
	"sentry_platform",
	"zabbix",
	"prtg",
	"slack_channel",
	"inbound_email",
}

func ResourceAmixrIntegration() *schema.Resource {
	return &schema.Resource{
		Description: `
* [Official documentation](https://grafana.com/docs/grafana-cloud/oncall/integrations/)
* [HTTP API](https://grafana.com/docs/grafana-cloud/oncall/oncall-api-reference/)
`,

		Create: resourceAmixrIntegrationCreate,
		Read:   resourceAmixrIntegrationRead,
		Update: resourceAmixrIntegrationUpdate,
		Delete: resourceAmixrIntegrationDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
				Description:  "The name of the service integration",
			},
			"team_id": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The id of the team",
			},
			"type": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice(integrationTypes, false),
				ForceNew:     true,
				Description:  "The type of integration",
			},
			"default_route": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"escalation_chain_id": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"slack": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"channel_id": {
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
							MaxItems: 1,
						},
					},
				},
				MaxItems:    1,
				Description: "The Default route for all alerts from the given integration",
			},
			"link": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The link for using in an integrated tool.",
			},
			"templates": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"resolve_signal": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"grouping_key": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"slack": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"title": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"message": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"image_url": {
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
							MaxItems: 1,
						},
					},
				},
				MaxItems:    1,
				Description: "Jinja2 templates for Alert payload.",
			},
		},
	}
}

func resourceAmixrIntegrationCreate(d *schema.ResourceData, m interface{}) error {
	client := m.(*client).amixrAPI

	teamIdData := d.Get("team_id").(string)
	nameData := d.Get("name").(string)
	typeData := d.Get("type").(string)
	templatesData := d.Get("templates").([]interface{})

	createOptions := &amixrAPI.CreateIntegrationOptions{
		TeamId:    teamIdData,
		Name:      nameData,
		Type:      typeData,
		Templates: expandTemplates(templatesData),
	}

	integration, _, err := client.Integrations.CreateIntegration(createOptions)
	if err != nil {
		return err
	}

	d.SetId(integration.ID)

	return resourceAmixrIntegrationRead(d, m)
}

func resourceAmixrIntegrationUpdate(d *schema.ResourceData, m interface{}) error {
	client := m.(*client).amixrAPI

	nameData := d.Get("name").(string)
	templateData := d.Get("templates").([]interface{})
	defaultRouteData := d.Get("default_route").([]interface{})

	updateOptions := &amixrAPI.UpdateIntegrationOptions{
		Name:         nameData,
		Templates:    expandTemplates(templateData),
		DefaultRoute: expandDefaultRoute(defaultRouteData),
	}

	integration, _, err := client.Integrations.UpdateIntegration(d.Id(), updateOptions)
	if err != nil {
		return err
	}

	d.SetId(integration.ID)

	return resourceAmixrIntegrationRead(d, m)
}

func resourceAmixrIntegrationRead(d *schema.ResourceData, m interface{}) error {
	client := m.(*client).amixrAPI
	options := &amixrAPI.GetIntegrationOptions{}
	integration, r, err := client.Integrations.GetIntegration(d.Id(), options)
	if err != nil {
		if r != nil && r.StatusCode == http.StatusNotFound {
			log.Printf("[WARN] removing integreation %s from state because it no longer exists", d.Get("name").(string))
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("team_id", integration.TeamId)
	d.Set("default_route", flattenDefaultRoute(integration.DefaultRoute))
	d.Set("name", integration.Name)
	d.Set("type", integration.Type)
	d.Set("templates", flattenTemplates(integration.Templates))
	d.Set("link", integration.Link)

	return nil
}

func resourceAmixrIntegrationDelete(d *schema.ResourceData, m interface{}) error {
	log.Printf("[DEBUG] delete amixr integration")

	client := m.(*client).amixrAPI
	options := &amixrAPI.DeleteIntegrationOptions{}
	_, err := client.Integrations.DeleteIntegration(d.Id(), options)
	if err != nil {
		return err
	}

	d.SetId("")

	return nil
}

func flattenRouteSlack(in *amixrAPI.SlackRoute) []map[string]interface{} {
	slack := make([]map[string]interface{}, 0, 1)

	out := make(map[string]interface{})

	if in.ChannelId != nil {
		out["channel_id"] = in.ChannelId
		slack = append(slack, out)
	}
	return slack
}

func expandRouteSlack(in []interface{}) *amixrAPI.SlackRoute {
	slackRoute := amixrAPI.SlackRoute{}

	for _, r := range in {
		inputMap := r.(map[string]interface{})
		if inputMap["channel_id"] != "" {
			channelId := inputMap["channel_id"].(string)
			slackRoute.ChannelId = &channelId
		}
	}

	return &slackRoute
}

func flattenTemplates(in *amixrAPI.Templates) []map[string]interface{} {
	templates := make([]map[string]interface{}, 0, 1)
	out := make(map[string]interface{})

	out["grouping_key"] = in.GroupingKey
	out["resolve_signal"] = in.ResolveSignal
	out["slack"] = flattenSlackTemplate(in.Slack)

	add := false

	if in.GroupingKey != nil {
		out["grouping_key"] = in.GroupingKey
		add = true
	}
	if in.ResolveSignal != nil {
		out["resolve_signal"] = in.ResolveSignal
		add = true
	}
	if in.Slack != nil {
		flattenSlackTemplate := flattenSlackTemplate(in.Slack)
		if len(flattenSlackTemplate) > 0 {
			out["resolve_signal"] = in.ResolveSignal
			add = true
		}
	}

	if add {
		templates = append(templates, out)
	}

	return templates
}

func flattenSlackTemplate(in *amixrAPI.SlackTemplate) []map[string]interface{} {
	slackTemplates := make([]map[string]interface{}, 0, 1)

	add := false

	slackTemplate := make(map[string]interface{})

	if in.Title != nil {
		slackTemplate["title"] = in.Title
		add = true
	}
	if in.ImageURL != nil {
		slackTemplate["image_url"] = in.ImageURL
		add = true
	}
	if in.Message != nil {
		slackTemplate["message"] = in.Message
		add = true
	}

	if add {
		slackTemplates = append(slackTemplates, slackTemplate)
	}

	return slackTemplates
}

func expandTemplates(input []interface{}) *amixrAPI.Templates {
	templates := amixrAPI.Templates{}

	for _, r := range input {
		inputMap := r.(map[string]interface{})
		if inputMap["grouping_key"] != "" {
			gk := inputMap["grouping_key"].(string)
			templates.GroupingKey = &gk
		}
		if inputMap["resolve_signal"] != "" {
			rs := inputMap["resolve_signal"].(string)
			templates.ResolveSignal = &rs
		}
		if inputMap["slack"] == nil {
			templates.Slack = nil
		} else {
			templates.Slack = expandSlackTemplate(inputMap["slack"].([]interface{}))
		}
	}
	return &templates
}

func expandSlackTemplate(in []interface{}) *amixrAPI.SlackTemplate {
	slackTemplate := amixrAPI.SlackTemplate{}
	for _, r := range in {
		inputMap := r.(map[string]interface{})
		if inputMap["title"] != "" {
			t := inputMap["title"].(string)
			slackTemplate.Title = &t
		}
		if inputMap["message"] != "" {
			m := inputMap["message"].(string)
			slackTemplate.Message = &m
		}
		if inputMap["image_url"] != "" {
			iu := inputMap["image_url"].(string)
			slackTemplate.ImageURL = &iu
		}
	}
	return &slackTemplate
}

func flattenDefaultRoute(in *amixrAPI.DefaultRoute) []map[string]interface{} {
	defaultRoute := make([]map[string]interface{}, 0, 1)
	out := make(map[string]interface{})
	out["id"] = in.ID
	out["escalation_chain_id"] = in.EscalationChainId
	out["slack"] = flattenRouteSlack(in.SlackRoute)

	defaultRoute = append(defaultRoute, out)
	return defaultRoute
}

func expandDefaultRoute(input []interface{}) *amixrAPI.DefaultRoute {
	defaultRoute := amixrAPI.DefaultRoute{}

	for _, r := range input {
		inputMap := r.(map[string]interface{})
		id := inputMap["id"].(string)
		defaultRoute.ID = id
		if inputMap["escalation_chain_id"] != "" {
			escalation_chain_id := inputMap["escalation_chain_id"].(string)
			defaultRoute.EscalationChainId = &escalation_chain_id
		}
		if inputMap["slack"] == nil {
			defaultRoute.SlackRoute = nil
		} else {
			defaultRoute.SlackRoute = expandRouteSlack(inputMap["slack"].([]interface{}))
		}
	}
	return &defaultRoute
}
