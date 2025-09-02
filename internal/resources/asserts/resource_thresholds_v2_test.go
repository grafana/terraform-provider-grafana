package asserts_test

import (
	"fmt"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccAssertsThresholdsV2_basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	rName := fmt.Sprintf("test-thresholds-v2-%s", acctest.RandString(6))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAssertsThresholdsV2Config(rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_asserts_thresholds_v2.test", "id", "custom_thresholds"),
					resource.TestCheckResourceAttr("grafana_asserts_thresholds_v2.test", "request_thresholds.0.assertion_name", "ErrorRatioBreach"),
					resource.TestCheckResourceAttr("grafana_asserts_thresholds_v2.test", "resource_thresholds.0.severity", "warning"),
					resource.TestCheckResourceAttr("grafana_asserts_thresholds_v2.test", "health_thresholds.0.assertion_name", rName),
				),
			},
			{
				ResourceName:      "grafana_asserts_thresholds_v2.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccAssertsThresholdsV2ConfigUpdated(rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_asserts_thresholds_v2.test", "request_thresholds.0.value", "0.02"),
					resource.TestCheckResourceAttr("grafana_asserts_thresholds_v2.test", "resource_thresholds.0.severity", "critical"),
					resource.TestCheckResourceAttr("grafana_asserts_thresholds_v2.test", "health_thresholds.0.expression", "up == 0"),
				),
			},
		},
	})
}

func testAccAssertsThresholdsV2Config(name string) string {
	return fmt.Sprintf(`
resource "grafana_asserts_thresholds_v2" "test" {
  request_thresholds = [{
    entity_name     = "svc-%s"
    assertion_name  = "ErrorRatioBreach"
    request_type    = "inbound"
    request_context = "/login"
    value           = 0.01
  }]

  resource_thresholds = [{
    assertion_name = "Saturation"
    resource_type  = "container"
    container_name = "web"
    source         = "metrics"
    severity       = "warning"
    value          = 75
  }]

  health_thresholds = [{
    assertion_name = "%s"
    expression     = "up < 1"
  }]
}
`, name, name)
}

func testAccAssertsThresholdsV2ConfigUpdated(name string) string {
	return fmt.Sprintf(`
resource "grafana_asserts_thresholds_v2" "test" {
  request_thresholds = [{
    entity_name     = "svc-%s"
    assertion_name  = "ErrorRatioBreach"
    request_type    = "inbound"
    request_context = "/login"
    value           = 0.02
  }]

  resource_thresholds = [{
    assertion_name = "Saturation"
    resource_type  = "container"
    container_name = "web"
    source         = "metrics"
    severity       = "critical"
    value          = 80
  }]

  health_thresholds = [{
    assertion_name = "%s"
    expression     = "up == 0"
  }]
}
`, name, name)
}


