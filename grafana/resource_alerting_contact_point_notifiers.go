package grafana

import (
	"strings"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type discordNotifier struct{}

var _ notifier = (*discordNotifier)(nil)

func (d discordNotifier) meta() notifierMeta {
	return notifierMeta{
		typeStr: "discord",
		desc:    "A contact point that sends notifications to Discord.",
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

func (d discordNotifier) pack(p gapi.ContactPoint) interface{} {
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
	return notifier
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

func (e emailNotifier) pack(p gapi.ContactPoint) interface{} {
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
	return notifier
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

type alertmanagerNotifier struct{}

var _ notifier = (*alertmanagerNotifier)(nil)

func (a alertmanagerNotifier) meta() notifierMeta {
	return notifierMeta{
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
		Type:        schema.TypeString,
		Optional:    true,
		Sensitive:   true,
		Description: "The password component of the basic auth credentials to use.",
	}
	return r
}

func (a alertmanagerNotifier) pack(p gapi.ContactPoint) interface{} {
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
	return notifier
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
