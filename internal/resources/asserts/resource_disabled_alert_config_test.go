package asserts_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

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
					testAccAssertsDisabledAlertConfigCheckExists("grafana_asserts_disabled_alert_config.test", stackID, rName),
					resource.TestCheckResourceAttr("grafana_asserts_disabled_alert_config.test", "name", rName),
					resource.TestCheckResourceAttr("grafana_asserts_disabled_alert_config.test", "match_labels.alertname", rName),
					testutils.CheckLister("grafana_asserts_disabled_alert_config.test"),
				),
			},
			{
				// Test import
				ResourceName:      "grafana_asserts_disabled_alert_config.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// Test update
				Config: testAccAssertsDisabledAlertConfigConfigUpdated(stackID, rName),
				Check: resource.ComposeTestCheckFunc(
					testAccAssertsDisabledAlertConfigCheckExists("grafana_asserts_disabled_alert_config.test", stackID, rName+"-updated"),
					resource.TestCheckResourceAttr("grafana_asserts_disabled_alert_config.test", "name", rName+"-updated"),
					resource.TestCheckResourceAttr("grafana_asserts_disabled_alert_config.test", "match_labels.alertname", rName+"-updated"),
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
					testAccAssertsDisabledAlertConfigCheckExists("grafana_asserts_disabled_alert_config.test", stackID, rName),
					resource.TestCheckResourceAttr("grafana_asserts_disabled_alert_config.test", "name", rName),
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
		request := client.DisabledAlertConfigControllerAPI.GetAllDisabledAlertConfigs(ctx).
			XScopeOrgID(fmt.Sprintf("%d", stackID))

		disabledAlertConfigs, _, err := request.Execute()
		if err != nil {
			return fmt.Errorf("error getting disabled alert configs: %s", err)
		}

		// Find our specific config
		for _, config := range disabledAlertConfigs.DisabledAlertConfigs {
			if config.Name != nil && *config.Name == name {
				return nil // Found it
			}
		}

		return fmt.Errorf("disabled alert config with name %s not found", name)
	}
}

func testAccAssertsDisabledAlertConfigCheckDestroy(s *terraform.State) error {
	client := testutils.Provider.Meta().(*common.Client).AssertsAPIClient
	ctx := context.Background()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "grafana_asserts_disabled_alert_config" {
			continue
		}

		// Parse the ID to get stack_id and name
		parts := strings.SplitN(rs.Primary.ID, ":", 2)
		if len(parts) != 2 {
			continue
		}

		stackID := parts[0]
		name := parts[1]

		// Get all disabled alert configs
		request := client.DisabledAlertConfigControllerAPI.GetAllDisabledAlertConfigs(ctx).
			XScopeOrgID(stackID)

		disabledAlertConfigs, _, err := request.Execute()
		if err != nil {
			// If we can't get configs, assume it's because they don't exist
			if common.IsNotFoundError(err) {
				continue
			}
			return fmt.Errorf("error checking disabled alert config destruction: %s", err)
		}

		// Check if our config still exists
		for _, config := range disabledAlertConfigs.DisabledAlertConfigs {
			if config.Name != nil && *config.Name == name {
				return fmt.Errorf("disabled alert config %s still exists", name)
			}
		}
	}

	return nil
}

func testAccAssertsDisabledAlertConfigConfig(stackID int64, name string) string {
	return fmt.Sprintf(`
resource "grafana_asserts_disabled_alert_config" "test" {
  stack_id = %d
  name     = "%s"

  match_labels = {
    alertname = "%s"
  }
}
`, stackID, name, name)
}

func testAccAssertsDisabledAlertConfigConfigUpdated(stackID int64, name string) string {
	return fmt.Sprintf(`
resource "grafana_asserts_disabled_alert_config" "test" {
  stack_id = %d
  name     = "%s-updated"

  match_labels = {
    alertname = "%s-updated"
  }
}
`, stackID, name, name)
}

func testAccAssertsDisabledAlertConfigConfigMinimal(stackID int64, name string) string {
	return fmt.Sprintf(`
resource "grafana_asserts_disabled_alert_config" "test" {
  stack_id = %d
  name     = "%s"
  
  match_labels = {
    alertname = "%s"
  }
}
`, stackID, name, name)
}
