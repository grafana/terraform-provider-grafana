package asserts_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// TestAccResourceStack tests the grafana_asserts_stack resource.
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
// Additional required environment variables for Stack resource tests:
//   - GRAFANA_ASSERTS_CLOUD_ACCESS_POLICY_TOKEN: A Cloud Access Policy token
//     with scopes: stacks:read, metrics:read, metrics:write
//
// Optional:
//   - GRAFANA_ASSERTS_GRAFANA_TOKEN: A Grafana Service Account token with Admin role
//     (for testing with Grafana Managed Alerts)
//
// IMPORTANT: This test will UPDATE the stack's token configuration. The existing
// stack should have Asserts already enabled, and the tokens should be valid.
// After the test, the stack will be left in a configured state with the test tokens.
func TestAccResourceStack(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	// Get required token from environment
	capToken := os.Getenv("GRAFANA_ASSERTS_CLOUD_ACCESS_POLICY_TOKEN")
	if capToken == "" {
		t.Skip("GRAFANA_ASSERTS_CLOUD_ACCESS_POLICY_TOKEN must be set for Asserts Stack tests")
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			// Create/Update and Read
			{
				Config: testAccStackConfig(capToken),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_asserts_stack.test", "id"),
					resource.TestCheckResourceAttrSet("grafana_asserts_stack.test", "enabled"),
					resource.TestCheckResourceAttrSet("grafana_asserts_stack.test", "status"),
					resource.TestCheckResourceAttrSet("grafana_asserts_stack.test", "version"),
					testutils.CheckLister("grafana_asserts_stack.test"),
				),
			},
			// Import - tokens are write-only so we ignore them on import verification
			{
				ResourceName:            "grafana_asserts_stack.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"cloud_access_policy_token", "grafana_token"},
			},
		},
	})
}

// TestAccResourceStack_withGrafanaToken tests the stack with both tokens provided.
func TestAccResourceStack_withGrafanaToken(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	// Get required tokens from environment
	capToken := os.Getenv("GRAFANA_ASSERTS_CLOUD_ACCESS_POLICY_TOKEN")
	if capToken == "" {
		t.Skip("GRAFANA_ASSERTS_CLOUD_ACCESS_POLICY_TOKEN must be set for Asserts Stack tests")
	}

	grafanaToken := os.Getenv("GRAFANA_ASSERTS_GRAFANA_TOKEN")
	if grafanaToken == "" {
		t.Skip("GRAFANA_ASSERTS_GRAFANA_TOKEN must be set for this test")
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStackConfigWithGrafanaToken(capToken, grafanaToken),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_asserts_stack.test", "id"),
					resource.TestCheckResourceAttrSet("grafana_asserts_stack.test", "enabled"),
					resource.TestCheckResourceAttrSet("grafana_asserts_stack.test", "status"),
				),
			},
		},
	})
}

// TestAccResourceStack_readOnly tests reading an existing stack without modifying tokens.
// This test can be run safely as it only reads the stack status and doesn't perform
// any create/update/delete operations.
func TestAccResourceStack_readOnly(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	client := testutils.Provider.Meta().(*common.Client).AssertsAPIClient
	stackID := fmt.Sprintf("%d", testutils.Provider.Meta().(*common.Client).GrafanaStackID)
	ctx := context.Background()

	// Just verify we can read the stack status via the API
	request := client.StackControllerAPI.GetStatus(ctx).XScopeOrgID(stackID)
	stackStatus, _, err := request.Execute()

	if err != nil {
		t.Skipf("Stack not configured for Asserts, skipping read test: %v", err)
	}

	t.Logf("Stack status: enabled=%v, status=%s, version=%d",
		stackStatus.GetEnabled(),
		stackStatus.GetStatus(),
		stackStatus.GetVersion())
}

func testAccStackConfig(capToken string) string {
	return `
resource "grafana_asserts_stack" "test" {
  cloud_access_policy_token = "` + capToken + `"
}
`
}

func testAccStackConfigWithGrafanaToken(capToken, grafanaToken string) string {
	return `
resource "grafana_asserts_stack" "test" {
  cloud_access_policy_token = "` + capToken + `"
  grafana_token             = "` + grafanaToken + `"
}
`
}
