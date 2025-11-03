package asserts

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"

	assertsapi "github.com/grafana/grafana-asserts-public-clients/go/gcom"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

// terraformManagedBy is the value used to mark resources as managed by Terraform
const terraformManagedBy = "terraform"

// getManagedByTerraform returns a pointer to the Terraform managed-by string.
// This is used to set the managedBy field on Asserts resources to indicate
// they are managed by Terraform (as opposed to the UI or other sources).
func getManagedByTerraform() *string {
	s := terraformManagedBy
	return &s
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
