package asserts_test

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"
)

// TestAccAssertsAlertConfig_basic tests the creation, import, and update of an Asserts alert configuration.
// It also covers the eventual consistency case by immediately reading the resource after creation.
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

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "grafana_asserts_notification_alerts_config" {
			continue
		}

		// Resource ID is just the name now
		name := rs.Primary.ID
		stackID := fmt.Sprintf("%d", testutils.Provider.Meta().(*common.Client).GrafanaStackID)

		// Get all alert configs
		request := client.AlertConfigurationAPI.GetAllAlertConfigs(ctx).
			XScopeOrgID(stackID)

		alertConfigs, _, err := request.Execute()
		if err != nil {
			// If we can't get configs, assume it's because they don't exist
			if common.IsNotFoundError(err) {
				continue
			}
			return fmt.Errorf("error checking alert config destruction: %s", err)
		}

		// Check if our config still exists
		for _, config := range alertConfigs.AlertConfigs {
			if config.Name != nil && *config.Name == name {
				return fmt.Errorf("alert config %s still exists", name)
			}
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
