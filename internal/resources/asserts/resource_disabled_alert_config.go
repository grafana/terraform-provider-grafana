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

func makeResourceDisabledAlertConfig() *common.Resource {
	schema := &schema.Resource{
		Description: "Manages Asserts Disabled Alert Configurations through Grafana API.",

		CreateContext: resourceDisabledAlertConfigCreate,
		ReadContext:   resourceDisabledAlertConfigRead,
		UpdateContext: resourceDisabledAlertConfigUpdate,
		DeleteContext: resourceDisabledAlertConfigDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true, // Force recreation if name changes
				Description: "The name of the disabled alert configuration.",
			},
			"match_labels": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "Labels to match for this disabled alert configuration.",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
		},
	}

	return common.NewLegacySDKResource(
		common.CategoryAsserts,
		"grafana_asserts_suppressed_assertions_config",
		common.NewResourceID(common.StringIDField("name")),
		schema,
	).WithLister(assertsListerFunction(listDisabledAlertConfigs))
}

func resourceDisabledAlertConfigCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, stackID, diags := validateAssertsClient(meta)
	if diags.HasError() {
		return diags
	}
	name := d.Get("name").(string)
	matchLabels := make(map[string]string)

	if v, ok := d.GetOk("match_labels"); ok {
		for k, val := range v.(map[string]interface{}) {
			matchLabels[k] = val.(string)
		}
	}

	// Create DisabledAlertConfigDto using the generated client models
	disabledAlertConfig := assertsapi.DisabledAlertConfigDto{
		Name: &name,
	}

	// Only set matchLabels if not empty
	if len(matchLabels) > 0 {
		disabledAlertConfig.MatchLabels = matchLabels
	}

	// Call the generated client API
	request := client.AlertConfigurationAPI.PutDisabledAlertConfig(ctx).
		DisabledAlertConfigDto(disabledAlertConfig).
		XScopeOrgID(fmt.Sprintf("%d", stackID))

	_, err := request.Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create disabled alert configuration: %w", err))
	}

	d.SetId(name)

	return resourceDisabledAlertConfigRead(ctx, d, meta)
}

func resourceDisabledAlertConfigRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, stackID, diags := validateAssertsClient(meta)
	if diags.HasError() {
		return diags
	}
	name := d.Id()

	// Retry logic for read operation to handle eventual consistency
	var foundConfig *assertsapi.DisabledAlertConfigDto
	err := withRetryRead(ctx, func(retryCount, maxRetries int) *retry.RetryError {
		// Get all disabled alert configs using the generated client API
		request := client.AlertConfigurationAPI.GetAllDisabledAlertConfigs(ctx).
			XScopeOrgID(fmt.Sprintf("%d", stackID))

		configs, _, err := request.Execute()
		if err != nil {
			return createAPIError("get disabled alert configurations", retryCount, maxRetries, err)
		}

		// Find our specific config
		for _, config := range configs.DisabledAlertConfigs {
			if config.Name != nil && *config.Name == name {
				foundConfig = &config
				return nil
			}
		}

		// Check if we should give up or retry
		if retryCount >= maxRetries {
			return createNonRetryableError("disabled alert configuration", name, retryCount)
		}
		return createRetryableError("disabled alert configuration", name, retryCount, maxRetries)
	})

	if err != nil {
		return diag.FromErr(err)
	}

	if foundConfig == nil {
		d.SetId("")
		return nil
	}

	// Set the resource data
	if foundConfig.Name != nil {
		if err := d.Set("name", *foundConfig.Name); err != nil {
			return diag.FromErr(err)
		}
	}
	if foundConfig.MatchLabels != nil {
		if err := d.Set("match_labels", foundConfig.MatchLabels); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

func resourceDisabledAlertConfigUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, stackID, diags := validateAssertsClient(meta)
	if diags.HasError() {
		return diags
	}

	name := d.Get("name").(string)
	matchLabels := make(map[string]string)

	if v, ok := d.GetOk("match_labels"); ok {
		for k, val := range v.(map[string]interface{}) {
			matchLabels[k] = val.(string)
		}
	}

	// Create DisabledAlertConfigDto using the generated client models
	disabledAlertConfig := assertsapi.DisabledAlertConfigDto{
		Name: &name,
	}

	// Only set matchLabels if not empty
	if len(matchLabels) > 0 {
		disabledAlertConfig.MatchLabels = matchLabels
	}

	// Update Disabled Alert Configuration using the generated client API
	// Note: For disabled configs, update is effectively a re-create
	request := client.AlertConfigurationAPI.PutDisabledAlertConfig(ctx).
		DisabledAlertConfigDto(disabledAlertConfig).
		XScopeOrgID(fmt.Sprintf("%d", stackID))

	_, err := request.Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to update disabled alert configuration: %w", err))
	}

	return resourceDisabledAlertConfigRead(ctx, d, meta)
}

func resourceDisabledAlertConfigDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, stackID, diags := validateAssertsClient(meta)
	if diags.HasError() {
		return diags
	}
	name := d.Id()

	// Delete Disabled Alert Configuration using the generated client API
	request := client.AlertConfigurationAPI.DeleteDisabledAlertConfig(ctx, name).
		XScopeOrgID(fmt.Sprintf("%d", stackID))

	_, err := request.Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to delete disabled alert configuration: %w", err))
	}

	return nil
}
