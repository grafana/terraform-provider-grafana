package grafana

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strings"
	"time"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceAlertRule() *schema.Resource {
	return &schema.Resource{
		Description: `TODO`,

		ReadContext:   readAlertRule,
		CreateContext: createAlertRule,
		DeleteContext: deleteAlertRule,
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
				Description: "The UID of the group that the folder belongs to.",
			},
			"interval_seconds": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true, // TODO: remove
				Description: "The interval, in seconds, at which all rules in the group are evaluated. If a group contains many rules, the rules are evaluated sequentially.",
			},
			"org_id": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "TODO",
			},
			"rules": {
				Type:        schema.TypeList,
				Required:    true,
				ForceNew:    true, // TODO: remove
				Description: "The rules within the group.",
				MinItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"uid": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "TODO",
						},
						"name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "TODO",
						},
						"for": {
							Type:        schema.TypeInt,
							Required:    true, // TODO
							Description: "TODO",
						},
						"no_data_state": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "NoData",
							Description: "TODO",
						},
						"exec_err_state": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "Alerting",
							Description: "TODO",
						},
						"condition": {
							Type:        schema.TypeString,
							Required:    true, // TODO??
							Description: "TODO",
						},
						"data": {
							Type:             schema.TypeList,
							Required:         false,
							Optional:         true, // TODO: make required
							Description:      "TODO",
							MinItems:         1,
							DiffSuppressFunc: diffSuppressJSON,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"ref_id": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "TODO",
									},
									"datasource_uid": {
										Type:        schema.TypeString,
										Optional:    true, // TODO
										Description: "TODO",
									},
									"query_type": {
										Type:        schema.TypeString,
										Required:    true, // TODO
										Description: "TODO",
									},
									"model": {
										Required:    true,
										Type:        schema.TypeString,
										Description: "TODO",
									},
									"relative_time_range": {
										Type:        schema.TypeList,
										Optional:    true, // TODO
										Description: "TODO",
										MaxItems:    1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"from": {
													Type:        schema.TypeInt,
													Required:    true,
													Description: "TODO",
												},
												"to": {
													Type:        schema.TypeInt,
													Required:    true,
													Description: "TODO",
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
							Description: "TODO",
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"annotations": {
							Type:        schema.TypeMap,
							Optional:    true,
							Default:     map[string]interface{}{},
							Description: "TODO",
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

func readAlertRule(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi

	key := unpackGroupID(data.Id())

	group, err := client.AlertRuleGroup(key.folderUID, key.name)
	if err != nil {
		if strings.HasPrefix(err.Error(), "status: 404") {
			log.Printf("[WARN] removing rule group %s/%s from state because it no longer exists in grafana", key.folderUID, key.name)
			data.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	packRuleGroup(group, data)
	data.SetId(packGroupID(ruleKeyFromGroup(group)))

	return nil
}

func createAlertRule(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi

	group := unpackRuleGroup(data)
	key := ruleKeyFromGroup(group)

	for i := range group.Rules {

		_, err := client.NewAlertRule(&group.Rules[i])
		if err != nil {
			// TODO: remove
			panic(fmt.Sprintf("%s", jsonifyRuleTODORemove(group.Rules[i])))
			return diag.FromErr(err)
		}
	}

	data.SetId(packGroupID(key))
	return readAlertRule(ctx, data, meta)
}

func jsonifyRuleTODORemove(g gapi.AlertRule) string {
	bytes, err := json.Marshal(g)
	if err != nil {
		panic(err)
	}
	return string(bytes)
}

func deleteAlertRule(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi

	key := unpackGroupID(data.Id())

	group, err := client.AlertRuleGroup(key.folderUID, key.name)
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

func packRuleGroup(g gapi.RuleGroup, data *schema.ResourceData) {
	data.Set("name", g.Title)
	data.Set("folder_uid", g.FolderUID)
	data.Set("interval_seconds", g.Interval)
	rules := make([]interface{}, 0, len(g.Rules))
	for _, r := range g.Rules {
		data.Set("org_id", r.OrgID)
		rules = append(rules, packAlertRule(r))
	}
	data.Set("rules", rules)
}

func unpackRuleGroup(data *schema.ResourceData) gapi.RuleGroup {
	group := data.Get("name").(string)
	folder := data.Get("folder_uid").(string)
	interval := data.Get("interval_seconds").(int)
	packedRules := data.Get("rules").([]interface{})
	orgID := data.Get("org_id").(int)

	rules := make([]gapi.AlertRule, 0, len(packedRules))
	for i := range packedRules {
		rule := unpackAlertRule(packedRules[i], group, folder, interval, orgID)
		rules = append(rules, rule)
	}

	return gapi.RuleGroup{
		Title:     group,
		FolderUID: folder,
		Interval:  int64(interval),
		Rules:     rules,
	}
}

func packAlertRule(r gapi.AlertRule) interface{} {
	json := map[string]interface{}{
		"uid":            r.UID,
		"name":           r.Title,
		"for":            r.ForDuration,
		"no_data_state":  string(r.NoDataState),
		"exec_err_state": string(r.ExecErrState),
		"condition":      r.Condition,
		"labels":         r.Labels,
		"annotations":    r.Annotations,
		"data":           packRuleData(r.Data),
	}
	return json
}

func unpackAlertRule(raw interface{}, groupName string, folderUID string, interval int, orgID int) gapi.AlertRule {
	json := raw.(map[string]interface{})

	return gapi.AlertRule{
		Title:     json["name"].(string),
		FolderUID: folderUID,
		RuleGroup: groupName,
		OrgID:     int64(orgID),
		// TODO: interval
		ExecErrState: gapi.ExecErrState(json["exec_err_state"].(string)),
		NoDataState:  gapi.NoDataState(json["no_data_state"].(string)),
		ForDuration:  time.Duration(json["for"].(int)),
		Data:         unpackRuleData(json["data"]),
		Condition:    json["condition"].(string),
		Labels:       unpackMap(json["labels"]),
		Annotations:  unpackMap(json["annotations"]),
	}
}

func packRuleData(queries []*gapi.AlertQuery) interface{} {
	result := []interface{}{}
	for i := range queries {
		if queries[i] == nil {
			continue
		}

		model, err := json.Marshal(queries[i].Model)
		if err != nil {
			panic(err) // TODO: propagate
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
	return result
}

func unpackRuleData(raw interface{}) []*gapi.AlertQuery {
	rows := raw.([]interface{})
	result := make([]*gapi.AlertQuery, 0, len(rows))
	for i := range rows {
		row := rows[i].(map[string]interface{})

		stage := &gapi.AlertQuery{
			RefID:         row["ref_id"].(string),
			QueryType:     row["query_type"].(string),
			DatasourceUID: row["datasource_uid"].(string),

			// TODO
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
			panic(err) // TODO
		}
		stage.Model = decodedModelJSON
		result = append(result, stage)
	}
	return result
}

func unpackMap(raw interface{}) map[string]string {
	json := raw.(map[string]interface{})
	result := map[string]string{}
	for k, v := range json {
		result[k] = v.(string)
	}
	return result
}

type alertRuleGroupKey struct {
	folderUID string
	name      string
}

func ruleKeyFromGroup(g gapi.RuleGroup) alertRuleGroupKey {
	return alertRuleGroupKey{
		folderUID: g.FolderUID,
		name:      g.Title,
	}
}

const groupIDSeparator = ";"

func packGroupID(key alertRuleGroupKey) string {
	return key.folderUID + ";" + key.name
}

func unpackGroupID(tfID string) alertRuleGroupKey {
	vals := strings.SplitN(tfID, groupIDSeparator, 2)
	if len(vals) != 2 {
		return alertRuleGroupKey{}
	}
	return alertRuleGroupKey{
		folderUID: vals[0],
		name:      vals[1],
	}
}
