package grafana

import (
	"context"
	"fmt"
	"log"
	"strings"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/grafana/terraform-provider-grafana/internal/common"
)

var notifiers = []notifier{
	alertmanagerNotifier{},
	dingDingNotifier{},
	discordNotifier{},
	emailNotifier{},
	googleChatNotifier{},
	kafkaNotifier{},
	lineNotifier{},
	oncallNotifier{},
	opsGenieNotifier{},
	pagerDutyNotifier{},
	pushoverNotifier{},
	sensugoNotifier{},
	slackNotifier{},
	teamsNotifier{},
	telegramNotifier{},
	threemaNotifier{},
	victorOpsNotifier{},
	webexNotifier{},
	webhookNotifier{},
	wecomNotifier{},
}

func ResourceContactPoint() *schema.Resource {
	resource := &schema.Resource{
		Description: `
Manages Grafana Alerting contact points.

* [Official documentation](https://grafana.com/docs/grafana/next/alerting/fundamentals/contact-points/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/alerting_provisioning/#contact-points)

This resource requires Grafana 9.1.0 or later.
`,
		CreateContext: createContactPoint,
		ReadContext:   readContactPoint,
		UpdateContext: updateContactPoint,
		DeleteContext: deleteContactPoint,

		Importer: &schema.ResourceImporter{
			StateContext: importContactPoint,
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

	// Build list of available notifier fields, at least one has to be specified
	notifierFields := make([]string, len(notifiers))
	for i, n := range notifiers {
		notifierFields[i] = n.meta().field
	}

	for _, n := range notifiers {
		resource.Schema[n.meta().field] = &schema.Schema{
			Type:         schema.TypeList,
			Optional:     true,
			Description:  n.meta().desc,
			Elem:         n.schema(),
			AtLeastOneOf: notifierFields,
		}
	}

	return resource
}

func importContactPoint(ctx context.Context, data *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	name := data.Id()
	client := meta.(*common.Client).GrafanaAPI

	ps, err := client.ContactPointsByName(name)
	if err != nil {
		return nil, err
	}

	if len(ps) == 0 {
		return nil, fmt.Errorf("no contact points with the given name were found to import")
	}

	uids := make([]string, 0, len(ps))
	for _, p := range ps {
		uids = append(uids, p.UID)
	}

	data.SetId(packUIDs(uids))
	return []*schema.ResourceData{data}, nil
}

func readContactPoint(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).GrafanaAPI

	uidsToFetch := unpackUIDs(data.Id())

	points := []gapi.ContactPoint{}
	for _, uid := range uidsToFetch {
		p, err := client.ContactPoint(uid)
		if err != nil {
			if strings.HasPrefix(err.Error(), "status: 404") || strings.Contains(err.Error(), "not found") {
				log.Printf("[WARN] removing contact point %s from state because it no longer exists in grafana", uid)
				continue
			}
			return diag.FromErr(err)
		}
		points = append(points, p)
	}

	err := packContactPoints(points, data)
	if err != nil {
		return diag.FromErr(err)
	}
	uids := make([]string, 0, len(points))
	for _, p := range points {
		uids = append(uids, p.UID)
	}
	data.SetId(packUIDs(uids))

	return nil
}

func createContactPoint(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	lock := &meta.(*common.Client).AlertingMutex
	client := meta.(*common.Client).GrafanaAPI

	ps := unpackContactPoints(data)
	uids := make([]string, 0, len(ps))

	lock.Lock()
	defer lock.Unlock()
	for i := range ps {
		p := ps[i]
		uid, err := client.NewContactPoint(&p.gfState)
		if err != nil {
			return diag.FromErr(err)
		}
		uids = append(uids, uid)

		// Since this is a new resource, the proposed state won't have a UID.
		// We need the UID so that we can later associate it with the config returned in the api response.
		p.tfState["uid"] = uid
	}

	data.SetId(packUIDs(uids))
	return readContactPoint(ctx, data, meta)
}

func updateContactPoint(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	lock := &meta.(*common.Client).AlertingMutex
	client := meta.(*common.Client).GrafanaAPI

	existingUIDs := unpackUIDs(data.Id())
	ps := unpackContactPoints(data)

	unprocessedUIDs := toUIDSet(existingUIDs)
	newUIDs := make([]string, 0, len(ps))
	lock.Lock()
	defer lock.Unlock()
	for i := range ps {
		p := ps[i].gfState
		delete(unprocessedUIDs, p.UID)
		err := client.UpdateContactPoint(&p)
		if err != nil {
			if strings.HasPrefix(err.Error(), "status: 404") {
				uid, err := client.NewContactPoint(&p)
				newUIDs = append(newUIDs, uid)
				if err != nil {
					return diag.FromErr(err)
				}
				continue
			}
			return diag.FromErr(err)
		}
		newUIDs = append(newUIDs, p.UID)
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
	lock := &meta.(*common.Client).AlertingMutex
	client := meta.(*common.Client).GrafanaAPI

	uids := unpackUIDs(data.Id())

	lock.Lock()
	defer lock.Unlock()
	for _, uid := range uids {
		if err := client.DeleteContactPoint(uid); err != nil {
			return diag.FromErr(err)
		}
	}

	return diag.Diagnostics{}
}

func unpackContactPoints(data *schema.ResourceData) []statePair {
	result := make([]statePair, 0)
	name := data.Get("name").(string)
	for _, n := range notifiers {
		if points, ok := data.GetOk(n.meta().field); ok {
			for _, p := range points.([]interface{}) {
				result = append(result, statePair{
					tfState: p.(map[string]interface{}),
					gfState: unpackPointConfig(n, p, name),
				})
			}
		}
	}

	return result
}

func unpackPointConfig(n notifier, data interface{}, name string) gapi.ContactPoint {
	pt := n.unpack(data, name)
	// Treat settings like `omitempty`. Workaround for versions affected by https://github.com/grafana/grafana/issues/55139
	for k, v := range pt.Settings {
		if v == "" {
			delete(pt.Settings, k)
		}
	}
	return pt
}

func packContactPoints(ps []gapi.ContactPoint, data *schema.ResourceData) error {
	pointsPerNotifier := map[notifier][]interface{}{}
	for _, p := range ps {
		data.Set("name", p.Name)

		for _, n := range notifiers {
			if p.Type == n.meta().typeStr {
				packed, err := n.pack(p, data)
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
	pack(p gapi.ContactPoint, data *schema.ResourceData) (interface{}, error)
	unpack(raw interface{}, name string) gapi.ContactPoint
}

type notifierMeta struct {
	field        string
	typeStr      string
	desc         string
	secureFields []string
}

type statePair struct {
	tfState map[string]interface{}
	gfState gapi.ContactPoint
}

func packNotifierStringField(gfSettings, tfSettings *map[string]interface{}, gfKey, tfKey string) {
	if v, ok := (*gfSettings)[gfKey]; ok && v != nil {
		(*tfSettings)[tfKey] = v.(string)
		delete(*gfSettings, gfKey)
	}
}

func packSecureFields(tfSettings, state map[string]interface{}, secureFields []string) {
	for _, tfKey := range secureFields {
		if v, ok := state[tfKey]; ok && v != nil {
			tfSettings[tfKey] = v.(string)
		}
	}
}

func unpackNotifierStringField(tfSettings, gfSettings *map[string]interface{}, tfKey, gfKey string) {
	if v, ok := (*tfSettings)[tfKey]; ok && v != nil {
		(*gfSettings)[gfKey] = v.(string)
	}
}

func getNotifierConfigFromStateWithUID(data *schema.ResourceData, n notifier, uid string) map[string]interface{} {
	if points, ok := data.GetOk(n.meta().field); ok {
		for _, pt := range points.([]interface{}) {
			config := pt.(map[string]interface{})
			if config["uid"] == uid {
				return config
			}
		}
	}

	return nil
}
