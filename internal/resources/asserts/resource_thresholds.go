package asserts

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	assertsapi "github.com/grafana/grafana-asserts-public-clients/go/gcom"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

// makeResourceThresholds creates the grafana_asserts_thresholds resource which manages
// request/resource/health thresholds through Knowledge Graph Thresholds V2 bulk endpoints.
func makeResourceThresholds() *common.Resource {
	sch := &schema.Resource{
		Description: "Manages Knowledge Graph Thresholds configuration (request, resource, health) via bulk endpoints.",

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
							ValidateFunc: validation.StringInSlice([]string{
								"RequestRateAnomaly",
								"ErrorRatioAnomaly",
								"ErrorRatioBreach",
								"ErrorBuildup",
								"InboundClientErrorAnomaly",
								"ErrorLogRateBreach",
								"LatencyAverageAnomaly",
								"LatencyAverageBreach",
								"LatencyP99ErrorBuildup",
								"LoggerRateAnomaly",
							}, false),
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
							ValidateFunc: validation.StringInSlice([]string{
								"Saturation",
								"ResourceRateBreach",
								"ResourceMayExhaust",
								"ResourceRateAnomaly",
							}, false),
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
							Type:         schema.TypeString,
							Required:     true,
							Description:  "Severity (warning or critical).",
							ValidateFunc: validation.StringInSlice([]string{"warning", "critical"}, false),
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
						"entity_type": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Entity type for the health threshold (e.g., Service, Pod, Namespace, Volume).",
						},
						"alert_category": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Optional alert category label for the health threshold.",
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
	client, stackID, diags := validateAssertsClient(meta)
	if diags.HasError() {
		return diags
	}

	// Build DTO from schema
	dto := buildThresholdsV2Dto(d)

	// Call bulk update endpoint
	req := client.ThresholdsV2ConfigControllerAPI.UpdateAllThresholds(ctx).
		ThresholdsV2Dto(dto).
		XScopeOrgID(fmt.Sprintf("%d", stackID))

	_, err := req.Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to apply thresholds: %w", err))
	}

	// Set a stable ID
	d.SetId("custom_thresholds")
	return resourceThresholdsRead(ctx, d, meta)
}

func resourceThresholdsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, stackID, diags := validateAssertsClient(meta)
	if diags.HasError() {
		return diags
	}

	// Retry logic for read operation to handle eventual consistency
	var resp *assertsapi.ThresholdsV2Dto
	err := withRetryRead(ctx, func(retryCount, maxRetries int) *retry.RetryError {
		// Read current thresholds
		request := client.ThresholdsV2ConfigControllerAPI.GetThresholds(ctx).
			XScopeOrgID(fmt.Sprintf("%d", stackID))

		result, _, err := request.Execute()
		if err != nil {
			return createAPIError("get thresholds", retryCount, maxRetries, err)
		}

		resp = result
		return nil
	})

	if err != nil {
		return diag.FromErr(err)
	}

	if resp == nil {
		d.SetId("")
		return nil
	}

	// Map back to state
	if err := d.Set("request_thresholds", flattenRequestThresholds(resp.GetRequestThresholds())); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("resource_thresholds", flattenResourceThresholds(resp.GetResourceThresholds())); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("health_thresholds", flattenHealthThresholds(resp.GetHealthThresholds())); err != nil {
		return diag.FromErr(err)
	}

	if d.Id() == "" {
		d.SetId("custom_thresholds")
	}
	return nil
}

func resourceThresholdsDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, stackID, diags := validateAssertsClient(meta)
	if diags.HasError() {
		return diags
	}

	// The DELETE endpoint supports two modes via the request body:
	//   1. Empty DTO with all lists nil/empty → Delete ALL custom thresholds
	//   2. DTO with specific threshold types populated → Delete only those types
	//
	// For Terraform destroy, we want to delete everything that was managed by this resource.
	// We build a DTO with the current state to tell the server what to delete.
	dto := buildThresholdsV2Dto(d)

	req := client.ThresholdsV2ConfigControllerAPI.DeleteThresholds(ctx).
		ThresholdsV2Dto(dto).
		XScopeOrgID(fmt.Sprintf("%d", stackID))

	_, err := req.Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to delete thresholds: %w", err))
	}

	return nil
}

func flattenRequestThresholds(in []assertsapi.RequestThresholdV2Dto) []map[string]interface{} {
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

func flattenResourceThresholds(in []assertsapi.ResourceThresholdV2Dto) []map[string]interface{} {
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

func flattenHealthThresholds(in []assertsapi.HealthThresholdV2Dto) []map[string]interface{} {
	if len(in) == 0 {
		return nil
	}
	out := make([]map[string]interface{}, 0, len(in))
	for _, v := range in {
		m := map[string]interface{}{
			"assertion_name": v.GetAssertionName(),
			"expression":     v.GetExpression(),
		}
		if et := v.GetEntityType(); et != "" {
			m["entity_type"] = et
		}
		if ac := v.GetAlertCategory(); ac != "" {
			m["alert_category"] = ac
		}
		out = append(out, m)
	}
	return out
}

// buildThresholdsV2Dto constructs a ThresholdsV2Dto from the Terraform schema.
// This is used for both create/update and delete operations.
func buildThresholdsV2Dto(d *schema.ResourceData) assertsapi.ThresholdsV2Dto {
	dto := assertsapi.ThresholdsV2Dto{}

	// Build request thresholds
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
			r.SetManagedBy(getManagedByTerraformValue())
			reqs = append(reqs, r)
		}
		dto.SetRequestThresholds(reqs)
	}

	// Build resource thresholds
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
			r.SetManagedBy(getManagedByTerraformValue())
			ress = append(ress, r)
		}
		dto.SetResourceThresholds(ress)
	}

	// Build health thresholds
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
			if s, ok := m["entity_type"].(string); ok {
				h.SetEntityType(s)
			}
			if s, ok := m["alert_category"].(string); ok && s != "" {
				h.SetAlertCategory(s)
			}
			h.SetManagedBy(getManagedByTerraformValue())
			healths = append(healths, h)
		}
		dto.SetHealthThresholds(healths)
	}

	return dto
}
