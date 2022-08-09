package grafana

import (
	"context"
	"strings"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceAlertRule() *schema.Resource {
	return &schema.Resource{
		Description: `TODO`,

		ReadContext: readAlertRule,
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
							Type:        schema.TypeList,
							Required:    false,
							Optional:    true, // TODO: make required
							Description: "TODO",
							MinItems:    1,
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
										// TypeMap with no elem is equivalent to a JSON object.
										Type:        schema.TypeMap,
										Required:    true,
										Description: "TODO",
									},
									"relative_time_range": {
										Type:        schema.TypeMap,
										Optional:    true, // TODO
										Description: "TODO",
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
		return diag.FromErr(err)
	}

	packRuleGroup(group, data)
	data.SetId(packGroupID(ruleKeyFromGroup(group)))

	return nil
}

func packRuleGroup(g gapi.RuleGroup, data *schema.ResourceData) {
	data.Set("name", g.Title)
	data.Set("folder_uid", g.FolderUID)
	data.Set("interval_seconds", g.Interval)
	rules := make([]interface{}, 0, len(g.Rules))
	for _, r := range g.Rules {
		rules = append(rules, packAlertRule(r))
	}
	data.Set("rules", rules)
}

func packAlertRule(r gapi.AlertRule) interface{} {
	json := map[string]interface{}{
		"uid":            r.UID,
		"name":           r.Title,
		"for":            r.ForDuration,
		"no_data_state":  r.NoDataState,
		"exec_err_state": r.ExecErrState,
		"condition":      r.Condition,
		"labels":         r.Labels,
		"annotations":    r.Annotations,
	}
	// TODO: data
	return json
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
