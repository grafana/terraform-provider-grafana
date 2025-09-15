package asserts_test

import (
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// TestAccAssertsLogConfig_readOnly tests the GET endpoint functionality
// This test imports an existing log config and verifies it can be read
func TestAccAssertsLogConfig_readOnly(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	// Use a known existing log config name for testing
	rName := "test-read-only"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Import an existing log config and verify it can be read
				ResourceName:      "grafana_asserts_log_config.test",
				ImportState:       true,
				ImportStateVerify: true,
				Config:            testAccAssertsLogConfigReadOnlyConfig(rName),
			},
		},
	})
}

func testAccAssertsLogConfigReadOnlyConfig(name string) string {
	return `
resource "grafana_asserts_log_config" "test" {
  name = "` + name + `"
  config = "placeholder"
}
`
}
