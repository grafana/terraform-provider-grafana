package asserts

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	assertsapi "github.com/grafana/grafana-asserts-public-clients/go/gcom"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

func makeResourcePromRules() *common.Resource {
	schema := &schema.Resource{
		Description: "Manages Prometheus Rules configurations through Grafana Asserts API. " +
			"Allows creation and management of custom Prometheus recording and alerting rules.",

		CreateContext: resourcePromRulesCreate,
		ReadContext:   resourcePromRulesRead,
		UpdateContext: resourcePromRulesUpdate,
		DeleteContext: resourcePromRulesDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true, // Force recreation if name changes
				Description: "The name of the Prometheus rules file. This will be stored with a .custom extension. " +
					"Must follow naming validation rules (alphanumeric, hyphens, underscores).",
			},
			"active": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Whether the rules file is active. Inactive rules are not evaluated.",
			},
			"group": {
				Type:     schema.TypeList,
				Required: true,
				Description: "List of Prometheus rule groups. Each group contains one or more rules " +
					"and can have its own evaluation interval.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The name of the rule group (e.g., 'latency_monitoring').",
						},
						"interval": {
							Type:     schema.TypeString,
							Optional: true,
							Description: "Evaluation interval for this group (e.g., '30s', '1m'). " +
								"If not specified, uses the global evaluation interval.",
						},
						"rule": {
							Type:        schema.TypeList,
							Required:    true,
							Description: "List of Prometheus rules in this group.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"record": {
										Type:     schema.TypeString,
										Optional: true,
										Description: "The name of the time series to output for recording rules. " +
											"Either 'record' or 'alert' must be specified, but not both.",
									},
									"alert": {
										Type:     schema.TypeString,
										Optional: true,
										Description: "The name of the alert for alerting rules. " +
											"Either 'record' or 'alert' must be specified, but not both.",
									},
									"expr": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "The PromQL expression to evaluate.",
									},
									"duration": {
										Type:     schema.TypeString,
										Optional: true,
										Description: "How long the condition must be true before firing the alert " +
											"(e.g., '5m'). Only applicable for alerting rules. Maps to 'for' in Prometheus.",
									},
									"active": {
										Type:        schema.TypeBool,
										Optional:    true,
										Default:     true,
										Description: "Whether this specific rule is active.",
									},
									"labels": {
										Type:        schema.TypeMap,
										Optional:    true,
										Description: "Labels to attach to the resulting time series or alert.",
										Elem:        &schema.Schema{Type: schema.TypeString},
									},
									"annotations": {
										Type:        schema.TypeMap,
										Optional:    true,
										Description: "Annotations to add to alerts (e.g., summary, description).",
										Elem:        &schema.Schema{Type: schema.TypeString},
									},
									"disable_in_groups": {
										Type:     schema.TypeSet,
										Optional: true,
										Description: "List of group names where this rule should be disabled. " +
											"Useful for conditional rule enablement.",
										Elem: &schema.Schema{Type: schema.TypeString},
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
		common.CategoryAsserts,
		"grafana_asserts_prom_rule_file",
		common.NewResourceID(common.StringIDField("name")),
		schema,
	).WithLister(assertsListerFunction(listPromRules))
}

func resourcePromRulesCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, stackID, diags := validateAssertsClient(meta)
	if diags.HasError() {
		return diags
	}

	name := d.Get("name").(string)
	active := d.Get("active").(bool)

	// Build the PrometheusRulesDto
	rulesDto := assertsapi.PrometheusRulesDto{
		Name: &name,
	}

	// Only set active if false (true is the default)
	if !active {
		rulesDto.Active = &active
	}

	// Build groups
	groups, err := buildRuleGroups(d.Get("group").([]interface{}))
	if err != nil {
		return diag.FromErr(err)
	}
	rulesDto.Groups = groups

	// Call the API to create/update the rules file
	// Note: PUT is idempotent, so create and update use the same operation
	request := client.PromRulesConfigControllerAPI.PutPromRules(ctx).
		PrometheusRulesDto(rulesDto).
		XScopeOrgID(fmt.Sprintf("%d", stackID))

	_, err = request.Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create Prometheus rules file: %w", err))
	}

	d.SetId(name)

	return resourcePromRulesRead(ctx, d, meta)
}

func resourcePromRulesRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, stackID, diags := validateAssertsClient(meta)
	if diags.HasError() {
		return diags
	}

	name := d.Id()

	// Retry logic for read operation to handle eventual consistency
	var foundRules *assertsapi.PrometheusRulesDto
	err := withRetryRead(ctx, func(retryCount, maxRetries int) *retry.RetryError {
		// Get specific rules file
		request := client.PromRulesConfigControllerAPI.GetPromRules(ctx, name).
			XScopeOrgID(fmt.Sprintf("%d", stackID))

		rules, resp, err := request.Execute()
		if err != nil {
			// If 404, the resource doesn't exist
			if resp != nil && resp.StatusCode == 404 {
				// Check if we should give up or retry
				if retryCount >= maxRetries {
					return createNonRetryableError("Prometheus rules file", name, retryCount)
				}
				return createRetryableError("Prometheus rules file", name, retryCount, maxRetries)
			}
			return createAPIError("get Prometheus rules file", retryCount, maxRetries, err)
		}

		foundRules = rules
		return nil
	})

	if err != nil {
		// If not found after retries, remove from state
		if foundRules == nil {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	// Set the resource data
	if foundRules.Name != nil {
		if err := d.Set("name", *foundRules.Name); err != nil {
			return diag.FromErr(err)
		}
	}

	// Only set active if explicitly false (true is the schema default)
	if foundRules.Active != nil && !*foundRules.Active {
		if err := d.Set("active", *foundRules.Active); err != nil {
			return diag.FromErr(err)
		}
	}

	// Flatten groups back into Terraform state
	if len(foundRules.Groups) > 0 {
		groups, err := flattenRuleGroups(foundRules.Groups)
		if err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set("group", groups); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

func resourcePromRulesUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, stackID, diags := validateAssertsClient(meta)
	if diags.HasError() {
		return diags
	}

	name := d.Get("name").(string)
	active := d.Get("active").(bool)

	// Build the PrometheusRulesDto
	rulesDto := assertsapi.PrometheusRulesDto{
		Name: &name,
	}

	// Only set active if false (true is the default)
	if !active {
		rulesDto.Active = &active
	}

	// Build groups
	groups, err := buildRuleGroups(d.Get("group").([]interface{}))
	if err != nil {
		return diag.FromErr(err)
	}
	rulesDto.Groups = groups

	// Update using PUT (idempotent)
	request := client.PromRulesConfigControllerAPI.PutPromRules(ctx).
		PrometheusRulesDto(rulesDto).
		XScopeOrgID(fmt.Sprintf("%d", stackID))

	_, err = request.Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to update Prometheus rules file: %w", err))
	}

	return resourcePromRulesRead(ctx, d, meta)
}

func resourcePromRulesDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, stackID, diags := validateAssertsClient(meta)
	if diags.HasError() {
		return diags
	}

	name := d.Id()

	// Delete the rules file
	request := client.PromRulesConfigControllerAPI.DeletePromRules(ctx, name).
		XScopeOrgID(fmt.Sprintf("%d", stackID))

	_, err := request.Execute()
	if err != nil {
		// Ignore 404 errors - resource already deleted
		if !common.IsNotFoundError(err) {
			return diag.FromErr(fmt.Errorf("failed to delete Prometheus rules file: %w", err))
		}
	}

	return nil
}

// buildRuleGroups converts Terraform schema data into PrometheusRuleGroupDto slice
func buildRuleGroups(groupsData []interface{}) ([]assertsapi.PrometheusRuleGroupDto, error) {
	if len(groupsData) == 0 {
		return nil, fmt.Errorf("at least one rule group is required")
	}

	groups := make([]assertsapi.PrometheusRuleGroupDto, 0, len(groupsData))

	for _, groupItem := range groupsData {
		groupMap := groupItem.(map[string]interface{})

		groupName := groupMap["name"].(string)
		group := assertsapi.PrometheusRuleGroupDto{
			Name: &groupName,
		}

		// Optional interval
		if interval, ok := groupMap["interval"].(string); ok && interval != "" {
			group.Interval = &interval
		}

		// Build rules
		rulesData := groupMap["rule"].([]interface{})
		if len(rulesData) == 0 {
			return nil, fmt.Errorf("group '%s' must have at least one rule", groupName)
		}

		rules, err := buildRules(rulesData, groupName)
		if err != nil {
			return nil, err
		}

		group.Rules = rules
		groups = append(groups, group)
	}

	return groups, nil
}

// buildRules converts Terraform schema data for rules into PrometheusRuleDto slice
func buildRules(rulesData []interface{}, groupName string) ([]assertsapi.PrometheusRuleDto, error) {
	rules := make([]assertsapi.PrometheusRuleDto, 0, len(rulesData))

	for _, ruleItem := range rulesData {
		ruleMap := ruleItem.(map[string]interface{})

		rule, err := buildRule(ruleMap, groupName)
		if err != nil {
			return nil, err
		}

		rules = append(rules, rule)
	}

	return rules, nil
}

// buildRule converts a single rule from Terraform schema data into PrometheusRuleDto
func buildRule(ruleMap map[string]interface{}, groupName string) (assertsapi.PrometheusRuleDto, error) {
	// Validate record/alert fields
	record, hasRecord := ruleMap["record"].(string)
	alert, hasAlert := ruleMap["alert"].(string)

	if (hasRecord && record != "") && (hasAlert && alert != "") {
		return assertsapi.PrometheusRuleDto{}, fmt.Errorf("rule in group '%s' cannot have both 'record' and 'alert' specified", groupName)
	}
	if (!hasRecord || record == "") && (!hasAlert || alert == "") {
		return assertsapi.PrometheusRuleDto{}, fmt.Errorf("rule in group '%s' must have either 'record' or 'alert' specified", groupName)
	}

	expr := ruleMap["expr"].(string)
	if expr == "" {
		return assertsapi.PrometheusRuleDto{}, fmt.Errorf("rule in group '%s' must have 'expr' specified", groupName)
	}

	rule := assertsapi.PrometheusRuleDto{
		Expr: &expr,
	}

	if hasRecord && record != "" {
		rule.Record = &record
	}

	if hasAlert && alert != "" {
		rule.Alert = &alert
	}

	// Optional fields
	if duration, ok := ruleMap["duration"].(string); ok && duration != "" {
		rule.For = &duration
	}

	// Only set active if explicitly set to false (don't send true as it's the default)
	if active, ok := ruleMap["active"].(bool); ok && !active {
		rule.Active = &active
	}

	// Labels
	if labelsData, ok := ruleMap["labels"].(map[string]interface{}); ok && len(labelsData) > 0 {
		labels := make(map[string]string)
		for k, v := range labelsData {
			labels[k] = v.(string)
		}
		rule.Labels = labels
	}

	// Annotations
	if annotationsData, ok := ruleMap["annotations"].(map[string]interface{}); ok && len(annotationsData) > 0 {
		annotations := make(map[string]string)
		for k, v := range annotationsData {
			annotations[k] = v.(string)
		}
		rule.Annotations = annotations
	}

	// Disable in groups
	if disableInGroupsData, ok := ruleMap["disable_in_groups"].(*schema.Set); ok && disableInGroupsData.Len() > 0 {
		disableInGroups := make([]string, 0, disableInGroupsData.Len())
		for _, item := range disableInGroupsData.List() {
			disableInGroups = append(disableInGroups, item.(string))
		}
		rule.DisableInGroups = disableInGroups
	}

	return rule, nil
}

// flattenRuleGroups converts PrometheusRuleGroupDto slice into Terraform schema data
func flattenRuleGroups(groups []assertsapi.PrometheusRuleGroupDto) ([]interface{}, error) {
	result := make([]interface{}, 0, len(groups))

	for _, group := range groups {
		groupMap := make(map[string]interface{})

		if group.Name != nil {
			groupMap["name"] = *group.Name
		}

		if group.Interval != nil {
			groupMap["interval"] = *group.Interval
		}

		// Flatten rules
		rules := make([]interface{}, 0, len(group.Rules))
		for _, rule := range group.Rules {
			ruleMap := make(map[string]interface{})

			if rule.Record != nil {
				ruleMap["record"] = *rule.Record
			}

			if rule.Alert != nil {
				ruleMap["alert"] = *rule.Alert
			}

			if rule.Expr != nil {
				ruleMap["expr"] = *rule.Expr
			}

			if rule.For != nil {
				ruleMap["duration"] = *rule.For
			}

			// Only set active if explicitly false (default is true in schema)
			if rule.Active != nil && !*rule.Active {
				ruleMap["active"] = *rule.Active
			}

			// Only set collections if they have values
			if len(rule.Labels) > 0 {
				ruleMap["labels"] = rule.Labels
			}

			if len(rule.Annotations) > 0 {
				ruleMap["annotations"] = rule.Annotations
			}

			if len(rule.DisableInGroups) > 0 {
				ruleMap["disable_in_groups"] = rule.DisableInGroups
			}

			rules = append(rules, ruleMap)
		}

		groupMap["rule"] = rules
		result = append(result, groupMap)
	}

	return result, nil
}
