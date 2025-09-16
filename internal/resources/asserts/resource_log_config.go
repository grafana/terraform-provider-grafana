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
				Optional:    true,
				Description: "(Deprecated) Raw YAML for the log configuration. Prefer typed fields.",
			},
			// Terraform-friendly typed attributes that the API returns via GET
			"envs_for_log": {
				Type:        schema.TypeList,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "List of environment names that this configuration applies to.",
			},
			"sites_for_log": {
				Type:        schema.TypeList,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "List of site identifiers that this configuration applies to.",
			},
			"default_config": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Whether this is the default configuration.",
			},
			"log_config": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Description: "Typed log configuration block.",
				Elem: &schema.Resource{Schema: map[string]*schema.Schema{
					"tool":                     {Type: schema.TypeString, Optional: true},
					"url":                      {Type: schema.TypeString, Optional: true},
					"date_format":              {Type: schema.TypeString, Optional: true},
					"correlation_labels":       {Type: schema.TypeList, Optional: true, Elem: &schema.Schema{Type: schema.TypeString}},
					"default_search_text":      {Type: schema.TypeString, Optional: true},
					"error_filter":             {Type: schema.TypeString, Optional: true},
					"columns":                  {Type: schema.TypeList, Optional: true, Elem: &schema.Schema{Type: schema.TypeString}},
					"index":                    {Type: schema.TypeString, Optional: true},
					"interval":                 {Type: schema.TypeString, Optional: true},
					"query":                    {Type: schema.TypeMap, Optional: true, Elem: &schema.Schema{Type: schema.TypeString}},
					"sort":                     {Type: schema.TypeList, Optional: true, Elem: &schema.Schema{Type: schema.TypeString}},
					"http_response_code_field": {Type: schema.TypeString, Optional: true},
					"org_id":                   {Type: schema.TypeInt, Optional: true},
					"data_source":              {Type: schema.TypeString, Optional: true},
				}},
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

	mergedYAML, err := buildMergedYAMLFromTyped(d)
	if err != nil {
		return diag.FromErr(err)
	}

	// Parse merged YAML config into EnvironmentDto
	var env assertsapi.EnvironmentDto
	if err := yaml.Unmarshal([]byte(mergedYAML), &env); err != nil {
		return diag.FromErr(fmt.Errorf("failed to unmarshal merged config YAML: %w", err))
	}

	// Ensure name is set from resource name (overrides YAML if present)
	env.SetName(name)

	// Call the generated client API
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
		// Get tenant log config using the generated client API
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

	// Find our specific environment config
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

	// Set the resource data
	if err := d.Set("name", foundEnv.GetName()); err != nil {
		return diag.FromErr(err)
	}

	// Set typed attributes from API summary (these are returned by GET)
	if err := d.Set("envs_for_log", stringSliceToInterface(foundEnv.GetEnvsForLog())); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("sites_for_log", stringSliceToInterface(foundEnv.GetSitesForLog())); err != nil {
		return diag.FromErr(err)
	}
	// default_config may be omitted; use getter
	if err := d.Set("default_config", foundEnv.GetDefaultConfig()); err != nil {
		return diag.FromErr(err)
	}

	// Preserve user-specified config/log_config to avoid diffs when API does not echo back full details
	if v, ok := d.GetOk("config"); ok {
		_ = d.Set("config", v)
	}
	if v, ok := d.GetOk("log_config"); ok {
		_ = d.Set("log_config", v)
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

	mergedYAML, err := buildMergedYAMLFromTyped(d)
	if err != nil {
		return diag.FromErr(err)
	}

	// Parse merged YAML config into EnvironmentDto
	var env assertsapi.EnvironmentDto
	if err := yaml.Unmarshal([]byte(mergedYAML), &env); err != nil {
		return diag.FromErr(fmt.Errorf("failed to unmarshal merged config YAML: %w", err))
	}

	// Ensure name is set from resource name (overrides YAML if present)
	env.SetName(name)

	// Update Log Configuration using the generated client API
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

func stringSliceToInterface(items []string) []interface{} {
	result := make([]interface{}, 0, len(items))
	for _, v := range items {
		result = append(result, v)
	}
	return result
}

func applyListInto(target map[string]interface{}, key string, list []interface{}) {
	if len(list) == 0 {
		return
	}
	arr := make([]interface{}, 0, len(list))
	for _, v := range list {
		arr = append(arr, v)
	}
	target[key] = arr
}

func buildMergedYAMLFromTyped(d *schema.ResourceData) (string, error) {
	// Start with existing YAML (if any)
	raw := map[string]interface{}{}
	if cfg, ok := d.GetOk("config"); ok {
		if s, ok := cfg.(string); ok && s != "" {
			if err := yaml.Unmarshal([]byte(s), &raw); err != nil {
				return "", fmt.Errorf("invalid config YAML: %w", err)
			}
		}
	}

	// Overlay top-level typed fields
	if v, ok := d.GetOk("envs_for_log"); ok {
		applyListInto(raw, "envsForLog", v.([]interface{}))
	}
	if v, ok := d.GetOk("sites_for_log"); ok {
		applyListInto(raw, "sitesForLog", v.([]interface{}))
	}
	// bool always present (defaults false)
	raw["defaultConfig"] = d.Get("default_config").(bool)

	// Overlay log_config block if provided
	if v, ok := d.GetOk("log_config"); ok {
		list := v.([]interface{})
		if len(list) > 0 && list[0] != nil {
			block := list[0].(map[string]interface{})
			lc := map[string]interface{}{}
			for k, val := range block {
				if val == nil {
					continue
				}
				switch k {
				case "tool":
					lc["tool"] = val
				case "url":
					lc["url"] = val
				case "date_format":
					lc["dateFormat"] = val
				case "correlation_labels":
					applyListInto(lc, "correlationLabels", val.([]interface{}))
				case "default_search_text":
					lc["defaultSearchText"] = val
				case "error_filter":
					lc["errorFilter"] = val
				case "columns":
					applyListInto(lc, "columns", val.([]interface{}))
				case "index":
					lc["index"] = val
				case "interval":
					lc["interval"] = val
				case "query":
					// map[string]string
					lc["query"] = val
				case "sort":
					applyListInto(lc, "sort", val.([]interface{}))
				case "http_response_code_field":
					lc["httpResponseCodeField"] = val
				case "org_id":
					lc["orgId"] = val
				case "data_source":
					lc["dataSource"] = val
				}
			}
			raw["logConfig"] = lc
		}
	}

	out, err := yaml.Marshal(raw)
	if err != nil {
		return "", fmt.Errorf("failed to marshal merged config: %w", err)
	}
	return string(out), nil
}
