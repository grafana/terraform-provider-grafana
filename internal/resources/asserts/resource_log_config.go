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
					"correlation_labels":       {Type: schema.TypeString, Optional: true},
					"default_search_text":      {Type: schema.TypeString, Optional: true},
					"error_filter":             {Type: schema.TypeString, Optional: true},
					"columns":                  {Type: schema.TypeList, Optional: true, Elem: &schema.Schema{Type: schema.TypeString}},
					"index":                    {Type: schema.TypeString, Optional: true},
					"interval":                 {Type: schema.TypeString, Optional: true},
					"query":                    {Type: schema.TypeMap, Optional: true, Elem: &schema.Schema{Type: schema.TypeString}},
					"sort":                     {Type: schema.TypeList, Optional: true, Elem: &schema.Schema{Type: schema.TypeString}},
					"http_response_code_field": {Type: schema.TypeString, Optional: true},
					"org_id":                   {Type: schema.TypeString, Optional: true},
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

	// Build DTO from typed fields only
	var env assertsapi.EnvironmentDto
	env.SetName(name)
	applyTypedAttributesToEnv(d, &env)
	applyTypedLogConfigToEnv(d, &env)

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

	// Preserve user-specified log_config to avoid diffs when API does not echo back full details
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

	// Build DTO from typed fields only
	var env assertsapi.EnvironmentDto
	env.SetName(name)
	applyTypedAttributesToEnv(d, &env)
	applyTypedLogConfigToEnv(d, &env)

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

func stringSliceToInterface(items []string) []interface{} {
	result := make([]interface{}, 0, len(items))
	for _, v := range items {
		result = append(result, v)
	}
	return result
}

func applyTypedAttributesToEnv(d *schema.ResourceData, env *assertsapi.EnvironmentDto) {
	if v, ok := d.GetOk("envs_for_log"); ok {
		var values []string
		for _, x := range v.([]interface{}) {
			if s, ok := x.(string); ok {
				values = append(values, s)
			}
		}
		env.SetEnvsForLog(values)
	}
	if v, ok := d.GetOk("sites_for_log"); ok {
		var values []string
		for _, x := range v.([]interface{}) {
			if s, ok := x.(string); ok {
				values = append(values, s)
			}
		}
		env.SetSitesForLog(values)
	}
	// Set default_config unconditionally; false is a valid explicit value
	b := d.Get("default_config").(bool)
	env.SetDefaultConfig(b)
}

func applyTypedLogConfigToEnv(d *schema.ResourceData, env *assertsapi.EnvironmentDto) {
	v, ok := d.GetOk("log_config")
	if !ok {
		return
	}
	list := v.([]interface{})
	if len(list) == 0 || list[0] == nil {
		return
	}
	block := list[0].(map[string]interface{})
	lc := assertsapi.LogConfigDto{}

	assignString(block, "tool", lc.SetTool)
	assignString(block, "url", lc.SetUrl)
	assignString(block, "date_format", lc.SetDateFormat)
	assignString(block, "correlation_labels", lc.SetCorrelationLabels)
	assignString(block, "default_search_text", lc.SetDefaultSearchText)
	assignString(block, "error_filter", lc.SetErrorFilter)
	assignString(block, "http_response_code_field", lc.SetHttpResponseCodeField)
	assignString(block, "index", lc.SetIndex)
	assignString(block, "interval", lc.SetInterval)
	assignString(block, "data_source", lc.SetDataSource)
	assignString(block, "org_id", lc.SetOrgId)

	assignStringSlice(block, "columns", lc.SetColumns)
	assignStringSlice(block, "sort", lc.SetSort)
	assignStringMap(block, "query", lc.SetQuery)

	env.SetLogConfig(lc)
}

func assignString(block map[string]interface{}, key string, apply func(string)) {
	if x, ok := block[key]; ok && x != nil {
		if s, ok := x.(string); ok {
			apply(s)
		}
	}
}

func assignStringSlice(block map[string]interface{}, key string, apply func([]string)) {
	x, ok := block[key]
	if !ok || x == nil {
		return
	}
	arr := []string{}
	for _, v := range x.([]interface{}) {
		if s, ok := v.(string); ok {
			arr = append(arr, s)
		}
	}
	apply(arr)
}

func assignStringMap(block map[string]interface{}, key string, apply func(map[string]string)) {
	x, ok := block[key]
	if !ok || x == nil {
		return
	}
	m := map[string]string{}
	for k, v := range x.(map[string]interface{}) {
		if sv, ok := v.(string); ok {
			m[k] = sv
		}
	}
	apply(m)
}
