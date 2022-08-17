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
	alertmanagerNotifier{},
	dingDingNotifier{},
	discordNotifier{},
	emailNotifier{},
	googleChatNotifier{},
	kafkaNotifier{},
	opsGenieNotifier{},
	pagerDutyNotifier{},
	pushoverNotifier{},
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
		resource.Schema[n.meta().field] = &schema.Schema{
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

	err := packContactPoints(points, data)
	if err != nil {
		return diag.FromErr(err)
	}
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
		if points, ok := data.GetOk(n.meta().field); ok {
			for _, p := range points.([]interface{}) {
				result = append(result, n.unpack(p, name))
			}
		}
	}

	return result
}

func packContactPoints(ps []gapi.ContactPoint, data *schema.ResourceData) error {
	pointsPerNotifier := map[notifier][]interface{}{}
	for _, p := range ps {
		data.Set("name", p.Name)

		for _, n := range notifiers {
			if p.Type == n.meta().typeStr {
				packed, err := n.pack(p)
				if err != nil {
					return err
				}
				pointsPerNotifier[n] = append(pointsPerNotifier[n], packed)
				continue
			}
		}
	}

	for n, pts := range pointsPerNotifier {
		data.Set(n.meta().field, pts)
	}

	return nil
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

const RedactedContactPointField = "[REDACTED]"

func redactedContactPointDiffSuppress(k, oldValue, newValue string, d *schema.ResourceData) bool {
	if oldValue == RedactedContactPointField {
		return true
	}
	return false
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
	pack(p gapi.ContactPoint) (interface{}, error)
	unpack(raw interface{}, name string) gapi.ContactPoint
}

type notifierMeta struct {
	field   string
	typeStr string
	desc    string
}
