package grafana

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/go-openapi/strfmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/client/provisioning"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

func resourceAlertingRule() *common.Resource {
	schema := &schema.Resource{

		Description: `
Manages Grafana Alerting rules.

* [Official documentation](https://grafana.com/docs/grafana/latest/alerting/set-up/provision-alerting-resources/terraform-provisioning/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/alerting_provisioning/#alert-rules)

This resource requires Grafana 9.1.0 or later.
`,

		CreateContext: createRule,
		ReadContext:   readRule,
		UpdateContext: updateRule,
		DeleteContext: deleteRule,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		CustomizeDiff: func(ctx context.Context, d *schema.ResourceDiff, meta interface{}) error {
			oldVal, newVal := d.GetChange("config_json")
			oldUID := extractRuleUID(oldVal.(string))
			newUID := extractRuleUID(newVal.(string))
			if oldUID != newUID && oldUID != "" {
				d.ForceNew("config_json")
			}
			return nil
		},

		Schema: map[string]*schema.Schema{
			"org_id": orgIDAttribute(),
			"uid": {
				Type:     schema.TypeString,
				Computed: true,
				Description: "The unique identifier of the alert rule. This is automatically generated if not provided when creating a rule. " +
					"The uid allows having consistent URLs for accessing rules and when syncing rules between multiple Grafana installs.",
			},
			"config_json": {
				Type:         schema.TypeString,
				Required:     true,
				StateFunc:    NormalizeConfigJSON,
				ValidateFunc: validateConfigJSON,
				Description:  "The complete alert rule model JSON.",
			},
			"disable_provenance": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Allow modifying the rule from other sources than Terraform or the Grafana API.",
			},
		},
		SchemaVersion: 0,
	}

	return common.NewLegacySDKResource(
		common.CategoryAlerting,
		"grafana_alerting_rule",
		orgResourceIDString("uid"),
		schema,
	).WithLister(listerFunctionOrgResource(listRules))
}

func listRules(ctx context.Context, client *goapi.GrafanaHTTPAPI, orgID int64) ([]string, error) {
	resp, err := client.Provisioning.GetAlertRules()
	if err != nil {
		return nil, err
	}

	uids := []string{}
	for _, rule := range resp.Payload {
		uids = append(uids, MakeOrgResourceID(orgID, rule.UID))
	}

	return uids, nil
}

func createRule(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID := OAPIClientFromNewOrgResource(meta, d)

	rule, err := makeRule(d)
	if err != nil {
		return diag.FromErr(err)
	}

	params := provisioning.NewPostAlertRuleParams().WithBody(&rule)
	if d.Get("disable_provenance").(bool) {
		params.SetXDisableProvenance(&provenanceDisabled)
	}

	resp, err := client.Provisioning.PostAlertRule(params)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(MakeOrgResourceID(orgID, resp.Payload.UID))
	return readRule(ctx, d, meta)
}

func readRule(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID, uid := OAPIClientFromExistingOrgResource(meta, d.Id())

	resp, err := client.Provisioning.GetAlertRule(uid)
	if err, shouldReturn := common.CheckReadError("alert rule", d, err); shouldReturn {
		return err
	}
	rule := resp.Payload

	d.SetId(MakeOrgResourceID(orgID, uid))
	d.Set("org_id", strconv.FormatInt(orgID, 10))
	d.Set("uid", rule.UID)

	disableProvenance := rule.Provenance == ""
	d.Set("disable_provenance", disableProvenance)

	configJSONBytes, err := json.Marshal(rule)
	if err != nil {
		return diag.FromErr(err)
	}
	remoteConfigJSON, err := UnmarshalConfigJSON(string(configJSONBytes))
	if err != nil {
		return diag.FromErr(err)
	}

	configJSON := d.Get("config_json").(string)

	// If certain fields are not set in configuration, we need to delete them from the
	// rule JSON we just read from the Grafana API. This is so it does not
	// create a diff.
	if configJSON != "" {
		configuredConfigJSON, err := UnmarshalConfigJSON(configJSON)
		if err != nil {
			return diag.FromErr(err)
		}
		// Remove uid if not in config
		if _, ok := configuredConfigJSON["uid"].(string); !ok {
			delete(remoteConfigJSON, "uid")
		}

		// Note: We intentionally do NOT remove relativeTimeRange from remote if not in config.
		// If someone added it via Grafana UI, that should show as drift so Terraform can revert it.
		// We only normalize away known Grafana defaults in NormalizeConfigJSON (empty objects, to:0)
	}
	configJSON = NormalizeConfigJSON(remoteConfigJSON)
	d.Set("config_json", configJSON)

	return nil
}

func updateRule(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID, uid := OAPIClientFromExistingOrgResource(meta, d.Id())

	rule, err := makeRule(d)
	if err != nil {
		return diag.FromErr(err)
	}

	params := provisioning.NewPutAlertRuleParams().WithUID(uid).WithBody(&rule)
	if d.Get("disable_provenance").(bool) {
		params.SetXDisableProvenance(&provenanceDisabled)
	}

	resp, err := client.Provisioning.PutAlertRule(params)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(MakeOrgResourceID(orgID, resp.Payload.UID))
	return readRule(ctx, d, meta)
}

func deleteRule(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, uid := OAPIClientFromExistingOrgResource(meta, d.Id())
	_, deleteErr := client.Provisioning.DeleteAlertRule(provisioning.NewDeleteAlertRuleParams().WithUID(uid))
	err, _ := common.CheckReadError("alert rule", d, deleteErr)
	return err
}

func makeRule(d *schema.ResourceData) (models.ProvisionedAlertRule, error) {
	rule := models.ProvisionedAlertRule{}

	configJSON := d.Get("config_json").(string)
	ruleMap, err := UnmarshalConfigJSON(configJSON)
	if err != nil {
		return rule, err
	}

	delete(ruleMap, "id")
	delete(ruleMap, "orgID")

	// Marshal back to JSON and unmarshal into the model
	// This is a simple way to convert map to struct
	ruleBytes, err := json.Marshal(ruleMap)
	if err != nil {
		return rule, err
	}

	err = json.Unmarshal(ruleBytes, &rule)
	if err != nil {
		return rule, err
	}

	return rule, nil
}

// UnmarshalConfigJSON is a convenience func for unmarshalling
// `config_json` field.
func UnmarshalConfigJSON(configJSON string) (map[string]interface{}, error) {
	ruleMap := map[string]interface{}{}
	err := json.Unmarshal([]byte(configJSON), &ruleMap)
	if err != nil {
		return nil, err
	}
	return ruleMap, nil
}

// validateConfigJSON is the ValidateFunc for `config_json`. It
// ensures its value is valid JSON.
func validateConfigJSON(config interface{}, k string) ([]string, []error) {
	configJSON := config.(string)
	configMap := map[string]interface{}{}
	err := json.Unmarshal([]byte(configJSON), &configMap)
	if err != nil {
		return nil, []error{err}
	}
	return nil, nil
}

// NormalizeConfigJSON is the StateFunc for the `config_json` field.
//
// It removes fields that are managed by Grafana and should not be in the configuration.
func NormalizeConfigJSON(config interface{}) string {
	var ruleMap map[string]interface{}
	switch c := config.(type) {
	case map[string]interface{}:
		ruleMap = c
	case string:
		var err error
		ruleMap, err = UnmarshalConfigJSON(c)
		if err != nil {
			return c
		}
	}

	// Remove fields that are managed by Grafana
	delete(ruleMap, "id")
	delete(ruleMap, "orgID")
	delete(ruleMap, "updated")
	delete(ruleMap, "provenance")

	// Normalize the "data" field - remove default values that Grafana adds
	if data, ok := ruleMap["data"].([]interface{}); ok {
		for _, item := range data {
			if dataItem, ok := item.(map[string]interface{}); ok {
				// Remove default values that Grafana adds automatically
				if model, ok := dataItem["model"].(map[string]interface{}); ok {
					// Remove default maxDataPoints (43200) if present
					if mdp, ok := model["maxDataPoints"].(float64); ok && mdp == 43200 {
						delete(model, "maxDataPoints")
					}
					// Remove default intervalMs (1000) if present
					if im, ok := model["intervalMs"].(float64); ok && im == 1000 {
						delete(model, "intervalMs")
					}
				}

				// Normalize relativeTimeRange - remove empty objects and default "to" value
				if rtr, ok := dataItem["relativeTimeRange"].(map[string]interface{}); ok {
					// Remove "to" field if it's 0 (the default value that Grafana omits)
					if toVal, ok := rtr["to"]; ok {
						if toFloat, ok := toVal.(float64); ok && toFloat == 0 {
							delete(rtr, "to")
						}
					}
					// Remove empty relativeTimeRange objects after cleanup
					if len(rtr) == 0 {
						delete(dataItem, "relativeTimeRange")
					}
				}
			}
		}
	}

	// Normalize duration fields to minute format (e.g., "5m" not "5m0s")
	if forValue, ok := ruleMap["for"]; ok {
		forStr, isString := forValue.(string)
		if !isString || forStr == "" {
			forStr = "0"
		}
		forDuration, err := strfmt.ParseDuration(forStr)
		if err == nil {
			// Convert to seconds format for consistency
			ruleMap["for"] = fmt.Sprintf("%ds", int(forDuration.Seconds()))
		}
	}

	// Normalize notification_settings - remove null/empty array fields
	if ns, ok := ruleMap["notification_settings"].(map[string]interface{}); ok {
		// Remove null or empty array fields
		for _, field := range []string{"mute_time_intervals", "active_time_intervals", "group_by", "receiver"} {
			if val, exists := ns[field]; exists {
				if val == nil {
					delete(ns, field)
				} else if arr, isArr := val.([]interface{}); isArr && len(arr) == 0 {
					delete(ns, field)
				}
			}
		}

		// Remove notification_settings entirely if it's empty
		if len(ns) == 0 {
			delete(ruleMap, "notification_settings")
		}
	}

	j, _ := json.Marshal(ruleMap)

	return string(j)
}

func extractRuleUID(jsonStr string) string {
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		return ""
	}
	if uid, ok := parsed["uid"].(string); ok {
		return uid
	}
	return ""
}
