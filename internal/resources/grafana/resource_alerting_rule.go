package grafana

import (
	"context"
	"strconv"
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

var resourceAlertRuleSchema = map[string]*schema.Schema{
	"org_id": orgIDAttribute(),
	"rule_group": {
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
	"disable_provenance": {
		Type:        schema.TypeBool,
		Optional:    true,
		Default:     false,
		Description: "Allow modifying the rule group from other sources than Terraform or the Grafana API.",
	},
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
		Default:          "0",
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
		Description: "Describes what state to enter when the rule's query returns No Data. Options are OK, NoData, KeepLast, and Alerting. Defaults to NoData if not set.",
		DiffSuppressFunc: func(k, oldValue, newValue string, d *schema.ResourceData) bool {
			// We default to this value later in the pipeline, so we need to account for that here.
			if newValue == "" {
				return oldValue == "NoData"
			}
			return oldValue == newValue
		},
	},
	"exec_err_state": {
		Type:        schema.TypeString,
		Optional:    true,
		Default:     "Alerting",
		Description: "Describes what state to enter when the rule's query is invalid and the rule cannot be executed. Options are OK, Error, KeepLast, and Alerting.  Defaults to Alerting if not set.",
		DiffSuppressFunc: func(k, oldValue, newValue string, d *schema.ResourceData) bool {
			// We default to this value later in the pipeline, so we need to account for that here.
			if newValue == "" {
				return oldValue == "Alerting"
			}
			return oldValue == newValue
		},
	},
	"condition": {
		Type:        schema.TypeString,
		Optional:    true,
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
}

func resourceRule() *common.Resource {
	schema := &schema.Resource{
		Description: `
Manages Grafana Alerting rules.

* [Official documentation](https://grafana.com/docs/grafana/latest/alerting/set-up/provision-alerting-resources/terraform-provisioning/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/alerting_provisioning/#alert-rules)

This resource requires Grafana 9.1.0 or later.
`,
		CreateContext: postAlertRule,
		ReadContext:   readAlertRule,
		UpdateContext: putAlertRule,
		DeleteContext: deleteAlertRule,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		SchemaVersion: 0,
		Schema:        resourceAlertRuleSchema,
	}

	return common.NewLegacySDKResource(
		common.CategoryAlerting,
		"grafana_rule",
		orgResourceIDString("uid"),
		schema,
	).WithLister(listerFunctionOrgResource(listRules))
}

func listRules(ctx context.Context, client *goapi.GrafanaHTTPAPI, orgID int64) ([]string, error) {
	uids := []string{}
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
			uids = append(uids, MakeOrgResourceID(orgID, rule.UID))
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return uids, nil
}

func readAlertRule(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID, uid := OAPIClientFromExistingOrgResource(meta, data.Id())

	ruleResp, err := client.Provisioning.GetAlertRule(uid)
	if err, shouldReturn := common.CheckReadError("rule", data, err); shouldReturn {
		return err
	}

	r := ruleResp.Payload

	ruleData, err := packRuleData(r.Data)
	if err != nil {
		return diag.FromErr(err)
	}
	ns, err := packNotificationSettings(r.NotificationSettings)
	if err != nil {
		return diag.FromErr(err)
	}

	data.SetId(MakeOrgResourceID(orgID, uid))
	data.Set("org_id", strconv.FormatInt(orgID, 10))
	data.Set("rule_group", r.RuleGroup)
	data.Set("folder_uid", r.FolderUID)
	data.Set("disable_provenance", r.Provenance == "")
	data.Set("uid", r.UID)
	data.Set("name", r.Title)
	data.Set("for", r.For.String())
	data.Set("no_data_state", *r.NoDataState)
	data.Set("exec_err_state", *r.ExecErrState)
	data.Set("condition", r.Condition)
	data.Set("labels", r.Labels)
	data.Set("annotations", r.Annotations)
	data.Set("is_paused", r.IsPaused)
	data.Set("data", ruleData)

	if ns != nil {
		data.Set("notification_settings", ns)
	}

	return nil
}

func postAlertRule(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID := OAPIClientFromNewOrgResource(meta, data)

	ruleToApply, err := unpackAlertRule(data, orgID)
	if err != nil {
		return diag.FromErr(err)
	}

	postParams := provisioning.NewPostAlertRuleParams().WithBody(ruleToApply)
	if data.Get("disable_provenance").(bool) {
		postParams.SetXDisableProvenance(&provenanceDisabled)
	}

	retryErr := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		resp, err := client.Provisioning.PostAlertRule(postParams)
		if orgID > 1 && err != nil {
			if apiError, ok := err.(*runtime.APIError); ok && (apiError.IsCode(500) || apiError.IsCode(404)) {
				return retry.RetryableError(err)
			}
		}
		if err != nil {
			return retry.NonRetryableError(err)
		}

		data.SetId(MakeOrgResourceID(orgID, resp.Payload.UID))
		return nil
	})

	if retryErr != nil {
		return diag.FromErr(retryErr)
	}

	return readAlertRule(ctx, data, meta)
}

func putAlertRule(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID, uid := OAPIClientFromExistingOrgResource(meta, data.Id())

	ruleToApply, err := unpackAlertRule(data, orgID)
	if err != nil {
		return diag.FromErr(err)
	}

	putParams := provisioning.NewPutAlertRuleParams().WithBody(ruleToApply).WithUID(uid)
	if data.Get("disable_provenance").(bool) {
		putParams.SetXDisableProvenance(&provenanceDisabled)
	}

	retryErr := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		resp, err := client.Provisioning.PutAlertRule(putParams)
		if orgID > 1 && err != nil {
			if apiError, ok := err.(*runtime.APIError); ok && (apiError.IsCode(500) || apiError.IsCode(404)) {
				return retry.RetryableError(err)
			}
		}
		if err != nil {
			return retry.NonRetryableError(err)
		}

		data.SetId(MakeOrgResourceID(orgID, resp.Payload.UID))
		return nil
	})

	if retryErr != nil {
		return diag.FromErr(retryErr)
	}

	return readAlertRule(ctx, data, meta)
}

func deleteAlertRule(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, uid := OAPIClientFromExistingOrgResource(meta, data.Id())
	_, deleteErr := client.Provisioning.DeleteAlertRule(provisioning.NewDeleteAlertRuleParams().WithUID(uid))
	err, _ := common.CheckReadError("rule group", data, deleteErr)
	return err
}

func unpackAlertRule(data *schema.ResourceData, orgID int64) (*models.ProvisionedAlertRule, error) {
	ruleData, err := unpackRuleData(data.Get("data"))
	if err != nil {
		return nil, err
	}

	forStr := data.Get("for").(string)
	if forStr == "" {
		forStr = "0"
	}
	forDuration, err := strfmt.ParseDuration(forStr)
	if err != nil {
		return nil, err
	}

	ns, err := unpackNotificationSettings(data.Get("notification_settings"))
	if err != nil {
		return nil, err
	}

	ruleToApply := models.ProvisionedAlertRule{
		Title:                common.Ref(data.Get("name").(string)),
		FolderUID:            common.Ref(data.Get("folder_uid").(string)),
		RuleGroup:            common.Ref(data.Get("rule_group").(string)),
		OrgID:                common.Ref(orgID),
		ExecErrState:         common.Ref(data.Get("exec_err_state").(string)),
		NoDataState:          common.Ref(data.Get("no_data_state").(string)),
		For:                  common.Ref(strfmt.Duration(forDuration)),
		Data:                 ruleData,
		Condition:            common.Ref(data.Get("condition").(string)),
		Labels:               unpackMap(data.Get("labels")),
		Annotations:          unpackMap(data.Get("annotations")),
		IsPaused:             data.Get("is_paused").(bool),
		NotificationSettings: ns,
	}

	if uid, ok := data.GetOk("uid"); ok {
		ruleToApply.UID = uid.(string)
	}

	return &ruleToApply, nil
}
