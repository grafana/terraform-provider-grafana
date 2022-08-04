package grafana

import (
	"context"
	"fmt"
	"log"
	"strings"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var notifiers = []notifier{
	discordNotifier{},
	emailNotifier{},
}

func ResourceContactPoint() *schema.Resource {
	resource := &schema.Resource{
		Description: `
Manages Grafana Alerting contact points.

* [Official documentation](https://grafana.com/docs/grafana/next/alerting/contact-points)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/alerting_provisioning/#contact-points)
`,
		CreateContext: createContactPoint,
		ReadContext:   readContactPoint,
		UpdateContext: updateContactPoint,
		DeleteContext: deleteContactPoint,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the contact point.",
			},
		},
	}

	for _, n := range notifiers {
		resource.Schema[n.meta().typeStr] = &schema.Schema{
			Type:        schema.TypeList,
			Optional:    true,
			Description: n.meta().desc,
			Elem:        n.schema(),
		}
	}

	return resource
}

func readContactPoint(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi

	uids := unpackUIDs(data.Id())

	points := []gapi.ContactPoint{}
	for _, uid := range uids {
		p, err := client.ContactPoint(uid)
		if err != nil {
			if strings.HasPrefix(err.Error(), "status: 404") {
				log.Printf("[WARN] removing contact point %s from state because it no longer exists in grafana", uid)
				data.SetId("")
				return nil
			}
			return diag.FromErr(err)
		}
		points = append(points, p)
	}

	packContactPoints(points, data)
	data.SetId(packUIDs(uids))

	return nil
}

func createContactPoint(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi

	ps := unpackContactPoints(data)
	uids := make([]string, 0, len(ps))
	for i := range ps {
		uid, err := client.NewContactPoint(&ps[i])
		if err != nil {
			return diag.FromErr(err)
		}
		uids = append(uids, uid)
	}

	data.SetId(packUIDs(uids))
	return readContactPoint(ctx, data, meta)
}

func updateContactPoint(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi

	existingUIDs := unpackUIDs(data.Id())
	ps := unpackContactPoints(data)

	unprocessedUIDs := toUIDSet(existingUIDs)
	newUIDs := make([]string, 0, len(ps))
	for i := range ps {
		delete(unprocessedUIDs, ps[i].UID)
		err := client.UpdateContactPoint(&ps[i])
		if err != nil {
			if strings.HasPrefix(err.Error(), "status: 404") {
				uid, err := client.NewContactPoint(&ps[i])
				newUIDs = append(newUIDs, uid)
				if err != nil {
					return diag.FromErr(err)
				}
				continue
			}
			return diag.FromErr(err)
		}
		newUIDs = append(newUIDs, ps[i].UID)
	}

	// Any UIDs still left in the state that we haven't seen must map to deleted receivers.
	// Delete them on the server and drop them from state.
	for u := range unprocessedUIDs {
		if err := client.DeleteContactPoint(u); err != nil {
			return diag.FromErr(err)
		}
	}

	data.SetId(packUIDs(newUIDs))

	return readContactPoint(ctx, data, meta)
}

func deleteContactPoint(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi

	uids := unpackUIDs(data.Id())
	for _, uid := range uids {
		if err := client.DeleteContactPoint(uid); err != nil {
			return diag.FromErr(err)
		}
	}

	return diag.Diagnostics{}
}

func unpackContactPoints(data *schema.ResourceData) []gapi.ContactPoint {
	result := make([]gapi.ContactPoint, 0)
	name := data.Get("name").(string)
	for _, n := range notifiers {
		if points, ok := data.GetOk(n.meta().typeStr); ok {
			for _, p := range points.([]interface{}) {
				result = append(result, n.unpack(p, name))
			}
		}
	}

	return result
}

func packContactPoints(ps []gapi.ContactPoint, data *schema.ResourceData) {
	pointsPerNotifier := map[notifier][]interface{}{}
	for _, p := range ps {
		data.Set("name", p.Name)

		for _, n := range notifiers {
			if p.Type == n.meta().typeStr {
				packed := n.pack(p)
				pointsPerNotifier[n] = append(pointsPerNotifier[n], packed)
				continue
			}
		}
	}

	for n, pts := range pointsPerNotifier {
		data.Set(n.meta().typeStr, pts)
	}
}

func unpackCommonNotifierFields(raw map[string]interface{}) (string, bool, map[string]interface{}) {
	return raw["uid"].(string), raw["disable_resolve_message"].(bool), raw["settings"].(map[string]interface{})
}

func packCommonNotifierFields(p *gapi.ContactPoint) map[string]interface{} {
	return map[string]interface{}{
		"uid":                     p.UID,
		"disable_resolve_message": p.DisableResolveMessage,
	}
}

func packSettings(p *gapi.ContactPoint) map[string]interface{} {
	settings := map[string]interface{}{}
	for k, v := range p.Settings {
		settings[k] = fmt.Sprintf("%#v", v)
	}
	return settings
}

func commonNotifierResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"uid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The UID of the contact point.",
			},
			"disable_resolve_message": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether to disable sending resolve messages.",
			},
			"settings": {
				Type:        schema.TypeMap,
				Optional:    true,
				Sensitive:   true,
				Default:     map[string]interface{}{},
				Description: "Additional custom properties to attach to the notifier.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

const UIDSeparator = ";"

func packUIDs(uids []string) string {
	return strings.Join(uids, UIDSeparator)
}

func unpackUIDs(packed string) []string {
	return strings.Split(packed, UIDSeparator)
}

func toUIDSet(uids []string) map[string]bool {
	set := map[string]bool{}
	for _, uid := range uids {
		set[uid] = true
	}
	return set
}

type notifier interface {
	meta() notifierMeta
	schema() *schema.Resource
	pack(p gapi.ContactPoint) interface{}
	unpack(raw interface{}, name string) gapi.ContactPoint
}

type notifierMeta struct {
	typeStr string
	desc    string
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
		Type:                  "email",
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
		Type:                  "discord",
		DisableResolveMessage: disableResolve,
		Settings:              settings,
	}
}
