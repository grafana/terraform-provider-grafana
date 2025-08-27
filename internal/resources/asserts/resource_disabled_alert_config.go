package asserts

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
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
			"stack_id": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "The Stack ID of the Grafana Cloud instance.",
			},
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
		"grafana_asserts_disabled_alert_config",
		common.NewResourceID(common.StringIDField("name")),
		schema,
	).WithLister(assertsListerFunction(listDisabledAlertConfigs))
}

func resourceDisabledAlertConfigCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).AssertsAPIClient
	if client == nil {
		return diag.Errorf("Asserts API client is not configured")
	}

	stackID := int64(d.Get("stack_id").(int))
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
	request := client.DisabledAlertConfigControllerAPI.PutDisabledAlertConfig(ctx).
		DisabledAlertConfigDto(disabledAlertConfig).
		XScopeOrgID(fmt.Sprintf("%d", stackID))

	_, err := request.Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create disabled alert configuration: %w", err))
	}

	d.SetId(fmt.Sprintf("%d:%s", stackID, name))
	return resourceDisabledAlertConfigRead(ctx, d, meta)
}

func resourceDisabledAlertConfigRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).AssertsAPIClient
	if client == nil {
		return diag.Errorf("Asserts API client is not configured")
	}

	// Parse ID to get stack_id and name
	parts := strings.Split(d.Id(), ":")
	if len(parts) != 2 {
		return diag.Errorf("invalid resource ID format: %s", d.Id())
	}

	stackID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return diag.FromErr(fmt.Errorf("invalid stack_id in resource ID: %w", err))
	}
	name := parts[1]

	// Get all disabled alert configs using the generated client API
	request := client.DisabledAlertConfigControllerAPI.GetAllDisabledAlertConfigs(ctx).
		XScopeOrgID(fmt.Sprintf("%d", stackID))

	configs, _, err := request.Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to get disabled alert configurations: %w", err))
	}

	// Find our specific config
	var foundConfig *assertsapi.DisabledAlertConfigDto
	for _, config := range configs.DisabledAlertConfigs {
		if config.Name != nil && *config.Name == name {
			foundConfig = &config
			break
		}
	}

	if foundConfig == nil {
		d.SetId("")
		return nil
	}

	// Set the resource data
	if err := d.Set("stack_id", int(stackID)); err != nil {
		return diag.FromErr(err)
	}
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
	client := meta.(*common.Client).AssertsAPIClient
	if client == nil {
		return diag.Errorf("Asserts API client is not configured")
	}

	// Parse ID to get stack_id and name
	parts := strings.Split(d.Id(), ":")
	if len(parts) != 2 {
		return diag.Errorf("invalid resource ID format: %s", d.Id())
	}

	stackID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return diag.FromErr(fmt.Errorf("invalid stack_id in resource ID: %w", err))
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
	request := client.DisabledAlertConfigControllerAPI.PutDisabledAlertConfig(ctx).
		DisabledAlertConfigDto(disabledAlertConfig).
		XScopeOrgID(fmt.Sprintf("%d", stackID))

	_, err = request.Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to update disabled alert configuration: %w", err))
	}

	return resourceDisabledAlertConfigRead(ctx, d, meta)
}

func resourceDisabledAlertConfigDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).AssertsAPIClient
	if client == nil {
		return diag.Errorf("Asserts API client is not configured")
	}

	// Parse ID to get stack_id and name
	parts := strings.Split(d.Id(), ":")
	if len(parts) != 2 {
		return diag.Errorf("invalid resource ID format: %s", d.Id())
	}

	stackID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return diag.FromErr(fmt.Errorf("invalid stack_id in resource ID: %w", err))
	}
	name := parts[1]

	// Delete Disabled Alert Configuration using the generated client API
	request := client.DisabledAlertConfigControllerAPI.DeleteDisabledAlertConfig(ctx, name).
		XScopeOrgID(fmt.Sprintf("%d", stackID))

	_, err = request.Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to delete disabled alert configuration: %w", err))
	}

	return nil
}
