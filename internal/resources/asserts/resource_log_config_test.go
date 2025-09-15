package asserts_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// TestAccAssertsLogConfig_readOnly tests the GET endpoint functionality
// This test assumes a log config already exists in the test environment
func TestAccAssertsLogConfig_readOnly(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	stackID := getTestStackID(t)
	rName := "test-read-only" // Use a fixed name for read-only testing

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Test that we can read existing log configs
				Config: testAccAssertsLogConfigReadOnlyConfig(stackID, rName),
				Check: resource.ComposeTestCheckFunc(
					// Just verify the resource can be read without errors
					resource.TestCheckResourceAttr("data.grafana_asserts_log_config.test", "name", rName),
				),
			},
		},
	})
}

func testAccAssertsLogConfigReadOnlyConfig(stackID int64, name string) string {
	return fmt.Sprintf(`
data "grafana_asserts_log_config" "test" {
  name = "%s"
}
`, name)
}

// TestAccAssertsLogConfig_lister tests the lister functionality
func TestAccAssertsLogConfig_lister(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	stackID := getTestStackID(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Test that the lister can discover existing log configs
				Config: testAccAssertsLogConfigListerConfig(stackID),
				Check: resource.ComposeTestCheckFunc(
					// Verify lister works without errors
					resource.TestCheckResourceAttrSet("data.grafana_asserts_log_configs.all", "ids.#"),
				),
			},
		},
	})
}

func testAccAssertsLogConfigListerConfig(stackID int64) string {
	return `
data "grafana_asserts_log_configs" "all" {
}
`
}