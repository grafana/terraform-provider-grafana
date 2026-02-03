package asserts_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
)

// TestAccResourceStack_readOnly tests reading an existing stack without modifying it.
//
// This test uses the same pattern as other Asserts tests - it operates on
// an existing Grafana Cloud stack configured via environment variables.
//
// Required environment variables (same as other Cloud Instance tests):
//   - TF_ACC_CLOUD_INSTANCE=1
//   - GRAFANA_URL: The Grafana instance URL (e.g., https://mystack.grafana.net)
//   - GRAFANA_AUTH: Authentication token for the Grafana API
//   - GRAFANA_CLOUD_PROVIDER_TEST_STACK_ID: The stack ID
//
// IMPORTANT: This test only reads the stack status. It does NOT enable or disable
// the stack, as the stack is expected to already be enabled in the test environment.
func TestAccResourceStack_readOnly(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	client := testutils.Provider.Meta().(*common.Client).AssertsAPIClient
	stackID := fmt.Sprintf("%d", testutils.Provider.Meta().(*common.Client).GrafanaStackID)
	ctx := context.Background()

	// Verify we can read the stack status via the API
	request := client.StackControllerAPI.GetStatus(ctx).XScopeOrgID(stackID)
	stackStatus, _, err := request.Execute()

	if err != nil {
		t.Skipf("Stack not configured for Asserts, skipping read test: %v", err)
	}

	// Verify the stack is enabled
	if !stackStatus.GetEnabled() {
		t.Errorf("Expected stack to be enabled, but got enabled=%v", stackStatus.GetEnabled())
	}

	t.Logf("Stack status: enabled=%v, status=%s, version=%d",
		stackStatus.GetEnabled(),
		stackStatus.GetStatus(),
		stackStatus.GetVersion())
}
