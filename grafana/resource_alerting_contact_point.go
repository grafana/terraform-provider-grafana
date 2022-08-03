package grafana

import (
	"context"
	"log"
	"strings"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceContactPoint() *schema.Resource {
	return &schema.Resource{
		Description: `
Manages Grafana Alerting contact points.

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
			"custom": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "An unspecified, customizable contact point.",
				Elem:        customContactResource(),
			},
			"email": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "The email contact point.",
				Elem:        emailContactResource(),
			},
		},
	}
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
	for _, p := range ps {
		uid, err := client.NewContactPoint(&p)
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

	ps := unpackContactPoints(data)
	for _, p := range ps {
		if err := client.UpdateContactPoint(&p); err != nil {
			return diag.FromErr(err)
		}
	}

	return diag.Diagnostics{}
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
	if custom, ok := data.GetOk("custom"); ok {
		for _, p := range custom.([]interface{}) {
			result = append(result, unpackCustomNotifier(p, name))
		}
	}
	/*if email, ok := data.GetOk("email"); ok {

	}*/

	return result
	/*typ := data.Get("type").(string)
	settings := data.Get("settings").(map[string]interface{})
	if settings == nil {
		settings = map[string]interface{}{}
	}
	if e, ok := data.GetOk("email"); ok {
		typ = "email"
		for _, notif := range e.(*schema.Set).List() {
			// TODO: This is horrible because we're stuffing potentially many notifiers into one.
			// TODO: Answer the "one vs many" notifiers per contact point question and fix accordingly.
			// TODO: this is just for testing
			n := notif.(map[string]interface{})
			adds := n["addresses"].([]interface{})
			addStrs := make([]string, len(adds))
			for i, a := range adds {
				addStrs[i] = a.(string)
			}
			settings["addresses"] = strings.Join(addStrs, ";")
			settings["singleEmail"] = n["single_email"].(bool)
			settings["message"] = n["message"].(string)
			settings["subject"] = n["subject"].(string)
			// TODO: merge in shared settings field too
		}
	}
	return gapi.ContactPoint{
		UID:                   data.Id(),
		Name:                  data.Get("name").(string),
		DisableResolveMessage: data.Get("disable_resolve_message").(bool),
		Type:                  typ,
		Settings:              settings,
	}*/
}

func packContactPoints(ps []gapi.ContactPoint, data *schema.ResourceData) {
	points := map[string][]interface{}{}
	for _, p := range ps {
		data.Set("name", p.Name)

		if p.Type == "email TODO" {

		} else {
			point := packCustomNotifier(p)
			points["custom"] = append(points["custom"], point)
		}
	}
	data.Set("custom", points["custom"])

	/*
		data.Set("type", p.Type)
		data.Set("disable_resolve_message", p.DisableResolveMessage)
		if p.Type == "email" {
			emailData := map[string]interface{}{}
			if v, ok := p.Settings["addresses"]; ok {
				addrs := strings.Split(v.(string), ";")
				for i, a := range addrs {
					addrs[i] = strings.TrimSpace(a)
				}
				emailData["addresses"] = addrs
			}
			if v, ok := p.Settings["singleEmail"]; ok {
				emailData["single_email"] = v.(bool)
			}
			if v, ok := p.Settings["message"]; ok {
				emailData["message"] = v.(string)
			}
			if v, ok := p.Settings["subject"]; ok {
				emailData["subject"] = v.(string)
			}
			data.Set("email", []interface{}{emailData})
		} else {
			data.Set("settings", p.Settings)
		}*/
}

func emailContactResource() *schema.Resource {
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
		Description: "TODO",
	}
	r.Schema["subject"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Default:     "",
		Description: "TODO",
	}
	return r
}

func unpackCustomNotifier(raw interface{}, name string) gapi.ContactPoint {
	json := raw.(map[string]interface{})
	uid, disableResolve, settings := unpackCommonNotifierFields(json)

	return gapi.ContactPoint{
		UID:                   uid,
		Name:                  name,
		Type:                  json["type"].(string),
		DisableResolveMessage: disableResolve,
		Settings:              settings,
	}
}

func packCustomNotifier(p gapi.ContactPoint) interface{} {
	notifier := packCommonNotifierFields(&p)
	notifier["type"] = p.Type
	return notifier
}

func customContactResource() *schema.Resource {
	r := commonNotifierResource()
	r.Schema["type"] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Description: "The type of the contact point.",
	}
	return r
}

func unpackCommonNotifierFields(raw map[string]interface{}) (string, bool, map[string]interface{}) {
	return raw["uid"].(string), raw["disable_resolve_message"].(bool), raw["settings"].(map[string]interface{})
}

func packCommonNotifierFields(p *gapi.ContactPoint) map[string]interface{} {
	settings := map[string]interface{}{}
	for k, v := range p.Settings {
		settings[k] = v
	}
	return map[string]interface{}{
		"uid":                     p.UID,
		"disable_resolve_message": p.DisableResolveMessage,
		"settings":                settings,
	}
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
