package asserts

import (
	"context"
	"fmt"
	"time"

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

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(5 * time.Minute),
			Read:   schema.DefaultTimeout(2 * time.Minute),
			Update: schema.DefaultTimeout(5 * time.Minute),
			Delete: schema.DefaultTimeout(5 * time.Minute),
		},

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

// expandEnvironmentFromYAML parses YAML into EnvironmentDto and sets the name if missing.
func expandEnvironmentFromYAML(name string, yamlConfig string) (assertsapi.EnvironmentDto, error) {
	var env assertsapi.EnvironmentDto
	if err := yaml.Unmarshal([]byte(yamlConfig), &env); err != nil {
		return env, fmt.Errorf("failed to unmarshal config YAML: %w", err)
	}
	if env.GetName() == "" {
		env.SetName(name)
	}
	return env, nil
}

// flattenEnvironmentToYAML marshals EnvironmentDto to YAML; caller may preserve original YAML to avoid diffs.
func flattenEnvironmentToYAML(env *assertsapi.EnvironmentDto) (string, error) {
	if env == nil {
		return "", nil
	}
	b, err := yaml.Marshal(env)
	if err != nil {
		return "", fmt.Errorf("failed to marshal config to YAML: %w", err)
	}
	return string(b), nil
}

// resourceLogConfigCreate - POST endpoint implementation for creating log configs
func resourceLogConfigCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, stackID, diags := validateAssertsClient(meta)
	if diags.HasError() {
		return diags
	}

	name := d.Get("name").(string)
	configYAML := d.Get("config").(string)

	env, err := expandEnvironmentFromYAML(name, configYAML)
	if err != nil {
		return diag.FromErr(err)
	}

	request := client.LogConfigControllerAPI.UpsertEnvironmentConfig(ctx).
		EnvironmentDto(env).
		XScopeOrgID(fmt.Sprintf("%d", stackID))

	_, err = request.Execute()
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

	// Retry logic for read operation to handle eventual consistency
	var tenantConfig *assertsapi.TenantEnvConfigResponseDto
	err := withRetryRead(ctx, func(retryCount, maxRetries int) *retry.RetryError {
		request := client.LogConfigControllerAPI.GetTenantEnvConfig(ctx).
			XScopeOrgID(fmt.Sprintf("%d", stackID))
		config, _, err := request.Execute()
		if err != nil {
			return createAPIError("get tenant log configuration", retryCount, maxRetries, err)
		}
		tenantConfig = config
		return nil
	})
	if err != nil {
		return diag.FromErr(err)
	}
	if tenantConfig == nil {
		d.SetId("")
		return nil
	}

	var foundEnv *assertsapi.EnvironmentDto
	for _, env := range tenantConfig.GetEnvironments() {
		if env.GetName() == name {
			foundEnv = &env
			break
		}
	}
	if foundEnv == nil {
		d.SetId("")
		return nil
	}

	if err := d.Set("name", foundEnv.GetName()); err != nil {
		return diag.FromErr(err)
	}

	currentConfig := d.Get("config").(string)
	if currentConfig != "" {
		if err := d.Set("config", currentConfig); err != nil {
			return diag.FromErr(err)
		}
	} else {
		marshaled, err := flattenEnvironmentToYAML(foundEnv)
		if err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set("config", marshaled); err != nil {
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

	env, err := expandEnvironmentFromYAML(name, configYAML)
	if err != nil {
		return diag.FromErr(err)
	}

	request := client.LogConfigControllerAPI.UpsertEnvironmentConfig(ctx).
		EnvironmentDto(env).
		XScopeOrgID(fmt.Sprintf("%d", stackID))

	_, err = request.Execute()
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
