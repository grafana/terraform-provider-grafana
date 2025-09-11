package asserts

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	assertsapi "github.com/grafana/grafana-asserts-public-clients/go/gcom"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

// makeResourceThresholds creates the grafana_asserts_thresholds resource which manages
// request/resource/health thresholds through Asserts Thresholds bulk endpoints.
// Note: This is a placeholder implementation that will need to be updated when the
// Thresholds API becomes available in the asserts client.
func makeResourceThresholds() *common.Resource {
	sch := &schema.Resource{
		Description: "Manages Asserts Thresholds configuration (request, resource, health) via bulk endpoints. Note: This resource is currently a placeholder and will be implemented when the Thresholds API becomes available.",

		CreateContext: resourceThresholdsUpsert,
		ReadContext:   resourceThresholdsRead,
		UpdateContext: resourceThresholdsUpsert,
		DeleteContext: resourceThresholdsDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			// Request thresholds
			"request_thresholds": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "List of request thresholds.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"entity_name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Entity name the threshold applies to.",
						},
						"assertion_name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Assertion name (e.g., RequestRateAnomaly, ErrorRatioBreach).",
						},
						"request_type": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Request type (e.g., inbound/outbound).",
						},
						"request_context": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Request context (e.g., path or context identifier).",
						},
						"value": {
							Type:        schema.TypeFloat,
							Required:    true,
							Description: "Threshold value.",
						},
					},
				},
			},

			// Resource thresholds
			"resource_thresholds": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "List of resource thresholds.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"assertion_name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Assertion name (e.g., Saturation, ResourceRateBreach).",
						},
						"resource_type": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Resource type (e.g., container/pod/node).",
						},
						"container_name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Container name.",
						},
						"source": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Data source for the threshold (e.g., metrics/logs).",
						},
						"severity": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Severity (warning or critical).",
						},
						"value": {
							Type:        schema.TypeFloat,
							Required:    true,
							Description: "Threshold value.",
						},
					},
				},
			},

			// Health thresholds
			"health_thresholds": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "List of health thresholds.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"assertion_name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Assertion name.",
						},
						"expression": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Prometheus expression.",
						},
					},
				},
			},
		},
	}

	return common.NewLegacySDKResource(
		common.CategoryAsserts,
		"grafana_asserts_thresholds",
		// Singleton-ish resource; use a fixed ID
		common.NewResourceID(common.StringIDField("id")),
		sch,
	).WithLister(assertsListerFunction(listThresholdsSingleton))
}

// listThresholdsSingleton returns a single synthetic ID to enable sweeping/lister checks.
func listThresholdsSingleton(ctx context.Context, client *assertsapi.APIClient, stackID string) ([]string, error) {
	// We could check if any custom rules exist; for simplicity always expose the singleton ID.
	return []string{"custom_thresholds"}, nil
}

func resourceThresholdsUpsert(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// TODO: Implement when Thresholds API becomes available
	// For now, this is a placeholder that stores the configuration in state
	// but doesn't actually interact with the Asserts API

	// Set a stable ID
	d.SetId("custom_thresholds")

	// Store the configuration in state (this will be read back in the Read function)
	return resourceThresholdsRead(ctx, d, meta)
}

func resourceThresholdsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// TODO: Implement when Thresholds API becomes available
	// For now, this just ensures the resource exists in state

	if d.Id() == "" {
		d.SetId("custom_thresholds")
	}

	return nil
}

func resourceThresholdsDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// TODO: Implement when Thresholds API becomes available
	// For now, this is a no-op since we're not actually managing any external resources

	return nil
}
