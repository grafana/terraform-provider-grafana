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
				Required:    true,
				Description: "The type of the contact point.",
			},
			"email": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "The email contact points.",
				Elem:        emailResource(),
			},
			"settings": {
				Type:        schema.TypeMap,
				Required:    true,
				Sensitive:   true,
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
	data.Set("settings", p.Settings)
	data.Set("disable_resolve_message", p.DisableResolveMessage)

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
	return gapi.ContactPoint{
		UID:                   data.Id(),
		Name:                  data.Get("name").(string),
		Type:                  data.Get("type").(string),
		Settings:              data.Get("settings").(map[string]interface{}),
		DisableResolveMessage: data.Get("disable_resolve_message").(bool),
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
	r.Schema["singleEmail"] = &schema.Schema{
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
