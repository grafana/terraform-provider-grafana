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

// convertTerraformToModelRules converts Terraform structured data to ModelRulesDto
func convertTerraformToModelRules(d *schema.ResourceData) (*assertsapi.ModelRulesDto, error) {
	rulesList := d.Get("rules").([]interface{})
	if len(rulesList) == 0 {
		return nil, fmt.Errorf("rules block is required")
	}

	rulesData := rulesList[0].(map[string]interface{})
	entitiesList := rulesData["entity"].([]interface{})

	var entities []assertsapi.EntityRuleDto
	for _, entityData := range entitiesList {
		entity := entityData.(map[string]interface{})

		definedByList := entity["defined_by"].([]interface{})
		var definedBy []assertsapi.PropertyRuleDto
		for _, definedByData := range definedByList {
			definedByItem := definedByData.(map[string]interface{})
			query := definedByItem["query"].(string)

			propertyRule := assertsapi.PropertyRuleDto{
				Query: &query,
			}

			// Handle optional fields
			if disabled, ok := definedByItem["disabled"].(bool); ok {
				propertyRule.Disabled = &disabled
			}

			// Handle labelValues map
			if labelValues, ok := definedByItem["label_values"].(map[string]interface{}); ok && len(labelValues) > 0 {
				labelValuesMap := make(map[string]string)
				for k, v := range labelValues {
					if str, ok := v.(string); ok {
						labelValuesMap[k] = str
					}
				}
				if len(labelValuesMap) > 0 {
					propertyRule.LabelValues = labelValuesMap
				}
			}

			// Handle literals map
			if literals, ok := definedByItem["literals"].(map[string]interface{}); ok && len(literals) > 0 {
				literalsMap := make(map[string]string)
				for k, v := range literals {
					if str, ok := v.(string); ok {
						literalsMap[k] = str
					}
				}
				if len(literalsMap) > 0 {
					propertyRule.Literals = literalsMap
				}
			}

			definedBy = append(definedBy, propertyRule)
		}

		entityType := entity["type"].(string)
		entityName := entity["name"].(string)

		entityRule := assertsapi.EntityRuleDto{
			Type:      &entityType,
			Name:      &entityName,
			DefinedBy: definedBy,
		}

		// Handle optional entity fields
		if scope, ok := entity["scope"].(map[string]interface{}); ok && len(scope) > 0 {
			scopeMap := make(map[string]string)
			for k, v := range scope {
				if str, ok := v.(string); ok {
					scopeMap[k] = str
				}
			}
			if len(scopeMap) > 0 {
				entityRule.Scope = scopeMap
			}
		}

		if lookup, ok := entity["lookup"].(map[string]interface{}); ok && len(lookup) > 0 {
			lookupMap := make(map[string]string)
			for k, v := range lookup {
				if str, ok := v.(string); ok {
					lookupMap[k] = str
				}
			}
			if len(lookupMap) > 0 {
				entityRule.Lookup = lookupMap
			}
		}

		if enrichedBy, ok := entity["enriched_by"].([]interface{}); ok && len(enrichedBy) > 0 {
			var enrichedByList []assertsapi.PropertyRuleDto
			for _, item := range enrichedBy {
				if str, ok := item.(string); ok {
					enrichedByList = append(enrichedByList, assertsapi.PropertyRuleDto{
						Query: &str,
					})
				}
			}
			if len(enrichedByList) > 0 {
				entityRule.EnrichedBy = enrichedByList
			}
		}

		entities = append(entities, entityRule)
	}

	return &assertsapi.ModelRulesDto{
		Entities: entities,
	}, nil
}

// convertModelRulesToTerraform converts ModelRulesDto to Terraform structured data
func convertModelRulesToTerraform(rules *assertsapi.ModelRulesDto) ([]interface{}, error) {
	if rules == nil || rules.Entities == nil {
		return []interface{}{}, nil
	}

	var entities []interface{}
	for _, entity := range rules.Entities {
		var definedBy []interface{}
		for _, db := range entity.DefinedBy {
			query := ""
			if db.Query != nil {
				query = *db.Query
			}

			definedByItem := map[string]interface{}{
				"query": query,
			}

			// Add optional fields if they exist
			if db.Disabled != nil {
				definedByItem["disabled"] = *db.Disabled
			}

			if len(db.LabelValues) > 0 {
				definedByItem["label_values"] = db.LabelValues
			}

			if len(db.Literals) > 0 {
				definedByItem["literals"] = db.Literals
			}

			definedBy = append(definedBy, definedByItem)
		}

		entityType := ""
		if entity.Type != nil {
			entityType = *entity.Type
		}
		entityName := ""
		if entity.Name != nil {
			entityName = *entity.Name
		}

		entityMap := map[string]interface{}{
			"type":       entityType,
			"name":       entityName,
			"defined_by": definedBy,
		}

		// Add optional entity fields if they exist
		if len(entity.Scope) > 0 {
			entityMap["scope"] = entity.Scope
		}

		if len(entity.Lookup) > 0 {
			entityMap["lookup"] = entity.Lookup
		}

		if len(entity.EnrichedBy) > 0 {
			var enrichedByList []string
			for _, enrichedBy := range entity.EnrichedBy {
				if enrichedBy.Query != nil {
					enrichedByList = append(enrichedByList, *enrichedBy.Query)
				}
			}
			if len(enrichedByList) > 0 {
				entityMap["enriched_by"] = enrichedByList
			}
		}

		entities = append(entities, entityMap)
	}

	return []interface{}{
		map[string]interface{}{
			"entity": entities,
		},
	}, nil
}

func makeResourceCustomModelRules() *common.Resource {
	sch := &schema.Resource{
		Description: "Manages Asserts Custom Model Rules through the Grafana API.",

		CreateContext: resourceCustomModelRulesCreate,
		ReadContext:   resourceCustomModelRulesRead,
		UpdateContext: resourceCustomModelRulesUpdate,
		DeleteContext: resourceCustomModelRulesDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of the custom model rules.",
			},
			"rules": {
				Type:        schema.TypeList,
				Required:    true,
				MaxItems:    1,
				Description: "The rules configuration for the custom model rules.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"entity": {
							Type:        schema.TypeList,
							Required:    true,
							Description: "List of entities to define in the custom model rules.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"type": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "The type of the entity (e.g., Service, Pod, Namespace).",
									},
									"name": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "The name of the entity.",
									},
									"scope": {
										Type:        schema.TypeMap,
										Optional:    true,
										Description: "Scope labels for the entity.",
										Elem:        &schema.Schema{Type: schema.TypeString},
									},
									"lookup": {
										Type:        schema.TypeMap,
										Optional:    true,
										Description: "Lookup mappings for the entity.",
										Elem:        &schema.Schema{Type: schema.TypeString},
									},
									"enriched_by": {
										Type:        schema.TypeList,
										Optional:    true,
										Description: "List of enrichment sources for the entity.",
										Elem:        &schema.Schema{Type: schema.TypeString},
									},
									"defined_by": {
										Type:        schema.TypeList,
										Required:    true,
										Description: "List of queries that define this entity.",
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"query": {
													Type:        schema.TypeString,
													Required:    true,
													Description: "The Prometheus query that defines this entity.",
												},
												"disabled": {
													Type:        schema.TypeBool,
													Optional:    true,
													Description: "Whether this query is disabled.",
												},
												"label_values": {
													Type:        schema.TypeMap,
													Optional:    true,
													Description: "Label value mappings for the query.",
													Elem:        &schema.Schema{Type: schema.TypeString},
												},
												"literals": {
													Type:        schema.TypeMap,
													Optional:    true,
													Description: "Literal value mappings for the query.",
													Elem:        &schema.Schema{Type: schema.TypeString},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	return common.NewLegacySDKResource(
		common.CategoryAsserts,
		"grafana_asserts_custom_model_rules",
		common.NewResourceID(common.StringIDField("name")),
		sch,
	)
}

func resourceCustomModelRulesCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, stackID, diags := validateAssertsClient(meta)
	if diags.HasError() {
		return diags
	}

	name := d.Get("name").(string)

	rules, err := convertTerraformToModelRules(d)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to convert rules: %w", err))
	}

	rules.Name = &name

	req := client.CustomModelRulesControllerAPI.PutModelRules(ctx).ModelRulesDto(*rules).XScopeOrgID(fmt.Sprintf("%d", stackID))
	_, err = req.Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create custom model rules: %w", err))
	}

	d.SetId(name)

	return resourceCustomModelRulesRead(ctx, d, meta)
}

func resourceCustomModelRulesRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, stackID, diags := validateAssertsClient(meta)
	if diags.HasError() {
		return diags
	}
	name := d.Id()

	// Retry logic for read operation to handle eventual consistency
	var rules *assertsapi.ModelRulesDto
	err := withRetryRead(ctx, func(retryCount, maxRetries int) *retry.RetryError {
		req := client.CustomModelRulesControllerAPI.GetModelRules(ctx, name).XScopeOrgID(fmt.Sprintf("%d", stackID))
		rulesResult, _, err := req.Execute()
		if err != nil {
			// If the error indicates "not found", check if we should retry or give up
			if _, ok := err.(*assertsapi.GenericOpenAPIError); ok {
				if retryCount >= maxRetries {
					return createNonRetryableError("custom model rules", name, retryCount)
				}
				return createRetryableError("custom model rules", name, retryCount, maxRetries)
			}

			// Other API errors
			return createAPIError("get custom model rules", retryCount, maxRetries, err)
		}

		rules = rulesResult
		return nil
	})

	if err != nil {
		return diag.FromErr(err)
	}

	if rules == nil {
		d.SetId("")
		return nil
	}

	if rules.Name != nil {
		d.Set("name", *rules.Name)
	}

	// Convert API response to Terraform structured data
	rulesCopy := *rules
	rulesCopy.Name = nil // Don't include name in the rules structure

	terraformRules, err := convertModelRulesToTerraform(&rulesCopy)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to convert rules to Terraform format: %w", err))
	}

	d.Set("rules", terraformRules)

	return nil
}

func resourceCustomModelRulesUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, stackID, diags := validateAssertsClient(meta)
	if diags.HasError() {
		return diags
	}

	name := d.Get("name").(string)

	rules, err := convertTerraformToModelRules(d)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to convert rules: %w", err))
	}

	rules.Name = &name

	req := client.CustomModelRulesControllerAPI.PutModelRules(ctx).ModelRulesDto(*rules).XScopeOrgID(fmt.Sprintf("%d", stackID))
	_, err = req.Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to update custom model rules: %w", err))
	}

	return resourceCustomModelRulesRead(ctx, d, meta)
}

func resourceCustomModelRulesDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, stackID, diags := validateAssertsClient(meta)
	if diags.HasError() {
		return diags
	}
	name := d.Id()

	req := client.CustomModelRulesControllerAPI.DeleteModelRules(ctx, name).XScopeOrgID(fmt.Sprintf("%d", stackID))
	_, err := req.Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to delete custom model rules: %w", err))
	}

	return nil
}
