package asserts_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// TestAccAssertsThresholds_basic tests the basic functionality of the thresholds resource.
func TestAccAssertsThresholds_basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	rName := fmt.Sprintf("test-thresholds-%s", acctest.RandString(6))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccAssertsThresholdsCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAssertsThresholdsConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_asserts_thresholds.test", "id", "custom_thresholds"),
					resource.TestCheckResourceAttr("grafana_asserts_thresholds.test", "request_thresholds.0.assertion_name", "ErrorRatioBreach"),
					resource.TestCheckResourceAttr("grafana_asserts_thresholds.test", "resource_thresholds.0.severity", "warning"),
					resource.TestCheckResourceAttr("grafana_asserts_thresholds.test", "health_thresholds.0.assertion_name", rName),
					resource.TestCheckResourceAttr("grafana_asserts_thresholds.test", "health_thresholds.0.entity_type", "Service"),
					testutils.CheckLister("grafana_asserts_thresholds.test"),
				),
			},
			{
				ResourceName:      "grafana_asserts_thresholds.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccAssertsThresholdsConfigUpdated(rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_asserts_thresholds.test", "request_thresholds.0.value", "0.02"),
					resource.TestCheckResourceAttr("grafana_asserts_thresholds.test", "resource_thresholds.0.severity", "critical"),
					resource.TestCheckResourceAttr("grafana_asserts_thresholds.test", "health_thresholds.0.expression", "up == 0"),
					resource.TestCheckResourceAttr("grafana_asserts_thresholds.test", "health_thresholds.0.entity_type", "Service"),
				),
			},
		},
	})
}

func testAccAssertsThresholdsConfig(name string) string {
	return fmt.Sprintf(`
resource "grafana_asserts_thresholds" "test" {
  request_thresholds {
    entity_name     = "svc-%s"
    assertion_name  = "ErrorRatioBreach"
    request_type    = "inbound"
    request_context = "/login"
    value           = 0.01
  }

  resource_thresholds {
    assertion_name = "Saturation"
    resource_type  = "container"
    container_name = "web"
    source         = "metrics"
    severity       = "warning"
    value          = 75
  }

  health_thresholds {
    assertion_name = "%s"
    expression     = "up < 1"
    entity_type    = "Service"
    alert_category = "error"
  }
}
`, name, name)
}

func testAccAssertsThresholdsConfigUpdated(name string) string {
	return fmt.Sprintf(`
resource "grafana_asserts_thresholds" "test" {
  request_thresholds {
    entity_name     = "svc-%s"
    assertion_name  = "ErrorRatioBreach"
    request_type    = "inbound"
    request_context = "/login"
    value           = 0.02
  }

  resource_thresholds {
    assertion_name = "Saturation"
    resource_type  = "container"
    container_name = "web"
    source         = "metrics"
    severity       = "critical"
    value          = 80
  }

  health_thresholds {
    assertion_name = "%s"
    expression     = "up == 0"
    entity_type    = "Service"
    alert_category = "error"
  }
}
`, name, name)
}

// TestAccAssertsThresholds_validation exercises schema validations for nested blocks.
func TestAccAssertsThresholds_validation(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Invalid request assertion_name should fail validation
				Config:      testAccAssertsThresholdsConfigInvalidRequest(),
				ExpectError: regexpMust("assertion_name|one of"),
			},
			{
				// Invalid resource severity should fail validation
				Config:      testAccAssertsThresholdsConfigInvalidResourceSeverity(),
				ExpectError: regexpMust("severity|one of"),
			},
		},
	})
}

func testAccAssertsThresholdsConfigInvalidRequest() string {
	return `
resource "grafana_asserts_thresholds" "test" {
  request_thresholds {
    entity_name     = "svc-invalid"
    assertion_name  = "NotARealAssertion"
    request_type    = "inbound"
    request_context = "/path"
    value           = 0.01
  }
}
`
}

func testAccAssertsThresholdsConfigInvalidResourceSeverity() string {
	return `
resource "grafana_asserts_thresholds" "test" {
  resource_thresholds {
    assertion_name = "Saturation"
    resource_type  = "container"
    container_name = "web"
    source         = "metrics"
    severity       = "not-valid"
    value          = 75
  }
}
`
}

// regexpMust compiles a regex and panics if it fails; keeps test definitions concise.
func regexpMust(expr string) *regexp.Regexp {
	r, err := regexp.Compile(expr)
	if err != nil {
		panic(err)
	}
	return r
}

// testAccAssertsThresholdsCheckDestroy verifies that clearing the thresholds removes custom rules.
func testAccAssertsThresholdsCheckDestroy(s *terraform.State) error {
	client := testutils.Provider.Meta().(*common.Client).AssertsAPIClient
	ctx := context.Background()

	deadline := time.Now().Add(60 * time.Second)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "grafana_asserts_thresholds" {
			continue
		}

		stackID := fmt.Sprintf("%d", testutils.Provider.Meta().(*common.Client).GrafanaStackID)

		for {
			req := client.ThresholdsV2ConfigControllerAPI.GetThresholds(ctx).
				XScopeOrgID(stackID)

			resp, _, err := req.Execute()
			if err != nil {
				if common.IsNotFoundError(err) {
					break
				}
				return fmt.Errorf("error checking thresholds destruction: %s", err)
			}

			// Consider destroyed when all lists are nil or empty
			if (resp.GetRequestThresholds() == nil || len(resp.GetRequestThresholds()) == 0) &&
				(resp.GetResourceThresholds() == nil || len(resp.GetResourceThresholds()) == 0) &&
				(resp.GetHealthThresholds() == nil || len(resp.GetHealthThresholds()) == 0) {
				break
			}

			if time.Now().After(deadline) {
				return fmt.Errorf("thresholds still present after delete")
			}
			time.Sleep(2 * time.Second)
		}
	}
	return nil
}
