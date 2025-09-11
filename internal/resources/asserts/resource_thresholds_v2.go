package asserts

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	assertsapi "github.com/grafana/grafana-asserts-public-clients/go/gcom"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

// makeResourceThresholdsV2 creates the grafana_asserts_thresholds_v2 resource which manages
// request/resource/health thresholds through Asserts Thresholds V2 bulk endpoints.
// Note: This is a placeholder implementation that will need to be updated when the
// ThresholdsV2 API becomes available in the asserts client.
func makeResourceThresholdsV2() *common.Resource {
	sch := &schema.Resource{
		Description: "Manages Asserts Thresholds V2 configuration (request, resource, health) via bulk endpoints. Note: This resource is currently a placeholder and will be implemented when the ThresholdsV2 API becomes available.",

		CreateContext: resourceThresholdsV2Upsert,
		ReadContext:   resourceThresholdsV2Read,
		UpdateContext: resourceThresholdsV2Upsert,
		DeleteContext: resourceThresholdsV2Delete,

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
		"grafana_asserts_thresholds_v2",
		// Singleton-ish resource; use a fixed ID
		common.NewResourceID(common.StringIDField("id")),
		sch,
	).WithLister(assertsListerFunction(listThresholdsV2Singleton))
}

// listThresholdsV2Singleton returns a single synthetic ID to enable sweeping/lister checks.
func listThresholdsV2Singleton(ctx context.Context, client *assertsapi.APIClient, stackID string) ([]string, error) {
	// We could check if any custom rules exist; for simplicity always expose the singleton ID.
	return []string{"custom_thresholds"}, nil
}

func resourceThresholdsV2Upsert(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// TODO: Implement when ThresholdsV2 API becomes available
	// For now, this is a placeholder that stores the configuration in state
	// but doesn't actually interact with the Asserts API

	// Set a stable ID
	d.SetId("custom_thresholds")

	// Store the configuration in state (this will be read back in the Read function)
	return resourceThresholdsV2Read(ctx, d, meta)
}

func resourceThresholdsV2Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// TODO: Implement when ThresholdsV2 API becomes available
	// For now, this just ensures the resource exists in state

	if d.Id() == "" {
		d.SetId("custom_thresholds")
	}

	return nil
}

func resourceThresholdsV2Delete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// TODO: Implement when ThresholdsV2 API becomes available
	// For now, this is a no-op since we're not actually managing any external resources

	return nil
}
