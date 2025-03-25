package grafana

import (
	"context"
	"fmt"
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

var resourceRuleID = common.NewResourceID(
	common.OptionalIntIDField("orgID"),
	common.StringIDField("folderUID"),
	common.StringIDField("groupName"),
	common.StringIDField("ruleUID"),
)

func resourceRule() *common.Resource {
	schema := &schema.Resource{
		Description: `
Manages individual Grafana Alerting rules.

* [Official documentation](https://grafana.com/docs/grafana/latest/alerting/set-up/provision-alerting-resources/terraform-provisioning/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/alerting_provisioning/#alert-rules)

This resource requires Grafana 9.1.0 or later.
`,
		CreateContext: createAlertRule,
		ReadContext:   readAlertRule,
		UpdateContext: updateAlertRule,
		DeleteContext: deleteAlertRule,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"org_id": orgIDAttribute(),
			"folder_uid": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The UID of the folder that the rule belongs to.",
				ValidateFunc: folderUIDValidation,
			},
			"rule_group": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of the rule group that the rule belongs to.",
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
				Description: "Describes what state to enter when the rule's query returns No Data. Options are OK, NoData, KeepLast, and Alerting.",
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
				Description: "Describes what state to enter when the rule's query is invalid and the rule cannot be executed. Options are OK, Error, KeepLast, and Alerting.",
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
			"disable_provenance": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Allow modifying the rule from other sources than Terraform or the Grafana API.",
			},
		},
	}

	return common.NewLegacySDKResource(
		common.CategoryAlerting,
		"grafana_rule",
		resourceRuleID,
		schema,
	).WithLister(listerFunctionOrgResource(listRules))
}

func listRules(ctx context.Context, client *goapi.GrafanaHTTPAPI, orgID int64) ([]string, error) {
	var ids []string

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
			ids = append(ids, resourceRuleID.Make(orgID, rule.FolderUID, rule.RuleGroup, rule.UID))
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return ids, nil
}

func createAlertRule(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID := OAPIClientFromNewOrgResource(meta, data)

	folderUID := data.Get("folder_uid").(string)
	groupName := data.Get("rule_group").(string)
	name := data.Get("name").(string)
	uid := data.Get("uid").(string)

	// Check if a rule with the same name already exists
	resp, err := client.Provisioning.GetAlertRules()
	if err != nil {
		return diag.FromErr(err)
	}

	for _, rule := range resp.Payload {
		if *rule.Title == name && *rule.FolderUID == folderUID {
			return diag.Errorf("a rule with name %q already exists in folder %q", name, folderUID)
		}
		if uid != "" && rule.UID == uid {
			return diag.Errorf("a rule with UID %q already exists", uid)
		}
	}

	rule, err := buildAlertRuleFromResourceData(data, groupName, folderUID, orgID)
	if err != nil {
		return diag.FromErr(err)
	}

	params := provisioning.NewPostAlertRuleParams().WithBody(rule)

	disableProvenance := data.Get("disable_provenance").(bool)
	if disableProvenance {
		var provenanceFlag = "false"
		params.SetXDisableProvenance(&provenanceFlag)
	}

	ruleResp, err := client.Provisioning.PostAlertRule(params)
	if err != nil {
		return diag.FromErr(err)
	}

	data.SetId(resourceRuleID.Make(orgID, folderUID, groupName, ruleResp.Payload.UID))
	data.Set("uid", ruleResp.Payload.UID)

	return readAlertRule(ctx, data, meta)
}

func readAlertRule(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, idWithoutOrg := OAPIClientFromExistingOrgResource(meta, data.Id())

	// Parse ID
	parts := strings.Split(idWithoutOrg, common.ResourceIDSeparator)
	if len(parts) != 3 {
		return diag.Errorf("invalid ID format: %s", idWithoutOrg)
	}
	ruleUID := parts[2] // only use the ruleUID

	resp, err := client.Provisioning.GetAlertRule(ruleUID)
	if err, shouldReturn := common.CheckReadError("rule", data, err); shouldReturn {
		return err
	}

	rule := resp.Payload
	data.Set("org_id", strconv.FormatInt(*rule.OrgID, 10))
	data.Set("folder_uid", rule.FolderUID)
	data.Set("rule_group", rule.RuleGroup)
	data.Set("uid", rule.UID)
	data.Set("name", rule.Title)
	data.Set("for", rule.For.String())
	data.Set("no_data_state", rule.NoDataState)
	data.Set("exec_err_state", rule.ExecErrState)
	data.Set("condition", rule.Condition)
	data.Set("is_paused", rule.IsPaused)
	data.Set("disable_provenance", rule.Provenance == "")

	// Handle data
	ruleData, err := packRuleData(rule.Data)
	if err != nil {
		return diag.FromErr(err)
	}
	data.Set("data", ruleData)

	// Handle labels and annotations
	data.Set("labels", rule.Labels)
	data.Set("annotations", rule.Annotations)

	// Handle notification settings
	ns, err := packNotificationSettings(rule.NotificationSettings)
	if err != nil {
		return diag.FromErr(err)
	}
	if ns != nil {
		data.Set("notification_settings", ns)
	}

	// Handle record
	record := packRecord(rule.Record)
	if record != nil {
		data.Set("record", record)
	}

	return nil
}

func updateAlertRule(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, idWithoutOrg := OAPIClientFromExistingOrgResource(meta, data.Id())

	// Parse ID
	parts := strings.Split(idWithoutOrg, common.ResourceIDSeparator)
	if len(parts) != 3 {
		return diag.Errorf("invalid ID format: %s", idWithoutOrg)
	}
	folderUID, groupName, ruleUID := parts[0], parts[1], parts[2]

	// Check if rule exists
	_, err := client.Provisioning.GetAlertRule(ruleUID)
	if err != nil {
		return diag.FromErr(err)
	}

	// Get the current resource's org_id
	orgIDStr := data.Get("org_id").(string)
	orgID, err := strconv.ParseInt(orgIDStr, 10, 64)
	if err != nil {
		orgID = 1 // Default to 1 if not specified
	}

	rule, err := buildAlertRuleFromResourceData(data, groupName, folderUID, orgID)
	if err != nil {
		return diag.FromErr(err)
	}
	rule.UID = ruleUID // Ensure we're updating the correct rule

	params := provisioning.NewPutAlertRuleParams().WithUID(ruleUID).WithBody(rule)

	disableProvenance := data.Get("disable_provenance").(bool)
	if disableProvenance {
		var provenanceFlag = "false"
		params.SetXDisableProvenance(&provenanceFlag)
	}

	_, err = client.Provisioning.PutAlertRule(params)
	if err != nil {
		return diag.FromErr(err)
	}

	return readAlertRule(ctx, data, meta)
}

func deleteAlertRule(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, idWithoutOrg := OAPIClientFromExistingOrgResource(meta, data.Id())

	// Parse ID
	parts := strings.Split(idWithoutOrg, common.ResourceIDSeparator)
	if len(parts) != 3 {
		return diag.Errorf("invalid ID format: %s", idWithoutOrg)
	}
	ruleUID := parts[2] // only use the ruleUID

	_, err := client.Provisioning.DeleteAlertRule(provisioning.NewDeleteAlertRuleParams().WithUID(ruleUID))
	if err, shouldReturn := common.CheckReadError("rule", data, err); shouldReturn {
		return err
	}

	return nil
}

func buildAlertRuleFromResourceData(data *schema.ResourceData, groupName, folderUID string, orgID int64) (*models.ProvisionedAlertRule, error) {
	// Get data fields
	forStr := data.Get("for").(string)
	if forStr == "" {
		forStr = "0"
	}
	forDuration, err := strfmt.ParseDuration(forStr)
	if err != nil {
		return nil, err
	}

	ruleData, err := unpackRuleData(data.Get("data"))
	if err != nil {
		return nil, err
	}

	ns, err := unpackNotificationSettings(data.Get("notification_settings"))
	if err != nil {
		return nil, err
	}

	// Check for conflicting fields if record is present
	record := unpackRecord(data.Get("record"))
	if record != nil {
		incompatFieldMsgFmt := `conflicting fields "record" and "%s"`
		if forDuration != 0 {
			return nil, fmt.Errorf(incompatFieldMsgFmt, "for")
		}
		if data.Get("no_data_state").(string) != "" {
			return nil, fmt.Errorf(incompatFieldMsgFmt, "no_data_state")
		}
		if data.Get("exec_err_state").(string) != "" {
			return nil, fmt.Errorf(incompatFieldMsgFmt, "exec_err_state")
		}
		if data.Get("condition").(string) != "" {
			return nil, fmt.Errorf(incompatFieldMsgFmt, "condition")
		}
	}

	if record == nil && data.Get("condition").(string) == "" {
		return nil, fmt.Errorf(`"condition" is required`)
	}

	// Convert maps to expected format
	labels := make(map[string]string)
	for k, v := range data.Get("labels").(map[string]interface{}) {
		labels[k] = v.(string)
	}

	annotations := make(map[string]string)
	for k, v := range data.Get("annotations").(map[string]interface{}) {
		annotations[k] = v.(string)
	}

	// Build the rule
	noDataState := data.Get("no_data_state").(string)
	execErrState := data.Get("exec_err_state").(string)
	condition := data.Get("condition").(string)

	rule := models.ProvisionedAlertRule{
		Title:                common.Ref(data.Get("name").(string)),
		UID:                  data.Get("uid").(string),
		FolderUID:            common.Ref(folderUID),
		RuleGroup:            common.Ref(groupName),
		OrgID:                common.Ref(orgID),
		For:                  common.Ref(strfmt.Duration(forDuration)),
		Condition:            common.Ref(condition),
		NoDataState:          common.Ref(noDataState),
		ExecErrState:         common.Ref(execErrState),
		Data:                 ruleData,
		Labels:               labels,
		Annotations:          annotations,
		IsPaused:             data.Get("is_paused").(bool),
		NotificationSettings: ns,
		Record:               record,
	}

	return &rule, nil
}
