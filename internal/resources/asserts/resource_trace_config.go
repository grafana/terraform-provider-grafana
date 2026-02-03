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

func makeResourceTraceConfig() *common.Resource {
	resourceSchema := &schema.Resource{
		Description: "Manages Knowledge Graph Trace Configuration through Grafana API.",

		CreateContext: resourceTraceConfigCreate,
		ReadContext:   resourceTraceConfigRead,
		UpdateContext: resourceTraceConfigUpdate,
		DeleteContext: resourceTraceConfigDelete,

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
				Description: "The name of the trace configuration.",
			},
			"priority": {
				Type:         schema.TypeInt,
				Required:     true,
				Description:  "Priority of the trace configuration. A lower number means a higher priority.",
				ValidateFunc: validation.IntBetween(0, 2147483647),
			},
			"match": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "List of match rules for entity properties.",
				Elem: &schema.Resource{
					Schema: getMatchRulesSchema(),
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
				Description: "DataSource to be queried (e.g., a Tempo instance).",
			},
			"entity_property_to_trace_label_mapping": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "Mapping of entity properties to trace labels.",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
		},
	}

	return common.NewLegacySDKResource(
		common.CategoryAsserts,
		"grafana_asserts_trace_config",
		common.NewResourceID(common.StringIDField("name")),
		resourceSchema,
	).WithLister(assertsListerFunction(listTraceConfigs))
}

// resourceTraceConfigCreate - POST endpoint implementation for creating trace configs
func resourceTraceConfigCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, stackID, diags := validateAssertsClient(meta)
	if diags.HasError() {
		return diags
	}

	name := d.Get("name").(string)

	// Build DTO from typed fields
	config := buildTraceDrilldownConfigDto(d)
	config.SetName(name)

	// Call the generated client API
	request := client.TraceDrilldownConfigControllerAPI.UpsertTraceDrilldownConfig(ctx).
		TraceDrilldownConfigDto(*config).
		XScopeOrgID(fmt.Sprintf("%d", stackID))

	_, err := request.Execute()
	if err != nil {
		return diag.FromErr(formatAPIError("failed to create trace configuration", err))
	}

	d.SetId(name)

	return resourceTraceConfigRead(ctx, d, meta)
}

// resourceTraceConfigRead - GET endpoint implementation
func resourceTraceConfigRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, stackID, diags := validateAssertsClient(meta)
	if diags.HasError() {
		return diags
	}

	name := d.Id()

	// Retry logic for read operation to handle eventual consistency
	var tenantConfig *assertsapi.TenantTraceConfigResponseDto
	err := withRetryRead(ctx, func(retryCount, maxRetries int) *retry.RetryError {
		// Get tenant trace config using the generated client API
		request := client.TraceDrilldownConfigControllerAPI.GetTenantTraceConfig(ctx).
			XScopeOrgID(fmt.Sprintf("%d", stackID))

		config, _, err := request.Execute()
		if err != nil {
			return createAPIError("get tenant trace configuration", retryCount, maxRetries, err)
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
	var foundConfig *assertsapi.TraceDrilldownConfigDto
	for _, config := range tenantConfig.GetTraceDrilldownConfigs() {
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
	// Priority is required, so always set it
	if err := d.Set("priority", int(foundConfig.GetPriority())); err != nil {
		return diag.FromErr(err)
	}

	// Set match rules
	if foundConfig.HasMatch() {
		matchRules := matchRulesToSchemaData(foundConfig.GetMatch())
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
	if foundConfig.HasEntityPropertyToTraceLabelMapping() {
		if err := d.Set("entity_property_to_trace_label_mapping", foundConfig.GetEntityPropertyToTraceLabelMapping()); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

// resourceTraceConfigUpdate - POST endpoint implementation for updating trace configs
func resourceTraceConfigUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, stackID, diags := validateAssertsClient(meta)
	if diags.HasError() {
		return diags
	}

	name := d.Get("name").(string)

	// Build DTO from typed fields
	config := buildTraceDrilldownConfigDto(d)
	config.SetName(name)

	// Update Trace Configuration using the generated client API
	request := client.TraceDrilldownConfigControllerAPI.UpsertTraceDrilldownConfig(ctx).
		TraceDrilldownConfigDto(*config).
		XScopeOrgID(fmt.Sprintf("%d", stackID))

	_, err := request.Execute()
	if err != nil {
		return diag.FromErr(formatAPIError("failed to update trace configuration", err))
	}

	return resourceTraceConfigRead(ctx, d, meta)
}

// resourceTraceConfigDelete - DELETE endpoint implementation
func resourceTraceConfigDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, stackID, diags := validateAssertsClient(meta)
	if diags.HasError() {
		return diags
	}

	name := d.Id()

	// Call the generated client API to delete the configuration
	request := client.TraceDrilldownConfigControllerAPI.DeleteConfig(ctx, name).
		XScopeOrgID(fmt.Sprintf("%d", stackID))

	_, err := request.Execute()
	if err != nil {
		return diag.FromErr(formatAPIError("failed to delete trace configuration", err))
	}

	d.SetId("")
	return nil
}

func buildTraceDrilldownConfigDto(d *schema.ResourceData) *assertsapi.TraceDrilldownConfigDto {
	config := assertsapi.NewTraceDrilldownConfigDto()
	config.SetManagedBy(getManagedByTerraformValue())

	// Set required fields - priority is required
	priority := d.Get("priority").(int)
	// Safe conversion to int32 - validated by schema IntBetween(0, 2147483647)
	config.SetPriority(int32(priority)) //nolint:gosec

	// Set match rules
	if v, ok := d.GetOk("match"); ok {
		matches := buildMatchRules(v)
		config.SetMatch(matches)
	}

	// Set required fields
	config.SetDefaultConfig(d.Get("default_config").(bool))
	config.SetDataSourceUid(d.Get("data_source_uid").(string))

	if v, ok := d.GetOk("entity_property_to_trace_label_mapping"); ok {
		mapping := make(map[string]string)
		for k, val := range v.(map[string]interface{}) {
			mapping[k] = val.(string)
		}
		config.SetEntityPropertyToTraceLabelMapping(mapping)
	}

	return config
}
