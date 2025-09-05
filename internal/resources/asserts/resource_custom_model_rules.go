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
			definedBy = append(definedBy, assertsapi.PropertyRuleDto{
				Query: &query,
			})
		}

		entityType := entity["type"].(string)
		entityName := entity["name"].(string)
		entities = append(entities, assertsapi.EntityRuleDto{
			Type:      &entityType,
			Name:      &entityName,
			DefinedBy: definedBy,
		})
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
			definedBy = append(definedBy, map[string]interface{}{
				"query": query,
			})
		}

		entityType := ""
		if entity.Type != nil {
			entityType = *entity.Type
		}
		entityName := ""
		if entity.Name != nil {
			entityName = *entity.Name
		}

		entities = append(entities, map[string]interface{}{
			"type":       entityType,
			"name":       entityName,
			"defined_by": definedBy,
		})
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
