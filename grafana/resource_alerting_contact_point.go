package grafana

import (
	"context"
	"log"
	"strings"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var notifiers = []notifier{
	customNotifier{},
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

	existingUIDs := unpackUIDs(data.Id())
	ps := unpackContactPoints(data)

	unprocessedUIDs := toUIDSet(existingUIDs)
	newUIDs := make([]string, 0, len(ps))
	for _, p := range ps {
		delete(unprocessedUIDs, p.UID)
		err := client.UpdateContactPoint(&p)
		if err == nil {
			newUIDs = append(newUIDs, p.UID)
		} else if strings.HasPrefix(err.Error(), "status: 404") {
			uid, err := client.NewContactPoint(&p)
			newUIDs = append(newUIDs, uid)
			if err != nil {
				return diag.FromErr(err)
			}
		} else {
			return diag.FromErr(err)
		}
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
			packed := n.pack(p)
			pointsPerNotifier[n] = append(pointsPerNotifier[n], packed)
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
	settings := map[string]interface{}{}
	// TODO: Terraform expects values to be strings. Convert here.
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
	}
	if v, ok := p.Settings["singleEmail"]; ok && v != nil {
		notifier["single_email"] = v.(bool)
	}
	if v, ok := p.Settings["message"]; ok && v != nil {
		notifier["message"] = v.(string)
	}
	if v, ok := p.Settings["subject"]; ok && v != nil {
		notifier["subject"] = v.(string)
	}
	return notifier
}

func (e emailNotifier) unpack(raw interface{}, name string) gapi.ContactPoint {
	json := raw.(map[string]interface{})
	uid, disableResolve, settings := unpackCommonNotifierFields(json)

	addrs := unpackAddrs(json["addresses"].([]string))
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

func unpackAddrs(addrs []string) string {
	return strings.Join(addrs, addrSeparator)
}

type customNotifier struct{}

var _ notifier = (*customNotifier)(nil)

func (c customNotifier) meta() notifierMeta {
	return notifierMeta{
		typeStr: "custom",
		desc:    "An unspecified, customizable contact point.",
	}
}

func (c customNotifier) schema() *schema.Resource {
	r := commonNotifierResource()
	r.Schema["type"] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Description: "The type of the contact point.",
	}
	return r
}

func (c customNotifier) pack(p gapi.ContactPoint) interface{} {
	notifier := packCommonNotifierFields(&p)
	notifier["type"] = p.Type
	return notifier
}

func (c customNotifier) unpack(raw interface{}, name string) gapi.ContactPoint {
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
