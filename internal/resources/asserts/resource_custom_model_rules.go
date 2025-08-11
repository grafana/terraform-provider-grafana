package asserts

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"gopkg.in/yaml.v2"

	assertsapi "github.com/grafana/grafana-asserts-public-clients/go/gcom"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

func makeResourceCustomModelRules() *common.Resource {
	sch := &schema.Resource{
		Description: "Manages Asserts Custom Model Rules through the Grafana API.",

		CreateContext: resourceCustomModelRulesCreate,
		ReadContext:   resourceCustomModelRulesRead,
		UpdateContext: resourceCustomModelRulesUpdate,
		DeleteContext: resourceCustomModelRulesDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of the custom model rules.",
			},
			"rules": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The rules of the custom model rules, in YAML format.",
			},
		},
	}

	return common.NewLegacySDKResource(
		common.CategoryAsserts,
		"grafana_asserts_custom_model_rules",
		common.NewResourceID(common.StringIDField("name")),
		sch,
	)
}

func resourceCustomModelRulesCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, stackID, diags := validateAssertsClient(meta)
	if diags.HasError() {
		return diags
	}

	name := d.Get("name").(string)
	rulesYAML := d.Get("rules").(string)

	var rules assertsapi.ModelRulesDto
	if err := yaml.Unmarshal([]byte(rulesYAML), &rules); err != nil {
		return diag.FromErr(fmt.Errorf("failed to unmarshal rules YAML: %w", err))
	}
	rules.Name = &name

	req := client.CustomModelRulesControllerAPI.PutModelRules(ctx).ModelRulesDto(rules).XScopeOrgID(fmt.Sprintf("%d", stackID))
	_, err := req.Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create custom model rules: %w", err))
	}

	d.SetId(name)

	return resourceCustomModelRulesRead(ctx, d, meta)
}

func resourceCustomModelRulesRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, stackID, diags := validateAssertsClient(meta)
	if diags.HasError() {
		return diags
	}
	name := d.Id()

	// Retry logic for read operation to handle eventual consistency
	var rules *assertsapi.ModelRulesDto
	err := withRetryRead(ctx, func(retryCount, maxRetries int) *retry.RetryError {
		req := client.CustomModelRulesControllerAPI.GetModelRules(ctx, name).XScopeOrgID(fmt.Sprintf("%d", stackID))
		rulesResult, _, err := req.Execute()
		if err != nil {
			// If the error indicates "not found", check if we should retry or give up
			if _, ok := err.(*assertsapi.GenericOpenAPIError); ok {
				if retryCount >= maxRetries {
					return createNonRetryableError("custom model rules", name, retryCount)
				}
				return createRetryableError("custom model rules", name, retryCount, maxRetries)
			}

			// Other API errors
			return createAPIError("get custom model rules", retryCount, maxRetries, err)
		}

		rules = rulesResult
		return nil
	})

	if err != nil {
		return diag.FromErr(err)
	}

	if rules == nil {
		d.SetId("")
		return nil
	}

	if rules.Name != nil {
		d.Set("name", *rules.Name)
	}

	rules.Name = nil

	rulesYAML, err := yaml.Marshal(rules)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to marshal rules to YAML: %w", err))
	}
	d.Set("rules", string(rulesYAML))

	return nil
}

func resourceCustomModelRulesUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, stackID, diags := validateAssertsClient(meta)
	if diags.HasError() {
		return diags
	}

	name := d.Get("name").(string)
	rulesYAML := d.Get("rules").(string)

	var rules assertsapi.ModelRulesDto
	if err := yaml.Unmarshal([]byte(rulesYAML), &rules); err != nil {
		return diag.FromErr(fmt.Errorf("failed to unmarshal rules YAML: %w", err))
	}
	rules.Name = &name

	req := client.CustomModelRulesControllerAPI.PutModelRules(ctx).ModelRulesDto(rules).XScopeOrgID(fmt.Sprintf("%d", stackID))
	_, err := req.Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to update custom model rules: %w", err))
	}

	return resourceCustomModelRulesRead(ctx, d, meta)
}

func resourceCustomModelRulesDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, stackID, diags := validateAssertsClient(meta)
	if diags.HasError() {
		return diags
	}
	name := d.Id()

	req := client.CustomModelRulesControllerAPI.DeleteModelRules(ctx, name).XScopeOrgID(fmt.Sprintf("%d", stackID))
	_, err := req.Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to delete custom model rules: %w", err))
	}

	return nil
}
