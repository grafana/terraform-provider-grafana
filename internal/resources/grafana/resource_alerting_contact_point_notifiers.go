package grafana

import (
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

type alertmanagerNotifier struct{}

var _ notifier = (*alertmanagerNotifier)(nil)

func (a alertmanagerNotifier) meta() notifierMeta {
	return notifierMeta{
		field:   "alertmanager",
		typeStr: "prometheus-alertmanager",
		desc:    "A contact point that sends notifications to other Alertmanager instances.",
		fieldMapper: map[string]fieldMapper{
			"basic_auth_user":     newKeyMapper("basicAuthUser"),
			"basic_auth_password": newKeyMapper("basicAuthPassword"),
		},
	}
}

func (a alertmanagerNotifier) schema() *schema.Resource {
	r := commonNotifierResource()
	r.Schema["url"] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Description: "The URL of the Alertmanager instance.",
	}
	r.Schema["basic_auth_user"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The username component of the basic auth credentials to use.",
	}
	r.Schema["basic_auth_password"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Sensitive:   true,
		Description: "The password component of the basic auth credentials to use.",
	}
	return r
}

type dingDingNotifier struct{}

var _ notifier = (*dingDingNotifier)(nil)

func (d dingDingNotifier) meta() notifierMeta {
	return notifierMeta{
		field:   "dingding",
		typeStr: "dingding",
		desc:    "A contact point that sends notifications to DingDing.",
		fieldMapper: map[string]fieldMapper{
			"message_type": newKeyMapper("msgType"),
		},
	}
}

func (d dingDingNotifier) schema() *schema.Resource {
	r := commonNotifierResource()
	r.Schema["url"] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Sensitive:   true,
		Description: "The DingDing webhook URL.",
	}
	r.Schema["message_type"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The format of message to send - either 'link' or 'actionCard'",
	}
	r.Schema["message"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The templated content of the message.",
	}
	r.Schema["title"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The templated title of the message.",
	}
	return r
}

type discordNotifier struct{}

var _ notifier = (*discordNotifier)(nil)

func (d discordNotifier) meta() notifierMeta {
	return notifierMeta{
		field:       "discord",
		typeStr:     "discord",
		desc:        "A contact point that sends notifications as Discord messages",
		fieldMapper: map[string]fieldMapper{},
	}
}

func (d discordNotifier) schema() *schema.Resource {
	r := commonNotifierResource()
	r.Schema["url"] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Sensitive:   true,
		Description: "The discord webhook URL.",
	}
	r.Schema["title"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The templated content of the title.",
	}
	r.Schema["message"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Default:     "",
		Description: "The templated content of the message.",
	}
	r.Schema["avatar_url"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Default:     "",
		Description: "The URL of a custom avatar image to use.",
	}
	r.Schema["use_discord_username"] = &schema.Schema{
		Type:        schema.TypeBool,
		Optional:    true,
		Default:     false,
		Description: "Whether to use the bot account's plain username instead of \"Grafana.\"",
	}
	return r
}

type emailNotifier struct{}

var _ notifier = (*emailNotifier)(nil)

func (e emailNotifier) meta() notifierMeta {
	return notifierMeta{
		field:   "email",
		typeStr: "email",
		desc:    "A contact point that sends notifications to an email address.",
		fieldMapper: map[string]fieldMapper{
			"addresses":    newFieldMapper("", packAddrs, unpackAddrs),
			"single_email": newKeyMapper("singleEmail"),
		},
	}
}

func (e emailNotifier) schema() *schema.Resource {
	r := commonNotifierResource()
	r.Schema["addresses"] = &schema.Schema{
		Type:        schema.TypeList,
		Required:    true,
		Description: "The addresses to send emails to.",
		Elem: &schema.Schema{
			Type:         schema.TypeString,
			ValidateFunc: validation.StringIsNotEmpty,
		},
	}
	r.Schema["single_email"] = &schema.Schema{
		Type:        schema.TypeBool,
		Optional:    true,
		Default:     false,
		Description: "Whether to send a single email CC'ing all addresses, rather than a separate email to each address.",
	}
	r.Schema["message"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Default:     "",
		Description: "The templated content of the email.",
	}
	r.Schema["subject"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Default:     "",
		Description: "The templated subject line of the email.",
	}
	return r
}

const addrSeparator = ';'

func packAddrs(addrs any) any {
	return strings.FieldsFunc(addrs.(string), func(r rune) bool {
		switch r {
		case ',', addrSeparator, '\n':
			return true
		}
		return false
	})
}

func unpackAddrs(addrs any) any {
	strs := common.ListToStringSlice(addrs.([]any))
	return strings.Join(strs, string(addrSeparator))
}

type googleChatNotifier struct{}

var _ notifier = (*googleChatNotifier)(nil)

func (g googleChatNotifier) meta() notifierMeta {
	return notifierMeta{
		field:       "googlechat",
		typeStr:     "googlechat",
		desc:        "A contact point that sends notifications to Google Chat.",
		fieldMapper: map[string]fieldMapper{},
	}
}

func (g googleChatNotifier) schema() *schema.Resource {
	r := commonNotifierResource()
	r.Schema["url"] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Sensitive:   true,
		Description: "The Google Chat webhook URL.",
	}
	r.Schema["title"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The templated content of the title.",
	}
	r.Schema["message"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The templated content of the message.",
	}
	return r
}

type kafkaNotifier struct{}

var _ notifier = (*kafkaNotifier)(nil)

func (k kafkaNotifier) meta() notifierMeta {
	return notifierMeta{
		field:   "kafka",
		typeStr: "kafka",
		desc:    "A contact point that publishes notifications to Apache Kafka topics.",
		fieldMapper: map[string]fieldMapper{
			"rest_proxy_url": newKeyMapper("kafkaRestProxy"),
			"topic":          newKeyMapper("kafkaTopic"),
			"api_version":    newKeyMapper("apiVersion"),
			"cluster_id":     newKeyMapper("kafkaClusterId"),
		},
	}
}

func (k kafkaNotifier) schema() *schema.Resource {
	r := commonNotifierResource()
	r.Schema["rest_proxy_url"] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Sensitive:   true,
		Description: "The URL of the Kafka REST proxy to send requests to.",
	}
	r.Schema["topic"] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Description: "The name of the Kafka topic to publish to.",
	}
	r.Schema["description"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The templated description of the Kafka message.",
	}
	r.Schema["details"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The templated details to include with the message.",
	}
	r.Schema["username"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The user name to use when making a call to the Kafka REST Proxy",
	}
	r.Schema["password"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Sensitive:   true,
		Description: "The password to use when making a call to the Kafka REST Proxy",
	}
	r.Schema["api_version"] = &schema.Schema{
		Type:         schema.TypeString,
		Optional:     true,
		Default:      "v2",
		ValidateFunc: validation.StringInSlice([]string{"v2", "v3"}, false),
		Description:  "The API version to use when contacting the Kafka REST Server. Supported: v2 (default) and v3.",
	}
	r.Schema["cluster_id"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The Id of cluster to use when contacting the Kafka REST Server. Required api_version to be 'v3'",
	}
	return r
}

type lineNotifier struct{}

var _ notifier = (*lineNotifier)(nil)

func (o lineNotifier) meta() notifierMeta {
	return notifierMeta{
		field:       "line",
		typeStr:     "LINE",
		desc:        "A contact point that sends notifications to LINE.me.",
		fieldMapper: map[string]fieldMapper{},
	}
}

func (o lineNotifier) schema() *schema.Resource {
	r := commonNotifierResource()
	r.Schema["token"] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Sensitive:   true,
		Description: "The bearer token used to authorize the client.",
	}
	r.Schema["title"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The templated title of the message.",
	}
	r.Schema["description"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The templated description of the message.",
	}
	return r
}

type oncallNotifier struct{}

var _ notifier = (*oncallNotifier)(nil)

func (w oncallNotifier) meta() notifierMeta {
	return notifierMeta{
		field:   "oncall",
		typeStr: "oncall",
		desc:    "A contact point that sends notifications to Grafana On-Call.",
		fieldMapper: map[string]fieldMapper{
			"http_method":         newKeyMapper("httpMethod"),
			"basic_auth_user":     newKeyMapper("username"),
			"basic_auth_password": newKeyMapper("password"),
			"max_alerts":          newFieldMapper("maxAlerts", valueAsInt, valueAsInt),
		},
	}
}

func (w oncallNotifier) schema() *schema.Resource {
	r := commonNotifierResource()
	r.Schema["url"] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Description: "The URL to send webhook requests to.",
	}
	r.Schema["http_method"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The HTTP method to use in the request. Defaults to `POST`.",
	}
	r.Schema["basic_auth_user"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The username to use in basic auth headers attached to the request. If omitted, basic auth will not be used.",
	}
	r.Schema["basic_auth_password"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Sensitive:   true,
		Description: "The username to use in basic auth headers attached to the request. If omitted, basic auth will not be used.",
	}
	r.Schema["authorization_scheme"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "Allows a custom authorization scheme - attaches an auth header with this name. Do not use in conjunction with basic auth parameters.",
	}
	r.Schema["authorization_credentials"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Sensitive:   true,
		Description: "Allows a custom authorization scheme - attaches an auth header with this value. Do not use in conjunction with basic auth parameters.",
	}
	r.Schema["max_alerts"] = &schema.Schema{
		Type:        schema.TypeInt,
		Optional:    true,
		Description: "The maximum number of alerts to send in a single request. This can be helpful in limiting the size of the request body. The default is 0, which indicates no limit.",
	}
	r.Schema["message"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "Custom message. You can use template variables.",
	}
	r.Schema["title"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "Templated title of the message.",
	}
	return r
}

type opsGenieNotifier struct{}

var _ notifier = (*opsGenieNotifier)(nil)

func (o opsGenieNotifier) meta() notifierMeta {
	return notifierMeta{
		field:   "opsgenie",
		typeStr: "opsgenie",
		desc:    "A contact point that sends notifications to OpsGenie.",
		fieldMapper: map[string]fieldMapper{
			"url":               newKeyMapper("apiUrl"),
			"api_key":           newKeyMapper("apiKey"),
			"auto_close":        newKeyMapper("autoClose"),
			"override_priority": newKeyMapper("overridePriority"),
			"send_tags_as":      newKeyMapper("sendTagsAs"),
		},
	}
}

func (o opsGenieNotifier) schema() *schema.Resource {
	r := commonNotifierResource()
	r.Schema["url"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "Allows customization of the OpsGenie API URL.",
	}
	r.Schema["api_key"] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Sensitive:   true,
		Description: "The OpsGenie API key to use.",
	}
	r.Schema["message"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The templated content of the message.",
	}
	r.Schema["description"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "A templated high-level description to use for the alert.",
	}
	r.Schema["auto_close"] = &schema.Schema{
		Type:        schema.TypeBool,
		Optional:    true,
		Description: "Whether to auto-close alerts in OpsGenie when they resolve in the Alertmanager.",
	}
	r.Schema["override_priority"] = &schema.Schema{
		Type:        schema.TypeBool,
		Optional:    true,
		Description: "Whether to allow the alert priority to be configured via the value of the `og_priority` annotation on the alert.",
	}
	r.Schema["send_tags_as"] = &schema.Schema{
		Type:         schema.TypeString,
		Optional:     true,
		ValidateFunc: validation.StringInSlice([]string{"tags", "details", "both"}, false),
		Description:  "Whether to send annotations to OpsGenie as Tags, Details, or both. Supported values are `tags`, `details`, `both`, or empty to use the default behavior of Tags.",
	}
	r.Schema["responders"] = &schema.Schema{
		Type:        schema.TypeList,
		Optional:    true,
		Description: "Teams, users, escalations and schedules that the alert will be routed to send notifications. If the API Key belongs to a team integration, this field will be overwritten with the owner team. This feature is available from Grafana 10.3+.",
		Elem: &schema.Resource{
			Description: "Defines a responder. Either id, name or username must be specified",
			Schema: map[string]*schema.Schema{
				"type": {
					Type:        schema.TypeString,
					Required:    true,
					Description: "Type of the responder. Supported: team, teams, user, escalation, schedule or a template that is expanded to one of these values.",
				},
				"name": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "Name of the responder. Must be specified if username and id are empty.",
				},
				"username": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "User name of the responder. Must be specified if name and id are empty.",
				},
				"id": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "ID of the responder. Must be specified if name and username are empty.",
				},
			},
		},
	}
	return r
}

type pagerDutyNotifier struct{}

var _ notifier = (*pagerDutyNotifier)(nil)

func (n pagerDutyNotifier) meta() notifierMeta {
	return notifierMeta{
		field:   "pagerduty",
		typeStr: "pagerduty",
		desc:    "A contact point that sends notifications to PagerDuty.",
		fieldMapper: map[string]fieldMapper{
			"integration_key": newKeyMapper("integrationKey"),
		},
	}
}

func (n pagerDutyNotifier) schema() *schema.Resource {
	r := commonNotifierResource()
	r.Schema["integration_key"] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Sensitive:   true,
		Description: "The PagerDuty API key.",
	}
	r.Schema["severity"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The PagerDuty event severity level. Default is `critical`.",
	}
	r.Schema["class"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The class or type of event, for example `ping failure`.",
	}
	r.Schema["component"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The component being affected by the event.",
	}
	r.Schema["group"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The group to which the provided component belongs to.",
	}
	r.Schema["summary"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The templated summary message of the event.",
	}
	r.Schema["source"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The unique location of the affected system.",
	}
	r.Schema["client"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The name of the monitoring client that is triggering this event.",
	}
	r.Schema["client_url"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The URL of the monitoring client that is triggering this event.",
	}
	r.Schema["details"] = &schema.Schema{
		Type:        schema.TypeMap,
		Optional:    true,
		Default:     nil,
		Description: "A set of arbitrary key/value pairs that provide further detail about the incident.",
		Elem: &schema.Schema{
			Type: schema.TypeString,
		},
	}
	r.Schema["url"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The URL to send API requests to",
	}
	return r
}

type pushoverNotifier struct{}

var _ notifier = (*pushoverNotifier)(nil)

func (n pushoverNotifier) meta() notifierMeta {
	return notifierMeta{
		field:   "pushover",
		typeStr: "pushover",
		desc:    "A contact point that sends notifications to Pushover.",
		fieldMapper: map[string]fieldMapper{
			"user_key":     newKeyMapper("userKey"),
			"api_token":    newKeyMapper("apiToken"),
			"ok_sound":     newKeyMapper("okSound"),
			"upload_image": newKeyMapper("uploadImage"),
			// For unclear legacy reasons, these are sent as a string to Grafana API.
			"ok_priority": newFieldMapper("okPriority", valueAsInt, valueAsString),
			"priority":    newFieldMapper("", valueAsInt, valueAsString),
			"retry":       newFieldMapper("", valueAsInt, valueAsString),
			"expire":      newFieldMapper("", valueAsInt, valueAsString),
		},
	}
}

func (n pushoverNotifier) schema() *schema.Resource {
	r := commonNotifierResource()
	r.Schema["user_key"] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Sensitive:   true,
		Description: "The Pushover user key.",
	}
	r.Schema["api_token"] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Sensitive:   true,
		Description: "The Pushover API token.",
	}
	r.Schema["priority"] = &schema.Schema{
		Type:        schema.TypeInt,
		Optional:    true,
		Description: "The priority level of the event.",
	}
	r.Schema["ok_priority"] = &schema.Schema{
		Type:        schema.TypeInt,
		Optional:    true,
		Description: "The priority level of the resolved event.",
	}
	r.Schema["retry"] = &schema.Schema{
		Type:        schema.TypeInt,
		Optional:    true,
		Description: "How often, in seconds, the Pushover servers will send the same notification to the user.",
	}
	r.Schema["expire"] = &schema.Schema{
		Type:        schema.TypeInt,
		Optional:    true,
		Description: "How many seconds for which the notification will continue to be retried by Pushover.",
	}
	r.Schema["device"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "Comma-separated list of devices to which the event is associated.",
	}
	r.Schema["sound"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The sound associated with the notification.",
	}
	r.Schema["ok_sound"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The sound associated with the resolved notification.",
	}
	r.Schema["title"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The templated title of the message.",
	}
	r.Schema["message"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The templated notification message content.",
	}
	r.Schema["upload_image"] = &schema.Schema{
		Type:        schema.TypeBool,
		Optional:    true,
		Description: "Whether to send images in the notification or not. Default is true. Requires Grafana to be configured to send images in notifications.",
	}
	return r
}

type sensugoNotifier struct{}

var _ notifier = (*sensugoNotifier)(nil)

func (s sensugoNotifier) meta() notifierMeta {
	return notifierMeta{
		field:   "sensugo",
		typeStr: "sensugo",
		desc:    "A contact point that sends notifications to SensuGo.",
		fieldMapper: map[string]fieldMapper{
			"api_key": newKeyMapper("apikey"),
		},
	}
}

func (s sensugoNotifier) schema() *schema.Resource {
	r := commonNotifierResource()
	r.Schema["url"] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Description: "The SensuGo URL to send requests to.",
	}
	r.Schema["api_key"] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Sensitive:   true,
		Description: "The SensuGo API key.",
	}
	r.Schema["entity"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The entity being monitored.",
	}
	r.Schema["check"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The SensuGo check to which the event should be routed.",
	}
	r.Schema["namespace"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The namespace in which the check resides.",
	}
	r.Schema["handler"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "A custom handler to execute in addition to the check.",
	}
	r.Schema["message"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "Templated message content describing the alert.",
	}
	return r
}

type slackNotifier struct{}

var _ notifier = (*slackNotifier)(nil)

func (s slackNotifier) HasData(data map[string]any) bool {
	// Slack has no simple required fields as they require mutual exclusivity. We rely on `Required` to test for
	// deletions on update, so instead we define a custom HasData method.
	return data["url"] != "" || data["token"] != ""
}

func (s slackNotifier) meta() notifierMeta {
	return notifierMeta{
		field:   "slack",
		typeStr: "slack",
		desc:    "A contact point that sends notifications to Slack.",
		fieldMapper: map[string]fieldMapper{
			"endpoint_url":    newKeyMapper("endpointUrl"),
			"mention_channel": newKeyMapper("mentionChannel"),
			"mention_users":   newKeyMapper("mentionUsers"),
			"mention_groups":  newKeyMapper("mentionGroups"),
		},
	}
}

func (s slackNotifier) schema() *schema.Resource {
	r := commonNotifierResource()
	r.Schema["endpoint_url"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "Use this to override the Slack API endpoint URL to send requests to.",
	}
	r.Schema["url"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Sensitive:   true,
		Description: "A Slack webhook URL,for sending messages via the webhook method.",
	}
	r.Schema["token"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Sensitive:   true,
		Description: "A Slack API token,for sending messages directly without the webhook method.",
	}
	r.Schema["recipient"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "Channel, private group, or IM channel (can be an encoded ID or a name) to send messages to.",
	}
	r.Schema["text"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "Templated content of the message.",
	}
	r.Schema["title"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "Templated title of the message.",
	}
	r.Schema["username"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "Username for the bot to use.",
	}
	r.Schema["icon_emoji"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The name of a Slack workspace emoji to use as the bot icon.",
	}
	r.Schema["icon_url"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "A URL of an image to use as the bot icon.",
	}
	r.Schema["mention_channel"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "Describes how to ping the slack channel that messages are being sent to. Options are `here` for an @here ping, `channel` for @channel, or empty for no ping.",
	}
	r.Schema["mention_users"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "Comma-separated list of users to mention in the message.",
	}
	r.Schema["mention_groups"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "Comma-separated list of groups to mention in the message.",
	}
	r.Schema["color"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "Templated color of the slack message.",
	}
	return r
}

type snsNotifier struct{}

var _ notifier = (*snsNotifier)(nil)

func (s snsNotifier) meta() notifierMeta {
	return notifierMeta{
		field:   "sns",
		typeStr: "sns",
		desc:    "A contact point that sends notifications to Amazon SNS. Requires Amazon Managed Grafana.",
		fieldMapper: map[string]fieldMapper{
			"auth_provider":   newKeyMapper("authProvider"),
			"access_key":      newKeyMapper("accessKey"),
			"secret_key":      newKeyMapper("secretKey"),
			"assume_role_arn": newKeyMapper("assumeRoleARN"),
			"message_format":  newKeyMapper("messageFormat"),
			"external_id":     newKeyMapper("externalId"),
		},
	}
}

func (s snsNotifier) schema() *schema.Resource {
	r := commonNotifierResource()
	r.Schema["topic"] = &schema.Schema{
		Type:        schema.TypeString,
		Description: "The Amazon SNS topic to send notifications to.",
		Required:    true,
	}
	r.Schema["auth_provider"] = &schema.Schema{
		Type:         schema.TypeString,
		Optional:     true,
		Description:  "The authentication provider to use. Valid values are `default`, `arn` and `keys`. Default is `default`.",
		Default:      "default",
		ValidateFunc: validation.StringInSlice([]string{"default", "arn", "keys"}, false),
	}
	r.Schema["access_key"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Sensitive:   true,
		Description: "AWS access key ID used to authenticate with Amazon SNS.",
	}
	r.Schema["secret_key"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Sensitive:   true,
		Description: "AWS secret access key used to authenticate with Amazon SNS.",
	}
	r.Schema["assume_role_arn"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The Amazon Resource Name (ARN) of the role to assume to send notifications to Amazon SNS.",
	}
	r.Schema["message_format"] = &schema.Schema{
		Type:         schema.TypeString,
		Optional:     true,
		Description:  "The format of the message to send. Valid values are `text`, `body` and `json`. Default is `text`.",
		ValidateFunc: validation.StringInSlice([]string{"text", "body", "json"}, false),
		Default:      "text",
	}
	r.Schema["body"] = &schema.Schema{
		Type:     schema.TypeString,
		Optional: true,
	}
	r.Schema["subject"] = &schema.Schema{
		Type:     schema.TypeString,
		Optional: true,
	}
	r.Schema["external_id"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The external ID to use when assuming the role.",
	}

	return r
}

type teamsNotifier struct{}

var _ notifier = (*teamsNotifier)(nil)

func (t teamsNotifier) meta() notifierMeta {
	return notifierMeta{
		field:   "teams",
		typeStr: "teams",
		desc:    "A contact point that sends notifications to Microsoft Teams.",
		fieldMapper: map[string]fieldMapper{
			"section_title": newKeyMapper("sectiontitle"),
			"chat_id":       newKeyMapper("chatid"),
		},
	}
}

func (t teamsNotifier) schema() *schema.Resource {
	r := commonNotifierResource()
	r.Schema["url"] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Sensitive:   true,
		Description: "A Teams webhook URL.",
	}
	r.Schema["message"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The templated message content to send.",
	}
	r.Schema["title"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The templated title of the message.",
	}
	r.Schema["section_title"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The templated subtitle for each message section.",
	}
	return r
}

type telegramNotifier struct{}

var _ notifier = (*telegramNotifier)(nil)

func (t telegramNotifier) meta() notifierMeta {
	return notifierMeta{
		field:   "telegram",
		typeStr: "telegram",
		desc:    "A contact point that sends notifications to Telegram.",
		fieldMapper: map[string]fieldMapper{
			"token":   newKeyMapper("bottoken"),
			"chat_id": newKeyMapper("chatid"),
		},
	}
}

func (t telegramNotifier) schema() *schema.Resource {
	r := commonNotifierResource()
	r.Schema["token"] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Sensitive:   true,
		Description: "The Telegram bot token.",
	}
	r.Schema["chat_id"] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Description: "The chat ID to send messages to.",
	}
	r.Schema["message"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The templated content of the message.",
	}
	r.Schema["message_thread_id"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The ID of the message thread to send the message to.",
	}
	r.Schema["parse_mode"] = &schema.Schema{
		Type:         schema.TypeString,
		Optional:     true,
		ValidateFunc: validation.StringInSlice([]string{"None", "Markdown", "MarkdownV2", "HTML"}, true),
		Description:  "Mode for parsing entities in the message text. Supported: None, Markdown, MarkdownV2, and HTML. HTML is the default.",
	}
	r.Schema["disable_web_page_preview"] = &schema.Schema{
		Type:        schema.TypeBool,
		Optional:    true,
		Description: "When set it disables link previews for links in the message.",
	}
	r.Schema["protect_content"] = &schema.Schema{
		Type:        schema.TypeBool,
		Optional:    true,
		Description: "When set it protects the contents of the message from forwarding and saving.",
	}
	r.Schema["disable_notifications"] = &schema.Schema{
		Type:        schema.TypeBool,
		Optional:    true,
		Description: "When set users will receive a notification with no sound.",
	}
	return r
}

type threemaNotifier struct{}

var _ notifier = (*threemaNotifier)(nil)

func (t threemaNotifier) meta() notifierMeta {
	return notifierMeta{
		field:       "threema",
		typeStr:     "threema",
		desc:        "A contact point that sends notifications to Threema.",
		fieldMapper: map[string]fieldMapper{},
	}
}

func (t threemaNotifier) schema() *schema.Resource {
	r := commonNotifierResource()
	r.Schema["gateway_id"] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Description: "The Threema gateway ID.",
	}
	r.Schema["recipient_id"] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Description: "The ID of the recipient of the message.",
	}
	r.Schema["api_secret"] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Sensitive:   true,
		Description: "The Threema API key.",
	}
	r.Schema["title"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The templated title of the message.",
	}
	r.Schema["description"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The templated description of the message.",
	}
	return r
}

type victorOpsNotifier struct{}

var _ notifier = (*victorOpsNotifier)(nil)

func (v victorOpsNotifier) meta() notifierMeta {
	return notifierMeta{
		field:   "victorops",
		typeStr: "victorops",
		desc:    "A contact point that sends notifications to VictorOps (now known as Splunk OnCall).",
		fieldMapper: map[string]fieldMapper{
			"message_type": newKeyMapper("messageType"),
		},
	}
}

func (v victorOpsNotifier) schema() *schema.Resource {
	r := commonNotifierResource()
	r.Schema["url"] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Sensitive:   true,
		Description: "The VictorOps webhook URL.",
	}
	r.Schema["message_type"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The VictorOps alert state - typically either `CRITICAL` or `RECOVERY`.",
	}
	r.Schema["title"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "Templated title to display.",
	}
	r.Schema["description"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "Templated description of the message.",
	}
	return r
}

type webexNotifier struct{}

var _ notifier = (*webexNotifier)(nil)

func (w webexNotifier) meta() notifierMeta {
	return notifierMeta{
		field:   "webex",
		typeStr: "webex",
		desc:    "A contact point that sends notifications to Cisco Webex.",
		fieldMapper: map[string]fieldMapper{
			"token": newKeyMapper("bot_token"),
		},
	}
}

func (w webexNotifier) schema() *schema.Resource {
	r := commonNotifierResource()
	r.Schema["token"] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Sensitive:   true,
		Description: "The bearer token used to authorize the client.",
	}
	r.Schema["api_url"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The URL to send webhook requests to.",
	}
	r.Schema["message"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The templated title of the message to send.",
	}
	r.Schema["room_id"] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Description: "ID of the Webex Teams room where to send the messages.",
	}
	return r
}

type webhookNotifier struct{}

var _ notifier = (*webhookNotifier)(nil)

func (w webhookNotifier) meta() notifierMeta {
	return notifierMeta{
		field:   "webhook",
		typeStr: "webhook",
		desc:    "A contact point that sends notifications to an arbitrary webhook, using the Prometheus webhook format defined here: https://prometheus.io/docs/alerting/latest/configuration/#webhook_config",
		fieldMapper: withCommonHTTPConfigFieldMappers(map[string]fieldMapper{
			"http_method":                  newKeyMapper("httpMethod"),
			"basic_auth_user":              newKeyMapper("username"),
			"basic_auth_password":          newKeyMapper("password"),
			"max_alerts":                   newFieldMapper("maxAlerts", valueAsInt, valueAsInt),
			"tls_config":                   newFieldMapper("tlsConfig", translateTLSConfigPack, translateTLSConfigUnpack),
			"headers":                      omitEmptyMapper(),
			"hmac_config":                  newKeyMapper("hmacConfig"),
			"hmac_config.timestamp_header": newKeyMapper("timestampHeader"),
		}),
	}
}

func (w webhookNotifier) schema() *schema.Resource {
	r := commonNotifierResource()
	r.Schema["url"] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Description: "The URL to send webhook requests to.",
	}
	r.Schema["http_method"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The HTTP method to use in the request. Defaults to `POST`.",
	}
	r.Schema["basic_auth_user"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The username to use in basic auth headers attached to the request. If omitted, basic auth will not be used.",
	}
	r.Schema["basic_auth_password"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Sensitive:   true,
		Description: "The username to use in basic auth headers attached to the request. If omitted, basic auth will not be used.",
	}
	r.Schema["authorization_scheme"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "Allows a custom authorization scheme - attaches an auth header with this name. Do not use in conjunction with basic auth parameters.",
	}
	r.Schema["authorization_credentials"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Sensitive:   true,
		Description: "Allows a custom authorization scheme - attaches an auth header with this value. Do not use in conjunction with basic auth parameters.",
	}
	r.Schema["max_alerts"] = &schema.Schema{
		Type:        schema.TypeInt,
		Optional:    true,
		Description: "The maximum number of alerts to send in a single request. This can be helpful in limiting the size of the request body. The default is 0, which indicates no limit.",
	}
	r.Schema["message"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "Custom message. You can use template variables.",
	}
	r.Schema["title"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "Templated title of the message.",
	}
	r.Schema["tls_config"] = &schema.Schema{
		Type:        schema.TypeMap,
		Optional:    true,
		Sensitive:   true,
		Description: "Allows configuring TLS for the webhook notifier.",
	}
	r.Schema["hmac_config"] = &schema.Schema{
		Type:        schema.TypeSet,
		Optional:    true,
		MaxItems:    1,
		Description: "HMAC signature configuration options.",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"secret": {
					Type:        schema.TypeString,
					Required:    true,
					Sensitive:   true,
					Description: "The secret key used to generate the HMAC signature.",
				},
				"header": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "The header in which the HMAC signature will be included. Defaults to `X-Grafana-Alerting-Signature`.",
				},
				"timestamp_header": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "If set, the timestamp will be included in the HMAC signature. The value should be the name of the header to use.",
				},
			},
		},
	}
	r.Schema["headers"] = &schema.Schema{
		Type:        schema.TypeMap,
		Optional:    true,
		Description: "Custom headers to attach to the request.",
		Elem:        &schema.Schema{Type: schema.TypeString},
	}
	r.Schema["payload"] = &schema.Schema{
		Type:        schema.TypeSet,
		Optional:    true,
		MaxItems:    1,
		Description: "Optionally provide a templated payload. Overrides 'Message' and 'Title' field.",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"template": {
					Type:        schema.TypeString,
					Required:    true,
					Description: "Custom payload template.",
				},
				"vars": {
					Type:        schema.TypeMap,
					Optional:    true,
					Description: "Optionally provide a variables to be used in the payload template. They will be available in the template as `.Vars.<variable_name>`.",
					Elem:        &schema.Schema{Type: schema.TypeString},
				},
			},
		},
	}
	addCommonHTTPConfigResource(r)
	return r
}

type wecomNotifier struct{}

var _ notifier = (*wecomNotifier)(nil)

func (w wecomNotifier) HasData(data map[string]any) bool {
	// WeCom has no simple required fields as they require mutual exclusivity. We rely on `Required` to test for
	// deletions on update, so instead we define a custom HasData method.
	return data["url"] != "" || data["secret"] != ""
}

func (w wecomNotifier) meta() notifierMeta {
	return notifierMeta{
		field:   "wecom",
		typeStr: "wecom",
		desc:    "A contact point that sends notifications to WeCom.",
		fieldMapper: map[string]fieldMapper{
			"msg_type": newKeyMapper("msgtype"),
			"to_user":  newKeyMapper("touser"),
		},
	}
}

func (w wecomNotifier) schema() *schema.Resource {
	r := commonNotifierResource()
	r.Schema["url"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Sensitive:   true,
		Description: "The WeCom webhook URL. Required if using GroupRobot.",
	}
	r.Schema["secret"] = &schema.Schema{
		Type:        schema.TypeString,
		Sensitive:   true,
		Optional:    true,
		Description: "The secret key required to obtain access token when using APIAPP. See https://work.weixin.qq.com/wework_admin/frame#apps to create APIAPP.",
	}
	r.Schema["corp_id"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "Corp ID used to get token when using APIAPP.",
	}
	r.Schema["agent_id"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "Agent ID added to the request payload when using APIAPP.",
	}
	r.Schema["message"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The templated content of the message to send.",
	}
	r.Schema["title"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The templated title of the message to send.",
	}
	r.Schema["msg_type"] = &schema.Schema{
		Type:         schema.TypeString,
		Optional:     true,
		ValidateFunc: validation.StringInSlice([]string{"markdown", "text"}, false),
		Description:  "The type of them message. Supported: markdown, text. Default: text.",
	}
	r.Schema["to_user"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The ID of user that should receive the message. Multiple entries should be separated by '|'. Default: @all.",
	}
	return r
}
