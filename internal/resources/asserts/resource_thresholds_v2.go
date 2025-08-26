package asserts

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	assertsapi "github.com/grafana/grafana-asserts-public-clients/go/gcom"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

// makeResourceThresholdsV2 creates the grafana_asserts_thresholds_v2 resource which manages
// request/resource/health thresholds through Asserts Thresholds V2 bulk endpoints.
func makeResourceThresholdsV2() *common.Resource {
	sch := &schema.Resource{
		Description: "Manages Asserts Thresholds V2 configuration (request, resource, health) via bulk endpoints.",

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
	client, stackID, diags := validateAssertsClient(meta)
	if diags.HasError() {
		return diags
	}

	// Build DTO from schema
	dto := assertsapi.ThresholdsV2Dto{}

	if v, ok := d.GetOk("request_thresholds"); ok {
		items := v.([]interface{})
		reqs := make([]assertsapi.RequestThresholdV2Dto, 0, len(items))
		for _, it := range items {
			m := it.(map[string]interface{})
			r := assertsapi.RequestThresholdV2Dto{}
			if s, ok := m["entity_name"].(string); ok {
				r.SetEntityName(s)
			}
			if s, ok := m["assertion_name"].(string); ok {
				r.SetAssertionName(s)
			}
			if s, ok := m["request_type"].(string); ok {
				r.SetRequestType(s)
			}
			if s, ok := m["request_context"].(string); ok {
				r.SetRequestContext(s)
			}
			if f, ok := m["value"].(float64); ok {
				r.SetValue(f)
			}
			reqs = append(reqs, r)
		}
		dto.SetRequestThresholds(reqs)
	}

	if v, ok := d.GetOk("resource_thresholds"); ok {
		items := v.([]interface{})
		ress := make([]assertsapi.ResourceThresholdV2Dto, 0, len(items))
		for _, it := range items {
			m := it.(map[string]interface{})
			r := assertsapi.ResourceThresholdV2Dto{}
			if s, ok := m["assertion_name"].(string); ok {
				r.SetAssertionName(s)
			}
			if s, ok := m["resource_type"].(string); ok {
				r.SetResourceType(s)
			}
			if s, ok := m["container_name"].(string); ok {
				r.SetContainerName(s)
			}
			if s, ok := m["source"].(string); ok {
				r.SetSource(s)
			}
			if s, ok := m["severity"].(string); ok {
				r.SetSeverity(s)
			}
			if f, ok := m["value"].(float64); ok {
				r.SetValue(f)
			}
			ress = append(ress, r)
		}
		dto.SetResourceThresholds(ress)
	}

	if v, ok := d.GetOk("health_thresholds"); ok {
		items := v.([]interface{})
		healths := make([]assertsapi.HealthThresholdV2Dto, 0, len(items))
		for _, it := range items {
			m := it.(map[string]interface{})
			h := assertsapi.HealthThresholdV2Dto{}
			if s, ok := m["assertion_name"].(string); ok {
				h.SetAssertionName(s)
			}
			if s, ok := m["expression"].(string); ok {
				h.SetExpression(s)
			}
			healths = append(healths, h)
		}
		dto.SetHealthThresholds(healths)
	}

	// Call bulk update endpoint
	req := client.ThresholdsV2ConfigControllerAPI.UpdateAllThresholds(ctx).
		ThresholdsV2Dto(dto).
		XScopeOrgID(fmt.Sprintf("%d", stackID))

	_, err := req.Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to apply thresholds v2: %w", err))
	}

	// Set a stable ID
	d.SetId("custom_thresholds")
	return resourceThresholdsV2Read(ctx, d, meta)
}

func resourceThresholdsV2Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, stackID, diags := validateAssertsClient(meta)
	if diags.HasError() {
		return diags
	}

	// Read current thresholds
	req := client.ThresholdsV2ConfigControllerAPI.GetThresholds(ctx).
		XScopeOrgID(fmt.Sprintf("%d", stackID))

	resp, _, err := req.Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to read thresholds v2: %w", err))
	}

	if resp == nil {
		d.SetId("")
		return nil
	}

	// Map back to state
	if err := d.Set("request_thresholds", flattenRequestThresholdsV2(resp.GetRequestThresholds())); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("resource_thresholds", flattenResourceThresholdsV2(resp.GetResourceThresholds())); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("health_thresholds", flattenHealthThresholdsV2(resp.GetHealthThresholds())); err != nil {
		return diag.FromErr(err)
	}

	if d.Id() == "" {
		d.SetId("custom_thresholds")
	}
	return nil
}

func resourceThresholdsV2Delete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, stackID, diags := validateAssertsClient(meta)
	if diags.HasError() {
		return diags
	}

	// Clearing the group by sending empty lists
	empty := assertsapi.ThresholdsV2Dto{}
	req := client.ThresholdsV2ConfigControllerAPI.UpdateAllThresholds(ctx).
		ThresholdsV2Dto(empty).
		XScopeOrgID(fmt.Sprintf("%d", stackID))

	_, err := req.Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to clear thresholds v2: %w", err))
	}

	return nil
}

func flattenRequestThresholdsV2(in []assertsapi.RequestThresholdV2Dto) []map[string]interface{} {
	if len(in) == 0 {
		return nil
	}
	out := make([]map[string]interface{}, 0, len(in))
	for _, v := range in {
		m := map[string]interface{}{
			"entity_name":     v.GetEntityName(),
			"assertion_name":  v.GetAssertionName(),
			"request_type":    v.GetRequestType(),
			"request_context": v.GetRequestContext(),
			"value":           v.GetValue(),
		}
		out = append(out, m)
	}
	return out
}

func flattenResourceThresholdsV2(in []assertsapi.ResourceThresholdV2Dto) []map[string]interface{} {
	if len(in) == 0 {
		return nil
	}
	out := make([]map[string]interface{}, 0, len(in))
	for _, v := range in {
		m := map[string]interface{}{
			"assertion_name": v.GetAssertionName(),
			"resource_type":  v.GetResourceType(),
			"container_name": v.GetContainerName(),
			"source":         v.GetSource(),
			"severity":       v.GetSeverity(),
			"value":          v.GetValue(),
		}
		out = append(out, m)
	}
	return out
}

func flattenHealthThresholdsV2(in []assertsapi.HealthThresholdV2Dto) []map[string]interface{} {
	if len(in) == 0 {
		return nil
	}
	out := make([]map[string]interface{}, 0, len(in))
	for _, v := range in {
		m := map[string]interface{}{
			"assertion_name": v.GetAssertionName(),
			"expression":     v.GetExpression(),
		}
		out = append(out, m)
	}
	return out
}

// v2 client exposes getters/setters; no ptr helpers required
