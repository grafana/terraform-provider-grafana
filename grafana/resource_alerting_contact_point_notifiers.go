package grafana

import (
	"strconv"
	"strings"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type alertmanagerNotifier struct{}

var _ notifier = (*alertmanagerNotifier)(nil)

func (a alertmanagerNotifier) meta() notifierMeta {
	return notifierMeta{
		field:   "alertmanager",
		typeStr: "prometheus-alertmanager",
		desc:    "A contact point that sends notifications to other Alertmanager instances.",
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
		Type:             schema.TypeString,
		Optional:         true,
		Sensitive:        true,
		DiffSuppressFunc: redactedContactPointDiffSuppress,
		Description:      "The password component of the basic auth credentials to use.",
	}
	return r
}

func (a alertmanagerNotifier) pack(p gapi.ContactPoint) (interface{}, error) {
	notifier := packCommonNotifierFields(&p)
	if v, ok := p.Settings["url"]; ok && v != nil {
		notifier["url"] = v.(string)
		delete(p.Settings, "url")
	}
	if v, ok := p.Settings["basicAuthUser"]; ok && v != nil {
		notifier["basic_auth_user"] = v.(string)
		delete(p.Settings, "basicAuthUser")
	}
	if v, ok := p.Settings["basicAuthPassword"]; ok && v != nil {
		notifier["basic_auth_password"] = v.(string)
		delete(p.Settings, "basicAuthPassword")
	}
	notifier["settings"] = packSettings(&p)
	return notifier, nil
}

func (a alertmanagerNotifier) unpack(raw interface{}, name string) gapi.ContactPoint {
	json := raw.(map[string]interface{})
	uid, disableResolve, settings := unpackCommonNotifierFields(json)

	settings["url"] = json["url"].(string)
	if v, ok := json["basic_auth_user"]; ok && v != nil {
		settings["basicAuthUser"] = v.(string)
	}
	if v, ok := json["basic_auth_password"]; ok && v != nil {
		settings["basicAuthPassword"] = v.(string)
	}
	return gapi.ContactPoint{
		UID:                   uid,
		Name:                  name,
		Type:                  a.meta().typeStr,
		DisableResolveMessage: disableResolve,
		Settings:              settings,
	}
}

type dingDingNotifier struct{}

var _ notifier = (*dingDingNotifier)(nil)

func (d dingDingNotifier) meta() notifierMeta {
	return notifierMeta{
		field:   "dingding",
		typeStr: "dingding",
		desc:    "A contact point that sends notifications to DingDing.",
	}
}

func (d dingDingNotifier) schema() *schema.Resource {
	r := commonNotifierResource()
	r.Schema["url"] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
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
	return r
}

func (d dingDingNotifier) pack(p gapi.ContactPoint) (interface{}, error) {
	notifier := packCommonNotifierFields(&p)
	if v, ok := p.Settings["url"]; ok && v != nil {
		notifier["url"] = v.(string)
		delete(p.Settings, "url")
	}
	if v, ok := p.Settings["msgType"]; ok && v != nil {
		notifier["message_type"] = v.(string)
		delete(p.Settings, "msgType")
	}
	if v, ok := p.Settings["message"]; ok && v != nil {
		notifier["message"] = v.(string)
		delete(p.Settings, "message")
	}
	notifier["settings"] = packSettings(&p)
	return notifier, nil
}

func (d dingDingNotifier) unpack(raw interface{}, name string) gapi.ContactPoint {
	json := raw.(map[string]interface{})
	uid, disableResolve, settings := unpackCommonNotifierFields(json)

	settings["url"] = json["url"].(string)
	if v, ok := json["message_type"]; ok && v != nil {
		settings["msgType"] = v.(string)
	}
	if v, ok := json["message"]; ok && v != nil {
		settings["message"] = v.(string)
	}
	return gapi.ContactPoint{
		UID:                   uid,
		Name:                  name,
		Type:                  d.meta().typeStr,
		DisableResolveMessage: disableResolve,
		Settings:              settings,
	}
}

type discordNotifier struct{}

var _ notifier = (*discordNotifier)(nil)

func (d discordNotifier) meta() notifierMeta {
	return notifierMeta{
		field:   "discord",
		typeStr: "discord",
		desc:    "A contact point that sends notifications as Discord messages",
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

func (d discordNotifier) pack(p gapi.ContactPoint) (interface{}, error) {
	notifier := packCommonNotifierFields(&p)
	if v, ok := p.Settings["url"]; ok && v != nil {
		notifier["url"] = v.(string)
		delete(p.Settings, "url")
	}
	if v, ok := p.Settings["message"]; ok && v != nil {
		notifier["message"] = v.(string)
		delete(p.Settings, "message")
	}
	if v, ok := p.Settings["avatar_url"]; ok && v != nil {
		notifier["avatar_url"] = v.(string)
		delete(p.Settings, "avatar_url")
	}
	if v, ok := p.Settings["use_discord_username"]; ok && v != nil {
		notifier["use_discord_username"] = v.(bool)
		delete(p.Settings, "use_discord_username")
	}

	notifier["settings"] = packSettings(&p)
	return notifier, nil
}

func (d discordNotifier) unpack(raw interface{}, name string) gapi.ContactPoint {
	json := raw.(map[string]interface{})
	uid, disableResolve, settings := unpackCommonNotifierFields(json)

	settings["url"] = json["url"].(string)
	if v, ok := json["message"]; ok && v != nil {
		settings["message"] = v.(string)
	}
	if v, ok := json["avatar_url"]; ok && v != nil {
		settings["avatar_url"] = v.(string)
	}
	if v, ok := json["use_discord_username"]; ok && v != nil {
		settings["use_discord_username"] = v.(bool)
	}

	return gapi.ContactPoint{
		UID:                   uid,
		Name:                  name,
		Type:                  d.meta().typeStr,
		DisableResolveMessage: disableResolve,
		Settings:              settings,
	}
}

type emailNotifier struct{}

var _ notifier = (*emailNotifier)(nil)

func (e emailNotifier) meta() notifierMeta {
	return notifierMeta{
		field:   "email",
		typeStr: "email",
		desc:    "A contact point that sends notifications to an email address.",
	}
}

func (e emailNotifier) schema() *schema.Resource {
	r := commonNotifierResource()
	r.Schema["addresses"] = &schema.Schema{
		Type:        schema.TypeList,
		Required:    true,
		Description: "The addresses to send emails to.",
		Elem: &schema.Schema{
			Type: schema.TypeString,
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

func (e emailNotifier) pack(p gapi.ContactPoint) (interface{}, error) {
	notifier := packCommonNotifierFields(&p)
	if v, ok := p.Settings["addresses"]; ok && v != nil {
		notifier["addresses"] = packAddrs(v.(string))
		delete(p.Settings, "addresses")
	}
	if v, ok := p.Settings["singleEmail"]; ok && v != nil {
		notifier["single_email"] = v.(bool)
		delete(p.Settings, "singleEmail")
	}
	if v, ok := p.Settings["message"]; ok && v != nil {
		notifier["message"] = v.(string)
		delete(p.Settings, "message")
	}
	if v, ok := p.Settings["subject"]; ok && v != nil {
		notifier["subject"] = v.(string)
		delete(p.Settings, "subject")
	}
	notifier["settings"] = packSettings(&p)
	return notifier, nil
}

func (e emailNotifier) unpack(raw interface{}, name string) gapi.ContactPoint {
	json := raw.(map[string]interface{})
	uid, disableResolve, settings := unpackCommonNotifierFields(json)

	addrs := unpackAddrs(json["addresses"].([]interface{}))
	settings["addresses"] = addrs
	if v, ok := json["single_email"]; ok && v != nil {
		settings["singleEmail"] = v.(bool)
	}
	if v, ok := json["message"]; ok && v != nil {
		settings["message"] = v.(string)
	}
	if v, ok := json["subject"]; ok && v != nil {
		settings["subject"] = v.(string)
	}

	return gapi.ContactPoint{
		UID:                   uid,
		Name:                  name,
		Type:                  e.meta().typeStr,
		DisableResolveMessage: disableResolve,
		Settings:              settings,
	}
}

const addrSeparator = ";"

func packAddrs(addrs string) []string {
	return strings.Split(addrs, addrSeparator)
}

func unpackAddrs(addrs []interface{}) string {
	strs := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		strs = append(strs, addr.(string))
	}
	return strings.Join(strs, addrSeparator)
}

type googleChatNotifier struct{}

var _ notifier = (*googleChatNotifier)(nil)

func (g googleChatNotifier) meta() notifierMeta {
	return notifierMeta{
		field:   "googlechat",
		typeStr: "googlechat",
		desc:    "A contact point that sends notifications to Google Chat.",
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
	r.Schema["message"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The templated content of the message.",
	}
	return r
}

func (g googleChatNotifier) pack(p gapi.ContactPoint) (interface{}, error) {
	notifier := packCommonNotifierFields(&p)
	if v, ok := p.Settings["url"]; ok && v != nil {
		notifier["url"] = v.(string)
		delete(p.Settings, "url")
	}
	if v, ok := p.Settings["message"]; ok && v != nil {
		notifier["message"] = v.(string)
		delete(p.Settings, "message")
	}
	notifier["settings"] = packSettings(&p)
	return notifier, nil
}

func (g googleChatNotifier) unpack(raw interface{}, name string) gapi.ContactPoint {
	json := raw.(map[string]interface{})
	uid, disableResolve, settings := unpackCommonNotifierFields(json)

	settings["url"] = json["url"].(string)
	if v, ok := json["message"]; ok && v != nil {
		settings["message"] = v.(string)
	}
	return gapi.ContactPoint{
		UID:                   uid,
		Name:                  name,
		Type:                  g.meta().typeStr,
		DisableResolveMessage: disableResolve,
		Settings:              settings,
	}
}

type kafkaNotifier struct{}

var _ notifier = (*kafkaNotifier)(nil)

func (k kafkaNotifier) meta() notifierMeta {
	return notifierMeta{
		field:   "kafka",
		typeStr: "kafka",
		desc:    "A contact point that publishes notifications to Apache Kafka topics.",
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
	return r
}

func (k kafkaNotifier) pack(p gapi.ContactPoint) (interface{}, error) {
	notifier := packCommonNotifierFields(&p)
	if v, ok := p.Settings["kafkaRestProxy"]; ok && v != nil {
		notifier["rest_proxy_url"] = v.(string)
		delete(p.Settings, "kafkaRestProxy")
	}
	if v, ok := p.Settings["kafkaTopic"]; ok && v != nil {
		notifier["topic"] = v.(string)
		delete(p.Settings, "kafkaTopic")
	}
	notifier["settings"] = packSettings(&p)
	return notifier, nil
}

func (k kafkaNotifier) unpack(raw interface{}, name string) gapi.ContactPoint {
	json := raw.(map[string]interface{})
	uid, disableResolve, settings := unpackCommonNotifierFields(json)

	settings["kafkaRestProxy"] = json["rest_proxy_url"].(string)
	settings["kafkaTopic"] = json["topic"].(string)
	return gapi.ContactPoint{
		UID:                   uid,
		Name:                  name,
		Type:                  k.meta().typeStr,
		DisableResolveMessage: disableResolve,
		Settings:              settings,
	}
}

type opsGenieNotifier struct{}

var _ notifier = (*opsGenieNotifier)(nil)

func (o opsGenieNotifier) meta() notifierMeta {
	return notifierMeta{
		field:   "opsgenie",
		typeStr: "opsgenie",
		desc:    "A contact point that sends notifications to OpsGenie.",
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
		Type:             schema.TypeString,
		Required:         true,
		Sensitive:        true,
		DiffSuppressFunc: redactedContactPointDiffSuppress,
		Description:      "The OpsGenie API key to use.",
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
		Type:        schema.TypeString,
		Optional:    true,
		Description: "Whether to send annotations to OpsGenie as Tags, Details, or both. Supported values are `tags`, `details`, `both`, or empty to use the default behavior of Tags.",
	}
	return r
}

func (o opsGenieNotifier) pack(p gapi.ContactPoint) (interface{}, error) {
	notifier := packCommonNotifierFields(&p)
	if v, ok := p.Settings["apiUrl"]; ok && v != nil {
		notifier["url"] = v.(string)
		delete(p.Settings, "apiUrl")
	}
	if v, ok := p.Settings["apiKey"]; ok && v != nil {
		notifier["api_key"] = v.(string)
		delete(p.Settings, "apiKey")
	}
	if v, ok := p.Settings["message"]; ok && v != nil {
		notifier["message"] = v.(string)
		delete(p.Settings, "message")
	}
	if v, ok := p.Settings["description"]; ok && v != nil {
		notifier["description"] = v.(string)
		delete(p.Settings, "description")
	}
	if v, ok := p.Settings["autoClose"]; ok && v != nil {
		notifier["auto_close"] = v.(bool)
		delete(p.Settings, "autoClose")
	}
	if v, ok := p.Settings["overridePriority"]; ok && v != nil {
		notifier["override_priority"] = v.(bool)
		delete(p.Settings, "overridePriority")
	}
	if v, ok := p.Settings["sendTagsAs"]; ok && v != nil {
		notifier["send_tags_as"] = v.(string)
		delete(p.Settings, "sendTagsAs")
	}
	notifier["settings"] = packSettings(&p)
	return notifier, nil
}

func (o opsGenieNotifier) unpack(raw interface{}, name string) gapi.ContactPoint {
	json := raw.(map[string]interface{})
	uid, disableResolve, settings := unpackCommonNotifierFields(json)

	if v, ok := json["url"]; ok && v != nil {
		settings["apiUrl"] = v.(string)
	}
	if v, ok := json["api_key"]; ok && v != nil {
		settings["apiKey"] = v.(string)
	}
	if v, ok := json["message"]; ok && v != nil {
		settings["message"] = v.(string)
	}
	if v, ok := json["description"]; ok && v != nil {
		settings["description"] = v.(string)
	}
	if v, ok := json["auto_close"]; ok && v != nil {
		settings["autoClose"] = v.(bool)
	}
	if v, ok := json["override_priority"]; ok && v != nil {
		settings["overridePriority"] = v.(bool)
	}
	if v, ok := json["send_tags_as"]; ok && v != nil {
		settings["sendTagsAs"] = v.(string)
	}
	return gapi.ContactPoint{
		UID:                   uid,
		Name:                  name,
		Type:                  o.meta().typeStr,
		DisableResolveMessage: disableResolve,
		Settings:              settings,
	}
}

type pagerDutyNotifier struct{}

var _ notifier = (*pagerDutyNotifier)(nil)

func (p pagerDutyNotifier) meta() notifierMeta {
	return notifierMeta{
		field:   "pagerduty",
		typeStr: "pagerduty",
		desc:    "A contact point that sends notifications to PagerDuty.",
	}
}

func (p pagerDutyNotifier) schema() *schema.Resource {
	r := commonNotifierResource()
	r.Schema["integration_key"] = &schema.Schema{
		Type:             schema.TypeString,
		Required:         true,
		Sensitive:        true,
		DiffSuppressFunc: redactedContactPointDiffSuppress,
		Description:      "The PagerDuty API key.",
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
	return r
}

func (n pagerDutyNotifier) pack(p gapi.ContactPoint) (interface{}, error) {
	notifier := packCommonNotifierFields(&p)
	if v, ok := p.Settings["integrationKey"]; ok && v != nil {
		notifier["integration_key"] = v.(string)
		delete(p.Settings, "integrationKey")
	}
	if v, ok := p.Settings["severity"]; ok && v != nil {
		notifier["severity"] = v.(string)
		delete(p.Settings, "severity")
	}
	if v, ok := p.Settings["class"]; ok && v != nil {
		notifier["class"] = v.(string)
		delete(p.Settings, "class")
	}
	if v, ok := p.Settings["component"]; ok && v != nil {
		notifier["component"] = v.(string)
		delete(p.Settings, "component")
	}
	if v, ok := p.Settings["group"]; ok && v != nil {
		notifier["group"] = v.(string)
		delete(p.Settings, "group")
	}
	if v, ok := p.Settings["summary"]; ok && v != nil {
		notifier["summary"] = v.(string)
		delete(p.Settings, "summary")
	}
	notifier["settings"] = packSettings(&p)
	return notifier, nil
}

func (n pagerDutyNotifier) unpack(raw interface{}, name string) gapi.ContactPoint {
	json := raw.(map[string]interface{})
	uid, disableResolve, settings := unpackCommonNotifierFields(json)

	settings["integrationKey"] = json["integration_key"].(string)
	if v, ok := json["severity"]; ok && v != nil {
		settings["severity"] = v.(string)
	}
	if v, ok := json["class"]; ok && v != nil {
		settings["class"] = v.(string)
	}
	if v, ok := json["component"]; ok && v != nil {
		settings["component"] = v.(string)
	}
	if v, ok := json["group"]; ok && v != nil {
		settings["group"] = v.(string)
	}
	if v, ok := json["summary"]; ok && v != nil {
		settings["summary"] = v.(string)
	}
	return gapi.ContactPoint{
		UID:                   uid,
		Name:                  name,
		Type:                  n.meta().typeStr,
		DisableResolveMessage: disableResolve,
		Settings:              settings,
	}
}

type pushoverNotifier struct{}

var _ notifier = (*pushoverNotifier)(nil)

func (p pushoverNotifier) meta() notifierMeta {
	return notifierMeta{
		field:   "pushover",
		typeStr: "pushover",
		desc:    "A contact point that sends notifications to Pushover.",
	}
}

func (p pushoverNotifier) schema() *schema.Resource {
	r := commonNotifierResource()
	r.Schema["user_key"] = &schema.Schema{
		Type:             schema.TypeString,
		Required:         true,
		Sensitive:        true,
		DiffSuppressFunc: redactedContactPointDiffSuppress,
		Description:      "The Pushover user key.",
	}
	r.Schema["api_token"] = &schema.Schema{
		Type:             schema.TypeString,
		Required:         true,
		Sensitive:        true,
		DiffSuppressFunc: redactedContactPointDiffSuppress,
		Description:      "The Pushover API token.",
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
	r.Schema["message"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The templated notification message content.",
	}
	return r
}

func (n pushoverNotifier) pack(p gapi.ContactPoint) (interface{}, error) {
	notifier := packCommonNotifierFields(&p)
	if v, ok := p.Settings["userKey"]; ok && v != nil {
		notifier["user_key"] = v.(string)
		delete(p.Settings, "userKey")
	}
	if v, ok := p.Settings["apiToken"]; ok && v != nil {
		notifier["api_token"] = v.(string)
		delete(p.Settings, "apiToken")
	}
	if v, ok := p.Settings["priority"]; ok && v != nil {
		priority, err := strconv.Atoi(v.(string))
		if err != nil {
			return nil, err
		}
		notifier["priority"] = priority
		delete(p.Settings, "priority")
	}
	if v, ok := p.Settings["okPriority"]; ok && v != nil {
		priority, err := strconv.Atoi(v.(string))
		if err != nil {
			return nil, err
		}
		notifier["ok_priority"] = priority
		delete(p.Settings, "okPriority")
	}
	if v, ok := p.Settings["retry"]; ok && v != nil {
		priority, err := strconv.Atoi(v.(string))
		if err != nil {
			return nil, err
		}
		notifier["retry"] = priority
		delete(p.Settings, "retry")
	}
	if v, ok := p.Settings["expire"]; ok && v != nil {
		priority, err := strconv.Atoi(v.(string))
		if err != nil {
			return nil, err
		}
		notifier["expire"] = priority
		delete(p.Settings, "expire")
	}
	if v, ok := p.Settings["device"]; ok && v != nil {
		notifier["device"] = v.(string)
		delete(p.Settings, "device")
	}
	if v, ok := p.Settings["sound"]; ok && v != nil {
		notifier["sound"] = v.(string)
		delete(p.Settings, "sound")
	}
	if v, ok := p.Settings["okSound"]; ok && v != nil {
		notifier["ok_sound"] = v.(string)
		delete(p.Settings, "okSound")
	}
	if v, ok := p.Settings["message"]; ok && v != nil {
		notifier["message"] = v.(string)
		delete(p.Settings, "message")
	}
	notifier["settings"] = packSettings(&p)
	return notifier, nil
}

func (n pushoverNotifier) unpack(raw interface{}, name string) gapi.ContactPoint {
	json := raw.(map[string]interface{})
	uid, disableResolve, settings := unpackCommonNotifierFields(json)

	settings["userKey"] = json["user_key"].(string)
	settings["apiToken"] = json["api_token"].(string)
	if v, ok := json["priority"]; ok && v != nil {
		settings["priority"] = strconv.Itoa(v.(int))
	}
	if v, ok := json["ok_priority"]; ok && v != nil {
		settings["okPriority"] = strconv.Itoa(v.(int))
	}
	if v, ok := json["retry"]; ok && v != nil {
		settings["retry"] = strconv.Itoa(v.(int))
	}
	if v, ok := json["expire"]; ok && v != nil {
		settings["expire"] = strconv.Itoa(v.(int))
	}
	if v, ok := json["device"]; ok && v != nil {
		settings["device"] = v.(string)
	}
	if v, ok := json["sound"]; ok && v != nil {
		settings["sound"] = v.(string)
	}
	if v, ok := json["ok_sound"]; ok && v != nil {
		settings["okSound"] = v.(string)
	}
	if v, ok := json["message"]; ok && v != nil {
		settings["message"] = v.(string)
	}

	return gapi.ContactPoint{
		UID:                   uid,
		Name:                  name,
		Type:                  n.meta().typeStr,
		DisableResolveMessage: disableResolve,
		Settings:              settings,
	}
}

type sensugoNotifier struct{}

var _ notifier = (*sensugoNotifier)(nil)

func (s sensugoNotifier) meta() notifierMeta {
	return notifierMeta{
		field:   "sensugo",
		typeStr: "sensugo",
		desc:    "A contact point that sends notifications to SensuGo.",
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
		Type:             schema.TypeString,
		Required:         true,
		Sensitive:        true,
		DiffSuppressFunc: redactedContactPointDiffSuppress,
		Description:      "The SensuGo API key.",
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

func (s sensugoNotifier) pack(p gapi.ContactPoint) (interface{}, error) {
	notifier := packCommonNotifierFields(&p)
	if v, ok := p.Settings["url"]; ok && v != nil {
		notifier["url"] = v.(string)
		delete(p.Settings, "url")
	}
	if v, ok := p.Settings["apikey"]; ok && v != nil {
		notifier["api_key"] = v.(string)
		delete(p.Settings, "apikey")
	}
	if v, ok := p.Settings["entity"]; ok && v != nil {
		notifier["entity"] = v.(string)
		delete(p.Settings, "entity")
	}
	if v, ok := p.Settings["check"]; ok && v != nil {
		notifier["check"] = v.(string)
		delete(p.Settings, "check")
	}
	if v, ok := p.Settings["namespace"]; ok && v != nil {
		notifier["namespace"] = v.(string)
		delete(p.Settings, "namespace")
	}
	if v, ok := p.Settings["handler"]; ok && v != nil {
		notifier["handler"] = v.(string)
		delete(p.Settings, "handler")
	}
	if v, ok := p.Settings["message"]; ok && v != nil {
		notifier["message"] = v.(string)
		delete(p.Settings, "message")
	}
	notifier["settings"] = packSettings(&p)
	return notifier, nil
}

func (s sensugoNotifier) unpack(raw interface{}, name string) gapi.ContactPoint {
	json := raw.(map[string]interface{})
	uid, disableResolve, settings := unpackCommonNotifierFields(json)

	settings["url"] = json["url"].(string)
	settings["apikey"] = json["api_key"].(string)
	if v, ok := json["entity"]; ok && v != nil {
		settings["entity"] = v.(string)
	}
	if v, ok := json["check"]; ok && v != nil {
		settings["check"] = v.(string)
	}
	if v, ok := json["namespace"]; ok && v != nil {
		settings["namespace"] = v.(string)
	}
	if v, ok := json["handler"]; ok && v != nil {
		settings["handler"] = v.(string)
	}
	if v, ok := json["message"]; ok && v != nil {
		settings["message"] = v.(string)
	}
	return gapi.ContactPoint{
		UID:                   uid,
		Name:                  name,
		Type:                  s.meta().typeStr,
		DisableResolveMessage: disableResolve,
		Settings:              settings,
	}
}
