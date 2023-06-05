package oncall

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

var integrationTypes = []string{
	"grafana",
	"grafana_alerting",
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

func ResourceIntegration() *schema.Resource {
	return &schema.Resource{
		Description: `
* [Official documentation](https://grafana.com/docs/oncall/latest/integrations/)
* [HTTP API](https://grafana.com/docs/oncall/latest/oncall-api-reference/)
`,

		CreateContext: ResourceIntegrationCreate,
		ReadContext:   ResourceIntegrationRead,
		UpdateContext: ResourceIntegrationUpdate,
		DeleteContext: ResourceIntegrationDelete,
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
				Description: "The ID of the OnCall team. To get one, create a team in Grafana, and navigate to the OnCall plugin (to sync the team with OnCall). You can then get the ID using the `grafana_oncall_team` datasource.",
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
							Description: "Slack-specific settings for a route.",
							MaxItems:    1,
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
						"acknowledge_signal": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Template for sending a signal to acknowledge the Incident.",
						},
						"source_link": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Template for a source link.",
						},
						"slack":           onCallTemplate("Templates for Slack.", true, true),
						"web":             onCallTemplate("Templates for Web.", true, true),
						"telegram":        onCallTemplate("Templates for Telegram.", true, true),
						"microsoft_teams": onCallTemplate("Templates for Microsoft Teams.", true, true),
						"phone_call":      onCallTemplate("Templates for Phone Call.", false, false),
						"sms":             onCallTemplate("Templates for SMS.", false, false),
						"email":           onCallTemplate("Templates for Email.", true, false),
					},
				},
				MaxItems:    1,
				Description: "Jinja2 templates for Alert payload.",
			},
		},
	}
}

func onCallTemplate(description string, hasMessage, hasImage bool) *schema.Schema {
	elem := map[string]*schema.Schema{
		"title": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Template for Alert title.",
		},
	}

	if hasMessage {
		elem["message"] = &schema.Schema{
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Template for Alert message.",
		}
	}

	if hasImage {
		elem["image_url"] = &schema.Schema{
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Template for Alert image url.",
		}
	}

	templateSchema := schema.Schema{
		Type:        schema.TypeList,
		Optional:    true,
		Description: description,
		Elem: &schema.Resource{
			Schema: elem,
		},
		MaxItems: 1,
	}

	return &templateSchema
}

func ResourceIntegrationCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client).OnCallClient

	teamIDData := d.Get("team_id").(string)
	nameData := d.Get("name").(string)
	typeData := d.Get("type").(string)
	templatesData := d.Get("templates").([]interface{})
	defaultRouteData := d.Get("default_route").([]interface{})

	createOptions := &onCallAPI.CreateIntegrationOptions{
		TeamId:       teamIDData,
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

	return ResourceIntegrationRead(ctx, d, m)
}

func ResourceIntegrationUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client).OnCallClient

	nameData := d.Get("name").(string)
	teamIDData := d.Get("team_id").(string)
	templateData := d.Get("templates").([]interface{})
	defaultRouteData := d.Get("default_route").([]interface{})

	updateOptions := &onCallAPI.UpdateIntegrationOptions{
		Name:         nameData,
		TeamId:       teamIDData,
		Templates:    expandTemplates(templateData),
		DefaultRoute: expandDefaultRoute(defaultRouteData),
	}

	integration, _, err := client.Integrations.UpdateIntegration(d.Id(), updateOptions)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(integration.ID)

	return ResourceIntegrationRead(ctx, d, m)
}

func ResourceIntegrationRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client).OnCallClient
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
	d.Set("default_route", flattenDefaultRoute(integration.DefaultRoute, d))
	d.Set("name", integration.Name)
	d.Set("type", integration.Type)
	d.Set("templates", flattenTemplates(integration.Templates))
	d.Set("link", integration.Link)

	return nil
}

func ResourceIntegrationDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client).OnCallClient
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

	out["channel_id"] = in.ChannelId
	out["enabled"] = in.Enabled

	slack = append(slack, out)

	return slack
}

func expandRouteSlack(in []interface{}) *onCallAPI.SlackRoute {
	slackRoute := onCallAPI.SlackRoute{}

	for _, r := range in {
		inputMap := r.(map[string]interface{})
		if inputMap["channel_id"] != "" {
			channelID := inputMap["channel_id"].(string)
			slackRoute.ChannelId = &channelID
		}
		if enabled, ok := inputMap["enabled"].(bool); ok {
			slackRoute.Enabled = enabled
		}
	}

	return &slackRoute
}

func flattenRouteTelegram(in *onCallAPI.TelegramRoute) []map[string]interface{} {
	telegram := make([]map[string]interface{}, 0, 1)

	out := make(map[string]interface{})

	out["id"] = in.Id
	out["enabled"] = in.Enabled
	telegram = append(telegram, out)
	return telegram
}

func expandRouteTelegram(in []interface{}) *onCallAPI.TelegramRoute {
	telegramRoute := onCallAPI.TelegramRoute{}

	for _, r := range in {
		inputMap := r.(map[string]interface{})
		if inputMap["id"] != "" {
			id := inputMap["id"].(string)
			telegramRoute.Id = &id
		}
		if enabled, ok := inputMap["enabled"].(bool); ok {
			telegramRoute.Enabled = enabled
		}
	}

	return &telegramRoute
}

func flattenRouteMSTeams(in *onCallAPI.MSTeamsRoute) []map[string]interface{} {
	msTeams := make([]map[string]interface{}, 0, 1)

	out := make(map[string]interface{})

	if in != nil {
		out["id"] = in.Id
		out["enabled"] = in.Enabled
		msTeams = append(msTeams, out)
	}

	return msTeams
}

func expandRouteMSTeams(in []interface{}) *onCallAPI.MSTeamsRoute {
	msTeamsRoute := onCallAPI.MSTeamsRoute{}

	for _, r := range in {
		inputMap := r.(map[string]interface{})
		if inputMap["id"] != "" {
			id := inputMap["id"].(string)
			msTeamsRoute.Id = &id
		}
		if enabled, ok := inputMap["enabled"].(bool); ok {
			msTeamsRoute.Enabled = enabled
		}
	}

	return &msTeamsRoute
}

func flattenTemplates(in *onCallAPI.Templates) []map[string]interface{} {
	templates := make([]map[string]interface{}, 0, 1)
	out := make(map[string]interface{})
	add := false

	if in.GroupingKey != nil {
		out["grouping_key"] = in.GroupingKey
		add = true
	}
	if in.ResolveSignal != nil {
		out["resolve_signal"] = in.ResolveSignal
		add = true
	}
	if in.AcknowledgeSignal != nil {
		out["acknowledge_signal"] = in.AcknowledgeSignal
		add = true
	}
	if in.SourceLink != nil {
		out["source_link"] = in.SourceLink
		add = true
	}
	if in.Slack != nil {
		flattenSlackTemplate := flattenTitleMessageImageTemplate(in.Slack)
		if len(flattenSlackTemplate) > 0 {
			out["slack"] = flattenSlackTemplate
			add = true
		}
	}

	if in.Web != nil {
		flattenWebTemplate := flattenTitleMessageImageTemplate(in.Web)
		if len(flattenWebTemplate) > 0 {
			out["web"] = flattenWebTemplate
			add = true
		}
	}

	if in.MSTeams != nil {
		flattenMSTeamsTemplate := flattenTitleMessageImageTemplate(in.MSTeams)
		if len(flattenMSTeamsTemplate) > 0 {
			out["microsoft_teams"] = flattenMSTeamsTemplate
			add = true
		}
	}

	if in.Telegram != nil {
		flattenTelegramTemplate := flattenTitleMessageImageTemplate(in.Telegram)
		if len(flattenTelegramTemplate) > 0 {
			out["telegram"] = flattenTelegramTemplate
			add = true
		}
	}

	if in.Email != nil {
		flattenEmailTemplate := flattenTitleMessageTemplate(in.Email)
		if len(flattenEmailTemplate) > 0 {
			out["email"] = flattenEmailTemplate
			add = true
		}
	}

	if in.PhoneCall != nil {
		flattenPhoneCallTemplate := flattenTitleTemplate(in.PhoneCall)
		if len(flattenPhoneCallTemplate) > 0 {
			out["phone_call"] = flattenPhoneCallTemplate
			add = true
		}
	}
	if in.SMS != nil {
		flattenSMSTemplate := flattenTitleTemplate(in.SMS)
		if len(flattenSMSTemplate) > 0 {
			out["sms"] = flattenSMSTemplate
			add = true
		}
	}
	if add {
		templates = append(templates, out)
	}

	return templates
}

func flattenTitleMessageImageTemplate(in *onCallAPI.TitleMessageImageTemplate) []map[string]interface{} {
	templates := make([]map[string]interface{}, 0, 1)

	add := false

	template := make(map[string]interface{})

	if in.Title != nil {
		template["title"] = in.Title
		add = true
	}
	if in.ImageURL != nil {
		template["image_url"] = in.ImageURL
		add = true
	}
	if in.Message != nil {
		template["message"] = in.Message
		add = true
	}
	if add {
		templates = append(templates, template)
	}

	return templates
}

func flattenTitleMessageTemplate(in *onCallAPI.TitleMessageTemplate) []map[string]interface{} {
	templates := make([]map[string]interface{}, 0, 1)

	add := false

	template := make(map[string]interface{})

	if in.Title != nil {
		template["title"] = in.Title
		add = true
	}
	if in.Message != nil {
		template["message"] = in.Message
		add = true
	}
	if add {
		templates = append(templates, template)
	}

	return templates
}

func flattenTitleTemplate(in *onCallAPI.TitleTemplate) []map[string]interface{} {
	templates := make([]map[string]interface{}, 0, 1)

	add := false

	template := make(map[string]interface{})

	if in.Title != nil {
		template["title"] = in.Title
		add = true
	}
	if add {
		templates = append(templates, template)
	}

	return templates
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
		if inputMap["acknowledge_signal"] != "" {
			rs := inputMap["acknowledge_signal"].(string)
			templates.AcknowledgeSignal = &rs
		}
		if inputMap["source_link"] != "" {
			rs := inputMap["source_link"].(string)
			templates.SourceLink = &rs
		}

		if inputMap["slack"] == nil {
			templates.Slack = nil
		} else {
			templates.Slack = expandTitleMessageImageTemplate(inputMap["slack"].([]interface{}))
		}

		if inputMap["web"] == nil {
			templates.Web = nil
		} else {
			templates.Web = expandTitleMessageImageTemplate(inputMap["web"].([]interface{}))
		}

		if inputMap["microsoft_teams"] == nil {
			templates.MSTeams = nil
		} else {
			templates.MSTeams = expandTitleMessageImageTemplate(inputMap["microsoft_teams"].([]interface{}))
		}

		if inputMap["telegram"] == nil {
			templates.Telegram = nil
		} else {
			templates.Telegram = expandTitleMessageImageTemplate(inputMap["telegram"].([]interface{}))
		}

		if inputMap["phone_call"] == nil {
			templates.PhoneCall = nil
		} else {
			templates.PhoneCall = expandTitleTemplate(inputMap["phone_call"].([]interface{}))
		}

		if inputMap["sms"] == nil {
			templates.SMS = nil
		} else {
			templates.SMS = expandTitleTemplate(inputMap["sms"].([]interface{}))
		}

		if inputMap["email"] == nil {
			templates.Email = nil
		} else {
			templates.Email = expandTitleMessageTemplate(inputMap["email"].([]interface{}))
		}
	}
	return &templates
}

func expandTitleMessageImageTemplate(in []interface{}) *onCallAPI.TitleMessageImageTemplate {
	template := onCallAPI.TitleMessageImageTemplate{}
	for _, r := range in {
		inputMap := r.(map[string]interface{})
		if inputMap["title"] != "" {
			t := inputMap["title"].(string)
			template.Title = &t
		}
		if inputMap["message"] != "" {
			m := inputMap["message"].(string)
			template.Message = &m
		}
		if inputMap["image_url"] != "" {
			iu := inputMap["image_url"].(string)
			template.ImageURL = &iu
		}
	}
	return &template
}

func expandTitleTemplate(in []interface{}) *onCallAPI.TitleTemplate {
	template := onCallAPI.TitleTemplate{}
	for _, r := range in {
		inputMap := r.(map[string]interface{})
		if inputMap["title"] != "" {
			t := inputMap["title"].(string)
			template.Title = &t
		}
	}
	return &template
}

func expandTitleMessageTemplate(in []interface{}) *onCallAPI.TitleMessageTemplate {
	template := onCallAPI.TitleMessageTemplate{}
	for _, r := range in {
		inputMap := r.(map[string]interface{})
		if inputMap["title"] != "" {
			t := inputMap["title"].(string)
			template.Title = &t
		}
		if inputMap["message"] != "" {
			m := inputMap["message"].(string)
			template.Message = &m
		}
	}
	return &template
}

func flattenDefaultRoute(in *onCallAPI.DefaultRoute, d *schema.ResourceData) []map[string]interface{} {
	defaultRoute := make([]map[string]interface{}, 0, 1)
	out := make(map[string]interface{})
	out["id"] = in.ID
	out["escalation_chain_id"] = in.EscalationChainId
	// Set messengers data only if related fields are present
	_, slackOk := d.GetOk("default_route.0.slack")
	if slackOk {
		out["slack"] = flattenRouteSlack(in.SlackRoute)
	}
	_, telegramOk := d.GetOk("default_route.0.telegram")
	if telegramOk {
		out["telegram"] = flattenRouteTelegram(in.TelegramRoute)
	}
	_, msteamsOk := d.GetOk("default_route.0.msteams")
	if msteamsOk {
		out["msteams"] = flattenRouteMSTeams(in.MSTeamsRoute)
	}

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
			escalationChainID := inputMap["escalation_chain_id"].(string)
			defaultRoute.EscalationChainId = &escalationChainID
		}
		if inputMap["slack"] == nil {
			defaultRoute.SlackRoute = nil
		} else {
			defaultRoute.SlackRoute = expandRouteSlack(inputMap["slack"].([]interface{}))
		}
		if inputMap["telegram"] == nil {
			defaultRoute.TelegramRoute = nil
		} else {
			defaultRoute.TelegramRoute = expandRouteTelegram(inputMap["telegram"].([]interface{}))
		}
		if inputMap["msteams"] == nil {
			defaultRoute.MSTeamsRoute = nil
		} else {
			defaultRoute.MSTeamsRoute = expandRouteMSTeams(inputMap["msteams"].([]interface{}))
		}
	}
	return &defaultRoute
}
