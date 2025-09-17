package asserts_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccAssertsThresholdsV2_basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	stackID := getTestStackID(t)
	rName := fmt.Sprintf("test-thresholds-v2-%s", acctest.RandString(6))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccAssertsThresholdsV2CheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAssertsThresholdsV2Config(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccAssertsThresholdsV2CheckExists("grafana_asserts_thresholds_v2.test", stackID, rName),
					resource.TestCheckResourceAttr("grafana_asserts_thresholds_v2.test", "id", "custom_thresholds"),
					resource.TestCheckResourceAttr("grafana_asserts_thresholds_v2.test", "request_thresholds.0.assertion_name", "ErrorRatioBreach"),
					resource.TestCheckResourceAttr("grafana_asserts_thresholds_v2.test", "resource_thresholds.0.severity", "warning"),
					resource.TestCheckResourceAttr("grafana_asserts_thresholds_v2.test", "health_thresholds.0.assertion_name", rName),
					testutils.CheckLister("grafana_asserts_thresholds_v2.test"),
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

func TestAccAssertsThresholdsV2_minimal(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	stackID := getTestStackID(t)
	rName := fmt.Sprintf("test-thresholds-v2-min-%s", acctest.RandString(6))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccAssertsThresholdsV2CheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAssertsThresholdsV2ConfigMinimal(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccAssertsThresholdsV2CheckExists("grafana_asserts_thresholds_v2.test", stackID, rName),
					resource.TestCheckResourceAttr("grafana_asserts_thresholds_v2.test", "id", "custom_thresholds"),
					resource.TestCheckResourceAttr("grafana_asserts_thresholds_v2.test", "health_thresholds.0.assertion_name", rName),
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

func testAccAssertsThresholdsV2ConfigMinimal(name string) string {
	return fmt.Sprintf(`
resource "grafana_asserts_thresholds_v2" "test" {
  health_thresholds = [{
    assertion_name = "%s"
    expression     = "up < 1"
  }]
}
`, name)
}

func testAccAssertsThresholdsV2CheckExists(rn string, stackID int64, name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s\n %#v", rn, s.RootModule().Resources)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		client := testutils.Provider.Meta().(*common.Client).AssertsAPIClient
		ctx := context.Background()

		// Get current thresholds
		request := client.ThresholdsV2ConfigControllerAPI.GetThresholds(ctx).
			XScopeOrgID(fmt.Sprintf("%d", stackID))

		thresholds, _, err := request.Execute()
		if err != nil {
			return fmt.Errorf("error getting thresholds v2: %s", err)
		}

		// Verify our health threshold exists by assertion name
		for _, h := range thresholds.GetHealthThresholds() {
			if h.GetAssertionName() == name {
				return nil
			}
		}

		return fmt.Errorf("thresholds v2 health assertion %s not found", name)
	}
}

func testAccAssertsThresholdsV2CheckDestroy(s *terraform.State) error {
	client := testutils.Provider.Meta().(*common.Client).AssertsAPIClient
	ctx := context.Background()

	deadline := time.Now().Add(60 * time.Second)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "grafana_asserts_thresholds_v2" {
			continue
		}

		stackID := fmt.Sprintf("%d", testutils.Provider.Meta().(*common.Client).GrafanaStackID)

		for {
			request := client.ThresholdsV2ConfigControllerAPI.GetThresholds(ctx).
				XScopeOrgID(stackID)

			thresholds, _, err := request.Execute()
			if err != nil {
				if common.IsNotFoundError(err) {
					break
				}
				return fmt.Errorf("error checking thresholds v2 destruction: %s", err)
			}

			// Consider destroyed if all lists are empty or nil
			reqEmpty := thresholds == nil || len(thresholds.GetRequestThresholds()) == 0
			resEmpty := thresholds == nil || len(thresholds.GetResourceThresholds()) == 0
			healthEmpty := thresholds == nil || len(thresholds.GetHealthThresholds()) == 0

			if reqEmpty && resEmpty && healthEmpty {
				break
			}

			if time.Now().After(deadline) {
				return fmt.Errorf("thresholds v2 still present")
			}
			time.Sleep(2 * time.Second)
		}
	}

	return nil
}
