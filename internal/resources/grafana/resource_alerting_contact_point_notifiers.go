package grafana

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
)

type alertmanagerNotifier struct{}

var _ notifier = (*alertmanagerNotifier)(nil)

func (a alertmanagerNotifier) meta() notifierMeta {
	return notifierMeta{
		field:        "alertmanager",
		typeStr:      "prometheus-alertmanager",
		desc:         "A contact point that sends notifications to other Alertmanager instances.",
		secureFields: []string{"basic_auth_password"},
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

func (a alertmanagerNotifier) pack(p *models.EmbeddedContactPoint, data *schema.ResourceData) (interface{}, error) {
	notifier := packCommonNotifierFields(p)
	settings := p.Settings.(map[string]interface{})
	if v, ok := settings["url"]; ok && v != nil {
		notifier["url"] = v.(string)
		delete(settings, "url")
	}
	if v, ok := settings["basicAuthUser"]; ok && v != nil {
		notifier["basic_auth_user"] = v.(string)
		delete(settings, "basicAuthUser")
	}
	if v, ok := settings["basicAuthPassword"]; ok && v != nil {
		notifier["basic_auth_password"] = v.(string)
		delete(settings, "basicAuthPassword")
	}

	packSecureFields(notifier, getNotifierConfigFromStateWithUID(data, a, p.UID), a.meta().secureFields)

	notifier["settings"] = packSettings(p)
	return notifier, nil
}

func (a alertmanagerNotifier) unpack(raw interface{}, name string) *models.EmbeddedContactPoint {
	json := raw.(map[string]interface{})
	uid, disableResolve, settings := unpackCommonNotifierFields(json)

	settings["url"] = json["url"].(string)
	if v, ok := json["basic_auth_user"]; ok && v != nil {
		settings["basicAuthUser"] = v.(string)
	}
	if v, ok := json["basic_auth_password"]; ok && v != nil {
		settings["basicAuthPassword"] = v.(string)
	}
	return &models.EmbeddedContactPoint{
		UID:                   uid,
		Name:                  name,
		Type:                  common.Ref(a.meta().typeStr),
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
	r.Schema["title"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The templated title of the message.",
	}
	return r
}

func (d dingDingNotifier) pack(p *models.EmbeddedContactPoint, data *schema.ResourceData) (interface{}, error) {
	notifier := packCommonNotifierFields(p)
	settings := p.Settings.(map[string]interface{})
	if v, ok := settings["url"]; ok && v != nil {
		notifier["url"] = v.(string)
		delete(settings, "url")
	}
	if v, ok := settings["msgType"]; ok && v != nil {
		notifier["message_type"] = v.(string)
		delete(settings, "msgType")
	}
	if v, ok := settings["message"]; ok && v != nil {
		notifier["message"] = v.(string)
		delete(settings, "message")
	}
	if v, ok := settings["title"]; ok && v != nil {
		notifier["title"] = v.(string)
		delete(settings, "title")
	}
	notifier["settings"] = packSettings(p)
	return notifier, nil
}

func (d dingDingNotifier) unpack(raw interface{}, name string) *models.EmbeddedContactPoint {
	json := raw.(map[string]interface{})
	uid, disableResolve, settings := unpackCommonNotifierFields(json)

	settings["url"] = json["url"].(string)
	if v, ok := json["message_type"]; ok && v != nil {
		settings["msgType"] = v.(string)
	}
	if v, ok := json["message"]; ok && v != nil {
		settings["message"] = v.(string)
	}
	if v, ok := json["title"]; ok && v != nil {
		settings["title"] = v.(string)
	}
	return &models.EmbeddedContactPoint{
		UID:                   uid,
		Name:                  name,
		Type:                  common.Ref(d.meta().typeStr),
		DisableResolveMessage: disableResolve,
		Settings:              settings,
	}
}

type discordNotifier struct{}

var _ notifier = (*discordNotifier)(nil)

func (d discordNotifier) meta() notifierMeta {
	return notifierMeta{
		field:        "discord",
		typeStr:      "discord",
		desc:         "A contact point that sends notifications as Discord messages",
		secureFields: []string{"url"},
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

func (d discordNotifier) pack(p *models.EmbeddedContactPoint, data *schema.ResourceData) (interface{}, error) {
	notifier := packCommonNotifierFields(p)
	settings := p.Settings.(map[string]interface{})
	if v, ok := settings["url"]; ok && v != nil {
		notifier["url"] = v.(string)
		delete(settings, "url")
	}
	packNotifierStringField(&settings, &notifier, "title", "title")
	if v, ok := settings["message"]; ok && v != nil {
		notifier["message"] = v.(string)
		delete(settings, "message")
	}
	if v, ok := settings["avatar_url"]; ok && v != nil {
		notifier["avatar_url"] = v.(string)
		delete(settings, "avatar_url")
	}
	if v, ok := settings["use_discord_username"]; ok && v != nil {
		notifier["use_discord_username"] = v.(bool)
		delete(settings, "use_discord_username")
	}

	packSecureFields(notifier, getNotifierConfigFromStateWithUID(data, d, p.UID), d.meta().secureFields)

	notifier["settings"] = packSettings(p)
	return notifier, nil
}

func (d discordNotifier) unpack(raw interface{}, name string) *models.EmbeddedContactPoint {
	json := raw.(map[string]interface{})
	uid, disableResolve, settings := unpackCommonNotifierFields(json)

	settings["url"] = json["url"].(string)
	unpackNotifierStringField(&json, &settings, "title", "title")
	if v, ok := json["message"]; ok && v != nil {
		settings["message"] = v.(string)
	}
	if v, ok := json["avatar_url"]; ok && v != nil {
		settings["avatar_url"] = v.(string)
	}
	if v, ok := json["use_discord_username"]; ok && v != nil {
		settings["use_discord_username"] = v.(bool)
	}

	return &models.EmbeddedContactPoint{
		UID:                   uid,
		Name:                  name,
		Type:                  common.Ref(d.meta().typeStr),
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

func (e emailNotifier) pack(p *models.EmbeddedContactPoint, data *schema.ResourceData) (interface{}, error) {
	notifier := packCommonNotifierFields(p)
	settings := p.Settings.(map[string]interface{})
	if v, ok := settings["addresses"]; ok && v != nil {
		notifier["addresses"] = packAddrs(v.(string))
		delete(settings, "addresses")
	}
	if v, ok := settings["singleEmail"]; ok && v != nil {
		notifier["single_email"] = v.(bool)
		delete(settings, "singleEmail")
	}
	if v, ok := settings["message"]; ok && v != nil {
		notifier["message"] = v.(string)
		delete(settings, "message")
	}
	if v, ok := settings["subject"]; ok && v != nil {
		notifier["subject"] = v.(string)
		delete(settings, "subject")
	}
	notifier["settings"] = packSettings(p)
	return notifier, nil
}

func (e emailNotifier) unpack(raw interface{}, name string) *models.EmbeddedContactPoint {
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

	return &models.EmbeddedContactPoint{
		UID:                   uid,
		Name:                  name,
		Type:                  common.Ref(e.meta().typeStr),
		DisableResolveMessage: disableResolve,
		Settings:              settings,
	}
}

const addrSeparator = ';'

func packAddrs(addrs string) []string {
	return strings.FieldsFunc(addrs, func(r rune) bool {
		switch r {
		case ',', addrSeparator, '\n':
			return true
		}
		return false
	})
}

func unpackAddrs(addrs []interface{}) string {
	strs := common.ListToStringSlice(addrs)
	return strings.Join(strs, string(addrSeparator))
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

func (g googleChatNotifier) pack(p *models.EmbeddedContactPoint, data *schema.ResourceData) (interface{}, error) {
	notifier := packCommonNotifierFields(p)
	settings := p.Settings.(map[string]interface{})
	if v, ok := settings["url"]; ok && v != nil {
		notifier["url"] = v.(string)
		delete(settings, "url")
	}
	packNotifierStringField(&settings, &notifier, "title", "title")
	if v, ok := settings["message"]; ok && v != nil {
		notifier["message"] = v.(string)
		delete(settings, "message")
	}
	notifier["settings"] = packSettings(p)
	return notifier, nil
}

func (g googleChatNotifier) unpack(raw interface{}, name string) *models.EmbeddedContactPoint {
	json := raw.(map[string]interface{})
	uid, disableResolve, settings := unpackCommonNotifierFields(json)

	settings["url"] = json["url"].(string)
	unpackNotifierStringField(&json, &settings, "title", "title")
	if v, ok := json["message"]; ok && v != nil {
		settings["message"] = v.(string)
	}
	return &models.EmbeddedContactPoint{
		UID:                   uid,
		Name:                  name,
		Type:                  common.Ref(g.meta().typeStr),
		DisableResolveMessage: disableResolve,
		Settings:              settings,
	}
}

type kafkaNotifier struct{}

var _ notifier = (*kafkaNotifier)(nil)

func (k kafkaNotifier) meta() notifierMeta {
	return notifierMeta{
		field:        "kafka",
		typeStr:      "kafka",
		desc:         "A contact point that publishes notifications to Apache Kafka topics.",
		secureFields: []string{"rest_proxy_url", "password"},
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

func (k kafkaNotifier) pack(p *models.EmbeddedContactPoint, data *schema.ResourceData) (interface{}, error) {
	notifier := packCommonNotifierFields(p)
	settings := p.Settings.(map[string]interface{})
	if v, ok := settings["kafkaRestProxy"]; ok && v != nil {
		notifier["rest_proxy_url"] = v.(string)
		delete(settings, "kafkaRestProxy")
	}
	if v, ok := settings["kafkaTopic"]; ok && v != nil {
		notifier["topic"] = v.(string)
		delete(settings, "kafkaTopic")
	}
	packNotifierStringField(&settings, &notifier, "description", "description")
	packNotifierStringField(&settings, &notifier, "details", "details")
	packNotifierStringField(&settings, &notifier, "username", "username")
	packNotifierStringField(&settings, &notifier, "password", "password")
	packNotifierStringField(&settings, &notifier, "apiVersion", "api_version")
	packNotifierStringField(&settings, &notifier, "kafkaClusterId", "cluster_id")

	packSecureFields(notifier, getNotifierConfigFromStateWithUID(data, k, p.UID), k.meta().secureFields)

	notifier["settings"] = packSettings(p)
	return notifier, nil
}

func (k kafkaNotifier) unpack(raw interface{}, name string) *models.EmbeddedContactPoint {
	json := raw.(map[string]interface{})
	uid, disableResolve, settings := unpackCommonNotifierFields(json)

	settings["kafkaRestProxy"] = json["rest_proxy_url"].(string)
	settings["kafkaTopic"] = json["topic"].(string)
	unpackNotifierStringField(&json, &settings, "description", "description")
	unpackNotifierStringField(&json, &settings, "details", "details")
	unpackNotifierStringField(&json, &settings, "username", "username")
	unpackNotifierStringField(&json, &settings, "password", "password")
	unpackNotifierStringField(&json, &settings, "api_version", "apiVersion")
	unpackNotifierStringField(&json, &settings, "cluster_id", "kafkaClusterId")

	return &models.EmbeddedContactPoint{
		UID:                   uid,
		Name:                  name,
		Type:                  common.Ref(k.meta().typeStr),
		DisableResolveMessage: disableResolve,
		Settings:              settings,
	}
}

type lineNotifier struct{}

var _ notifier = (*lineNotifier)(nil)

func (o lineNotifier) meta() notifierMeta {
	return notifierMeta{
		field:        "line",
		typeStr:      "LINE",
		desc:         "A contact point that sends notifications to LINE.me.",
		secureFields: []string{"token"},
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

func (o lineNotifier) pack(p *models.EmbeddedContactPoint, data *schema.ResourceData) (interface{}, error) {
	notifier := packCommonNotifierFields(p)
	settings := p.Settings.(map[string]interface{})

	packNotifierStringField(&settings, &notifier, "token", "token")
	packNotifierStringField(&settings, &notifier, "title", "title")
	packNotifierStringField(&settings, &notifier, "description", "description")

	packSecureFields(notifier, getNotifierConfigFromStateWithUID(data, o, p.UID), o.meta().secureFields)

	notifier["settings"] = packSettings(p)
	return notifier, nil
}

func (o lineNotifier) unpack(raw interface{}, name string) *models.EmbeddedContactPoint {
	json := raw.(map[string]interface{})
	uid, disableResolve, settings := unpackCommonNotifierFields(json)

	unpackNotifierStringField(&json, &settings, "token", "token")
	unpackNotifierStringField(&json, &settings, "title", "title")
	unpackNotifierStringField(&json, &settings, "description", "description")

	return &models.EmbeddedContactPoint{
		UID:                   uid,
		Name:                  name,
		Type:                  common.Ref(o.meta().typeStr),
		DisableResolveMessage: disableResolve,
		Settings:              settings,
	}
}

type mqttNotifier struct{}

var _ notifier = (*mqttNotifier)(nil)

func (o mqttNotifier) meta() notifierMeta {
	return notifierMeta{
		field:        "mqtt",
		typeStr:      "mqtt",
		desc:         "A contact point that sends notifications to an MQTT broker.",
		secureFields: []string{"password"},
	}
}

func (o mqttNotifier) schema() *schema.Resource {
	r := commonNotifierResource()
	r.Schema["broker_url"] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Description: "The URL of the MQTT broker.",
	}
	r.Schema["topic"] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Description: "The topic to publish messages to.",
	}
	r.Schema["client_id"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The client ID to use when connecting to the broker.",
	}
	r.Schema["message_format"] = &schema.Schema{
		Type:         schema.TypeString,
		Optional:     true,
		ValidateFunc: validation.StringInSlice([]string{"json", "text"}, false),
		Description:  "The format of the message to send. Supported values are `json` and `text`.",
	}
	r.Schema["username"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The username to use when connecting to the broker.",
	}
	r.Schema["password"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Sensitive:   true,
		Description: "The password to use when connecting to the broker.",
	}
	r.Schema["qos"] = &schema.Schema{
		Type:         schema.TypeInt,
		Optional:     true,
		Default:      0,
		ValidateFunc: validation.IntBetween(0, 2),
		Description:  "The quality of service to use when sending messages. Supported values are 0, 1, and 2.",
	}
	r.Schema["retain"] = &schema.Schema{
		Type:        schema.TypeBool,
		Optional:    true,
		Default:     false,
		Description: "Whether to retain messages on the broker.",
	}

	r.Schema["tls_config"] = &schema.Schema{
		Type:        schema.TypeSet,
		Optional:    true,
		Description: "TLS configuration for the connection.",
		Elem:        tlsConfig{}.schema(),
	}

	return r
}

func (o mqttNotifier) pack(p *models.EmbeddedContactPoint, data *schema.ResourceData) (interface{}, error) {
	notifier := packCommonNotifierFields(p)
	settings := p.Settings.(map[string]interface{})

	packNotifierStringField(&settings, &notifier, "brokerUrl", "broker_url")
	packNotifierStringField(&settings, &notifier, "topic", "topic")
	packNotifierStringField(&settings, &notifier, "clientId", "client_id")
	packNotifierStringField(&settings, &notifier, "messageFormat", "message_format")
	packNotifierStringField(&settings, &notifier, "username", "username")
	if v, ok := settings["insecureSkipVerify"]; ok && v != nil {
		notifier["insecure_skip_verify"] = v.(bool)
		delete(settings, "insecureSkipVerify")
	}

	packSecureFields(notifier, getNotifierConfigFromStateWithUID(data, o, p.UID), o.meta().secureFields)

	notifier["settings"] = packSettings(p)
	return notifier, nil
}

func (o mqttNotifier) unpack(raw interface{}, name string) *models.EmbeddedContactPoint {
	json := raw.(map[string]interface{})
	uid, disableResolve, settings := unpackCommonNotifierFields(json)

	unpackNotifierStringField(&json, &settings, "broker_url", "brokerUrl")
	unpackNotifierStringField(&json, &settings, "topic", "topic")
	unpackNotifierStringField(&json, &settings, "client_id", "clientId")
	unpackNotifierStringField(&json, &settings, "message_format", "messageFormat")
	unpackNotifierStringField(&json, &settings, "username", "username")
	unpackNotifierStringField(&json, &settings, "password", "password")
	if v, ok := json["insecure_skip_verify"]; ok && v != nil {
		settings["insecureSkipVerify"] = v.(bool)
	}

	return &models.EmbeddedContactPoint{
		UID:                   uid,
		Name:                  name,
		Type:                  common.Ref(o.meta().typeStr),
		DisableResolveMessage: disableResolve,
		Settings:              settings,
	}
}

type tlsConfig struct{}

func (t tlsConfig) schema() *schema.Resource {
	r := &schema.Resource{
		Schema: make(map[string]*schema.Schema),
	}

	r.Schema["insecure_skip_verify"] = &schema.Schema{
		Type:        schema.TypeBool,
		Optional:    true,
		Default:     false,
		Description: "Whether to skip verification of the server's certificate chain and host name.",
	}
	r.Schema["ca_certificate"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Sensitive:   true,
		Description: "The CA certificate to use when verifying the server's certificate.",
	}
	r.Schema["client_certificate"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Sensitive:   true,
		Description: "The client certificate to use when connecting to the server.",
	}
	r.Schema["client_key"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Sensitive:   true,
		Description: "The client key to use when connecting to the server.",
	}

	return r
}

func (t tlsConfig) pack(p *models.EmbeddedContactPoint, data *schema.ResourceData) (interface{}, error) {
	settings := p.Settings.(map[string]interface{})
	tls := make(map[string]interface{})

	if v, ok := settings["insecureSkipVerify"]; ok && v != nil {
		tls["insecure_skip_verify"] = v.(bool)
		delete(settings, "insecureSkipVerify")
	}
	packNotifierStringField(&settings, &tls, "caCertificate", "ca_certificate")
	packNotifierStringField(&settings, &tls, "clientCertificate", "client_certificate")
	packNotifierStringField(&settings, &tls, "clientKey", "client_key")

	return tls, nil
}

func (t tlsConfig) unpack(raw interface{}) map[string]interface{} {
	json := raw.(map[string]interface{})
	tls := make(map[string]interface{})

	if v, ok := json["insecure_skip_verify"]; ok && v != nil {
		tls["insecureSkipVerify"] = v.(bool)
	}
	unpackNotifierStringField(&json, &tls, "ca_certificate", "caCertificate")
	unpackNotifierStringField(&json, &tls, "client_certificate", "clientCertificate")
	unpackNotifierStringField(&json, &tls, "client_key", "clientKey")

	return tls
}

type oncallNotifier struct {
}

var _ notifier = (*oncallNotifier)(nil)

func (w oncallNotifier) meta() notifierMeta {
	return notifierMeta{
		field:        "oncall",
		typeStr:      "oncall",
		desc:         "A contact point that sends notifications to Grafana On-Call.",
		secureFields: []string{"basic_auth_password", "authorization_credentials"},
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

func (w oncallNotifier) pack(p *models.EmbeddedContactPoint, data *schema.ResourceData) (interface{}, error) {
	notifier := packCommonNotifierFields(p)
	settings := p.Settings.(map[string]interface{})

	packNotifierStringField(&settings, &notifier, "url", "url")
	packNotifierStringField(&settings, &notifier, "httpMethod", "http_method")
	packNotifierStringField(&settings, &notifier, "username", "basic_auth_user")
	packNotifierStringField(&settings, &notifier, "password", "basic_auth_password")
	packNotifierStringField(&settings, &notifier, "authorization_scheme", "authorization_scheme")
	packNotifierStringField(&settings, &notifier, "authorization_credentials", "authorization_credentials")
	packNotifierStringField(&settings, &notifier, "message", "message")
	packNotifierStringField(&settings, &notifier, "title", "title")
	if v, ok := settings["maxAlerts"]; ok && v != nil {
		switch typ := v.(type) {
		case int:
			notifier["max_alerts"] = v.(int)
		case float64:
			notifier["max_alerts"] = int(v.(float64))
		case string:
			val, err := strconv.Atoi(typ)
			if err != nil {
				panic(fmt.Errorf("failed to parse value of 'maxAlerts' to integer: %w", err))
			}
			notifier["max_alerts"] = val
		default:
			panic(fmt.Sprintf("unexpected type %T for 'maxAlerts': %v", typ, typ))
		}
		delete(settings, "maxAlerts")
	}

	packSecureFields(notifier, getNotifierConfigFromStateWithUID(data, w, p.UID), w.meta().secureFields)

	notifier["settings"] = packSettings(p)
	return notifier, nil
}

func (w oncallNotifier) unpack(raw interface{}, name string) *models.EmbeddedContactPoint {
	json := raw.(map[string]interface{})
	uid, disableResolve, settings := unpackCommonNotifierFields(json)

	unpackNotifierStringField(&json, &settings, "url", "url")
	unpackNotifierStringField(&json, &settings, "http_method", "httpMethod")
	unpackNotifierStringField(&json, &settings, "basic_auth_user", "username")
	unpackNotifierStringField(&json, &settings, "basic_auth_password", "password")
	unpackNotifierStringField(&json, &settings, "authorization_scheme", "authorization_scheme")
	unpackNotifierStringField(&json, &settings, "authorization_credentials", "authorization_credentials")
	unpackNotifierStringField(&json, &settings, "message", "message")
	unpackNotifierStringField(&json, &settings, "title", "title")
	if v, ok := json["max_alerts"]; ok && v != nil {
		switch typ := v.(type) {
		case int:
			settings["maxAlerts"] = v.(int)
		case float64:
			settings["maxAlerts"] = int(v.(float64))
		default:
			panic(fmt.Sprintf("unexpected type for maxAlerts: %v", typ))
		}
	}

	return &models.EmbeddedContactPoint{
		UID:                   uid,
		Name:                  name,
		Type:                  common.Ref(w.meta().typeStr),
		DisableResolveMessage: disableResolve,
		Settings:              settings,
	}
}

type opsGenieNotifier struct{}

var _ notifier = (*opsGenieNotifier)(nil)

func (o opsGenieNotifier) meta() notifierMeta {
	return notifierMeta{
		field:        "opsgenie",
		typeStr:      "opsgenie",
		desc:         "A contact point that sends notifications to OpsGenie.",
		secureFields: []string{"api_key"},
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

func (o opsGenieNotifier) pack(p *models.EmbeddedContactPoint, data *schema.ResourceData) (interface{}, error) {
	notifier := packCommonNotifierFields(p)
	settings := p.Settings.(map[string]interface{})
	if v, ok := settings["apiUrl"]; ok && v != nil {
		notifier["url"] = v.(string)
		delete(settings, "apiUrl")
	}
	if v, ok := settings["apiKey"]; ok && v != nil {
		notifier["api_key"] = v.(string)
		delete(settings, "apiKey")
	}
	if v, ok := settings["message"]; ok && v != nil {
		notifier["message"] = v.(string)
		delete(settings, "message")
	}
	if v, ok := settings["description"]; ok && v != nil {
		notifier["description"] = v.(string)
		delete(settings, "description")
	}
	if v, ok := settings["autoClose"]; ok && v != nil {
		notifier["auto_close"] = v.(bool)
		delete(settings, "autoClose")
	}
	if v, ok := settings["overridePriority"]; ok && v != nil {
		notifier["override_priority"] = v.(bool)
		delete(settings, "overridePriority")
	}
	if v, ok := settings["sendTagsAs"]; ok && v != nil {
		notifier["send_tags_as"] = v.(string)
		delete(settings, "sendTagsAs")
	}
	if v, ok := settings["responders"]; ok && v != nil {
		items := v.([]any)
		responders := make([]map[string]interface{}, 0, len(items))
		for _, item := range items {
			itemMap := item.(map[string]interface{})
			responder := make(map[string]interface{}, 4)
			packNotifierStringField(&itemMap, &responder, "type", "type")
			packNotifierStringField(&itemMap, &responder, "id", "id")
			packNotifierStringField(&itemMap, &responder, "name", "name")
			packNotifierStringField(&itemMap, &responder, "username", "username")
			responders = append(responders, responder)
		}
		notifier["responders"] = responders
		delete(settings, "responders")
	}

	packSecureFields(notifier, getNotifierConfigFromStateWithUID(data, o, p.UID), o.meta().secureFields)

	notifier["settings"] = packSettings(p)
	return notifier, nil
}

func (o opsGenieNotifier) unpack(raw interface{}, name string) *models.EmbeddedContactPoint {
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
	if v, ok := json["responders"]; ok && v != nil {
		items := v.([]any)
		responders := make([]map[string]interface{}, 0, len(items))
		for _, item := range items {
			tfResponder := item.(map[string]interface{})
			responder := make(map[string]interface{}, 4)
			unpackNotifierStringField(&tfResponder, &responder, "type", "type")
			unpackNotifierStringField(&tfResponder, &responder, "id", "id")
			unpackNotifierStringField(&tfResponder, &responder, "name", "name")
			unpackNotifierStringField(&tfResponder, &responder, "username", "username")
			responders = append(responders, responder)
		}
		settings["responders"] = responders
	}
	return &models.EmbeddedContactPoint{
		UID:                   uid,
		Name:                  name,
		Type:                  common.Ref(o.meta().typeStr),
		DisableResolveMessage: disableResolve,
		Settings:              settings,
	}
}

type pagerDutyNotifier struct{}

var _ notifier = (*pagerDutyNotifier)(nil)

func (n pagerDutyNotifier) meta() notifierMeta {
	return notifierMeta{
		field:        "pagerduty",
		typeStr:      "pagerduty",
		desc:         "A contact point that sends notifications to PagerDuty.",
		secureFields: []string{"integration_key"},
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

func (n pagerDutyNotifier) pack(p *models.EmbeddedContactPoint, data *schema.ResourceData) (interface{}, error) {
	notifier := packCommonNotifierFields(p)
	settings := p.Settings.(map[string]interface{})
	if v, ok := settings["integrationKey"]; ok && v != nil {
		notifier["integration_key"] = v.(string)
		delete(settings, "integrationKey")
	}
	if v, ok := settings["severity"]; ok && v != nil {
		notifier["severity"] = v.(string)
		delete(settings, "severity")
	}
	if v, ok := settings["class"]; ok && v != nil {
		notifier["class"] = v.(string)
		delete(settings, "class")
	}
	if v, ok := settings["component"]; ok && v != nil {
		notifier["component"] = v.(string)
		delete(settings, "component")
	}
	if v, ok := settings["group"]; ok && v != nil {
		notifier["group"] = v.(string)
		delete(settings, "group")
	}
	if v, ok := settings["summary"]; ok && v != nil {
		notifier["summary"] = v.(string)
		delete(settings, "summary")
	}
	if v, ok := settings["source"]; ok && v != nil {
		notifier["source"] = v.(string)
		delete(settings, "source")
	}
	if v, ok := settings["client"]; ok && v != nil {
		notifier["client"] = v.(string)
		delete(settings, "client")
	}
	if v, ok := settings["client_url"]; ok && v != nil {
		notifier["client_url"] = v.(string)
		delete(settings, "client_url")
	}
	if v, ok := settings["details"]; ok && v != nil {
		notifier["details"] = unpackMap(v)
		delete(settings, "details")
	}
	if v, ok := settings["url"]; ok && v != nil {
		notifier["url"] = v.(string)
		delete(settings, "url")
	}

	packSecureFields(notifier, getNotifierConfigFromStateWithUID(data, n, p.UID), n.meta().secureFields)

	notifier["settings"] = packSettings(p)
	return notifier, nil
}

func (n pagerDutyNotifier) unpack(raw interface{}, name string) *models.EmbeddedContactPoint {
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
	if v, ok := json["source"]; ok && v != nil {
		settings["source"] = v.(string)
	}
	if v, ok := json["client"]; ok && v != nil {
		settings["client"] = v.(string)
	}
	if v, ok := json["client_url"]; ok && v != nil {
		settings["client_url"] = v.(string)
	}
	if v, ok := json["details"]; ok && v != nil {
		settings["details"] = unpackMap(v)
	}
	if v, ok := json["url"]; ok && v != nil {
		settings["url"] = v.(string)
	}
	return &models.EmbeddedContactPoint{
		UID:                   uid,
		Name:                  name,
		Type:                  common.Ref(n.meta().typeStr),
		DisableResolveMessage: disableResolve,
		Settings:              settings,
	}
}

type pushoverNotifier struct{}

var _ notifier = (*pushoverNotifier)(nil)

func (n pushoverNotifier) meta() notifierMeta {
	return notifierMeta{
		field:        "pushover",
		typeStr:      "pushover",
		desc:         "A contact point that sends notifications to Pushover.",
		secureFields: []string{"user_key", "api_token"},
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

func (n pushoverNotifier) pack(p *models.EmbeddedContactPoint, data *schema.ResourceData) (interface{}, error) {
	notifier := packCommonNotifierFields(p)
	settings := p.Settings.(map[string]interface{})
	if v, ok := settings["userKey"]; ok && v != nil {
		notifier["user_key"] = v.(string)
		delete(settings, "userKey")
	}
	if v, ok := settings["apiToken"]; ok && v != nil {
		notifier["api_token"] = v.(string)
		delete(settings, "apiToken")
	}
	if v, ok := settings["priority"]; ok && v != nil {
		priority, err := strconv.Atoi(v.(string))
		if err != nil {
			return nil, err
		}
		notifier["priority"] = priority
		delete(settings, "priority")
	}
	if v, ok := settings["okPriority"]; ok && v != nil {
		priority, err := strconv.Atoi(v.(string))
		if err != nil {
			return nil, err
		}
		notifier["ok_priority"] = priority
		delete(settings, "okPriority")
	}
	if v, ok := settings["retry"]; ok && v != nil {
		priority, err := strconv.Atoi(v.(string))
		if err != nil {
			return nil, err
		}
		notifier["retry"] = priority
		delete(settings, "retry")
	}
	if v, ok := settings["expire"]; ok && v != nil {
		priority, err := strconv.Atoi(v.(string))
		if err != nil {
			return nil, err
		}
		notifier["expire"] = priority
		delete(settings, "expire")
	}
	if v, ok := settings["device"]; ok && v != nil {
		notifier["device"] = v.(string)
		delete(settings, "device")
	}
	if v, ok := settings["sound"]; ok && v != nil {
		notifier["sound"] = v.(string)
		delete(settings, "sound")
	}
	if v, ok := settings["okSound"]; ok && v != nil {
		notifier["ok_sound"] = v.(string)
		delete(settings, "okSound")
	}
	if v, ok := settings["title"]; ok && v != nil {
		notifier["title"] = v.(string)
		delete(settings, "title")
	}
	if v, ok := settings["message"]; ok && v != nil {
		notifier["message"] = v.(string)
		delete(settings, "message")
	}
	if v, ok := settings["uploadImage"]; ok && v != nil {
		notifier["upload_image"] = v.(bool)
		delete(settings, "uploadImage")
	}

	packSecureFields(notifier, getNotifierConfigFromStateWithUID(data, n, p.UID), n.meta().secureFields)

	notifier["settings"] = packSettings(p)
	return notifier, nil
}

func (n pushoverNotifier) unpack(raw interface{}, name string) *models.EmbeddedContactPoint {
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
	if v, ok := json["title"]; ok && v != nil {
		settings["title"] = v.(string)
	}
	if v, ok := json["message"]; ok && v != nil {
		settings["message"] = v.(string)
	}
	if v, ok := json["upload_image"]; ok && v != nil {
		settings["uploadImage"] = v.(bool)
	}

	return &models.EmbeddedContactPoint{
		UID:                   uid,
		Name:                  name,
		Type:                  common.Ref(n.meta().typeStr),
		DisableResolveMessage: disableResolve,
		Settings:              settings,
	}
}

type sensugoNotifier struct{}

var _ notifier = (*sensugoNotifier)(nil)

func (s sensugoNotifier) meta() notifierMeta {
	return notifierMeta{
		field:        "sensugo",
		typeStr:      "sensugo",
		desc:         "A contact point that sends notifications to SensuGo.",
		secureFields: []string{"api_key"},
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

func (s sensugoNotifier) pack(p *models.EmbeddedContactPoint, data *schema.ResourceData) (interface{}, error) {
	notifier := packCommonNotifierFields(p)
	settings := p.Settings.(map[string]interface{})
	if v, ok := settings["url"]; ok && v != nil {
		notifier["url"] = v.(string)
		delete(settings, "url")
	}
	if v, ok := settings["apikey"]; ok && v != nil {
		notifier["api_key"] = v.(string)
		delete(settings, "apikey")
	}
	if v, ok := settings["entity"]; ok && v != nil {
		notifier["entity"] = v.(string)
		delete(settings, "entity")
	}
	if v, ok := settings["check"]; ok && v != nil {
		notifier["check"] = v.(string)
		delete(settings, "check")
	}
	if v, ok := settings["namespace"]; ok && v != nil {
		notifier["namespace"] = v.(string)
		delete(settings, "namespace")
	}
	if v, ok := settings["handler"]; ok && v != nil {
		notifier["handler"] = v.(string)
		delete(settings, "handler")
	}
	if v, ok := settings["message"]; ok && v != nil {
		notifier["message"] = v.(string)
		delete(settings, "message")
	}

	packSecureFields(notifier, getNotifierConfigFromStateWithUID(data, s, p.UID), s.meta().secureFields)

	notifier["settings"] = packSettings(p)
	return notifier, nil
}

func (s sensugoNotifier) unpack(raw interface{}, name string) *models.EmbeddedContactPoint {
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
	return &models.EmbeddedContactPoint{
		UID:                   uid,
		Name:                  name,
		Type:                  common.Ref(s.meta().typeStr),
		DisableResolveMessage: disableResolve,
		Settings:              settings,
	}
}

type slackNotifier struct{}

var _ notifier = (*slackNotifier)(nil)

func (s slackNotifier) meta() notifierMeta {
	return notifierMeta{
		field:        "slack",
		typeStr:      "slack",
		desc:         "A contact point that sends notifications to Slack.",
		secureFields: []string{"url", "token"},
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
	return r
}

func (s slackNotifier) pack(p *models.EmbeddedContactPoint, data *schema.ResourceData) (interface{}, error) {
	notifier := packCommonNotifierFields(p)
	settings := p.Settings.(map[string]interface{})

	packNotifierStringField(&settings, &notifier, "endpointUrl", "endpoint_url")
	packNotifierStringField(&settings, &notifier, "url", "url")
	packNotifierStringField(&settings, &notifier, "token", "token")
	packNotifierStringField(&settings, &notifier, "recipient", "recipient")
	packNotifierStringField(&settings, &notifier, "text", "text")
	packNotifierStringField(&settings, &notifier, "title", "title")
	packNotifierStringField(&settings, &notifier, "username", "username")
	packNotifierStringField(&settings, &notifier, "icon_emoji", "icon_emoji")
	packNotifierStringField(&settings, &notifier, "icon_url", "icon_url")
	packNotifierStringField(&settings, &notifier, "mentionChannel", "mention_channel")
	packNotifierStringField(&settings, &notifier, "mentionUsers", "mention_users")
	packNotifierStringField(&settings, &notifier, "mentionGroups", "mention_groups")

	packSecureFields(notifier, getNotifierConfigFromStateWithUID(data, s, p.UID), s.meta().secureFields)

	notifier["settings"] = packSettings(p)

	return notifier, nil
}

func (s slackNotifier) unpack(raw interface{}, name string) *models.EmbeddedContactPoint {
	json := raw.(map[string]interface{})
	uid, disableResolve, settings := unpackCommonNotifierFields(json)

	unpackNotifierStringField(&json, &settings, "endpoint_url", "endpointUrl")
	unpackNotifierStringField(&json, &settings, "url", "url")
	unpackNotifierStringField(&json, &settings, "token", "token")
	unpackNotifierStringField(&json, &settings, "recipient", "recipient")
	unpackNotifierStringField(&json, &settings, "text", "text")
	unpackNotifierStringField(&json, &settings, "title", "title")
	unpackNotifierStringField(&json, &settings, "username", "username")
	unpackNotifierStringField(&json, &settings, "icon_emoji", "icon_emoji")
	unpackNotifierStringField(&json, &settings, "icon_url", "icon_url")
	unpackNotifierStringField(&json, &settings, "mention_channel", "mentionChannel")
	unpackNotifierStringField(&json, &settings, "mention_users", "mentionUsers")
	unpackNotifierStringField(&json, &settings, "mention_groups", "mentionGroups")

	return &models.EmbeddedContactPoint{
		UID:                   uid,
		Name:                  name,
		Type:                  common.Ref(s.meta().typeStr),
		DisableResolveMessage: disableResolve,
		Settings:              settings,
	}
}

type snsNotifier struct{}

var _ notifier = (*snsNotifier)(nil)

func (s snsNotifier) meta() notifierMeta {
	return notifierMeta{
		field:        "sns",
		typeStr:      "sns",
		desc:         "A contact point that sends notifications to Amazon SNS. Requires Amazon Managed Grafana.",
		secureFields: []string{"access_key", "secret_key"},
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

func (s snsNotifier) pack(p *models.EmbeddedContactPoint, data *schema.ResourceData) (interface{}, error) {
	notifier := packCommonNotifierFields(p)
	settings := p.Settings.(map[string]interface{})

	packNotifierStringField(&settings, &notifier, "topic", "topic")
	packNotifierStringField(&settings, &notifier, "authProvider", "auth_provider")
	packNotifierStringField(&settings, &notifier, "accessKey", "access_key")
	packNotifierStringField(&settings, &notifier, "secretKey", "secret_key")
	packNotifierStringField(&settings, &notifier, "assumeRoleARN", "assume_role_arn")
	packNotifierStringField(&settings, &notifier, "messageFormat", "message_format")
	packNotifierStringField(&settings, &notifier, "body", "body")
	packNotifierStringField(&settings, &notifier, "subject", "subject")
	packNotifierStringField(&settings, &notifier, "externalId", "external_id")

	packSecureFields(notifier, getNotifierConfigFromStateWithUID(data, s, p.UID), s.meta().secureFields)

	notifier["settings"] = packSettings(p)

	return notifier, nil
}

func (s snsNotifier) unpack(raw interface{}, name string) *models.EmbeddedContactPoint {
	json := raw.(map[string]interface{})
	uid, disableResolve, settings := unpackCommonNotifierFields(json)

	unpackNotifierStringField(&json, &settings, "topic", "topic")
	unpackNotifierStringField(&json, &settings, "auth_provider", "authProvider")
	unpackNotifierStringField(&json, &settings, "access_key", "accessKey")
	unpackNotifierStringField(&json, &settings, "secret_key", "secretKey")
	unpackNotifierStringField(&json, &settings, "assume_role_arn", "assumeRoleARN")
	unpackNotifierStringField(&json, &settings, "message_format", "messageFormat")
	unpackNotifierStringField(&json, &settings, "body", "body")
	unpackNotifierStringField(&json, &settings, "subject", "subject")
	unpackNotifierStringField(&json, &settings, "external_id", "externalId")

	return &models.EmbeddedContactPoint{
		UID:                   uid,
		Name:                  name,
		Type:                  common.Ref(s.meta().typeStr),
		DisableResolveMessage: disableResolve,
		Settings:              settings,
	}
}

type teamsNotifier struct{}

var _ notifier = (*teamsNotifier)(nil)

func (t teamsNotifier) meta() notifierMeta {
	return notifierMeta{
		field:        "teams",
		typeStr:      "teams",
		desc:         "A contact point that sends notifications to Microsoft Teams.",
		secureFields: []string{"url"},
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

func (t teamsNotifier) pack(p *models.EmbeddedContactPoint, data *schema.ResourceData) (interface{}, error) {
	notifier := packCommonNotifierFields(p)
	settings := p.Settings.(map[string]interface{})

	packNotifierStringField(&settings, &notifier, "url", "url")
	packNotifierStringField(&settings, &notifier, "message", "message")
	packNotifierStringField(&settings, &notifier, "title", "title")
	packNotifierStringField(&settings, &notifier, "sectiontitle", "section_title")

	packSecureFields(notifier, getNotifierConfigFromStateWithUID(data, t, p.UID), t.meta().secureFields)

	notifier["settings"] = packSettings(p)
	return notifier, nil
}

func (t teamsNotifier) unpack(raw interface{}, name string) *models.EmbeddedContactPoint {
	json := raw.(map[string]interface{})
	uid, disableResolve, settings := unpackCommonNotifierFields(json)

	unpackNotifierStringField(&json, &settings, "url", "url")
	unpackNotifierStringField(&json, &settings, "message", "message")
	unpackNotifierStringField(&json, &settings, "title", "title")
	unpackNotifierStringField(&json, &settings, "section_title", "sectiontitle")

	return &models.EmbeddedContactPoint{
		UID:                   uid,
		Name:                  name,
		Type:                  common.Ref(t.meta().typeStr),
		DisableResolveMessage: disableResolve,
		Settings:              settings,
	}
}

type telegramNotifier struct{}

var _ notifier = (*telegramNotifier)(nil)

func (t telegramNotifier) meta() notifierMeta {
	return notifierMeta{
		field:        "telegram",
		typeStr:      "telegram",
		desc:         "A contact point that sends notifications to Telegram.",
		secureFields: []string{"token"},
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

func (t telegramNotifier) pack(p *models.EmbeddedContactPoint, data *schema.ResourceData) (interface{}, error) {
	notifier := packCommonNotifierFields(p)
	settings := p.Settings.(map[string]interface{})

	packNotifierStringField(&settings, &notifier, "bottoken", "token")
	packNotifierStringField(&settings, &notifier, "chatid", "chat_id")
	packNotifierStringField(&settings, &notifier, "message_thread_id", "message_thread_id")
	packNotifierStringField(&settings, &notifier, "message", "message")
	packNotifierStringField(&settings, &notifier, "parse_mode", "parse_mode")

	if v, ok := settings["disable_web_page_preview"]; ok && v != nil {
		notifier["disable_web_page_preview"] = v.(bool)
		delete(settings, "disable_web_page_preview")
	}
	if v, ok := settings["protect_content"]; ok && v != nil {
		notifier["protect_content"] = v.(bool)
		delete(settings, "protect_content")
	}
	if v, ok := settings["disable_notifications"]; ok && v != nil {
		notifier["disable_notifications"] = v.(bool)
		delete(settings, "disable_notifications")
	}

	packSecureFields(notifier, getNotifierConfigFromStateWithUID(data, t, p.UID), t.meta().secureFields)

	notifier["settings"] = packSettings(p)
	return notifier, nil
}

func (t telegramNotifier) unpack(raw interface{}, name string) *models.EmbeddedContactPoint {
	json := raw.(map[string]interface{})
	uid, disableResolve, settings := unpackCommonNotifierFields(json)

	unpackNotifierStringField(&json, &settings, "token", "bottoken")
	unpackNotifierStringField(&json, &settings, "chat_id", "chatid")
	unpackNotifierStringField(&json, &settings, "message_thread_id", "message_thread_id")
	unpackNotifierStringField(&json, &settings, "message", "message")
	unpackNotifierStringField(&json, &settings, "parse_mode", "parse_mode")

	if v, ok := json["disable_web_page_preview"]; ok && v != nil {
		settings["disable_web_page_preview"] = v.(bool)
	}
	if v, ok := json["protect_content"]; ok && v != nil {
		settings["protect_content"] = v.(bool)
	}
	if v, ok := json["disable_notifications"]; ok && v != nil {
		settings["disable_notifications"] = v.(bool)
	}

	return &models.EmbeddedContactPoint{
		UID:                   uid,
		Name:                  name,
		Type:                  common.Ref(t.meta().typeStr),
		DisableResolveMessage: disableResolve,
		Settings:              settings,
	}
}

type threemaNotifier struct{}

var _ notifier = (*threemaNotifier)(nil)

func (t threemaNotifier) meta() notifierMeta {
	return notifierMeta{
		field:        "threema",
		typeStr:      "threema",
		desc:         "A contact point that sends notifications to Threema.",
		secureFields: []string{"api_secret"},
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

func (t threemaNotifier) pack(p *models.EmbeddedContactPoint, data *schema.ResourceData) (interface{}, error) {
	notifier := packCommonNotifierFields(p)
	settings := p.Settings.(map[string]interface{})

	packNotifierStringField(&settings, &notifier, "gateway_id", "gateway_id")
	packNotifierStringField(&settings, &notifier, "recipient_id", "recipient_id")
	packNotifierStringField(&settings, &notifier, "api_secret", "api_secret")
	packNotifierStringField(&settings, &notifier, "title", "title")
	packNotifierStringField(&settings, &notifier, "description", "description")

	packSecureFields(notifier, getNotifierConfigFromStateWithUID(data, t, p.UID), t.meta().secureFields)

	notifier["settings"] = packSettings(p)
	return notifier, nil
}

func (t threemaNotifier) unpack(raw interface{}, name string) *models.EmbeddedContactPoint {
	json := raw.(map[string]interface{})
	uid, disableResolve, settings := unpackCommonNotifierFields(json)

	unpackNotifierStringField(&json, &settings, "gateway_id", "gateway_id")
	unpackNotifierStringField(&json, &settings, "recipient_id", "recipient_id")
	unpackNotifierStringField(&json, &settings, "api_secret", "api_secret")
	unpackNotifierStringField(&json, &settings, "title", "title")
	unpackNotifierStringField(&json, &settings, "description", "description")

	return &models.EmbeddedContactPoint{
		UID:                   uid,
		Name:                  name,
		Type:                  common.Ref(t.meta().typeStr),
		DisableResolveMessage: disableResolve,
		Settings:              settings,
	}
}

type victorOpsNotifier struct{}

var _ notifier = (*victorOpsNotifier)(nil)

func (v victorOpsNotifier) meta() notifierMeta {
	return notifierMeta{
		field:   "victorops",
		typeStr: "victorops",
		desc:    "A contact point that sends notifications to VictorOps (now known as Splunk OnCall).",
	}
}

func (v victorOpsNotifier) schema() *schema.Resource {
	r := commonNotifierResource()
	r.Schema["url"] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
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

func (v victorOpsNotifier) pack(p *models.EmbeddedContactPoint, data *schema.ResourceData) (interface{}, error) {
	notifier := packCommonNotifierFields(p)
	settings := p.Settings.(map[string]interface{})

	packNotifierStringField(&settings, &notifier, "url", "url")
	packNotifierStringField(&settings, &notifier, "messageType", "message_type")
	packNotifierStringField(&settings, &notifier, "title", "title")
	packNotifierStringField(&settings, &notifier, "description", "description")

	notifier["settings"] = packSettings(p)
	return notifier, nil
}

func (v victorOpsNotifier) unpack(raw interface{}, name string) *models.EmbeddedContactPoint {
	json := raw.(map[string]interface{})
	uid, disableResolve, settings := unpackCommonNotifierFields(json)

	unpackNotifierStringField(&json, &settings, "url", "url")
	unpackNotifierStringField(&json, &settings, "message_type", "messageType")
	unpackNotifierStringField(&json, &settings, "title", "title")
	unpackNotifierStringField(&json, &settings, "description", "description")

	return &models.EmbeddedContactPoint{
		UID:                   uid,
		Name:                  name,
		Type:                  common.Ref(v.meta().typeStr),
		DisableResolveMessage: disableResolve,
		Settings:              settings,
	}
}

type webexNotifier struct{}

var _ notifier = (*webexNotifier)(nil)

func (w webexNotifier) meta() notifierMeta {
	return notifierMeta{
		field:        "webex",
		typeStr:      "webex",
		desc:         "A contact point that sends notifications to Cisco Webex.",
		secureFields: []string{"token"},
	}
}

func (w webexNotifier) schema() *schema.Resource {
	r := commonNotifierResource()
	r.Schema["token"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
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
		Optional:    true,
		Description: "ID of the Webex Teams room where to send the messages.",
	}
	return r
}

func (w webexNotifier) pack(p *models.EmbeddedContactPoint, data *schema.ResourceData) (interface{}, error) {
	notifier := packCommonNotifierFields(p)
	settings := p.Settings.(map[string]interface{})

	packNotifierStringField(&settings, &notifier, "bot_token", "token")
	packNotifierStringField(&settings, &notifier, "api_url", "api_url")
	packNotifierStringField(&settings, &notifier, "message", "message")
	packNotifierStringField(&settings, &notifier, "room_id", "room_id")

	packSecureFields(notifier, getNotifierConfigFromStateWithUID(data, w, p.UID), w.meta().secureFields)

	notifier["settings"] = packSettings(p)
	return notifier, nil
}

func (w webexNotifier) unpack(raw interface{}, name string) *models.EmbeddedContactPoint {
	json := raw.(map[string]interface{})
	uid, disableResolve, settings := unpackCommonNotifierFields(json)

	unpackNotifierStringField(&json, &settings, "token", "bot_token")
	unpackNotifierStringField(&json, &settings, "api_url", "api_url")
	unpackNotifierStringField(&json, &settings, "message", "message")
	unpackNotifierStringField(&json, &settings, "room_id", "room_id")

	return &models.EmbeddedContactPoint{
		UID:                   uid,
		Name:                  name,
		Type:                  common.Ref(w.meta().typeStr),
		DisableResolveMessage: disableResolve,
		Settings:              settings,
	}
}

type webhookNotifier struct{}

var _ notifier = (*webhookNotifier)(nil)

func (w webhookNotifier) meta() notifierMeta {
	return notifierMeta{
		field:        "webhook",
		typeStr:      "webhook",
		desc:         "A contact point that sends notifications to an arbitrary webhook, using the Prometheus webhook format defined here: https://prometheus.io/docs/alerting/latest/configuration/#webhook_config",
		secureFields: []string{"basic_auth_password", "authorization_credentials"},
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
	return r
}

func (w webhookNotifier) pack(p *models.EmbeddedContactPoint, data *schema.ResourceData) (interface{}, error) {
	notifier := packCommonNotifierFields(p)
	settings := p.Settings.(map[string]interface{})

	packNotifierStringField(&settings, &notifier, "url", "url")
	packNotifierStringField(&settings, &notifier, "httpMethod", "http_method")
	packNotifierStringField(&settings, &notifier, "username", "basic_auth_user")
	packNotifierStringField(&settings, &notifier, "password", "basic_auth_password")
	packNotifierStringField(&settings, &notifier, "authorization_scheme", "authorization_scheme")
	packNotifierStringField(&settings, &notifier, "authorization_credentials", "authorization_credentials")
	packNotifierStringField(&settings, &notifier, "message", "message")
	packNotifierStringField(&settings, &notifier, "title", "title")
	if v, ok := settings["maxAlerts"]; ok && v != nil {
		switch typ := v.(type) {
		case int:
			notifier["max_alerts"] = v.(int)
		case float64:
			notifier["max_alerts"] = int(v.(float64))
		case string:
			val, err := strconv.Atoi(typ)
			if err != nil {
				panic(fmt.Errorf("failed to parse value of 'maxAlerts' to integer: %w", err))
			}
			notifier["max_alerts"] = val
		default:
			panic(fmt.Sprintf("unexpected type %T for 'maxAlerts': %v", typ, typ))
		}
		delete(settings, "maxAlerts")
	}

	packSecureFields(notifier, getNotifierConfigFromStateWithUID(data, w, p.UID), w.meta().secureFields)

	notifier["settings"] = packSettings(p)
	return notifier, nil
}

func (w webhookNotifier) unpack(raw interface{}, name string) *models.EmbeddedContactPoint {
	json := raw.(map[string]interface{})
	uid, disableResolve, settings := unpackCommonNotifierFields(json)

	unpackNotifierStringField(&json, &settings, "url", "url")
	unpackNotifierStringField(&json, &settings, "http_method", "httpMethod")
	unpackNotifierStringField(&json, &settings, "basic_auth_user", "username")
	unpackNotifierStringField(&json, &settings, "basic_auth_password", "password")
	unpackNotifierStringField(&json, &settings, "authorization_scheme", "authorization_scheme")
	unpackNotifierStringField(&json, &settings, "authorization_credentials", "authorization_credentials")
	unpackNotifierStringField(&json, &settings, "message", "message")
	unpackNotifierStringField(&json, &settings, "title", "title")
	if v, ok := json["max_alerts"]; ok && v != nil {
		switch typ := v.(type) {
		case int:
			settings["maxAlerts"] = v.(int)
		case float64:
			settings["maxAlerts"] = int(v.(float64))
		default:
			panic(fmt.Sprintf("unexpected type for maxAlerts: %v", typ))
		}
	}

	return &models.EmbeddedContactPoint{
		UID:                   uid,
		Name:                  name,
		Type:                  common.Ref(w.meta().typeStr),
		DisableResolveMessage: disableResolve,
		Settings:              settings,
	}
}

type wecomNotifier struct{}

var _ notifier = (*wecomNotifier)(nil)

func (w wecomNotifier) meta() notifierMeta {
	return notifierMeta{
		field:        "wecom",
		typeStr:      "wecom",
		desc:         "A contact point that sends notifications to WeCom.",
		secureFields: []string{"url", "secret"},
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

func (w wecomNotifier) pack(p *models.EmbeddedContactPoint, data *schema.ResourceData) (interface{}, error) {
	notifier := packCommonNotifierFields(p)
	settings := p.Settings.(map[string]interface{})

	packNotifierStringField(&settings, &notifier, "url", "url")
	packNotifierStringField(&settings, &notifier, "message", "message")
	packNotifierStringField(&settings, &notifier, "title", "title")
	packNotifierStringField(&settings, &notifier, "secret", "secret")
	packNotifierStringField(&settings, &notifier, "corp_id", "corp_id")
	packNotifierStringField(&settings, &notifier, "agent_id", "agent_id")
	packNotifierStringField(&settings, &notifier, "msgtype", "msg_type")
	packNotifierStringField(&settings, &notifier, "touser", "to_user")

	packSecureFields(notifier, getNotifierConfigFromStateWithUID(data, w, p.UID), w.meta().secureFields)

	notifier["settings"] = packSettings(p)
	return notifier, nil
}

func (w wecomNotifier) unpack(raw interface{}, name string) *models.EmbeddedContactPoint {
	json := raw.(map[string]interface{})
	uid, disableResolve, settings := unpackCommonNotifierFields(json)

	unpackNotifierStringField(&json, &settings, "url", "url")
	unpackNotifierStringField(&json, &settings, "message", "message")
	unpackNotifierStringField(&json, &settings, "title", "title")
	unpackNotifierStringField(&json, &settings, "secret", "secret")
	unpackNotifierStringField(&json, &settings, "corp_id", "corp_id")
	unpackNotifierStringField(&json, &settings, "agent_id", "agent_id")
	unpackNotifierStringField(&json, &settings, "msg_type", "msgtype")
	unpackNotifierStringField(&json, &settings, "to_user", "touser")

	return &models.EmbeddedContactPoint{
		UID:                   uid,
		Name:                  name,
		Type:                  common.Ref(w.meta().typeStr),
		DisableResolveMessage: disableResolve,
		Settings:              settings,
	}
}
