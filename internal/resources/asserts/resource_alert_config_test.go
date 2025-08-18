package asserts_test

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"
)

func TestAccAssertsAlertConfig_basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	stackID := getTestStackID(t)
	rName := fmt.Sprintf("test-acc-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccAssertsAlertConfigCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAssertsAlertConfigConfig(stackID, rName),
				Check: resource.ComposeTestCheckFunc(
					testAccAssertsAlertConfigCheckExists("grafana_asserts_notification_alerts_config.test", stackID, rName),
					resource.TestCheckResourceAttr("grafana_asserts_notification_alerts_config.test", "name", rName),
					resource.TestCheckResourceAttr("grafana_asserts_notification_alerts_config.test", "duration", "5m"),
					resource.TestCheckResourceAttr("grafana_asserts_notification_alerts_config.test", "silenced", "false"),
					resource.TestCheckResourceAttr("grafana_asserts_notification_alerts_config.test", "match_labels.alertname", rName),
					testutils.CheckLister("grafana_asserts_notification_alerts_config.test"),
				),
			},
			{
				// Test import
				ResourceName:      "grafana_asserts_notification_alerts_config.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// Test update
				Config: testAccAssertsAlertConfigConfigUpdated(stackID, rName),
				Check: resource.ComposeTestCheckFunc(
					testAccAssertsAlertConfigCheckExists("grafana_asserts_notification_alerts_config.test", stackID, rName+"-updated"),
					resource.TestCheckResourceAttr("grafana_asserts_notification_alerts_config.test", "name", rName+"-updated"),
					resource.TestCheckResourceAttr("grafana_asserts_notification_alerts_config.test", "duration", "10m"),
					resource.TestCheckResourceAttr("grafana_asserts_notification_alerts_config.test", "silenced", "true"),
					resource.TestCheckResourceAttr("grafana_asserts_notification_alerts_config.test", "match_labels.alertname", rName+"-updated"),
				),
			},
		},
	})
}

func TestAccAssertsAlertConfig_minimal(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	stackID := getTestStackID(t)
	rName := fmt.Sprintf("test-minimal-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccAssertsAlertConfigCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAssertsAlertConfigConfigMinimal(stackID, rName),
				Check: resource.ComposeTestCheckFunc(
					testAccAssertsAlertConfigCheckExists("grafana_asserts_notification_alerts_config.test", stackID, rName),
					resource.TestCheckResourceAttr("grafana_asserts_notification_alerts_config.test", "name", rName),
					resource.TestCheckResourceAttr("grafana_asserts_notification_alerts_config.test", "silenced", "false"), // default value
				),
			},
		},
	})
}

func testAccAssertsAlertConfigCheckExists(rn string, stackID int64, name string) resource.TestCheckFunc {
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

		// Get all alert configs and find ours
		request := client.AlertConfigurationAPI.GetAllAlertConfigs(ctx).
			XScopeOrgID(fmt.Sprintf("%d", stackID))

		alertConfigs, _, err := request.Execute()
		if err != nil {
			return fmt.Errorf("error getting alert configs: %s", err)
		}

		// Find our specific config
		for _, config := range alertConfigs.AlertConfigs {
			if config.Name != nil && *config.Name == name {
				return nil // Found it
			}
		}

		return fmt.Errorf("alert config with name %s not found", name)
	}
}

func testAccAssertsAlertConfigCheckDestroy(s *terraform.State) error {
	client := testutils.Provider.Meta().(*common.Client).AssertsAPIClient
	ctx := context.Background()

	deadline := time.Now().Add(60 * time.Second)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "grafana_asserts_notification_alerts_config" {
			continue
		}

		// Resource ID is just the name now
		name := rs.Primary.ID
		stackID := fmt.Sprintf("%d", testutils.Provider.Meta().(*common.Client).GrafanaStackID)

		for {
			// Get all alert configs
			request := client.AlertConfigurationAPI.GetAllAlertConfigs(ctx).
				XScopeOrgID(stackID)

			alertConfigs, _, err := request.Execute()
			if err != nil {
				// If we can't get configs, assume it's because they don't exist
				if common.IsNotFoundError(err) {
					break
				}
				return fmt.Errorf("error checking alert config destruction: %s", err)
			}

			// Check if our config still exists
			stillExists := false
			for _, config := range alertConfigs.AlertConfigs {
				if config.Name != nil && *config.Name == name {
					stillExists = true
					break
				}
			}

			if !stillExists {
				break
			}

			if time.Now().After(deadline) {
				return fmt.Errorf("alert config %s still exists", name)
			}
			time.Sleep(2 * time.Second)
		}
	}

	return nil
}

func getTestStackID(t require.TestingT) int64 {
	stackIDStr := os.Getenv("GRAFANA_CLOUD_PROVIDER_TEST_STACK_ID")
	require.NotEmpty(t, stackIDStr, "GRAFANA_CLOUD_PROVIDER_TEST_STACK_ID must be set")

	stackID, err := strconv.ParseInt(stackIDStr, 10, 64)
	require.NoError(t, err, "GRAFANA_CLOUD_PROVIDER_TEST_STACK_ID must be a valid integer")

	return stackID
}

func testAccAssertsAlertConfigConfig(stackID int64, name string) string {
	return fmt.Sprintf(`
resource "grafana_asserts_notification_alerts_config" "test" {
  name = "%s"

  match_labels = {
    alertname = "%s"
  }

  duration = "5m"
  silenced = false
}
`, name, name)
}

func testAccAssertsAlertConfigConfigUpdated(stackID int64, name string) string {
	return fmt.Sprintf(`
resource "grafana_asserts_notification_alerts_config" "test" {
  name = "%s-updated"

  match_labels = {
    alertname = "%s-updated"
  }

  duration = "10m"
  silenced = true
}
`, name, name)
}

func testAccAssertsAlertConfigConfigMinimal(stackID int64, name string) string {
	return fmt.Sprintf(`
resource "grafana_asserts_notification_alerts_config" "test" {
  name = "%s"
  
  match_labels = {
    alertname = "%s"
  }
  
  duration = "5m"
}
`, name, name)
}

// TestAccAssertsAlertConfig_eventualConsistencyStress tests multiple resources created simultaneously
// to verify the retry logic handles eventual consistency properly
func TestAccAssertsAlertConfig_eventualConsistencyStress(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	// Skip this flaky test unless explicitly enabled
	if !testutils.AccTestsEnabled("TF_ACC_STRESS_TESTS") {
		t.Skip("TF_ACC_STRESS_TESTS must be set to a truthy value for stress tests")
	}

	stackID := getTestStackID(t)
	baseName := fmt.Sprintf("stress-test-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccAssertsAlertConfigCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAssertsAlertConfigStressConfig(stackID, baseName),
				Check: resource.ComposeTestCheckFunc(
					// Check that all resources were created successfully
					testAccAssertsAlertConfigCheckExists("grafana_asserts_notification_alerts_config.test1", stackID, baseName+"-1"),
					testAccAssertsAlertConfigCheckExists("grafana_asserts_notification_alerts_config.test2", stackID, baseName+"-2"),
					testAccAssertsAlertConfigCheckExists("grafana_asserts_notification_alerts_config.test3", stackID, baseName+"-3"),
					resource.TestCheckResourceAttr("grafana_asserts_notification_alerts_config.test1", "name", baseName+"-1"),
					resource.TestCheckResourceAttr("grafana_asserts_notification_alerts_config.test2", "name", baseName+"-2"),
					resource.TestCheckResourceAttr("grafana_asserts_notification_alerts_config.test3", "name", baseName+"-3"),
				),
			},
		},
	})
}

func testAccAssertsAlertConfigStressConfig(stackID int64, baseName string) string {
	return fmt.Sprintf(`
resource "grafana_asserts_notification_alerts_config" "test1" {
  name = "%s-1"
  
  match_labels = {
    alertname = "%s-1"
  }
  
  duration = "5m"
}

resource "grafana_asserts_notification_alerts_config" "test2" {
  name = "%s-2"
  
  match_labels = {
    alertname = "%s-2"
  }
  
  duration = "10m"
}

resource "grafana_asserts_notification_alerts_config" "test3" {
  name = "%s-3"
  
  match_labels = {
    alertname = "%s-3"
  }
  
  duration = "15m"
}

`, baseName, baseName, baseName, baseName, baseName, baseName)
}
