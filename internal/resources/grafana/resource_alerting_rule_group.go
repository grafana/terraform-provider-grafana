package grafana

import (
	"context"
	"encoding/json"
	"log"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/grafana/grafana-openapi-client-go/client/provisioning"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func ResourceRuleGroup() *schema.Resource {
	return &schema.Resource{
		Description: `
Manages Grafana Alerting rule groups.

* [Official documentation](https://grafana.com/docs/grafana/latest/alerting/alerting-rules/)
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
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The UID of the folder that the group belongs to.",
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
							Description: "Describes what state to enter when the rule's query returns No Data. Options are OK, NoData, and Alerting.",
						},
						"exec_err_state": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "Alerting",
							Description: "Describes what state to enter when the rule's query is invalid and the rule cannot be executed. Options are OK, Error, and Alerting.",
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
							Description: "Key-value pairs of metadata to attach to the alert rule that may add user-defined context, but cannot be used for matching, grouping, or routing.",
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
					},
				},
			},
		},
	}
}

func readAlertRuleGroup(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID, idStr := OAPIClientFromExistingOrgResource(meta, data.Id())

	key := UnpackGroupID(idStr)

	resp, err := client.Provisioning.GetAlertRuleGroup(key.Name, key.FolderUID)
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
	data.SetId(MakeOrgResourceID(orgID, packGroupID(key)))

	return nil
}

func putAlertRuleGroup(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID := OAPIClientFromNewOrgResource(meta, data)

	group := data.Get("name").(string)
	folder := data.Get("folder_uid").(string)
	interval := data.Get("interval_seconds").(int)

	packedRules := data.Get("rule").([]interface{})
	rules := make([]*models.ProvisionedAlertRule, 0, len(packedRules))
	for i := range packedRules {
		rule, err := unpackAlertRule(packedRules[i], group, folder, orgID)
		if err != nil {
			return diag.FromErr(err)
		}
		rules = append(rules, rule)
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
		return diag.FromErr(err)
	}

	key := packGroupID(AlertRuleGroupKey{resp.Payload.FolderUID, resp.Payload.Title})
	data.SetId(MakeOrgResourceID(orgID, key))
	return readAlertRuleGroup(ctx, data, meta)
}

func deleteAlertRuleGroup(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, idStr := OAPIClientFromExistingOrgResource(meta, data.Id())

	key := UnpackGroupID(idStr)

	resp, err := client.Provisioning.GetAlertRuleGroup(key.Name, key.FolderUID)
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

	rule := models.ProvisionedAlertRule{
		UID:          json["uid"].(string),
		Title:        common.Ref(json["name"].(string)),
		FolderUID:    common.Ref(folderUID),
		RuleGroup:    common.Ref(groupName),
		OrgID:        common.Ref(orgID),
		ExecErrState: common.Ref(json["exec_err_state"].(string)),
		NoDataState:  common.Ref(json["no_data_state"].(string)),
		For:          common.Ref(strfmt.Duration(forDuration)),
		Data:         data,
		Condition:    common.Ref(json["condition"].(string)),
		Labels:       unpackMap(json["labels"]),
		Annotations:  unpackMap(json["annotations"]),
		IsPaused:     json["is_paused"].(bool),
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

type AlertRuleGroupKey struct {
	FolderUID string
	Name      string
}

const groupIDSeparator = ";"

func packGroupID(key AlertRuleGroupKey) string {
	return key.FolderUID + ";" + key.Name
}

func UnpackGroupID(tfID string) AlertRuleGroupKey {
	vals := strings.SplitN(tfID, groupIDSeparator, 2)
	if len(vals) != 2 {
		return AlertRuleGroupKey{}
	}
	return AlertRuleGroupKey{
		FolderUID: vals[0],
		Name:      vals[1],
	}
}
