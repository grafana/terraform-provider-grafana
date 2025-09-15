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

func makeResourceLogConfig() *common.Resource {
	schema := &schema.Resource{
		Description: "Manages Asserts Log Configuration through Grafana API.",

		CreateContext: resourceLogConfigCreate,
		ReadContext:   resourceLogConfigRead,
		UpdateContext: resourceLogConfigUpdate,
		DeleteContext: resourceLogConfigDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true, // Force recreation if name changes
				Description: "The name of the log configuration environment.",
			},
			"config": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The log configuration in YAML format.",
			},
		},
	}

	return common.NewLegacySDKResource(
		common.CategoryAsserts,
		"grafana_asserts_log_config",
		common.NewResourceID(common.StringIDField("name")),
		schema,
	).WithLister(assertsListerFunction(listLogConfigs))
}

// resourceLogConfigCreate - POST endpoint implementation for creating log configs
func resourceLogConfigCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, stackID, diags := validateAssertsClient(meta)
	if diags.HasError() {
		return diags
	}

	name := d.Get("name").(string)
	configYAML := d.Get("config").(string)

	// Parse YAML config into EnvironmentDto
	var env assertsapi.EnvironmentDto
	if err := yaml.Unmarshal([]byte(configYAML), &env); err != nil {
		return diag.FromErr(fmt.Errorf("failed to unmarshal config YAML: %w", err))
	}

	// Ensure name is set from resource name
	env.SetName(name)

	// Call the generated client API
	request := client.LogConfigControllerAPI.UpsertEnvironmentConfig(ctx).
		EnvironmentDto(env).
		XScopeOrgID(fmt.Sprintf("%d", stackID))

	_, err := request.Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create log configuration: %w", err))
	}

	d.SetId(name)

	return resourceLogConfigRead(ctx, d, meta)
}

// resourceLogConfigRead - GET endpoint implementation
func resourceLogConfigRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, stackID, diags := validateAssertsClient(meta)
	if diags.HasError() {
		return diags
	}

	name := d.Id()

	var foundEnv *assertsapi.EnvironmentDto
	// Retry logic for read operation to handle eventual consistency.
	err := retry.RetryContext(ctx, d.Timeout(schema.TimeoutRead), func() *retry.RetryError {
		request := client.LogConfigControllerAPI.GetTenantEnvConfig(ctx).
			XScopeOrgID(fmt.Sprintf("%d", stackID))

		tenantConfig, httpResp, err := request.Execute()
		if err != nil {
			// The lister can 404 if there are no log configs. Treat this as "not found".
			if httpResp != nil && httpResp.StatusCode == 404 {
				return nil
			}
			return retry.RetryableError(fmt.Errorf("error getting tenant log configuration: %w", err))
		}

		if tenantConfig != nil {
			for i, env := range tenantConfig.GetEnvironments() {
				if env.GetName() == name {
					foundEnv = &tenantConfig.GetEnvironments()[i]
					return nil // Found it
				}
			}
		}

		// Not found yet, retry
		return retry.RetryableError(fmt.Errorf("log config with name '%s' not found yet", name))
	})

	if err != nil {
		// If we timed out, it means the resource is gone.
		d.SetId("")
		return nil
	}

	if foundEnv == nil {
		// Not found after retries or because of 404 on the API call.
		d.SetId("")
		return nil
	}

	// Set the resource data
	if err := d.Set("name", foundEnv.GetName()); err != nil {
		return diag.FromErr(err)
	}

	// For the config field, we need to marshal the environment back to YAML
	// But we want to preserve the original YAML structure to avoid plan diffs
	currentConfig := d.Get("config").(string)
	if currentConfig != "" {
		// Keep the original config YAML to prevent unnecessary diffs
		if err := d.Set("config", currentConfig); err != nil {
			return diag.FromErr(err)
		}
	} else {
		// Fallback for import case - marshal what we got from API
		configYAML, err := yaml.Marshal(foundEnv)
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to marshal config to YAML: %w", err))
		}
		if err := d.Set("config", string(configYAML)); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

// resourceLogConfigUpdate - POST endpoint implementation for updating log configs
func resourceLogConfigUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, stackID, diags := validateAssertsClient(meta)
	if diags.HasError() {
		return diags
	}

	name := d.Get("name").(string)
	configYAML := d.Get("config").(string)

	// Parse YAML config into EnvironmentDto
	var env assertsapi.EnvironmentDto
	if err := yaml.Unmarshal([]byte(configYAML), &env); err != nil {
		return diag.FromErr(fmt.Errorf("failed to unmarshal config YAML: %w", err))
	}

	// Ensure name is set from resource name
	env.SetName(name)

	// Update Log Configuration using the generated client API
	request := client.LogConfigControllerAPI.UpsertEnvironmentConfig(ctx).
		EnvironmentDto(env).
		XScopeOrgID(fmt.Sprintf("%d", stackID))

	_, err := request.Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to update log configuration: %w", err))
	}

	return resourceLogConfigRead(ctx, d, meta)
}

// resourceLogConfigDelete - DELETE endpoint implementation
func resourceLogConfigDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, stackID, diags := validateAssertsClient(meta)
	if diags.HasError() {
		return diags
	}

	name := d.Id()

	// Call the generated client API to delete the configuration
	request := client.LogConfigControllerAPI.DeleteConfig(ctx, name).
		XScopeOrgID(fmt.Sprintf("%d", stackID))

	_, err := request.Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to delete log configuration: %w", err))
	}

	d.SetId("")
	return nil
}
