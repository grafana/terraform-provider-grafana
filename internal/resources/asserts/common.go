package asserts

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	assertsapi "github.com/grafana/grafana-asserts-public-clients/go/gcom"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

// TerraformManagedBy is the value used to mark resources as managed by Terraform.
// This constant is exported for use in tests.
const TerraformManagedBy = "terraform"

// getManagedByTerraform returns a pointer to the Terraform managed-by string.
// This is used to set the managedBy field on Asserts resources to indicate
// they are managed by Terraform (as opposed to the UI or other sources).
// Use this for DTOs that expect *string (e.g., AlertConfigDto, DisabledAlertConfigDto).
func getManagedByTerraform() *string {
	s := TerraformManagedBy
	return &s
}

// getManagedByTerraformValue returns the Terraform managed-by string value.
// Use this for DTOs with SetManagedBy methods that expect string values
// (e.g., LogDrilldownConfigDto, threshold DTOs).
func getManagedByTerraformValue() string {
	return TerraformManagedBy
}

// validateAssertsClient checks if the Asserts API client is properly configured
func validateAssertsClient(meta interface{}) (*assertsapi.APIClient, int64, diag.Diagnostics) {
	client := meta.(*common.Client).AssertsAPIClient
	if client == nil {
		return nil, 0, diag.Errorf("Asserts API client is not configured")
	}

	stackID := meta.(*common.Client).GrafanaStackID
	if stackID == 0 {
		return nil, 0, diag.Errorf("stack_id must be set in provider configuration for Asserts resources")
	}

	return client, stackID, nil
}

// retryReadFunc is a function that performs a read operation with retry logic
type retryReadFunc func(retryCount, maxRetries int) *retry.RetryError

// withRetryRead wraps a read operation with consistent retry logic and exponential backoff
func withRetryRead(ctx context.Context, operation retryReadFunc) error {
	retryCount := 0
	maxRetries := 40

	// Increase overall timeout to better handle eventual consistency when
	// multiple resources are created concurrently (e.g., stress tests)
	return retry.RetryContext(ctx, 600*time.Second, func() *retry.RetryError {
		retryCount++

		// Backoff with jitter to reduce request stampeding
		var baseSleep time.Duration
		if retryCount == 1 {
			baseSleep = 1 * time.Second
		} else {
			// Exponential backoff: 1s, 2s, 4s, 8s, 16s (capped at 16s)
			baseSleep = time.Duration(1<<int(math.Min(float64(retryCount-2), 4))) * time.Second
		}

		// Apply jitter: sleep in [base/2, base]
		minSleep := baseSleep / 2
		maxJitter := baseSleep - minSleep
		if maxJitter > 0 {
			//nolint:gosec // Using math/rand for jitter in retry logic, not cryptographic purposes
			j := time.Duration(rand.Int63n(int64(maxJitter)))
			time.Sleep(minSleep + j)
		} else {
			time.Sleep(baseSleep)
		}

		// Execute the operation with retry count
		return operation(retryCount, maxRetries)
	})
}

// createRetryableError creates a retryable error with consistent formatting
func createRetryableError(resourceType, resourceName string, retryCount, maxRetries int) *retry.RetryError {
	return retry.RetryableError(fmt.Errorf("%s %s not found (attempt %d/%d)", resourceType, resourceName, retryCount, maxRetries))
}

// createNonRetryableError creates a non-retryable error with consistent formatting
func createNonRetryableError(resourceType, resourceName string, retryCount int) *retry.RetryError {
	return retry.NonRetryableError(fmt.Errorf("%s %s not found after %d retries - may indicate a permanent issue", resourceType, resourceName, retryCount))
}

// createAPIError creates a retryable or non-retryable API error based on retry count
func createAPIError(operation string, retryCount, maxRetries int, err error) *retry.RetryError {
	if retryCount >= maxRetries {
		return retry.NonRetryableError(fmt.Errorf("failed to %s after %d retries: %w", operation, retryCount, err))
	}
	return retry.RetryableError(fmt.Errorf("failed to %s: %w", operation, err))
}

// formatAPIError extracts detailed error information from API errors.
// When the OpenAPI client fails to parse error responses (e.g., oneOf schema mismatch),
// this function extracts the raw response body to provide more context.
func formatAPIError(operation string, err error) error {
	if err == nil {
		return nil
	}

	// Check if the error is a GenericOpenAPIError with a raw body
	if apiErr, ok := err.(*assertsapi.GenericOpenAPIError); ok {
		body := apiErr.Body()
		if len(body) > 0 {
			// If the error message contains "oneOf" parsing issues, include the raw body
			errMsg := err.Error()
			if strings.Contains(errMsg, "oneOf") || strings.Contains(errMsg, "failed to match schemas") {
				return fmt.Errorf("%s: %s (raw response: %s)", operation, errMsg, string(body))
			}
			// For other errors, still include the body for context
			return fmt.Errorf("%s: %s (response: %s)", operation, errMsg, string(body))
		}
	}

	return fmt.Errorf("%s: %w", operation, err)
}

// stringSliceToInterface converts a slice of strings to a slice of interfaces for Terraform schema
func stringSliceToInterface(items []string) []interface{} {
	result := make([]interface{}, 0, len(items))
	for _, v := range items {
		result = append(result, v)
	}
	return result
}

// getMatchRulesSchema returns the common schema definition for match rules used across drilldown configs
func getMatchRulesSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"property": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "Entity property to match.",
		},
		"op": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "Operation to use for matching. One of: =, <>, <, >, <=, >=, IS NULL, IS NOT NULL, STARTS WITH, CONTAINS.",
			ValidateFunc: validation.StringInSlice([]string{
				"=", "<>", "<", ">", "<=", ">=", "IS NULL", "IS NOT NULL", "STARTS WITH", "CONTAINS",
			}, false),
		},
		"values": {
			Type:        schema.TypeList,
			Required:    true,
			Description: "Values to match against.",
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
	}
}

// buildMatchRules converts Terraform schema match data to PropertyMatchEntryDto slice
func buildMatchRules(matchData interface{}) []assertsapi.PropertyMatchEntryDto {
	if matchData == nil {
		return nil
	}

	matchList := matchData.([]interface{})
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

	return matches
}

// matchRulesToSchemaData converts PropertyMatchEntryDto slice to Terraform schema format
func matchRulesToSchemaData(matches []assertsapi.PropertyMatchEntryDto) []map[string]interface{} {
	if len(matches) == 0 {
		return nil
	}

	matchRules := make([]map[string]interface{}, 0, len(matches))
	for _, match := range matches {
		rule := map[string]interface{}{
			"property": match.GetProperty(),
			"op":       match.GetOp(),
			"values":   stringSliceToInterface(match.GetValues()),
		}
		matchRules = append(matchRules, rule)
	}
	return matchRules
}
