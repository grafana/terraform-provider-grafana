package grafana

import (
	"context"
	"fmt"
	"log"
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

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
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
			if err.(runtime.ClientResponseStatus).IsCode(500) || err.(runtime.ClientResponseStatus).IsCode(403) {
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

func readContactPoint(ctx context.Context, data *schema.ResourceData, meta any) diag.Diagnostics {
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

func updateContactPoint(ctx context.Context, data *schema.ResourceData, meta any) diag.Diagnostics {
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
				if err != nil {
					if err.(runtime.ClientResponseStatus).IsCode(500) {
						return retry.RetryableError(err)
					}
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

func deleteContactPoint(ctx context.Context, data *schema.ResourceData, meta any) diag.Diagnostics {
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
			pointMap := p.(map[string]any)
			if uid, ok := pointMap["uid"]; ok && uid != "" {
				deleted = true
				processedUIDs[uid.(string)] = true
			}
			if nonEmptyNotifier(n, pointMap) {
				deleted = false
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
			pointMap := p.(map[string]any)
			if uid, ok := pointMap["uid"]; ok && uid != "" && !processedUIDs[uid.(string)] {
				result = append(result, statePair{
					tfState: p.(map[string]any),
					gfState: nil,
					deleted: true,
				})
			}
		}
	}

	return result
}

type HasData interface {
	HasData(data map[string]any) bool
}

func nonEmptyNotifier(n notifier, data map[string]any) bool {
	if customEmpty, ok := n.(HasData); ok {
		return customEmpty.HasData(data)
	}
	for fieldName, fieldSchema := range n.schema().Schema {
		// We only check required fields to determine if the point is zeroed. This is because some optional fields,
		// such as nested schema.Set can be troublesome to check for zero values. Notifiers that lack required fields
		// (ex. wecom) can define a custom IsEmpty method to handle this.
		if !fieldSchema.Computed && fieldSchema.Required && !reflect.ValueOf(data[fieldName]).IsZero() {
			return true
		}
	}
	return false
}

func unpackPointConfig(n notifier, data any, name string) *models.EmbeddedContactPoint {
	pt := n.unpack(data, name)
	settings := pt.Settings.(map[string]any)
	// Treat settings like `omitempty`. Workaround for versions affected by https://github.com/grafana/grafana/issues/55139
	for k, v := range settings {
		if v == "" {
			delete(settings, k)
		}
	}
	return pt
}

func packContactPoints(ps []*models.EmbeddedContactPoint, data *schema.ResourceData) error {
	pointsPerNotifier := map[notifier][]any{}
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

	for _, n := range notifiers {
		data.Set(n.meta().field, pointsPerNotifier[n])
	}

	return nil
}

func unpackCommonNotifierFields(raw map[string]any) (string, bool, map[string]any) {
	return raw["uid"].(string), raw["disable_resolve_message"].(bool), raw["settings"].(map[string]any)
}

func packCommonNotifierFields(p *models.EmbeddedContactPoint) map[string]any {
	return map[string]any{
		"uid":                     p.UID,
		"disable_resolve_message": p.DisableResolveMessage,
	}
}

func packSettings(p *models.EmbeddedContactPoint) map[string]any {
	settings := map[string]any{}
	for k, v := range p.Settings.(map[string]any) {
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
				Default:     map[string]any{},
				Description: "Additional custom properties to attach to the notifier.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

// fieldMapper is a helper struct to map fields that differ between Terraform and Grafana schema. Such as field keys or type conversions.
type fieldMapper struct {
	newKey        string
	packValFunc   func(any) any
	unpackValFunc func(any) any
}

func newFieldMapper(newKey string, packValFunc, unpackValFunc func(any) any) fieldMapper {
	return fieldMapper{
		newKey:        newKey,
		packValFunc:   packValFunc,
		unpackValFunc: unpackValFunc,
	}
}

// newKeyMapper is a fieldMapper that only changes the key name in the schema.
func newKeyMapper(newKey string) fieldMapper {
	return fieldMapper{
		newKey: newKey,
	}
}

// valueAsInt is a fieldMapper function that converts a value to an integer.
func valueAsInt(value any) any {
	switch typ := value.(type) {
	case int:
		return typ
	case float64:
		return int(typ)
	case string:
		val, err := strconv.Atoi(typ)
		if err != nil {
			panic(fmt.Errorf("failed to parse value to integer: %w", err))
		}
		return val
	default:
		panic(fmt.Sprintf("unexpected type %T: %v", typ, typ))
	}
}

// unpackNotifier takes the Terraform-style settings and unpacks them into the grafana-style settings. It handles:
//   - Applying any transformation functions defined in fieldMapping to the keys and values in gfSettings. This is necessary
//     because some field names differ between Terraform and Grafana, and some values need to be transformed (e.g., converting a string to an integer).
//   - Flattening the "settings" field created by TF when unpacking the resource schema. This contains any unknown fields
//     not present in the resource schema.
func unpackNotifier(tfSettings map[string]any, name string, n notifier) *models.EmbeddedContactPoint {
	gfSettings := unpackFields(tfSettings, "", n.schema().Schema, n.meta().fieldMapper)

	// UID, disable_resolve_message, and leftover "settings" are part of the schema so are currently unpacked into gfSettings.
	// However, they are not part of the settings schema in Grafana, so we extract them.
	uid := tfSettings["uid"].(string)
	delete(gfSettings, "uid")

	disableResolve := tfSettings["disable_resolve_message"].(bool)
	delete(gfSettings, "disable_resolve_message")

	if settings, ok := gfSettings["settings"].(map[string]any); ok {
		for k, v := range settings {
			gfSettings[k] = v
		}
	}
	delete(gfSettings, "settings")

	// Treat settings like `omitempty`. Workaround for versions affected by https://github.com/grafana/grafana/issues/55139
	for k, v := range gfSettings {
		if v == "" {
			delete(gfSettings, k)
		}
	}

	return &models.EmbeddedContactPoint{
		UID:                   uid,
		Name:                  name,
		Type:                  common.Ref(n.meta().typeStr),
		DisableResolveMessage: disableResolve,
		Settings:              gfSettings,
	}
}

// unpackFields is the recursive counterpart to unpackNotifier.
func unpackFields(tfSettings map[string]any, prefix string, schemas map[string]*schema.Schema, fieldMapping map[string]fieldMapper) map[string]any {
	gfSettings := make(map[string]any, len(schemas))
	for tfKey, sch := range schemas {
		fullTfKey := tfKey
		if prefix != "" {
			fullTfKey = fmt.Sprintf("%s.%s", prefix, tfKey)
		}

		val, ok := tfSettings[tfKey]
		if !ok {
			continue // Skip if the key is not present in the resource map
		}

		gfKey := tfKey
		if fMap := fieldMapping[fullTfKey]; fMap.newKey != "" {
			gfKey = fMap.newKey
		}

		if unpackedVal := unpackedValue(val, fullTfKey, sch, fieldMapping); unpackedVal != nil {
			// Omit nil values, this is usually from a custom transform function or an empty set.
			gfSettings[gfKey] = unpackedVal
		}
	}
	return gfSettings
}

// unpackedValue recursively returns the appropriate Grafana representation of the TF field value based on the schema.
func unpackedValue(val any, tfKey string, sch *schema.Schema, fieldMapping map[string]fieldMapper) any {
	// Apply the transformation function if provided
	if fMap := fieldMapping[tfKey]; fMap.unpackValFunc != nil {
		val = fMap.unpackValFunc(val)
	}

	switch sch.Type {
	case schema.TypeSet:
		// We use TypeSet with MaxItems=1 to represent nested schemas (map[string]any), but they are technically slices
		// and need to be unpacked as such. This means extracting the first item in the set.
		set, ok := val.(*schema.Set)
		if !ok {
			log.Printf("[WARN] Unsupported value type '%s' for key '%s'", sch.Type.String(), tfKey)
			return val
		}

		items := set.List()
		if len(items) == 0 {
			return nil // empty set
		}
		if len(items) > 1 {
			log.Printf("[WARN] Multiple items found in set for path '%s', using the first one", tfKey)
		}
		// Use the first item in the set as the child map
		m, ok := items[0].(map[string]any)
		if !ok {
			log.Printf("[WARN] Unsupported value type '%s' for key '%s'", sch.Type.String(), tfKey)
			return val
		}
		return unpackFields(m, tfKey, sch.Elem.(*schema.Resource).Schema, fieldMapping)
	default:
		return val
	}
}

// packNotifier takes the grafana-style settings and packs them into the Terraform-style settings. It handles:
//   - Applying any transformation functions defined in fieldMapping to the keys and values in gfSettings. This is necessary
//     because some field names differ between Terraform and Grafana, and some values need to be transformed (e.g., converting a string to an integer).
//   - Overriding sensitive fields with the state values if they are present in the Terraform state. This is necessary
//     because the API returns [REDACTED] for sensitive fields, and we want to preserve the original value in the Terraform state.
//   - Collecting all remaining fields from the Grafana settings that are not in the resource schema into a "settings" field.
func packNotifier(p *models.EmbeddedContactPoint, data *schema.ResourceData, n notifier) map[string]any {
	gfSettings := p.Settings.(map[string]any)
	tfSettings := packFields(gfSettings, getNotifierConfigFromStateWithUID(data, n, p.UID), "", n.schema().Schema, n.meta().fieldMapper)

	// Add common fields to the Terraform settings as these aren't available in EmbeddedContactPoint settings.
	for k, v := range packCommonNotifierFields(p) {
		tfSettings[k] = v
	}

	// Collect all remaining fields from the Grafana settings that are not in the resource schema.
	settings := map[string]any{}
	for k, v := range gfSettings {
		settings[k] = fmt.Sprintf("%s", v)
	}
	tfSettings["settings"] = settings

	return tfSettings
}

// packFields is the recursive counterpart to packNotifier.
func packFields(gfSettings, state map[string]any, prefix string, schemas map[string]*schema.Schema, fieldMapping map[string]fieldMapper) map[string]any {
	settings := make(map[string]any, len(schemas))
	for tfKey, sch := range schemas {
		fullTfKey := tfKey
		if prefix != "" {
			fullTfKey = fmt.Sprintf("%s.%s", prefix, tfKey)
		}

		gfKey := tfKey
		if fMap := fieldMapping[fullTfKey]; fMap.newKey != "" {
			gfKey = fMap.newKey
		}

		val, ok := gfSettings[gfKey]
		if !ok {
			continue // Skip if the key is not present in the resource map
		}

		packedVal, remove := packedValue(val, state, fullTfKey, sch, fieldMapping)
		if packedVal != nil {
			// Omit nil values, this is usually from a custom transform function or an empty set.
			settings[tfKey] = packedVal
		}
		if remove {
			delete(gfSettings, gfKey) // Remove the key from the original map to avoid including it in leftover "settings"
		}
	}
	return settings
}

// packedValue recursively returns the appropriate TF representation of the Grafana field value based on the schema.
func packedValue(val any, state map[string]any, tfKey string, sch *schema.Schema, fieldMapping map[string]fieldMapper) (any, bool) {
	stateVal, hasState := state[tfKey]
	if sch.Sensitive && hasState {
		val = stateVal // Use the state value for sensitive fields as the API returns [REDACTED] for sensitive fields.
	}

	// Apply the transformation function if provided
	if fMap := fieldMapping[tfKey]; fMap.packValFunc != nil {
		val = fMap.packValFunc(val)
	}

	switch sch.Type {
	case schema.TypeSet:
		// We use TypeSet with MaxItems=1 to represent nested schemas (map[string]any), but they are technically slices
		// and need to be packed as such.
		m, ok := val.(map[string]any)
		if !ok {
			log.Printf("[WARN] Unsupported value type '%s' for key '%s'", sch.Type.String(), tfKey)
			return val, true
		}

		stateValMap := make(map[string]any)
		switch sv := stateVal.(type) {
		case map[string]any:
			stateValMap = sv
		case *schema.Set:
			items := sv.List()
			if len(items) != 0 {
				if len(items) > 1 {
					log.Printf("[WARN] Multiple items found in state for path '%s', using the first one", tfKey)
				}
				stateValMap = items[0].(map[string]any)
			}
		}

		return []any{packFields(m, stateValMap, tfKey, sch.Elem.(*schema.Resource).Schema, fieldMapping)}, len(m) == 0
	default:
		return val, true
	}
}

type notifier interface {
	meta() notifierMeta
	schema() *schema.Resource
	pack(p *models.EmbeddedContactPoint, data *schema.ResourceData) (any, error)
	unpack(raw any, name string) *models.EmbeddedContactPoint
}

type notifierMeta struct {
	field        string
	typeStr      string
	desc         string
	secureFields []string
	fieldMapper  map[string]fieldMapper
}

type statePair struct {
	tfState map[string]any
	gfState *models.EmbeddedContactPoint
	deleted bool
}

func packNotifierStringField(gfSettings, tfSettings *map[string]any, gfKey, tfKey string) {
	if v, ok := (*gfSettings)[gfKey]; ok && v != nil {
		(*tfSettings)[tfKey] = v.(string)
		delete(*gfSettings, gfKey)
	}
}

func packSecureFields(tfSettings, state map[string]any, secureFields []string) {
	for _, tfKey := range secureFields {
		if v, ok := state[tfKey]; ok && v != nil {
			tfSettings[tfKey] = v
		}
	}
}

func unpackNotifierStringField(tfSettings, gfSettings *map[string]any, tfKey, gfKey string) {
	if v, ok := (*tfSettings)[tfKey]; ok && v != nil {
		(*gfSettings)[gfKey] = v.(string)
	}
}

func getNotifierConfigFromStateWithUID(data *schema.ResourceData, n notifier, uid string) map[string]any {
	if points, ok := data.GetOk(n.meta().field); ok {
		for _, pt := range points.(*schema.Set).List() {
			config := pt.(map[string]any)
			if config["uid"] == uid {
				return config
			}
		}
	}

	return nil
}

// translateTLSConfigPack is necessary to convert the TLS configuration from the Grafana API format to the Terraform format.
// This is needed because tlsConfig was initially defined without a corresponding schema, so packNotifier cannot handle
// the field name conversions with fieldMapper.newKey.
func translateTLSConfigPack(value any) any {
	m, ok := value.(map[string]any)
	if !ok {
		panic(fmt.Sprintf("unexpected type for tls_config: %T", value))
	}
	if len(m) == 0 {
		return nil // Return nil if the map is empty, to avoid setting an empty map in the resource
	}
	// Convert the keys to the expected format
	newTLSConfig := make(map[string]any, len(m))
	for k, v := range m {
		switch k {
		case "insecureSkipVerify":
			if is, ok := v.(string); ok {
				if insecureSkipVerify, err := strconv.ParseBool(is); err != nil {
					log.Printf("[WARN] failed to parse 'insecureSkipVerify': %s", err)
				} else {
					newTLSConfig["insecure_skip_verify"] = insecureSkipVerify
				}
			}
		case "caCertificate":
			newTLSConfig["ca_certificate"] = v
		case "clientCertificate":
			newTLSConfig["client_certificate"] = v
		case "clientKey":
			newTLSConfig["client_key"] = v
		default:
			newTLSConfig[k] = v
		}
	}

	return newTLSConfig
}

// translateTLSConfigUnpack is necessary to convert the TLS configuration from the Terraform API format to the Grafana format.
// This is needed because tlsConfig was initially defined without a corresponding schema, so unpackNotifier cannot handle
// the field name conversions with fieldMapper.newKey.
func translateTLSConfigUnpack(value any) any {
	m, ok := value.(map[string]any)
	if !ok {
		panic(fmt.Sprintf("unexpected type for tlsConfig: %T", value))
	}
	if len(m) == 0 {
		return nil // Return nil if the map is empty, to avoid setting an empty map in the resource
	}
	// Convert the keys to the expected format
	newTLSConfig := make(map[string]any, len(m))
	for k, v := range m {
		switch k {
		case "insecure_skip_verify":
			if is, ok := v.(string); ok {
				if insecureSkipVerify, err := strconv.ParseBool(is); err != nil {
					log.Printf("[WARN] failed to parse 'insecure_skip_verify': %s", err)
				} else {
					newTLSConfig["insecureSkipVerify"] = insecureSkipVerify
				}
			}
		case "ca_certificate":
			newTLSConfig["caCertificate"] = v
		case "client_certificate":
			newTLSConfig["clientCertificate"] = v
		case "client_key":
			newTLSConfig["clientKey"] = v
		default:
			newTLSConfig[k] = v
		}
	}

	return newTLSConfig
}
