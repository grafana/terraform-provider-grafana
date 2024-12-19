package grafana

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/go-openapi/runtime"
	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/client/provisioning"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
)

var (
	provenanceDisabled = "disabled"
	notifiers          = []notifier{
		alertmanagerNotifier{},
		dingDingNotifier{},
		discordNotifier{},
		emailNotifier{},
		googleChatNotifier{},
		kafkaNotifier{},
		lineNotifier{},
		mqttNotifier{},
		oncallNotifier{},
		opsGenieNotifier{},
		pagerDutyNotifier{},
		pushoverNotifier{},
		sensugoNotifier{},
		slackNotifier{},
		snsNotifier{},
		teamsNotifier{},
		telegramNotifier{},
		threemaNotifier{},
		victorOpsNotifier{},
		webexNotifier{},
		webhookNotifier{},
		wecomNotifier{},
	}
)

func resourceContactPoint() *common.Resource {
	resource := &schema.Resource{
		Description: `
Manages Grafana Alerting contact points.

* [Official documentation](https://grafana.com/docs/grafana/latest/alerting/set-up/provision-alerting-resources/terraform-provisioning/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/alerting_provisioning/#contact-points)

This resource requires Grafana 9.1.0 or later.
`,
		CreateContext: common.WithAlertingMutex[schema.CreateContextFunc](updateContactPoint),
		ReadContext:   readContactPoint,
		UpdateContext: common.WithAlertingMutex[schema.UpdateContextFunc](updateContactPoint),
		DeleteContext: common.WithAlertingMutex[schema.DeleteContextFunc](deleteContactPoint),

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"org_id": orgIDAttribute(),
			"name": {
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
				Description: "The name of the contact point.",
			},
			"disable_provenance": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				ForceNew:    true, // Can't modify provenance on contact points
				Description: "Allow modifying the contact point from other sources than Terraform or the Grafana API.",
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
			Type:         schema.TypeSet,
			Optional:     true,
			Description:  n.meta().desc,
			Elem:         n.schema(),
			AtLeastOneOf: notifierFields,
		}
	}

	return common.NewLegacySDKResource(
		common.CategoryAlerting,
		"grafana_contact_point",
		orgResourceIDString("name"),
		resource,
	).WithLister(listerFunctionOrgResource(listContactPoints))
}

func listContactPoints(ctx context.Context, client *goapi.GrafanaHTTPAPI, orgID int64) ([]string, error) {
	idMap := map[string]bool{}
	// Retry if the API returns 500 because it may be that the alertmanager is not ready in the org yet.
	// The alertmanager is provisioned asynchronously when the org is created.
	if err := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		resp, err := client.Provisioning.GetContactpoints(provisioning.NewGetContactpointsParams())
		if err != nil {
			if orgID > 1 && (err.(*runtime.APIError).IsCode(500) || err.(*runtime.APIError).IsCode(403)) {
				return retry.RetryableError(err)
			}
			return retry.NonRetryableError(err)
		}

		for _, contactPoint := range resp.Payload {
			idMap[MakeOrgResourceID(orgID, contactPoint.Name)] = true
		}
		return nil
	}); err != nil {
		return nil, err
	}

	var ids []string
	for id := range idMap {
		ids = append(ids, id)
	}

	return ids, nil
}

func readContactPoint(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID, name := OAPIClientFromExistingOrgResource(meta, data.Id())

	resp, err := client.Provisioning.GetContactpoints(provisioning.NewGetContactpointsParams())
	if err != nil {
		return diag.FromErr(err)
	}
	var points []*models.EmbeddedContactPoint
	for _, p := range resp.Payload {
		if p.Name == name {
			points = append(points, p)
		}
	}
	if len(points) == 0 {
		return common.WarnMissing("contact point", data)
	}

	if err := packContactPoints(points, data); err != nil {
		return diag.FromErr(err)
	}
	data.Set("org_id", strconv.FormatInt(orgID, 10))
	data.SetId(MakeOrgResourceID(orgID, points[0].Name))

	return nil
}

func updateContactPoint(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID := OAPIClientFromNewOrgResource(meta, data)

	ps := unpackContactPoints(data)

	// Update + create notifiers
	for i := range ps {
		p := ps[i]

		if p.deleted { // We'll handle deletions later
			continue
		}

		var uid string
		if uid = p.tfState["uid"].(string); uid != "" {
			// If the contact point already has a UID, update it.
			params := provisioning.NewPutContactpointParams().WithUID(uid).WithBody(p.gfState)
			if data.Get("disable_provenance").(bool) {
				params.SetXDisableProvenance(&provenanceDisabled)
			}
			if _, err := client.Provisioning.PutContactpoint(params); err != nil {
				return diag.FromErr(err)
			}
		} else {
			// If the contact point does not have a UID, create it.
			// Retry if the API returns 500 because it may be that the alertmanager is not ready in the org yet.
			// The alertmanager is provisioned asynchronously when the org is created.
			err := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
				params := provisioning.NewPostContactpointsParams().WithBody(p.gfState)
				if data.Get("disable_provenance").(bool) {
					params.SetXDisableProvenance(&provenanceDisabled)
				}
				resp, err := client.Provisioning.PostContactpoints(params)
				if orgID > 1 && err != nil && err.(*runtime.APIError).IsCode(500) {
					return retry.RetryableError(err)
				} else if err != nil {
					return retry.NonRetryableError(err)
				}
				uid = resp.Payload.UID
				return nil
			})
			if err != nil {
				return diag.FromErr(err)
			}
		}

		// Since this is a new resource, the proposed state won't have a UID.
		// We need the UID so that we can later associate it with the config returned in the api response.
		ps[i].tfState["uid"] = uid
	}

	// Delete notifiers
	for _, p := range ps {
		if !p.deleted {
			continue
		}
		uid := p.tfState["uid"].(string)
		// If the contact point is not in the proposed state, delete it.
		if _, err := client.Provisioning.DeleteContactpoints(uid); err != nil {
			return diag.Errorf("failed to remove contact point notifier with UID %s from contact point %s: %v", uid, data.Id(), err)
		}
		continue
	}

	data.SetId(MakeOrgResourceID(orgID, data.Get("name").(string)))
	return readContactPoint(ctx, data, meta)
}

func deleteContactPoint(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, name := OAPIClientFromExistingOrgResource(meta, data.Id())

	resp, err := client.Provisioning.GetContactpoints(provisioning.NewGetContactpointsParams().WithName(&name))
	if err, shouldReturn := common.CheckReadError("contact point", data, err); shouldReturn {
		return err
	}

	for _, cp := range resp.Payload {
		if _, err := client.Provisioning.DeleteContactpoints(cp.UID); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

// unpackContactPoints unpacks the contact points from the Terraform state.
// It returns a slice of statePairs, which contain the Terraform state and the Grafana state for each contact point.
// It also tracks receivers that should be deleted. There are two cases where a receiver should be deleted:
// - The receiver is present in the "new" part of the diff, but all fields are zeroed out (except UID).
// - The receiver is present in the "old" part of the diff, but not in the "new" part.
func unpackContactPoints(data *schema.ResourceData) []statePair {
	result := make([]statePair, 0)
	name := data.Get("name").(string)
	for _, n := range notifiers {
		oldPoints, newPoints := data.GetChange(n.meta().field)
		oldPointsList := oldPoints.(*schema.Set).List()
		newPointsList := newPoints.(*schema.Set).List()
		if len(oldPointsList) == 0 && len(newPointsList) == 0 {
			continue
		}
		processedUIDs := map[string]bool{}
		for _, p := range newPointsList {
			// Checking if the point/receiver should be deleted
			// If all fields are zeroed out, except UID, then the receiver should be deleted
			deleted := false
			pointMap := p.(map[string]interface{})
			if uid, ok := pointMap["uid"]; ok && uid != "" {
				deleted = true
				processedUIDs[uid.(string)] = true
			}
			for fieldName, fieldSchema := range n.schema().Schema {
				if !fieldSchema.Computed && fieldSchema.Required && !reflect.ValueOf(pointMap[fieldName]).IsZero() {
					deleted = false
					break
				}
			}

			// Add the point/receiver to the result
			// If it's not deleted, it will either be created or updated
			result = append(result, statePair{
				tfState: pointMap,
				gfState: unpackPointConfig(n, p, name),
				deleted: deleted,
			})
		}
		// Checking if the point/receiver should be deleted
		// If the point is not present in the "new" part of the diff, but is present in the "old" part, then the receiver should be deleted
		for _, p := range oldPointsList {
			pointMap := p.(map[string]interface{})
			if uid, ok := pointMap["uid"]; ok && uid != "" && !processedUIDs[uid.(string)] {
				result = append(result, statePair{
					tfState: p.(map[string]interface{}),
					gfState: nil,
					deleted: true,
				})
			}
		}
	}

	return result
}

func unpackPointConfig(n notifier, data interface{}, name string) *models.EmbeddedContactPoint {
	pt := n.unpack(data, name)
	settings := pt.Settings.(map[string]interface{})
	// Treat settings like `omitempty`. Workaround for versions affected by https://github.com/grafana/grafana/issues/55139
	for k, v := range settings {
		if v == "" {
			delete(settings, k)
		}
	}
	return pt
}

func packContactPoints(ps []*models.EmbeddedContactPoint, data *schema.ResourceData) error {
	pointsPerNotifier := map[notifier][]interface{}{}
	disableProvenance := true
	for _, p := range ps {
		data.Set("name", p.Name)
		if p.Provenance != "" {
			disableProvenance = false
		}

		for _, n := range notifiers {
			if *p.Type == n.meta().typeStr {
				packed, err := n.pack(p, data)
				if err != nil {
					return err
				}
				pointsPerNotifier[n] = append(pointsPerNotifier[n], packed)
				continue
			}
		}
	}
	data.Set("disable_provenance", disableProvenance)

	for n, pts := range pointsPerNotifier {
		data.Set(n.meta().field, pts)
	}

	return nil
}

func unpackCommonNotifierFields(raw map[string]interface{}) (string, bool, map[string]interface{}) {
	return raw["uid"].(string), raw["disable_resolve_message"].(bool), raw["settings"].(map[string]interface{})
}

func packCommonNotifierFields(p *models.EmbeddedContactPoint) map[string]interface{} {
	return map[string]interface{}{
		"uid":                     p.UID,
		"disable_resolve_message": p.DisableResolveMessage,
	}
}

func packSettings(p *models.EmbeddedContactPoint) map[string]interface{} {
	settings := map[string]interface{}{}
	for k, v := range p.Settings.(map[string]interface{}) {
		settings[k] = fmt.Sprintf("%s", v)
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

type notifier interface {
	meta() notifierMeta
	schema() *schema.Resource
	pack(p *models.EmbeddedContactPoint, data *schema.ResourceData) (interface{}, error)
	unpack(raw interface{}, name string) *models.EmbeddedContactPoint
}

type notifierMeta struct {
	field        string
	typeStr      string
	desc         string
	secureFields []string
}

type statePair struct {
	tfState map[string]interface{}
	gfState *models.EmbeddedContactPoint
	deleted bool
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
		for _, pt := range points.(*schema.Set).List() {
			config := pt.(map[string]interface{})
			if config["uid"] == uid {
				return config
			}
		}
	}

	return nil
}
