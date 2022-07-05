package grafana

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
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

var integrationTypesVerbal = strings.Join(integrationTypes, ", ")

func ResourceOnCallIntegration() *schema.Resource {
	return &schema.Resource{
		Description: `
* [Official documentation](https://grafana.com/docs/grafana-cloud/oncall/integrations/)
* [HTTP API](https://grafana.com/docs/grafana-cloud/oncall/oncall-api-reference/)
`,

		CreateContext: ResourceOnCallIntegrationCreate,
		ReadContext:   ResourceOnCallIntegrationRead,
		UpdateContext: ResourceOnCallIntegrationUpdate,
		DeleteContext: ResourceOnCallIntegrationDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
				Description:  "The name of the service integration.",
			},
			"team_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The id of the team.",
			},
			"type": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice(integrationTypes, false),
				ForceNew:     true,
				Description:  fmt.Sprintf("The type of integration. Can be %s.", integrationTypesVerbal),
			},
			"default_route": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"escalation_chain_id": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "The ID of the escalation chain.",
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
							Description: "Slack-specific settings for a route.",
							MaxItems:    1,
						},
					},
				},
				MaxItems:    1,
				Description: "The Default route for all alerts from the given integration",
			},
			"link": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The link for using in an integrated tool.",
			},
			"templates": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"resolve_signal": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Template for sending a signal to resolve the Incident.",
						},
						"grouping_key": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Template for the key by which alerts are grouped.",
						},
						"slack": {
							Type:        schema.TypeList,
							Optional:    true,
							Description: "Templates for Slack.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"title": {
										Type:        schema.TypeString,
										Optional:    true,
										Description: "Template for Alert title.",
									},
									"message": {
										Type:        schema.TypeString,
										Optional:    true,
										Description: "Template for Alert message.",
									},
									"image_url": {
										Type:        schema.TypeString,
										Optional:    true,
										Description: "Template for Alert image url.",
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

func ResourceOnCallIntegrationCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*client).onCallAPI
	if client == nil {
		return diag.Errorf("grafana OnCall api client is not configured")
	}

	teamIdData := d.Get("team_id").(string)
	nameData := d.Get("name").(string)
	typeData := d.Get("type").(string)
	templatesData := d.Get("templates").([]interface{})
	defaultRouteData := d.Get("default_route").([]interface{})

	createOptions := &onCallAPI.CreateIntegrationOptions{
		TeamId:       teamIdData,
		Name:         nameData,
		Type:         typeData,
		Templates:    expandTemplates(templatesData),
		DefaultRoute: expandDefaultRoute(defaultRouteData),
	}

	integration, _, err := client.Integrations.CreateIntegration(createOptions)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(integration.ID)

	return ResourceOnCallIntegrationRead(ctx, d, m)
}

func ResourceOnCallIntegrationUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*client).onCallAPI
	if client == nil {
		return diag.Errorf("grafana OnCall api client is not configured")
	}

	nameData := d.Get("name").(string)
	templateData := d.Get("templates").([]interface{})
	defaultRouteData := d.Get("default_route").([]interface{})

	updateOptions := &onCallAPI.UpdateIntegrationOptions{
		Name:         nameData,
		Templates:    expandTemplates(templateData),
		DefaultRoute: expandDefaultRoute(defaultRouteData),
	}

	integration, _, err := client.Integrations.UpdateIntegration(d.Id(), updateOptions)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(integration.ID)

	return ResourceOnCallIntegrationRead(ctx, d, m)
}

func ResourceOnCallIntegrationRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*client).onCallAPI
	if client == nil {
		return diag.Errorf("grafana OnCall api client is not configured")
	}
	options := &onCallAPI.GetIntegrationOptions{}
	integration, r, err := client.Integrations.GetIntegration(d.Id(), options)
	if err != nil {
		if r != nil && r.StatusCode == http.StatusNotFound {
			log.Printf("[WARN] removing integreation %s from state because it no longer exists", d.Get("name").(string))
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	d.Set("team_id", integration.TeamId)
	d.Set("default_route", flattenDefaultRoute(integration.DefaultRoute))
	d.Set("name", integration.Name)
	d.Set("type", integration.Type)
	d.Set("templates", flattenTemplates(integration.Templates))
	d.Set("link", integration.Link)

	return nil
}

func ResourceOnCallIntegrationDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*client).onCallAPI
	if client == nil {
		return diag.Errorf("grafana OnCall api client is not configured")
	}
	options := &onCallAPI.DeleteIntegrationOptions{}
	_, err := client.Integrations.DeleteIntegration(d.Id(), options)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	return nil
}

func flattenRouteSlack(in *onCallAPI.SlackRoute) []map[string]interface{} {
	slack := make([]map[string]interface{}, 0, 1)

	out := make(map[string]interface{})

	if in.ChannelId != nil {
		out["channel_id"] = in.ChannelId
		slack = append(slack, out)
	}
	return slack
}

func expandRouteSlack(in []interface{}) *onCallAPI.SlackRoute {
	slackRoute := onCallAPI.SlackRoute{}

	for _, r := range in {
		inputMap := r.(map[string]interface{})
		if inputMap["channel_id"] != "" {
			channelId := inputMap["channel_id"].(string)
			slackRoute.ChannelId = &channelId
		}
	}

	return &slackRoute
}

func flattenTemplates(in *onCallAPI.Templates) []map[string]interface{} {
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

func flattenSlackTemplate(in *onCallAPI.SlackTemplate) []map[string]interface{} {
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

func expandTemplates(input []interface{}) *onCallAPI.Templates {
	templates := onCallAPI.Templates{}

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

func expandSlackTemplate(in []interface{}) *onCallAPI.SlackTemplate {
	slackTemplate := onCallAPI.SlackTemplate{}
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

func flattenDefaultRoute(in *onCallAPI.DefaultRoute) []map[string]interface{} {
	defaultRoute := make([]map[string]interface{}, 0, 1)
	out := make(map[string]interface{})
	out["id"] = in.ID
	out["escalation_chain_id"] = in.EscalationChainId
	out["slack"] = flattenRouteSlack(in.SlackRoute)

	defaultRoute = append(defaultRoute, out)
	return defaultRoute
}

func expandDefaultRoute(input []interface{}) *onCallAPI.DefaultRoute {
	defaultRoute := onCallAPI.DefaultRoute{}

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
