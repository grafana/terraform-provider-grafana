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

// TestAccAssertsDisabledAlertConfig_basic tests the creation, import, and update of an Asserts disabled alert configuration.
// It also covers the eventual consistency case by immediately reading the resource after creation.
func TestAccAssertsDisabledAlertConfig_basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	stackID := getTestStackID(t)
	rName := fmt.Sprintf("test-disabled-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccAssertsDisabledAlertConfigCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAssertsDisabledAlertConfigConfig(stackID, rName),
				Check: resource.ComposeTestCheckFunc(
					testAccAssertsDisabledAlertConfigCheckExists("grafana_asserts_suppressed_assertions_config.test", stackID, rName),
					resource.TestCheckResourceAttr("grafana_asserts_suppressed_assertions_config.test", "name", rName),
					resource.TestCheckResourceAttr("grafana_asserts_suppressed_assertions_config.test", "match_labels.alertname", rName),
					testutils.CheckLister("grafana_asserts_suppressed_assertions_config.test"),
				),
			},
			{
				// Test import
				ResourceName:      "grafana_asserts_suppressed_assertions_config.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// Test update
				Config: testAccAssertsDisabledAlertConfigConfigUpdated(stackID, rName),
				Check: resource.ComposeTestCheckFunc(
					testAccAssertsDisabledAlertConfigCheckExists("grafana_asserts_suppressed_assertions_config.test", stackID, rName+"-updated"),
					resource.TestCheckResourceAttr("grafana_asserts_suppressed_assertions_config.test", "name", rName+"-updated"),
					resource.TestCheckResourceAttr("grafana_asserts_suppressed_assertions_config.test", "match_labels.alertname", rName+"-updated"),
				),
			},
		},
	})
}

func TestAccAssertsDisabledAlertConfig_minimal(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	stackID := getTestStackID(t)
	rName := fmt.Sprintf("test-minimal-disabled-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccAssertsDisabledAlertConfigCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAssertsDisabledAlertConfigConfigMinimal(stackID, rName),
				Check: resource.ComposeTestCheckFunc(
					testAccAssertsDisabledAlertConfigCheckExists("grafana_asserts_suppressed_assertions_config.test", stackID, rName),
					resource.TestCheckResourceAttr("grafana_asserts_suppressed_assertions_config.test", "name", rName),
				),
			},
		},
	})
}

func testAccAssertsDisabledAlertConfigCheckExists(rn string, stackID int64, name string) resource.TestCheckFunc {
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

		// Get all disabled alert configs and find ours
		request := client.AlertConfigurationAPI.GetAllDisabledAlertConfigs(ctx).
			XScopeOrgID(fmt.Sprintf("%d", stackID))

		disabledAlertConfigs, _, err := request.Execute()
		if err != nil {
			return fmt.Errorf("error getting disabled alert configs: %s", err)
		}

		// Find our specific config
		for _, config := range disabledAlertConfigs.DisabledAlertConfigs {
			if config.Name != nil && *config.Name == name {
				// Verify managedBy field is set to terraform
				if config.ManagedBy == nil || *config.ManagedBy != "terraform" {
					return fmt.Errorf("disabled alert config %s has invalid managedBy field (expected 'terraform', got %v)", name, config.ManagedBy)
				}
				return nil // Found it
			}
		}

		return fmt.Errorf("disabled alert config with name %s not found", name)
	}
}

func testAccAssertsDisabledAlertConfigCheckDestroy(s *terraform.State) error {
	client := testutils.Provider.Meta().(*common.Client).AssertsAPIClient
	ctx := context.Background()

	deadline := time.Now().Add(60 * time.Second)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "grafana_asserts_suppressed_assertions_config" {
			continue
		}

		// Resource ID is just the name now
		name := rs.Primary.ID
		stackID := fmt.Sprintf("%d", testutils.Provider.Meta().(*common.Client).GrafanaStackID)

		for {
			// Get all disabled alert configs
			request := client.AlertConfigurationAPI.GetAllDisabledAlertConfigs(ctx).
				XScopeOrgID(stackID)

			disabledAlertConfigs, _, err := request.Execute()
			if err != nil {
				// If we can't get configs, assume it's because they don't exist
				if common.IsNotFoundError(err) {
					break
				}
				return fmt.Errorf("error checking disabled alert config destruction: %s", err)
			}

			// Check if our config still exists
			stillExists := false
			for _, config := range disabledAlertConfigs.DisabledAlertConfigs {
				if config.Name != nil && *config.Name == name {
					stillExists = true
					break
				}
			}

			if !stillExists {
				break
			}

			if time.Now().After(deadline) {
				return fmt.Errorf("disabled alert config %s still exists", name)
			}
			time.Sleep(2 * time.Second)
		}
	}

	return nil
}

func testAccAssertsDisabledAlertConfigConfig(stackID int64, name string) string {
	return fmt.Sprintf(`
resource "grafana_asserts_suppressed_assertions_config" "test" {
  name = "%s"

  match_labels = {
    alertname = "%s"
  }
}
`, name, name)
}

func testAccAssertsDisabledAlertConfigConfigUpdated(stackID int64, name string) string {
	return fmt.Sprintf(`
resource "grafana_asserts_suppressed_assertions_config" "test" {
  name = "%s-updated"

  match_labels = {
    alertname = "%s-updated"
  }
}
`, name, name)
}

func testAccAssertsDisabledAlertConfigConfigMinimal(stackID int64, name string) string {
	return fmt.Sprintf(`
resource "grafana_asserts_suppressed_assertions_config" "test" {
  name = "%s"
  
  match_labels = {
    alertname = "%s"
  }
}
`, name, name)
}

// TestAccAssertsDisabledAlertConfig_eventualConsistencyStress tests multiple resources created simultaneously
// to verify the retry logic handles eventual consistency properly
func TestAccAssertsDisabledAlertConfig_eventualConsistencyStress(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)
	testutils.CheckStressTestsEnabled(t)

	stackID := getTestStackID(t)
	baseName := fmt.Sprintf("stress-disabled-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccAssertsDisabledAlertConfigCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAssertsDisabledAlertConfigStressConfig(stackID, baseName),
				Check: resource.ComposeTestCheckFunc(
					// Check that all resources were created successfully
					testAccAssertsDisabledAlertConfigCheckExists("grafana_asserts_suppressed_assertions_config.test1", stackID, baseName+"-1"),
					testAccAssertsDisabledAlertConfigCheckExists("grafana_asserts_suppressed_assertions_config.test2", stackID, baseName+"-2"),
					testAccAssertsDisabledAlertConfigCheckExists("grafana_asserts_suppressed_assertions_config.test3", stackID, baseName+"-3"),
					resource.TestCheckResourceAttr("grafana_asserts_suppressed_assertions_config.test1", "name", baseName+"-1"),
					resource.TestCheckResourceAttr("grafana_asserts_suppressed_assertions_config.test2", "name", baseName+"-2"),
					resource.TestCheckResourceAttr("grafana_asserts_suppressed_assertions_config.test3", "name", baseName+"-3"),
				),
			},
		},
	})
}

func testAccAssertsDisabledAlertConfigStressConfig(stackID int64, baseName string) string {
	return fmt.Sprintf(`
resource "grafana_asserts_suppressed_assertions_config" "test1" {
  name = "%s-1"
  
  match_labels = {
    alertname = "%s-1"
  }
}

resource "grafana_asserts_suppressed_assertions_config" "test2" {
  name = "%s-2"
  
  match_labels = {
    alertname = "%s-2"
  }
}

resource "grafana_asserts_suppressed_assertions_config" "test3" {
  name = "%s-3"
  
  match_labels = {
    alertname = "%s-3"
  }
}
`, baseName, baseName, baseName, baseName, baseName, baseName)
}
