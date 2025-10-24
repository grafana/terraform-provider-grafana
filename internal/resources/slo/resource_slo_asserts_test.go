package slo_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/grafana/slo-openapi-client/go/slo"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccSLO_AssertsIntegration(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	resourceName := "grafana_slo.asserts_test"
	uid := "asserts-test-" + acctest.RandString(10)
	var sloObj slo.SloV00Slo

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccSloCheckDestroy(&sloObj),
		Steps: []resource.TestStep{
			{
				Config: testAccSLOAssertsConfig(uid),
				Check: resource.ComposeTestCheckFunc(
					testAccSloCheckExists(resourceName, &sloObj),
					// Verify the SLO has the correct provenance
					testAccSloCheckAssertsProvenance(resourceName),
					// Verify the SLO has the required Asserts labels
					resource.TestCheckResourceAttr(resourceName, "label.0.key", "grafana_slo_provenance"),
					resource.TestCheckResourceAttr(resourceName, "label.0.value", "asserts"),
					resource.TestCheckResourceAttr(resourceName, "label.1.key", "service_name"),
					resource.TestCheckResourceAttr(resourceName, "label.1.value", "test-service"),
					resource.TestCheckResourceAttr(resourceName, "label.2.key", "team_name"),
					resource.TestCheckResourceAttr(resourceName, "label.2.value", "test-team"),
				),
			},
			{
				Config: testAccSLOAssertsConfigUpdated(uid),
				Check: resource.ComposeTestCheckFunc(
					testAccSloCheckExists(resourceName, &sloObj),
					testAccSloCheckAssertsProvenance(resourceName),
					// Verify the SLO still has Asserts provenance after update
					resource.TestCheckResourceAttr(resourceName, "label.0.key", "grafana_slo_provenance"),
					resource.TestCheckResourceAttr(resourceName, "label.0.value", "asserts"),
				),
			},
		},
	})
}

func testAccSLOAssertsConfig(uid string) string {
	return fmt.Sprintf(`
resource "grafana_slo" "asserts_test" {
  name        = "Asserts SLO Test - %s"
  description = "Test SLO for Asserts integration"
  
  query {
    type = "ratio"
    ratio {
      success_metric  = "rate(http_requests_total{status!~\"5..\"}[5m])"
      total_metric    = "rate(http_requests_total[5m])"
      group_by_labels = ["service"]
    }
  }
  
  objectives {
    value  = 0.99
    window = "30d"
  }
  
  destination_datasource {
    uid = "grafanacloud-prom"
  }
  
  # Asserts integration labels
  label {
    key   = "grafana_slo_provenance"
    value = "asserts"
  }
  label {
    key   = "service_name"
    value = "test-service"
  }
  label {
    key   = "team_name"
    value = "test-team"
  }
  
  # Add search expression for Asserts RCA workbench
  search_expression = "service=test-service"
}
`, uid)
}

func testAccSLOAssertsConfigUpdated(uid string) string {
	return fmt.Sprintf(`
resource "grafana_slo" "asserts_test" {
  name        = "Asserts SLO Test Updated - %s"
  description = "Updated test SLO for Asserts integration"
  
  query {
    type = "ratio"
    ratio {
      success_metric  = "rate(http_requests_total{status!~\"5..\"}[5m])"
      total_metric    = "rate(http_requests_total[5m])"
      group_by_labels = ["service"]
    }
  }
  
  objectives {
    value  = 0.995
    window = "30d"
  }
  
  destination_datasource {
    uid = "grafanacloud-prom"
  }
  
  # Asserts integration labels
  label {
    key   = "grafana_slo_provenance"
    value = "asserts"
  }
  label {
    key   = "service_name"
    value = "test-service"
  }
  label {
    key   = "team_name"
    value = "test-team"
  }
  
  # Add search expression for Asserts RCA workbench
  search_expression = "service=test-service"
}
`, uid)
}

// testAccSloCheckAssertsProvenance verifies that the SLO has the correct Asserts provenance
func testAccSloCheckAssertsProvenance(rn string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		client := testutils.Provider.Meta().(*common.Client).SLOClient
		req := client.DefaultAPI.V1SloIdGet(context.Background(), rs.Primary.ID)
		gotSlo, _, err := req.Execute()
		if err != nil {
			return fmt.Errorf("error getting SLO: %s", err)
		}

		// Check that the SLO has the correct provenance
		if gotSlo.ReadOnly == nil || gotSlo.ReadOnly.Provenance == nil {
			return fmt.Errorf("SLO provenance is not set")
		}

		if *gotSlo.ReadOnly.Provenance != "asserts" {
			return fmt.Errorf("expected SLO provenance to be 'asserts', got '%s'", *gotSlo.ReadOnly.Provenance)
		}

		// Verify the SLO has the Asserts provenance label
		hasAssertsLabel := false
		for _, label := range gotSlo.Labels {
			if label.Key == "grafana_slo_provenance" && label.Value == "asserts" {
				hasAssertsLabel = true
				break
			}
		}

		if !hasAssertsLabel {
			return fmt.Errorf("SLO does not have the grafana_slo_provenance=asserts label")
		}

		return nil
	}
}

// TestAccSLO_AssertsIntegration_WithoutProvenance tests that regular SLOs still work
func TestAccSLO_AssertsIntegration_WithoutProvenance(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	resourceName := "grafana_slo.regular_test"
	uid := "regular-test-" + acctest.RandString(10)
	var sloObj slo.SloV00Slo

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccSloCheckDestroy(&sloObj),
		Steps: []resource.TestStep{
			{
				Config: testAccSLORegularConfig(uid),
				Check: resource.ComposeTestCheckFunc(
					testAccSloCheckExists(resourceName, &sloObj),
					// Verify the SLO has the correct provenance (should be "terraform" or "api")
					testAccSloCheckRegularProvenance(resourceName),
				),
			},
		},
	})
}

func testAccSLORegularConfig(uid string) string {
	return fmt.Sprintf(`
resource "grafana_slo" "regular_test" {
  name        = "Regular SLO Test - %s"
  description = "Test SLO without Asserts integration"
  
  query {
    type = "ratio"
    ratio {
      success_metric  = "rate(http_requests_total{status!~\"5..\"}[5m])"
      total_metric    = "rate(http_requests_total[5m])"
      group_by_labels = ["service"]
    }
  }
  
  objectives {
    value  = 0.99
    window = "30d"
  }
  
  destination_datasource {
    uid = "grafanacloud-prom"
  }
  
  label {
    key   = "environment"
    value = "test"
  }
}
`, uid)
}

// testAccSloCheckRegularProvenance verifies that regular SLOs have the correct provenance
func testAccSloCheckRegularProvenance(rn string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		client := testutils.Provider.Meta().(*common.Client).SLOClient
		req := client.DefaultAPI.V1SloIdGet(context.Background(), rs.Primary.ID)
		gotSlo, _, err := req.Execute()
		if err != nil {
			return fmt.Errorf("error getting SLO: %s", err)
		}

		// Check that the SLO has the correct provenance (should not be "asserts")
		if gotSlo.ReadOnly == nil || gotSlo.ReadOnly.Provenance == nil {
			return fmt.Errorf("SLO provenance is not set")
		}

		provenance := *gotSlo.ReadOnly.Provenance
		if provenance == "asserts" {
			return fmt.Errorf("regular SLO should not have 'asserts' provenance, got '%s'", provenance)
		}

		// Should be "terraform" or "api"
		if provenance != "terraform" && provenance != "api" {
			return fmt.Errorf("expected SLO provenance to be 'terraform' or 'api', got '%s'", provenance)
		}

		return nil
	}
}
