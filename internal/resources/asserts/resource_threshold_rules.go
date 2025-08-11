package asserts

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"gopkg.in/yaml.v2"

	assertsapi "github.com/grafana/grafana-asserts-public-clients/go/gcom"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

func makeResourceThresholdRules() *common.Resource {
	sch := &schema.Resource{
		Description: "Manages Asserts Threshold Rules through the Grafana API.",

		CreateContext: resourceThresholdRulesCreate,
		ReadContext:   resourceThresholdRulesRead,
		UpdateContext: resourceThresholdRulesUpdate,
		DeleteContext: resourceThresholdRulesDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of the threshold rules.",
			},
			"scope": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The scope of the threshold rules.",
			},
			"rules": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The rules of the threshold rules, in YAML format.",
			},
		},
	}

	return common.NewLegacySDKResource(
		common.CategoryAsserts,
		"grafana_asserts_threshold_rules",
		common.NewResourceID(common.StringIDField("name")),
		sch,
	)
}

func resourceThresholdRulesCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).AssertsAPIClient
	if client == nil {
		return diag.Errorf("Asserts API client is not configured")
	}

	stackID := meta.(*common.Client).GrafanaStackID
	if stackID == 0 {
		return diag.Errorf("stack_id must be set in provider configuration for Asserts resources")
	}

	name := d.Get("name").(string)
	scope := d.Get("scope").(string)
	rulesYAML := d.Get("rules").(string)

	var rules assertsapi.PrometheusRulesDto
	if err := yaml.Unmarshal([]byte(rulesYAML), &rules); err != nil {
		return diag.FromErr(fmt.Errorf("failed to unmarshal rules YAML: %w", err))
	}

	req := client.ThresholdRulesConfigControllerAPI.UpdateCustomThresholdRules(ctx).PrometheusRulesDto(rules).XScopeOrgID(fmt.Sprintf("%d", stackID))
	_, err := req.Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create threshold rules: %w", err))
	}

	d.SetId(fmt.Sprintf("%s/%s", scope, name))

	if err := waitForThresholdRulesVisible(ctx, client, stackID, scope, 2*time.Minute); err != nil {
		return diag.FromErr(err)
	}

	return resourceThresholdRulesRead(ctx, d, meta)
}

func resourceThresholdRulesRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).AssertsAPIClient
	if client == nil {
		return diag.Errorf("Asserts API client is not configured")
	}

	stackID := meta.(*common.Client).GrafanaStackID
	if stackID == 0 {
		return diag.Errorf("stack_id must be set in provider configuration for Asserts resources")
	}

	scope, _, err := parseThresholdRuleID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	var rules *assertsapi.ThresholdRulesDto
	var reqErr error

	if scope == "resource" {
		var resp *assertsapi.ThresholdRulesDto
		resp, _, reqErr = client.ThresholdRulesConfigControllerAPI.GetResourceThresholdRules(ctx).XScopeOrgID(fmt.Sprintf("%d", stackID)).Execute()
		if resp != nil {
			rules = resp
		}
	} else {
		var resp *assertsapi.ThresholdRulesDto
		resp, _, reqErr = client.ThresholdRulesConfigControllerAPI.GetRequestThresholdRules(ctx).XScopeOrgID(fmt.Sprintf("%d", stackID)).Execute()
		if resp != nil {
			rules = resp
		}
	}

	if reqErr != nil {
		if _, ok := reqErr.(*assertsapi.GenericOpenAPIError); ok {
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("failed to get threshold rules: %w", reqErr))
	}

	d.Set("name", d.Id()) // The API doesn't return a name, so we use the ID

	rulesYAML, err := yaml.Marshal(rules)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to marshal rules to YAML: %w", err))
	}
	d.Set("rules", string(rulesYAML))

	return nil
}

func resourceThresholdRulesUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).AssertsAPIClient
	if client == nil {
		return diag.Errorf("Asserts API client is not configured")
	}

	stackID := meta.(*common.Client).GrafanaStackID
	if stackID == 0 {
		return diag.Errorf("stack_id must be set in provider configuration for Asserts resources")
	}

	scope, _, err := parseThresholdRuleID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	rulesYAML := d.Get("rules").(string)

	var rules assertsapi.PrometheusRulesDto
	if err := yaml.Unmarshal([]byte(rulesYAML), &rules); err != nil {
		return diag.FromErr(fmt.Errorf("failed to unmarshal rules YAML: %w", err))
	}

	req := client.ThresholdRulesConfigControllerAPI.UpdateCustomThresholdRules(ctx).PrometheusRulesDto(rules).XScopeOrgID(fmt.Sprintf("%d", stackID))
	_, err = req.Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to update threshold rules: %w", err))
	}

	if err := waitForThresholdRulesVisible(ctx, client, stackID, scope, 2*time.Minute); err != nil {
		return diag.FromErr(err)
	}

	return resourceThresholdRulesRead(ctx, d, meta)
}

func resourceThresholdRulesDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).AssertsAPIClient
	if client == nil {
		return diag.Errorf("Asserts API client is not configured")
	}

	stackID := meta.(*common.Client).GrafanaStackID
	if stackID == 0 {
		return diag.Errorf("stack_id must be set in provider configuration for Asserts resources")
	}

	rulesYAML := d.Get("rules").(string)

	var rules assertsapi.PrometheusRulesDto
	if err := yaml.Unmarshal([]byte(rulesYAML), &rules); err != nil {
		return diag.FromErr(fmt.Errorf("failed to unmarshal rules YAML: %w", err))
	}

	req := client.ThresholdRulesConfigControllerAPI.DeleteCustomThresholdRule(ctx).PrometheusRuleDto(rules.Groups[0].Rules[0]).XScopeOrgID(fmt.Sprintf("%d", stackID))
	_, err := req.Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to delete threshold rules: %w", err))
	}

	return nil
}

func waitForThresholdRulesVisible(ctx context.Context, client *assertsapi.APIClient, stackID int64, scope string, timeout time.Duration) error {
	return retry.RetryContext(ctx, timeout, func() *retry.RetryError {
		var err error
		if scope == "resource" {
			_, _, err = client.ThresholdRulesConfigControllerAPI.GetResourceThresholdRules(ctx).XScopeOrgID(fmt.Sprintf("%d", stackID)).Execute()
		} else {
			_, _, err = client.ThresholdRulesConfigControllerAPI.GetRequestThresholdRules(ctx).XScopeOrgID(fmt.Sprintf("%d", stackID)).Execute()
		}

		if err != nil {
			if _, ok := err.(*assertsapi.GenericOpenAPIError); ok {
				return retry.RetryableError(fmt.Errorf("threshold rules for scope %q not yet visible", scope))
			}
			return retry.NonRetryableError(err)
		}
		return nil
	})
}

func parseThresholdRuleID(id string) (string, string, error) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("unexpected ID format (%s), expected scope/name", id)
	}
	return parts[0], parts[1], nil
}
