package grafana

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/client/provisioning"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
)

var resourceRuleGroupID = common.NewResourceID(
	common.OptionalIntIDField("orgID"),
	common.StringIDField("folderUID"),
	common.StringIDField("title"),
)

func resourceRuleGroup() *common.Resource {
	schema := &schema.Resource{
		Description: `
Manages Grafana Alerting rule groups.

* [Official documentation](https://grafana.com/docs/grafana/latest/alerting/set-up/provision-alerting-resources/terraform-provisioning/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/alerting_provisioning/#alert-rules)

This resource requires Grafana 9.1.0 or later.
`,
		CreateContext: putAlertRuleGroup,
		ReadContext:   readAlertRuleGroup,
		UpdateContext: putAlertRuleGroup,
		DeleteContext: deleteAlertRuleGroup,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"org_id": orgIDAttribute(),
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of the rule group.",
			},
			"folder_uid": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The UID of the folder that the group belongs to.",
				ValidateFunc: folderUIDValidation,
			},
			"interval_seconds": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "The interval, in seconds, at which all rules in the group are evaluated. If a group contains many rules, the rules are evaluated sequentially.",
			},
			"disable_provenance": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Allow modifying the rule group from other sources than Terraform or the Grafana API.",
			},
			"rule": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "The rules within the group.",
				MinItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"uid": {
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
							Description: "The unique identifier of the alert rule.",
						},
						"name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The name of the alert rule.",
						},
						"for": {
							Type:             schema.TypeString,
							Optional:         true,
							Default:          0,
							Description:      "The amount of time for which the rule must be breached for the rule to be considered to be Firing. Before this time has elapsed, the rule is only considered to be Pending.",
							ValidateDiagFunc: common.ValidateDurationWithDays,
							DiffSuppressFunc: func(k, oldValue, newValue string, d *schema.ResourceData) bool {
								oldDuration, _ := strfmt.ParseDuration(oldValue)
								newDuration, _ := strfmt.ParseDuration(newValue)
								return oldDuration == newDuration
							},
						},
						"no_data_state": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "NoData",
							Description: "Describes what state to enter when the rule's query returns No Data. Options are OK, NoData, KeepLast, and Alerting.",
						},
						"exec_err_state": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "Alerting",
							Description: "Describes what state to enter when the rule's query is invalid and the rule cannot be executed. Options are OK, Error, KeepLast, and Alerting.",
						},
						"condition": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The `ref_id` of the query node in the `data` field to use as the alert condition.",
						},
						"data": {
							Type:             schema.TypeList,
							Required:         true,
							MinItems:         1,
							Description:      "A sequence of stages that describe the contents of the rule.",
							DiffSuppressFunc: diffSuppressJSON,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"ref_id": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "A unique string to identify this query stage within a rule.",
									},
									"datasource_uid": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "The UID of the datasource being queried, or \"-100\" if this stage is an expression stage.",
									},
									"query_type": {
										Type:        schema.TypeString,
										Optional:    true,
										Default:     "",
										Description: "An optional identifier for the type of query being executed.",
									},
									"model": {
										Required:     true,
										Type:         schema.TypeString,
										Description:  "Custom JSON data to send to the specified datasource when querying.",
										ValidateFunc: validation.StringIsJSON,
										StateFunc:    normalizeModelJSON,
									},
									"relative_time_range": {
										Type:        schema.TypeList,
										Required:    true,
										Description: "The time range, relative to when the query is executed, across which to query.",
										MaxItems:    1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"from": {
													Type:        schema.TypeInt,
													Required:    true,
													Description: "The number of seconds in the past, relative to when the rule is evaluated, at which the time range begins.",
												},
												"to": {
													Type:        schema.TypeInt,
													Required:    true,
													Description: "The number of seconds in the past, relative to when the rule is evaluated, at which the time range ends.",
												},
											},
										},
									},
								},
							},
						},
						"labels": {
							Type:        schema.TypeMap,
							Optional:    true,
							Default:     map[string]interface{}{},
							Description: "Key-value pairs to attach to the alert rule that can be used in matching, grouping, and routing.",
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"annotations": {
							Type:        schema.TypeMap,
							Optional:    true,
							Default:     map[string]interface{}{},
							Description: "Key-value pairs of metadata to attach to the alert rule. They add additional information, such as a `summary` or `runbook_url`, to help identify and investigate alerts. The `dashboardUId` and `panelId` annotations, which link alerts to a panel, must be set together.",
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"is_paused": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "Sets whether the alert should be paused or not.",
						},
						"notification_settings": {
							Type:        schema.TypeList,
							MaxItems:    1,
							Optional:    true,
							Description: "Notification settings for the rule. If specified, it overrides the notification policies. Available since Grafana 10.4, requires feature flag 'alertingSimplifiedRouting' to be enabled.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"contact_point": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "The contact point to route notifications that match this rule to.",
									},
									"group_by": {
										Type:        schema.TypeList,
										Optional:    true,
										Description: "A list of alert labels to group alerts into notifications by. Use the special label `...` to group alerts by all labels, effectively disabling grouping. If empty, no grouping is used. If specified, requires labels 'alertname' and 'grafana_folder' to be included.",
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
									},
									"mute_timings": {
										Type:        schema.TypeList,
										Optional:    true,
										Description: "A list of mute timing names to apply to alerts that match this policy.",
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
									},
									"group_wait": {
										Type:        schema.TypeString,
										Optional:    true,
										Description: "Time to wait to buffer alerts of the same group before sending a notification. Default is 30 seconds.",
									},
									"group_interval": {
										Type:        schema.TypeString,
										Optional:    true,
										Description: "Minimum time interval between two notifications for the same group. Default is 5 minutes.",
									},
									"repeat_interval": {
										Type:        schema.TypeString,
										Optional:    true,
										Description: "Minimum time interval for re-sending a notification if an alert is still firing. Default is 4 hours.",
									},
								},
							},
						},
						"record": {
							Type:        schema.TypeList,
							MaxItems:    1,
							Optional:    true,
							Description: "Settings for a recording rule. Available since Grafana 11.2, requires feature flag 'grafanaManagedRecordingRules' to be enabled.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"metric": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "The name of the metric to write to.",
									},
									"from": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "The ref id of the query node in the data field to use as the source of the metric.",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	return common.NewLegacySDKResource(
		common.CategoryAlerting,
		"grafana_rule_group",
		resourceRuleGroupID,
		schema,
	).WithLister(listerFunctionOrgResource(listRuleGroups))
}

func listRuleGroups(ctx context.Context, client *goapi.GrafanaHTTPAPI, orgID int64) ([]string, error) {
	idMap := map[string]bool{}
	// Retry if the API returns 500 because it may be that the alertmanager is not ready in the org yet.
	// The alertmanager is provisioned asynchronously when the org is created.
	if err := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		resp, err := client.Provisioning.GetAlertRules()
		if err != nil {
			if orgID > 1 && (err.(*runtime.APIError).IsCode(500) || err.(*runtime.APIError).IsCode(403)) {
				return retry.RetryableError(err)
			}
			return retry.NonRetryableError(err)
		}

		for _, rule := range resp.Payload {
			idMap[resourceRuleGroupID.Make(orgID, rule.FolderUID, rule.RuleGroup)] = true
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

func readAlertRuleGroup(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID, idWithoutOrg := OAPIClientFromExistingOrgResource(meta, data.Id())

	folderUID, title, found := strings.Cut(idWithoutOrg, common.ResourceIDSeparator)
	if !found {
		return diag.Errorf("invalid ID %q", idWithoutOrg)
	}

	resp, err := client.Provisioning.GetAlertRuleGroup(title, folderUID)
	if err, shouldReturn := common.CheckReadError("rule group", data, err); shouldReturn {
		return err
	}

	g := resp.Payload
	data.Set("name", g.Title)
	data.Set("folder_uid", g.FolderUID)
	data.Set("interval_seconds", g.Interval)
	disableProvenance := true
	rules := make([]interface{}, 0, len(g.Rules))
	for _, r := range g.Rules {
		ruleResp, err := client.Provisioning.GetAlertRule(r.UID) // We need to get the rule through a separate API call to get the provenance.
		if err != nil {
			return diag.FromErr(err)
		}
		r := ruleResp.Payload
		data.Set("org_id", strconv.FormatInt(*r.OrgID, 10))
		packed, err := packAlertRule(r)
		if err != nil {
			return diag.FromErr(err)
		}
		if r.Provenance != "" {
			disableProvenance = false
		}
		rules = append(rules, packed)
	}
	data.Set("disable_provenance", disableProvenance)
	data.Set("rule", rules)
	data.SetId(resourceRuleGroupID.Make(orgID, folderUID, title))

	return nil
}

func putAlertRuleGroup(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID := OAPIClientFromNewOrgResource(meta, data)

	retryErr := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		respAlertRules, err := client.Provisioning.GetAlertRules()
		if err != nil {
			return retry.NonRetryableError(err)
		}

		if data.IsNewResource() {
			// Check if a rule group with the same name already exists. The API either:
			// - overwrites the existing rule group if it exists in the same folder, which is not expected of a TF provider.
			for _, rule := range respAlertRules.Payload {
				name := data.Get("name").(string)
				folder := data.Get("folder_uid").(string)
				if *rule.RuleGroup == name && *rule.FolderUID == folder {
					return retry.NonRetryableError(fmt.Errorf("rule group with name %q already exists", name))
				}
			}
		}

		group := data.Get("name").(string)
		folder := data.Get("folder_uid").(string)
		interval := data.Get("interval_seconds").(int)

		packedRules := data.Get("rule").([]interface{})
		rules := make([]*models.ProvisionedAlertRule, 0, len(packedRules))

		for i := range packedRules {
			ruleToApply, err := unpackAlertRule(packedRules[i], group, folder, orgID)
			if err != nil {
				return retry.NonRetryableError(err)
			}

			// Check if a rule with the same name or uid already exists within the same rule group
			for _, r := range rules {
				if *r.Title == *ruleToApply.Title {
					return retry.NonRetryableError(fmt.Errorf("rule with name %q is defined more than once", *ruleToApply.Title))
				}
				if ruleToApply.UID != "" && r.UID == ruleToApply.UID {
					return retry.NonRetryableError(fmt.Errorf("rule with UID %q is defined more than once. Rules with name %q and %q have the same uid", ruleToApply.UID, *r.Title, *ruleToApply.Title))
				}
			}

			// Check if a rule with the same name already exists within the same folder (changing the ordering is allowed within the same rule group)
			for _, existingRule := range respAlertRules.Payload {
				if *existingRule.Title == *ruleToApply.Title && *existingRule.FolderUID == *ruleToApply.FolderUID {
					if *ruleToApply.RuleGroup == *existingRule.RuleGroup {
						break
					}

					// Retry so that if the user is moving a rule from one group to another, it will pass on the next iteration.
					return retry.RetryableError(fmt.Errorf("rule with name %q already exists in the folder", *ruleToApply.Title))
				}
			}

			rules = append(rules, ruleToApply)
		}

		putParams := provisioning.NewPutAlertRuleGroupParams().
			WithFolderUID(folder).
			WithGroup(group).WithBody(&models.AlertRuleGroup{
			Title:     group,
			FolderUID: folder,
			Rules:     rules,
			Interval:  int64(interval),
		})

		if data.Get("disable_provenance").(bool) {
			putParams.SetXDisableProvenance(&provenanceDisabled)
		}

		resp, err := client.Provisioning.PutAlertRuleGroup(putParams)
		if err != nil {
			return retry.RetryableError(err)
		}

		data.SetId(resourceRuleGroupID.Make(orgID, resp.Payload.FolderUID, resp.Payload.Title))
		return nil
	})

	if retryErr != nil {
		return diag.FromErr(retryErr)
	}

	return readAlertRuleGroup(ctx, data, meta)
}

func deleteAlertRuleGroup(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, idWithoutOrg := OAPIClientFromExistingOrgResource(meta, data.Id())

	folderUID, title, found := strings.Cut(idWithoutOrg, common.ResourceIDSeparator)
	if !found {
		return diag.Errorf("invalid ID %q", idWithoutOrg)
	}

	// TODO use DeleteAlertRuleGroup method instead (available since Grafana 11)
	resp, err := client.Provisioning.GetAlertRuleGroup(title, folderUID)
	if err != nil {
		return diag.FromErr(err)
	}
	group := resp.Payload

	for _, r := range group.Rules {
		_, err := client.Provisioning.DeleteAlertRule(provisioning.NewDeleteAlertRuleParams().WithUID(r.UID))
		if diag, shouldReturn := common.CheckReadError("rule group", data, err); shouldReturn {
			return diag
		}
	}

	return nil
}

func diffSuppressJSON(k, oldValue, newValue string, data *schema.ResourceData) bool {
	var o, n interface{}
	d := json.NewDecoder(strings.NewReader(oldValue))
	if err := d.Decode(&o); err != nil {
		return false
	}
	d = json.NewDecoder(strings.NewReader(newValue))
	if err := d.Decode(&n); err != nil {
		return false
	}
	return reflect.DeepEqual(o, n)
}

func packAlertRule(r *models.ProvisionedAlertRule) (interface{}, error) {
	data, err := packRuleData(r.Data)
	if err != nil {
		return nil, err
	}
	json := map[string]interface{}{
		"uid":            r.UID,
		"name":           r.Title,
		"for":            r.For.String(),
		"no_data_state":  *r.NoDataState,
		"exec_err_state": *r.ExecErrState,
		"condition":      r.Condition,
		"labels":         r.Labels,
		"annotations":    r.Annotations,
		"data":           data,
		"is_paused":      r.IsPaused,
	}

	ns, err := packNotificationSettings(r.NotificationSettings)
	if err != nil {
		return nil, err
	}
	if ns != nil {
		json["notification_settings"] = ns
	}

	record := packRecord(r.Record)
	if record != nil {
		json["record"] = record
	}

	return json, nil
}

func unpackAlertRule(raw interface{}, groupName string, folderUID string, orgID int64) (*models.ProvisionedAlertRule, error) {
	json := raw.(map[string]interface{})
	data, err := unpackRuleData(json["data"])
	if err != nil {
		return nil, err
	}

	forStr := json["for"].(string)
	if forStr == "" {
		forStr = "0"
	}
	forDuration, err := strfmt.ParseDuration(forStr)
	if err != nil {
		return nil, err
	}

	ns, err := unpackNotificationSettings(json["notification_settings"])
	if err != nil {
		return nil, err
	}

	rule := models.ProvisionedAlertRule{
		UID:                  json["uid"].(string),
		Title:                common.Ref(json["name"].(string)),
		FolderUID:            common.Ref(folderUID),
		RuleGroup:            common.Ref(groupName),
		OrgID:                common.Ref(orgID),
		ExecErrState:         common.Ref(json["exec_err_state"].(string)),
		NoDataState:          common.Ref(json["no_data_state"].(string)),
		For:                  common.Ref(strfmt.Duration(forDuration)),
		Data:                 data,
		Condition:            common.Ref(json["condition"].(string)),
		Labels:               unpackMap(json["labels"]),
		Annotations:          unpackMap(json["annotations"]),
		IsPaused:             json["is_paused"].(bool),
		NotificationSettings: ns,
		Record:               unpackRecord(json["record"]),
	}

	return &rule, nil
}

func packRuleData(queries []*models.AlertQuery) (interface{}, error) {
	result := []interface{}{}
	for i := range queries {
		if queries[i] == nil {
			continue
		}

		model, err := json.Marshal(queries[i].Model)
		if err != nil {
			return nil, err
		}

		data := map[string]interface{}{}
		data["ref_id"] = queries[i].RefID
		data["datasource_uid"] = queries[i].DatasourceUID
		data["query_type"] = queries[i].QueryType
		timeRange := map[string]int{}
		timeRange["from"] = int(queries[i].RelativeTimeRange.From)
		timeRange["to"] = int(queries[i].RelativeTimeRange.To)
		data["relative_time_range"] = []interface{}{timeRange}
		data["model"] = normalizeModelJSON(string(model))
		result = append(result, data)
	}
	return result, nil
}

func unpackRuleData(raw interface{}) ([]*models.AlertQuery, error) {
	rows := raw.([]interface{})
	result := make([]*models.AlertQuery, 0, len(rows))
	for i := range rows {
		row := rows[i].(map[string]interface{})

		stage := &models.AlertQuery{
			RefID:         row["ref_id"].(string),
			QueryType:     row["query_type"].(string),
			DatasourceUID: row["datasource_uid"].(string),
		}
		if rtr, ok := row["relative_time_range"]; ok {
			listShim := rtr.([]interface{})
			rtr := listShim[0].(map[string]interface{})
			stage.RelativeTimeRange = &models.RelativeTimeRange{
				From: models.Duration(time.Duration(rtr["from"].(int))),
				To:   models.Duration(time.Duration(rtr["to"].(int))),
			}
		}
		var decodedModelJSON interface{}
		err := json.Unmarshal([]byte(row["model"].(string)), &decodedModelJSON)
		if err != nil {
			return nil, err
		}
		stage.Model = decodedModelJSON
		result = append(result, stage)
	}
	return result, nil
}

// normalizeModelJSON is the StateFunc for the `model`. It removes well-known default
// values from the model json, so that users do not see perma-diffs when not specifying
// the values explicitly in their Terraform.
func normalizeModelJSON(model interface{}) string {
	modelJSON := model.(string)
	var modelMap map[string]interface{}
	err := json.Unmarshal([]byte(modelJSON), &modelMap)
	if err != nil {
		// This should never happen if the field passes validation.
		log.Printf("[ERROR] Unexpected unmarshal failure for model: %v\n", err)
		return modelJSON
	}

	// The default values taken from:
	//   https://github.com/grafana/grafana/blob/ae688adabcfacd8bd0ac6ebaf8b78506f67962a9/pkg/services/ngalert/models/alert_query.go#L12-L13
	const defaultMaxDataPoints float64 = 43200
	const defaultIntervalMS float64 = 1000

	// https://github.com/grafana/grafana/blob/ae688adabcfacd8bd0ac6ebaf8b78506f67962a9/pkg/services/ngalert/models/alert_query.go#L127-L134
	iMaxDataPoints, ok := modelMap["maxDataPoints"]
	if ok {
		maxDataPoints, ok := iMaxDataPoints.(float64)
		if ok && maxDataPoints == defaultMaxDataPoints {
			log.Printf("[DEBUG] Removing maxDataPoints from state due to being set to default value (%f)", defaultMaxDataPoints)
			delete(modelMap, "maxDataPoints")
		}
	}

	// https://github.com/grafana/grafana/blob/ae688adabcfacd8bd0ac6ebaf8b78506f67962a9/pkg/services/ngalert/models/alert_query.go#L159-L166
	iIntervalMs, ok := modelMap["intervalMs"]
	if ok {
		intervalMs, ok := iIntervalMs.(float64)
		if ok && intervalMs == defaultIntervalMS {
			log.Printf("[DEBUG] Removing intervalMs from state due to being set to default value (%f)", defaultIntervalMS)
			delete(modelMap, "intervalMs")
		}
	}

	j, _ := json.Marshal(modelMap)
	resultJSON := string(j)
	return resultJSON
}

func unpackMap(raw interface{}) map[string]string {
	json := raw.(map[string]interface{})
	result := map[string]string{}
	for k, v := range json {
		result[k] = v.(string)
	}
	return result
}

func packNotificationSettings(settings *models.AlertRuleNotificationSettings) (interface{}, error) {
	if settings == nil {
		return nil, nil
	}

	rec := ""
	if settings.Receiver != nil {
		rec = *settings.Receiver
	}

	result := map[string]interface{}{
		"contact_point": rec,
	}

	if len(settings.GroupBy) > 0 {
		g := make([]interface{}, 0, len(settings.GroupBy))
		for _, s := range settings.GroupBy {
			g = append(g, s)
		}
		result["group_by"] = g
	}
	if len(settings.MuteTimeIntervals) > 0 {
		g := make([]interface{}, 0, len(settings.MuteTimeIntervals))
		for _, s := range settings.MuteTimeIntervals {
			g = append(g, s)
		}
		result["mute_timings"] = g
	}
	if settings.GroupWait != "" {
		result["group_wait"] = settings.GroupWait
	}
	if settings.GroupInterval != "" {
		result["group_interval"] = settings.GroupInterval
	}
	if settings.RepeatInterval != "" {
		result["repeat_interval"] = settings.RepeatInterval
	}
	return []interface{}{result}, nil
}

func unpackNotificationSettings(p interface{}) (*models.AlertRuleNotificationSettings, error) {
	if p == nil {
		return nil, nil
	}
	list := p.([]interface{})
	if len(list) == 0 {
		return nil, nil
	}

	jsonData := list[0].(map[string]interface{})

	receiver := jsonData["contact_point"].(string)
	result := models.AlertRuleNotificationSettings{
		Receiver: &receiver,
	}

	if g, ok := jsonData["group_by"]; ok {
		groupBy := common.ListToStringSlice(g.([]interface{}))
		if len(groupBy) > 0 {
			result.GroupBy = groupBy
		}
	}

	if v, ok := jsonData["mute_timings"]; ok && v != nil {
		result.MuteTimeIntervals = common.ListToStringSlice(v.([]interface{}))
	}
	if v, ok := jsonData["group_wait"]; ok && v != nil {
		result.GroupWait = v.(string)
	}
	if v, ok := jsonData["group_interval"]; ok && v != nil {
		result.GroupInterval = v.(string)
	}
	if v, ok := jsonData["repeat_interval"]; ok && v != nil {
		result.RepeatInterval = v.(string)
	}
	return &result, nil
}

func packRecord(r *models.Record) interface{} {
	if r == nil {
		return nil
	}
	res := map[string]interface{}{}
	if r.Metric != nil {
		res["metric"] = *r.Metric
	}
	if r.From != nil {
		res["from"] = *r.From
	}
	return []interface{}{res}
}

func unpackRecord(p interface{}) *models.Record {
	if p == nil {
		return nil
	}
	list, ok := p.([]interface{})
	if !ok || len(list) == 0 {
		return nil
	}
	jsonData := list[0].(map[string]interface{})
	res := &models.Record{}
	if v, ok := jsonData["metric"]; ok && v != nil {
		res.Metric = common.Ref(v.(string))
	}
	if v, ok := jsonData["from"]; ok && v != nil {
		res.From = common.Ref(v.(string))
	}
	return res
}
