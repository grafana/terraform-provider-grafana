package oncall

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
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
	"direct_paging",
	"jira",
	"zendesk",
}

var integrationTypesVerbal = strings.Join(integrationTypes, ", ")

func resourceIntegration() *common.Resource {
	schema := &schema.Resource{
		Description: `
* [Official documentation](https://grafana.com/docs/oncall/latest/configure/integrations/)
* [HTTP API](https://grafana.com/docs/oncall/latest/oncall-api-reference/)
`,

		CreateContext: withClient[schema.CreateContextFunc](resourceIntegrationCreate),
		ReadContext:   withClient[schema.ReadContextFunc](resourceIntegrationRead),
		UpdateContext: withClient[schema.UpdateContextFunc](resourceIntegrationUpdate),
		DeleteContext: withClient[schema.DeleteContextFunc](resourceIntegrationDelete),
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
				Description: "The ID of the OnCall team (using the `grafana_oncall_team` datasource).",
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
						"microsoft_teams": onCallTemplate("Templates for Microsoft Teams. **NOTE**: Microsoft Teams templates are only available on Grafana Cloud.", true, true),
						"mobile_app":      onCallTemplate("Templates for Mobile app push notifications.", true, false),
						"phone_call":      onCallTemplate("Templates for Phone Call.", false, false),
						"sms":             onCallTemplate("Templates for SMS.", false, false),
						"email":           onCallTemplate("Templates for Email.", true, false),
					},
				},
				MaxItems:    1,
				Description: "Jinja2 templates for Alert payload. An empty templates block will be ignored.",
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					if old != "" && new == "" || old == "" && new != "" {
						return false
					}

					oldTemplate, newTemplate := d.GetChange("templates")

					getTemplatesOrEmpty := func(template any) map[string]any {
						list := template.([]any)
						if len(list) > 0 && list[0] != nil {
							return list[0].(map[string]any)
						}
						return map[string]any{}
					}
					oldTemplateMap, newTemplateMap := getTemplatesOrEmpty(oldTemplate), getTemplatesOrEmpty(newTemplate)
					if len(oldTemplateMap) != len(newTemplateMap) {
						return false
					}
					for k, v := range oldTemplateMap {
						// Convert everything to string to be able to compare across types.
						// We're only interested in the actual value here,
						// and Terraform will implicitly convert a string to a number, and vice versa.
						if fmt.Sprintf("%v", newTemplateMap[k]) != fmt.Sprintf("%v", v) {
							return false
						}
					}
					return true
				},
			},
			"labels": {
				Type: schema.TypeSet,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": {
							Type:     schema.TypeString,
							Required: true,
						},
						"value": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
				Optional:    true,
				Description: "Static labels as key/value pairs.",
			},
			"dynamic_labels": {
				Type: schema.TypeSet,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": {
							Type:     schema.TypeString,
							Required: true,
						},
						"value": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
				Optional:    true,
				Description: "Dynamic labels as key/value pairs.",
			},
		},
	}

	return common.NewLegacySDKResource(
		common.CategoryOnCall,
		"grafana_oncall_integration",
		resourceID,
		schema,
	).
		WithLister(oncallListerFunction(listIntegrations)).
		WithPreferredResourceNameField("name")
}

func listIntegrations(client *onCallAPI.Client, listOptions onCallAPI.ListOptions) (ids []string, nextPage *string, err error) {
	resp, _, err := client.Integrations.ListIntegrations(&onCallAPI.ListIntegrationOptions{ListOptions: listOptions})
	if err != nil {
		return nil, nil, err
	}
	for _, i := range resp.Integrations {
		ids = append(ids, i.ID)
	}
	return ids, resp.Next, nil
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

func resourceIntegrationCreate(ctx context.Context, d *schema.ResourceData, client *onCallAPI.Client) diag.Diagnostics {
	teamIDData := d.Get("team_id").(string)
	nameData := d.Get("name").(string)
	typeData := d.Get("type").(string)
	templatesData := d.Get("templates").([]any)
	defaultRouteData := d.Get("default_route").([]any)
	labelsData := d.Get("labels").([]any)
	dynamicLabelsData := d.Get("dynamic_labels").([]any)

	createOptions := &onCallAPI.CreateIntegrationOptions{
		TeamId:        teamIDData,
		Name:          nameData,
		Type:          typeData,
		Templates:     expandTemplates(templatesData),
		DefaultRoute:  expandDefaultRoute(defaultRouteData),
		Labels:        expandLabels(labelsData),
		DynamicLabels: expandLabels(dynamicLabelsData),
	}

	integration, _, err := client.Integrations.CreateIntegration(createOptions)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(integration.ID)

	return resourceIntegrationRead(ctx, d, client)
}

func resourceIntegrationUpdate(ctx context.Context, d *schema.ResourceData, client *onCallAPI.Client) diag.Diagnostics {
	nameData := d.Get("name").(string)
	teamIDData := d.Get("team_id").(string)
	templateData := d.Get("templates").([]any)
	defaultRouteData := d.Get("default_route").([]any)
	labelsData := d.Get("labels").([]any)
	dynamicLabelsData := d.Get("dynamic_labels").([]any)

	updateOptions := &onCallAPI.UpdateIntegrationOptions{
		Name:          nameData,
		TeamId:        teamIDData,
		Templates:     expandTemplates(templateData),
		DefaultRoute:  expandDefaultRoute(defaultRouteData),
		Labels:        expandLabels(labelsData),
		DynamicLabels: expandLabels(dynamicLabelsData),
	}

	integration, _, err := client.Integrations.UpdateIntegration(d.Id(), updateOptions)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(integration.ID)

	return resourceIntegrationRead(ctx, d, client)
}

func resourceIntegrationRead(ctx context.Context, d *schema.ResourceData, client *onCallAPI.Client) diag.Diagnostics {
	options := &onCallAPI.GetIntegrationOptions{}
	integration, r, err := client.Integrations.GetIntegration(d.Id(), options)
	if err != nil {
		if r != nil && r.StatusCode == http.StatusNotFound {
			return common.WarnMissing("integration", d)
		}
		return diag.FromErr(err)
	}

	d.Set("team_id", integration.TeamId)
	d.Set("default_route", flattenDefaultRoute(integration.DefaultRoute, d))
	d.Set("name", integration.Name)
	d.Set("type", integration.Type)
	d.Set("templates", flattenTemplates(integration.Templates))
	d.Set("link", integration.Link)
	d.Set("labels", flattenLabels(integration.Labels))
	d.Set("dynamic_labels", flattenLabels(integration.DynamicLabels))

	return nil
}

func resourceIntegrationDelete(ctx context.Context, d *schema.ResourceData, client *onCallAPI.Client) diag.Diagnostics {
	options := &onCallAPI.DeleteIntegrationOptions{}
	_, err := client.Integrations.DeleteIntegration(d.Id(), options)
	return diag.FromErr(err)
}

func flattenRouteSlack(in *onCallAPI.SlackRoute) []map[string]any {
	slack := make([]map[string]any, 0, 1)

	out := make(map[string]any)

	out["channel_id"] = in.ChannelId
	out["enabled"] = in.Enabled

	slack = append(slack, out)

	return slack
}

func expandRouteSlack(in []any) *onCallAPI.SlackRoute {
	slackRoute := onCallAPI.SlackRoute{}

	for _, r := range in {
		inputMap := r.(map[string]any)
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

func flattenRouteTelegram(in *onCallAPI.TelegramRoute) []map[string]any {
	telegram := make([]map[string]any, 0, 1)

	out := make(map[string]any)

	out["id"] = in.Id
	out["enabled"] = in.Enabled
	telegram = append(telegram, out)
	return telegram
}

func expandRouteTelegram(in []any) *onCallAPI.TelegramRoute {
	telegramRoute := onCallAPI.TelegramRoute{}

	for _, r := range in {
		inputMap := r.(map[string]any)
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

func flattenRouteMSTeams(in *onCallAPI.MSTeamsRoute) []map[string]any {
	msTeams := make([]map[string]any, 0, 1)

	out := make(map[string]any)

	if in != nil {
		out["id"] = in.Id
		out["enabled"] = in.Enabled
		msTeams = append(msTeams, out)
	}

	return msTeams
}

func expandRouteMSTeams(in []any) *onCallAPI.MSTeamsRoute {
	msTeamsRoute := onCallAPI.MSTeamsRoute{}

	for _, r := range in {
		inputMap := r.(map[string]any)
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

func flattenTemplates(in *onCallAPI.Templates) []map[string]any {
	templates := make([]map[string]any, 0, 1)
	out := make(map[string]any)
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

	if in.MobileApp != nil {
		flattenMobileAppTemplate := flattenTitleMessageTemplate(in.MobileApp)
		if len(flattenMobileAppTemplate) > 0 {
			out["mobile_app"] = flattenMobileAppTemplate
			add = true
		}
	}

	if add {
		templates = append(templates, out)
	}

	return templates
}

func flattenTitleMessageImageTemplate(in *onCallAPI.TitleMessageImageTemplate) []map[string]any {
	templates := make([]map[string]any, 0, 1)

	add := false

	template := make(map[string]any)

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

func flattenTitleMessageTemplate(in *onCallAPI.TitleMessageTemplate) []map[string]any {
	templates := make([]map[string]any, 0, 1)

	add := false

	template := make(map[string]any)

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

func flattenTitleTemplate(in *onCallAPI.TitleTemplate) []map[string]any {
	templates := make([]map[string]any, 0, 1)

	add := false

	template := make(map[string]any)

	if in.Title != nil {
		template["title"] = in.Title
		add = true
	}
	if add {
		templates = append(templates, template)
	}

	return templates
}

func expandTemplates(input []any) *onCallAPI.Templates {
	templates := onCallAPI.Templates{}

	for _, r := range input {
		if r == nil {
			continue
		}

		inputMap := r.(map[string]any)
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
			templates.Slack = expandTitleMessageImageTemplate(inputMap["slack"].([]any))
		}

		if inputMap["web"] == nil {
			templates.Web = nil
		} else {
			templates.Web = expandTitleMessageImageTemplate(inputMap["web"].([]any))
		}

		if inputMap["microsoft_teams"] == nil {
			templates.MSTeams = nil
		} else {
			templates.MSTeams = expandTitleMessageImageTemplate(inputMap["microsoft_teams"].([]any))
		}

		if inputMap["telegram"] == nil {
			templates.Telegram = nil
		} else {
			templates.Telegram = expandTitleMessageImageTemplate(inputMap["telegram"].([]any))
		}

		if inputMap["phone_call"] == nil {
			templates.PhoneCall = nil
		} else {
			templates.PhoneCall = expandTitleTemplate(inputMap["phone_call"].([]any))
		}

		if inputMap["sms"] == nil {
			templates.SMS = nil
		} else {
			templates.SMS = expandTitleTemplate(inputMap["sms"].([]any))
		}

		if inputMap["email"] == nil {
			templates.Email = nil
		} else {
			templates.Email = expandTitleMessageTemplate(inputMap["email"].([]any))
		}

		if inputMap["mobile_app"] == nil {
			templates.MobileApp = nil
		} else {
			templates.MobileApp = expandTitleMessageTemplate(inputMap["mobile_app"].([]any))
		}
	}
	return &templates
}

func expandTitleMessageImageTemplate(in []any) *onCallAPI.TitleMessageImageTemplate {
	template := onCallAPI.TitleMessageImageTemplate{}
	for _, r := range in {
		inputMap := r.(map[string]any)
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

func expandTitleTemplate(in []any) *onCallAPI.TitleTemplate {
	template := onCallAPI.TitleTemplate{}
	for _, r := range in {
		inputMap := r.(map[string]any)
		if inputMap["title"] != "" {
			t := inputMap["title"].(string)
			template.Title = &t
		}
	}
	return &template
}

func expandTitleMessageTemplate(in []any) *onCallAPI.TitleMessageTemplate {
	template := onCallAPI.TitleMessageTemplate{}
	for _, r := range in {
		inputMap := r.(map[string]any)
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

func flattenDefaultRoute(in *onCallAPI.DefaultRoute, d *schema.ResourceData) []map[string]any {
	defaultRoute := make([]map[string]any, 0, 1)
	out := make(map[string]any)
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

func expandDefaultRoute(input []any) *onCallAPI.DefaultRoute {
	defaultRoute := onCallAPI.DefaultRoute{}

	for _, r := range input {
		inputMap := r.(map[string]any)
		id := inputMap["id"].(string)
		defaultRoute.ID = id
		if inputMap["escalation_chain_id"] != "" {
			escalationChainID := inputMap["escalation_chain_id"].(string)
			defaultRoute.EscalationChainId = &escalationChainID
		}
		if inputMap["slack"] == nil {
			defaultRoute.SlackRoute = nil
		} else {
			defaultRoute.SlackRoute = expandRouteSlack(inputMap["slack"].([]any))
		}
		if inputMap["telegram"] == nil {
			defaultRoute.TelegramRoute = nil
		} else {
			defaultRoute.TelegramRoute = expandRouteTelegram(inputMap["telegram"].([]any))
		}
		if inputMap["msteams"] == nil {
			defaultRoute.MSTeamsRoute = nil
		} else {
			defaultRoute.MSTeamsRoute = expandRouteMSTeams(inputMap["msteams"].([]any))
		}
	}
	return &defaultRoute
}

func expandLabels(input []any) []*onCallAPI.Label {
	labelsData := make([]*onCallAPI.Label, 0, 1)

	for _, r := range input {
		inputMap := r.(map[string]any)
		key, keyExists := inputMap["key"]
		value, valueExists := inputMap["value"]

		if keyExists && valueExists {
			label := onCallAPI.Label{
				Key:   onCallAPI.KeyValueName{Name: key.(string)},
				Value: onCallAPI.KeyValueName{Name: value.(string)},
			}
			labelsData = append(labelsData, &label)
		}
	}
	return labelsData
}

func flattenLabels(labels []*onCallAPI.Label) []map[string]string {
	flattenedLabels := make([]map[string]string, 0, 1)

	for _, l := range labels {
		flattenedLabels = append(flattenedLabels, map[string]string{
			"id":    l.Key.Name,
			"key":   l.Key.Name,
			"value": l.Value.Name,
		})
	}

	return flattenedLabels
}
