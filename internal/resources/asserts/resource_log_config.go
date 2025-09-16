package asserts

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

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
				Description: "The name of the log configuration.",
			},
			"priority": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Priority of the log configuration. (Note: Not yet supported by API)",
			},
			"match": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "List of match rules for entity properties.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"property": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Entity property to match.",
						},
						"op": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Operation to use for matching. One of: equals, not equals, contains, is null, is not null.",
							ValidateFunc: validation.StringInSlice([]string{
								"equals", "not equals", "contains", "is null", "is not null",
							}, false),
						},
						"values": {
							Type:        schema.TypeList,
							Required:    true,
							Description: "Values to match against.",
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
			"default_config": {
				Type:        schema.TypeBool,
				Required:    true,
				Description: "Is it the default config, therefore undeletable?",
			},
			"data_source_uid": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "DataSource to be queried (e.g., a Loki instance).",
			},
			"error_label": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Error label to filter logs.",
			},
			"entity_property_to_log_label_mapping": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "Mapping of entity properties to log labels.",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"filter_by_span_id": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Filter logs by span ID.",
			},
			"filter_by_trace_id": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Filter logs by trace ID.",
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

	// Build DTO from typed fields
	config := buildLogDrilldownConfigDto(d)
	config.SetName(name)

	// Call the generated client API
	request := client.LogDrilldownConfigControllerAPI.UpsertLogDrilldownConfig(ctx).
		LogDrilldownConfigDto(*config).
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
	var tenantConfig *assertsapi.TenantLogConfigResponseDto
	err := withRetryRead(ctx, func(retryCount, maxRetries int) *retry.RetryError {
		// Get tenant log config using the generated client API
		request := client.LogDrilldownConfigControllerAPI.GetTenantLogConfig(ctx).
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

	// Find our specific config
	var foundConfig *assertsapi.LogDrilldownConfigDto
	for _, config := range tenantConfig.GetLogDrilldownConfigs() {
		if config.GetName() == name {
			foundConfig = &config
			break
		}
	}

	if foundConfig == nil {
		d.SetId("")
		return nil
	}

	// Set the resource data
	if err := d.Set("name", foundConfig.GetName()); err != nil {
		return diag.FromErr(err)
	}

	// Set match rules
	if foundConfig.HasMatch() {
		matchRules := make([]map[string]interface{}, 0, len(foundConfig.GetMatch()))
		for _, match := range foundConfig.GetMatch() {
			rule := map[string]interface{}{
				"property": match.GetProperty(),
				"op":       match.GetOp(),
				"values":   stringSliceToInterface(match.GetValues()),
			}
			matchRules = append(matchRules, rule)
		}
		if err := d.Set("match", matchRules); err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set("default_config", foundConfig.GetDefaultConfig()); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("data_source_uid", foundConfig.GetDataSourceUid()); err != nil {
		return diag.FromErr(err)
	}
	if foundConfig.HasErrorLabel() {
		if err := d.Set("error_label", foundConfig.GetErrorLabel()); err != nil {
			return diag.FromErr(err)
		}
	}
	if foundConfig.HasEntityPropertyToLogLabelMapping() {
		if err := d.Set("entity_property_to_log_label_mapping", foundConfig.GetEntityPropertyToLogLabelMapping()); err != nil {
			return diag.FromErr(err)
		}
	}
	if foundConfig.HasFilterBySpanId() {
		if err := d.Set("filter_by_span_id", foundConfig.GetFilterBySpanId()); err != nil {
			return diag.FromErr(err)
		}
	}
	if foundConfig.HasFilterByTraceId() {
		if err := d.Set("filter_by_trace_id", foundConfig.GetFilterByTraceId()); err != nil {
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

	// Build DTO from typed fields
	config := buildLogDrilldownConfigDto(d)
	config.SetName(name)

	// Update Log Configuration using the generated client API
	request := client.LogDrilldownConfigControllerAPI.UpsertLogDrilldownConfig(ctx).
		LogDrilldownConfigDto(*config).
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
	request := client.LogDrilldownConfigControllerAPI.DeleteConfig(ctx, name).
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

func buildLogDrilldownConfigDto(d *schema.ResourceData) *assertsapi.LogDrilldownConfigDto {
	config := assertsapi.NewLogDrilldownConfigDto()

	// Set priority
	if priority, ok := d.GetOk("priority"); ok {
		_ = priority
	}

	// Set match rules
	if v, ok := d.GetOk("match"); ok {
		matchList := v.([]interface{})
		matches := make([]assertsapi.PropertyMatchEntryDto, 0, len(matchList))
		for _, item := range matchList {
			matchMap := item.(map[string]interface{})
			match := assertsapi.NewPropertyMatchEntryDto()

			if prop, ok := matchMap["property"]; ok {
				match.SetProperty(prop.(string))
			}
			if op, ok := matchMap["op"]; ok {
				match.SetOp(op.(string))
			}
			if vals, ok := matchMap["values"]; ok {
				values := make([]string, 0)
				for _, v := range vals.([]interface{}) {
					if s, ok := v.(string); ok {
						values = append(values, s)
					}
				}
				match.SetValues(values)
			}
			matches = append(matches, *match)
		}
		config.SetMatch(matches)
	}

	config.SetDefaultConfig(d.Get("default_config").(bool))
	config.SetDataSourceUid(d.Get("data_source_uid").(string))

	if v, ok := d.GetOk("error_label"); ok {
		config.SetErrorLabel(v.(string))
	}
	if v, ok := d.GetOk("entity_property_to_log_label_mapping"); ok {
		mapping := make(map[string]string)
		for k, val := range v.(map[string]interface{}) {
			mapping[k] = val.(string)
		}
		config.SetEntityPropertyToLogLabelMapping(mapping)
	}
	if v, ok := d.GetOk("filter_by_span_id"); ok {
		config.SetFilterBySpanId(v.(bool))
	}
	if v, ok := d.GetOk("filter_by_trace_id"); ok {
		config.SetFilterByTraceId(v.(bool))
	}

	return config
}
