package grafana

import (
	"context"
	"encoding/json"
	"log"
	"reflect"
	"strconv"
	"strings"
	"time"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceRuleGroup() *schema.Resource {
	return &schema.Resource{
		Description: `
Manages Grafana Alerting rule groups.

* [Official documentation](https://grafana.com/docs/grafana/latest/alerting/alerting-rules)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/alerting_provisioning/#alert-rules)

This resource requires Grafana 9.1.0 or later.
`,
		CreateContext: createAlertRuleGroup,
		ReadContext:   readAlertRuleGroup,
		UpdateContext: updateAlertRuleGroup,
		DeleteContext: deleteAlertRuleGroup,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
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
			"org_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The ID of the org to which the group belongs.",
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
							Type:        schema.TypeString,
							Optional:    true,
							Default:     0,
							Description: "The amount of time for which the rule must be breached for the rule to be considered to be Firing. Before this time has elapsed, the rule is only considered to be Pending.",
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
										Required:    true,
										Type:        schema.TypeString,
										Description: "Custom JSON data to send to the specified datasource when querying.",
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
					},
				},
			},
		},
	}
}

func readAlertRuleGroup(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).GrafanaAPI

	key := UnpackGroupID(data.Id())

	group, err := client.AlertRuleGroup(key.FolderUID, key.Name)
	if err != nil {
		if strings.HasPrefix(err.Error(), "status: 404") {
			log.Printf("[WARN] removing rule group %s/%s from state because it no longer exists in grafana", key.FolderUID, key.Name)
			data.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	if err := packRuleGroup(group, data); err != nil {
		return diag.FromErr(err)
	}
	data.SetId(packGroupID(ruleKeyFromGroup(group)))

	return nil
}

func createAlertRuleGroup(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).GrafanaAPI

	group, err := unpackRuleGroup(data)
	if err != nil {
		return diag.FromErr(err)
	}
	key := ruleKeyFromGroup(group)

	if err = client.SetAlertRuleGroup(group); err != nil {
		return diag.FromErr(err)
	}

	data.SetId(packGroupID(key))
	return readAlertRuleGroup(ctx, data, meta)
}

func updateAlertRuleGroup(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).GrafanaAPI

	group, err := unpackRuleGroup(data)
	if err != nil {
		return diag.FromErr(err)
	}
	key := ruleKeyFromGroup(group)

	if err = client.SetAlertRuleGroup(group); err != nil {
		return diag.FromErr(err)
	}

	data.SetId(packGroupID(key))
	return readAlertRuleGroup(ctx, data, meta)
}

func deleteAlertRuleGroup(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).GrafanaAPI

	key := UnpackGroupID(data.Id())

	group, err := client.AlertRuleGroup(key.FolderUID, key.Name)
	if err != nil {
		return diag.FromErr(err)
	}

	for _, r := range group.Rules {
		if err := client.DeleteAlertRule(r.UID); err != nil {
			return diag.FromErr(err)
		}
	}

	return diag.Diagnostics{}
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

func packRuleGroup(g gapi.RuleGroup, data *schema.ResourceData) error {
	data.Set("name", g.Title)
	data.Set("folder_uid", g.FolderUID)
	data.Set("interval_seconds", g.Interval)
	rules := make([]interface{}, 0, len(g.Rules))
	for _, r := range g.Rules {
		data.Set("org_id", strconv.FormatInt(r.OrgID, 10))
		packed, err := packAlertRule(r)
		if err != nil {
			return err
		}
		rules = append(rules, packed)
	}
	data.Set("rule", rules)
	return nil
}

func unpackRuleGroup(data *schema.ResourceData) (gapi.RuleGroup, error) {
	group := data.Get("name").(string)
	folder := data.Get("folder_uid").(string)
	interval := data.Get("interval_seconds").(int)
	packedRules := data.Get("rule").([]interface{})

	// org_id is a string to properly support referencing between resources. However, the API expects an int64.
	orgID, err := strconv.ParseInt(data.Get("org_id").(string), 10, 64)
	if err != nil {
		return gapi.RuleGroup{}, err
	}

	rules := make([]gapi.AlertRule, 0, len(packedRules))
	for i := range packedRules {
		rule, err := unpackAlertRule(packedRules[i], group, folder, orgID)
		if err != nil {
			return gapi.RuleGroup{}, err
		}
		rules = append(rules, rule)
	}

	return gapi.RuleGroup{
		Title:     group,
		FolderUID: folder,
		Interval:  int64(interval),
		Rules:     rules,
	}, nil
}

func packAlertRule(r gapi.AlertRule) (interface{}, error) {
	data, err := packRuleData(r.Data)
	if err != nil {
		return nil, err
	}
	json := map[string]interface{}{
		"uid":            r.UID,
		"name":           r.Title,
		"for":            r.For,
		"no_data_state":  string(r.NoDataState),
		"exec_err_state": string(r.ExecErrState),
		"condition":      r.Condition,
		"labels":         r.Labels,
		"annotations":    r.Annotations,
		"data":           data,
	}
	return json, nil
}

func unpackAlertRule(raw interface{}, groupName string, folderUID string, orgID int64) (gapi.AlertRule, error) {
	json := raw.(map[string]interface{})
	data, err := unpackRuleData(json["data"])
	if err != nil {
		return gapi.AlertRule{}, err
	}

	return gapi.AlertRule{
		UID:          json["uid"].(string),
		Title:        json["name"].(string),
		FolderUID:    folderUID,
		RuleGroup:    groupName,
		OrgID:        orgID,
		ExecErrState: gapi.ExecErrState(json["exec_err_state"].(string)),
		NoDataState:  gapi.NoDataState(json["no_data_state"].(string)),
		For:          json["for"].(string),
		Data:         data,
		Condition:    json["condition"].(string),
		Labels:       unpackMap(json["labels"]),
		Annotations:  unpackMap(json["annotations"]),
	}, nil
}

func packRuleData(queries []*gapi.AlertQuery) (interface{}, error) {
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
		data["model"] = string(model)
		result = append(result, data)
	}
	return result, nil
}

func unpackRuleData(raw interface{}) ([]*gapi.AlertQuery, error) {
	rows := raw.([]interface{})
	result := make([]*gapi.AlertQuery, 0, len(rows))
	for i := range rows {
		row := rows[i].(map[string]interface{})

		stage := &gapi.AlertQuery{
			RefID:         row["ref_id"].(string),
			QueryType:     row["query_type"].(string),
			DatasourceUID: row["datasource_uid"].(string),
		}
		if rtr, ok := row["relative_time_range"]; ok {
			listShim := rtr.([]interface{})
			rtr := listShim[0].(map[string]interface{})
			stage.RelativeTimeRange = gapi.RelativeTimeRange{
				From: time.Duration(rtr["from"].(int)),
				To:   time.Duration(rtr["to"].(int)),
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

func ruleKeyFromGroup(g gapi.RuleGroup) AlertRuleGroupKey {
	return AlertRuleGroupKey{
		FolderUID: g.FolderUID,
		Name:      g.Title,
	}
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
