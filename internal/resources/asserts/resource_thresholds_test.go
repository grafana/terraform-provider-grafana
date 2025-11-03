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

// NOTE: Threshold tests use resource.Test (not ParallelTest) because the thresholds resource
// is a singleton (all tests share the same ID "custom_thresholds"). Running tests in parallel
// would cause them to interfere with each other.

// TestAccAssertsThresholds_basic tests the basic functionality of the thresholds resource.
func TestAccAssertsThresholds_basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	stackID := getTestStackID(t)
	rName := fmt.Sprintf("test-thresholds-%s", acctest.RandString(6))

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccAssertsThresholdsCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAssertsThresholdsConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccAssertsThresholdsCheckExists(stackID),
					resource.TestCheckResourceAttr("grafana_asserts_thresholds.test", "id", "custom_thresholds"),
					resource.TestCheckResourceAttr("grafana_asserts_thresholds.test", "request_thresholds.0.assertion_name", "ErrorRatioBreach"),
					resource.TestCheckResourceAttr("grafana_asserts_thresholds.test", "request_thresholds.0.entity_name", fmt.Sprintf("svc-%s", rName)),
					resource.TestCheckResourceAttr("grafana_asserts_thresholds.test", "request_thresholds.0.value", "0.01"),
					resource.TestCheckResourceAttr("grafana_asserts_thresholds.test", "resource_thresholds.0.assertion_name", "Saturation"),
					resource.TestCheckResourceAttr("grafana_asserts_thresholds.test", "resource_thresholds.0.severity", "warning"),
					resource.TestCheckResourceAttr("grafana_asserts_thresholds.test", "resource_thresholds.0.value", "75"),
					resource.TestCheckResourceAttr("grafana_asserts_thresholds.test", "health_thresholds.0.assertion_name", rName),
					resource.TestCheckResourceAttr("grafana_asserts_thresholds.test", "health_thresholds.0.entity_type", "Service"),
					resource.TestCheckResourceAttr("grafana_asserts_thresholds.test", "health_thresholds.0.alert_category", "error"),
					testutils.CheckLister("grafana_asserts_thresholds.test"),
				),
			},
			{
				ResourceName:      "grafana_asserts_thresholds.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccAssertsThresholds_update tests updating thresholds.
func TestAccAssertsThresholds_update(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	stackID := getTestStackID(t)
	rName := fmt.Sprintf("test-update-%s", acctest.RandString(6))

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccAssertsThresholdsCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAssertsThresholdsConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccAssertsThresholdsCheckExists(stackID),
					resource.TestCheckResourceAttr("grafana_asserts_thresholds.test", "request_thresholds.0.value", "0.01"),
					resource.TestCheckResourceAttr("grafana_asserts_thresholds.test", "resource_thresholds.0.severity", "warning"),
				),
			},
			{
				Config: testAccAssertsThresholdsConfigUpdated(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccAssertsThresholdsCheckExists(stackID),
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

// TestAccAssertsThresholds_minimal tests thresholds with only one threshold type.
func TestAccAssertsThresholds_minimal(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	stackID := getTestStackID(t)
	rName := fmt.Sprintf("test-minimal-%s", acctest.RandString(6))

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccAssertsThresholdsCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAssertsThresholdsConfigMinimal(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccAssertsThresholdsCheckExists(stackID),
					resource.TestCheckResourceAttr("grafana_asserts_thresholds.test", "id", "custom_thresholds"),
					resource.TestCheckResourceAttr("grafana_asserts_thresholds.test", "request_thresholds.0.assertion_name", "RequestRateAnomaly"),
					resource.TestCheckResourceAttr("grafana_asserts_thresholds.test", "request_thresholds.0.value", "100"),
				),
			},
		},
	})
}

// TestAccAssertsThresholds_fullFields tests thresholds with all supported assertion types.
func TestAccAssertsThresholds_fullFields(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	stackID := getTestStackID(t)
	rName := fmt.Sprintf("test-full-%s", acctest.RandString(6))

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccAssertsThresholdsCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAssertsThresholdsConfigFull(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccAssertsThresholdsCheckExists(stackID),
					// Request thresholds - multiple assertions
					resource.TestCheckResourceAttr("grafana_asserts_thresholds.full", "request_thresholds.0.assertion_name", "ErrorRatioBreach"),
					resource.TestCheckResourceAttr("grafana_asserts_thresholds.full", "request_thresholds.1.assertion_name", "LatencyAverageBreach"),
					resource.TestCheckResourceAttr("grafana_asserts_thresholds.full", "request_thresholds.2.assertion_name", "RequestRateAnomaly"),
					// Resource thresholds - multiple assertions
					resource.TestCheckResourceAttr("grafana_asserts_thresholds.full", "resource_thresholds.0.assertion_name", "Saturation"),
					resource.TestCheckResourceAttr("grafana_asserts_thresholds.full", "resource_thresholds.0.severity", "warning"),
					resource.TestCheckResourceAttr("grafana_asserts_thresholds.full", "resource_thresholds.1.assertion_name", "ResourceMayExhaust"),
					resource.TestCheckResourceAttr("grafana_asserts_thresholds.full", "resource_thresholds.1.severity", "critical"),
					// Health thresholds
					resource.TestCheckResourceAttr("grafana_asserts_thresholds.full", "health_thresholds.0.assertion_name", fmt.Sprintf("%s-health", rName)),
					resource.TestCheckResourceAttr("grafana_asserts_thresholds.full", "health_thresholds.0.entity_type", "Service"),
					resource.TestCheckResourceAttr("grafana_asserts_thresholds.full", "health_thresholds.0.alert_category", "error"),
				),
			},
		},
	})
}

// TestAccAssertsThresholds_validation exercises schema validations for nested blocks.
func TestAccAssertsThresholds_validation(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	resource.Test(t, resource.TestCase{
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

func testAccAssertsThresholdsConfigMinimal(name string) string {
	return fmt.Sprintf(`
resource "grafana_asserts_thresholds" "test" {
  request_thresholds {
    entity_name     = "%s"
    assertion_name  = "RequestRateAnomaly"
    request_type    = "inbound"
    request_context = "/api"
    value           = 100
  }
}
`, name)
}

func testAccAssertsThresholdsConfigFull(name string) string {
	return fmt.Sprintf(`
resource "grafana_asserts_thresholds" "full" {
  request_thresholds {
    entity_name     = "%s-error"
    assertion_name  = "ErrorRatioBreach"
    request_type    = "inbound"
    request_context = "/api/error"
    value           = 0.05
  }

  request_thresholds {
    entity_name     = "%s-latency"
    assertion_name  = "LatencyAverageBreach"
    request_type    = "inbound"
    request_context = "/api/slow"
    value           = 500
  }

  request_thresholds {
    entity_name     = "%s-rate"
    assertion_name  = "RequestRateAnomaly"
    request_type    = "inbound"
    request_context = "/api/rate"
    value           = 1000
  }

  resource_thresholds {
    assertion_name = "Saturation"
    resource_type  = "cpu:usage"
    container_name = "app"
    source         = "prometheus"
    severity       = "warning"
    value          = 80
  }

  resource_thresholds {
    assertion_name = "ResourceMayExhaust"
    resource_type  = "memory:usage"
    container_name = "app"
    source         = "prometheus"
    severity       = "critical"
    value          = 90
  }

  health_thresholds {
    assertion_name = "%s-health"
    expression     = "up{service=\"%s\"} == 0"
    entity_type    = "Service"
    alert_category = "error"
  }
}
`, name, name, name, name, name)
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

func testAccAssertsThresholdsCheckExists(stackID int64) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testutils.Provider.Meta().(*common.Client).AssertsAPIClient
		ctx := context.Background()

		req := client.ThresholdsV2ConfigControllerAPI.GetThresholds(ctx).
			XScopeOrgID(fmt.Sprintf("%d", stackID))

		resp, _, err := req.Execute()
		if err != nil {
			return fmt.Errorf("error getting thresholds: %s", err)
		}

		// Verify managedBy field is set to terraform on all threshold types
		for _, threshold := range resp.GetRequestThresholds() {
			if !threshold.HasManagedBy() || threshold.GetManagedBy() != "terraform" {
				return fmt.Errorf("request threshold has invalid managedBy field (expected 'terraform', got %v)", threshold.ManagedBy)
			}
		}
		for _, threshold := range resp.GetResourceThresholds() {
			if !threshold.HasManagedBy() || threshold.GetManagedBy() != "terraform" {
				return fmt.Errorf("resource threshold has invalid managedBy field (expected 'terraform', got %v)", threshold.ManagedBy)
			}
		}
		for _, threshold := range resp.GetHealthThresholds() {
			if !threshold.HasManagedBy() || threshold.GetManagedBy() != "terraform" {
				return fmt.Errorf("health threshold has invalid managedBy field (expected 'terraform', got %v)", threshold.ManagedBy)
			}
		}

		return nil
	}
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
