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
			"type": {
				Type:        schema.TypeString,
				Optional:    true, // TODO changed to optional
				Default:     "",
				Description: "The type of the contact point.",
			},
			"email": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "The email contact point.",
				Elem:        emailResource(),
			},
			"settings": {
				Type:        schema.TypeMap,
				Optional:    true, // TODO changed to optional
				Sensitive:   true,
				Default:     map[string]interface{}{},
				Description: "Settings fields for the contact point.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"disable_resolve_message": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether to disable sending resolve messages.",
			},
		},
	}
}

func readContactPoint(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi

	uid := data.Id()
	p, err := client.ContactPoint(uid)
	if err != nil {
		if strings.HasPrefix(err.Error(), "status: 404") {
			log.Printf("[WARN] removing contact point %s from state because it no longer exists in grafana", uid)
			data.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	data.SetId(p.UID)
	data.Set("name", p.Name)
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
	}

	return nil
}

func createContactPoint(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi

	p := contactPointFromResourceData(data)
	uid, err := client.NewContactPoint(&p)
	if err != nil {
		return diag.FromErr(err)
	}

	data.SetId(uid)
	return readContactPoint(ctx, data, meta)
}

func updateContactPoint(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi

	p := contactPointFromResourceData(data)
	if err := client.UpdateContactPoint(&p); err != nil {
		return diag.FromErr(err)
	}

	return diag.Diagnostics{}
}

func deleteContactPoint(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi

	uid := data.Id()
	if err := client.DeleteContactPoint(uid); err != nil {
		return diag.FromErr(err)
	}

	return diag.Diagnostics{}
}

func contactPointFromResourceData(data *schema.ResourceData) gapi.ContactPoint {
	typ := data.Get("type").(string)
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
	}
}

func emailResource() *schema.Resource {
	r := baseChannelResource()
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

func baseChannelResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"settings": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "Additional custom properties to attach to the notifier.",
				Sensitive:   true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}
