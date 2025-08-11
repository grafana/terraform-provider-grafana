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

func makeResourceLogDrilldownConfig() *common.Resource {
	sch := &schema.Resource{
		Description: "Manages Asserts Log Drilldown Environment Configuration through the Grafana API.",

		CreateContext: resourceLogDrilldownConfigCreate,
		ReadContext:   resourceLogDrilldownConfigRead,
		UpdateContext: resourceLogDrilldownConfigUpdate,
		DeleteContext: resourceLogDrilldownConfigDelete,

		Importer: &schema.ResourceImporter{StateContext: schema.ImportStatePassthroughContext},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The environment name for the log drilldown configuration.",
			},
			"config": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The environment configuration (EnvironmentDto), in YAML format.",
			},
		},
	}

	return common.NewLegacySDKResource(
		common.CategoryAsserts,
		"grafana_asserts_log_drilldown_config",
		common.NewResourceID(common.StringIDField("name")),
		sch,
	)
}

func resourceLogDrilldownConfigCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, stackID, diags := validateAssertsClient(meta)
	if diags.HasError() {
		return diags
	}

	name := d.Get("name").(string)
	cfgYAML := d.Get("config").(string)

	var env assertsapi.EnvironmentDto
	if err := yaml.Unmarshal([]byte(cfgYAML), &env); err != nil {
		return diag.FromErr(fmt.Errorf("failed to unmarshal environment YAML: %w", err))
	}
	// Ensure name is set from resource name
	env.SetName(name)

	req := client.LogConfigControllerAPI.UpsertEnvironmentConfig(ctx).
		EnvironmentDto(env).
		XScopeOrgID(fmt.Sprintf("%d", stackID))
	_, err := req.Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to upsert log environment config: %w", err))
	}

	d.SetId(name)
	return resourceLogDrilldownConfigRead(ctx, d, meta)
}

func resourceLogDrilldownConfigRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, stackID, diags := validateAssertsClient(meta)
	if diags.HasError() {
		return diags
	}
	name := d.Id()

	var tenantCfg *assertsapi.TenantEnvConfigResponseDto
	err := withRetryRead(ctx, func(retryCount, maxRetries int) *retry.RetryError {
		resp, _, err := client.LogConfigControllerAPI.GetTenantEnvConfig(ctx).
			XScopeOrgID(fmt.Sprintf("%d", stackID)).
			Execute()
		if err != nil {
			return createAPIError("get tenant environment log config", retryCount, maxRetries, err)
		}
		tenantCfg = resp
		return nil
	})
	if err != nil {
		return diag.FromErr(err)
	}

	if tenantCfg == nil {
		d.SetId("")
		return nil
	}

	found := false
	for _, env := range tenantCfg.GetEnvironments() {
		if env.GetName() == name {
			found = true
			break
		}
	}
	if !found {
		d.SetId("")
		return nil
	}

	_ = d.Set("name", name)
	if v := d.Get("config").(string); v != "" {
		_ = d.Set("config", v)
	}
	return nil
}

func resourceLogDrilldownConfigUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, stackID, diags := validateAssertsClient(meta)
	if diags.HasError() {
		return diags
	}

	name := d.Get("name").(string)
	cfgYAML := d.Get("config").(string)

	var env assertsapi.EnvironmentDto
	if err := yaml.Unmarshal([]byte(cfgYAML), &env); err != nil {
		return diag.FromErr(fmt.Errorf("failed to unmarshal environment YAML: %w", err))
	}
	env.SetName(name)

	req := client.LogConfigControllerAPI.UpsertEnvironmentConfig(ctx).
		EnvironmentDto(env).
		XScopeOrgID(fmt.Sprintf("%d", stackID))
	_, err := req.Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to upsert log environment config: %w", err))
	}

	return resourceLogDrilldownConfigRead(ctx, d, meta)
}

func resourceLogDrilldownConfigDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, stackID, diags := validateAssertsClient(meta)
	if diags.HasError() {
		return diags
	}
	name := d.Id()

	req := client.LogConfigControllerAPI.DeleteConfig(ctx, name).
		XScopeOrgID(fmt.Sprintf("%d", stackID))
	_, err := req.Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to delete log environment config: %w", err))
	}
	return nil
}
