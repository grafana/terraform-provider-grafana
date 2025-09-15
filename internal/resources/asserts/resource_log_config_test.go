package asserts_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccAssertsLogConfig_basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccAssertsLogConfigCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAssertsLogConfigConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_asserts_log_config.test", "name", "test-env"),
					resource.TestCheckResourceAttrSet("grafana_asserts_log_config.test", "config"),
					testAccAssertsLogConfigCheckExists("grafana_asserts_log_config.test"),
				),
			},
			{
				ResourceName:      "grafana_asserts_log_config.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{"config"},
			},
		},
	})
}

func TestAccAssertsLogConfig_minimal(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccAssertsLogConfigCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAssertsLogConfigConfigMinimal,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_asserts_log_config.minimal", "name", "minimal-env"),
					resource.TestCheckResourceAttrSet("grafana_asserts_log_config.minimal", "config"),
					testAccAssertsLogConfigCheckExists("grafana_asserts_log_config.minimal"),
				),
			},
		},
	})
}

func TestAccAssertsLogConfig_update(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccAssertsLogConfigCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAssertsLogConfigConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_asserts_log_config.test", "name", "test-env"),
					testAccAssertsLogConfigCheckExists("grafana_asserts_log_config.test"),
				),
			},
			{
				Config: testAccAssertsLogConfigConfigUpdated,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_asserts_log_config.test", "name", "test-env"),
					testAccAssertsLogConfigCheckExists("grafana_asserts_log_config.test"),
				),
			},
		},
	})
}

func TestAccAssertsLogConfig_eventualConsistencyStress(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	// This test creates multiple resources rapidly to test eventual consistency
	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccAssertsLogConfigCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAssertsLogConfigConfigStress,
				Check: resource.ComposeTestCheckFunc(
					testAccAssertsLogConfigCheckExists("grafana_asserts_log_config.stress1"),
					testAccAssertsLogConfigCheckExists("grafana_asserts_log_config.stress2"),
					testAccAssertsLogConfigCheckExists("grafana_asserts_log_config.stress3"),
				),
			},
		},
	})
}

func testAccAssertsLogConfigCheckExists(rn string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		// Get the client and stack ID from the provider
		client := testutils.Provider.Meta().(*common.Client)
		if client.AssertsAPIClient == nil {
			return fmt.Errorf("client not configured for the Asserts API")
		}

		stackID := client.GrafanaStackID
		if stackID == 0 {
			return fmt.Errorf("stack_id must be set in provider configuration for Asserts resources")
		}

		// Try to read the resource with retry logic for eventual consistency
		var lastErr error
		for i := 0; i < 10; i++ {
			request := client.AssertsAPIClient.LogConfigControllerAPI.GetTenantEnvConfig(context.Background()).
				XScopeOrgID(fmt.Sprintf("%d", stackID))

			tenantConfig, _, err := request.Execute()
			if err != nil {
				lastErr = err
				time.Sleep(time.Second * 2)
				continue
			}

			// Check if our environment exists
			for _, env := range tenantConfig.GetEnvironments() {
				if env.GetName() == rs.Primary.ID {
					return nil // Found it!
				}
			}

			lastErr = fmt.Errorf("environment %s not found", rs.Primary.ID)
			time.Sleep(time.Second * 2)
		}

		return fmt.Errorf("environment %s not found after retries: %w", rs.Primary.ID, lastErr)
	}
}

func testAccAssertsLogConfigCheckDestroy(s *terraform.State) error {
	client := testutils.Provider.Meta().(*common.Client)
	if client.AssertsAPIClient == nil {
		return fmt.Errorf("client not configured for the Asserts API")
	}

	stackID := client.GrafanaStackID
	if stackID == 0 {
		return fmt.Errorf("stack_id must be set in provider configuration for Asserts resources")
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "grafana_asserts_log_config" {
			continue
		}

		// Try to read the resource with retry logic for eventual consistency
		var lastErr error
		for i := 0; i < 10; i++ {
			request := client.AssertsAPIClient.LogConfigControllerAPI.GetTenantEnvConfig(context.Background()).
				XScopeOrgID(fmt.Sprintf("%d", stackID))

			tenantConfig, _, err := request.Execute()
			if err != nil {
				lastErr = err
				time.Sleep(time.Second * 2)
				continue
			}

			// Check if our environment still exists
			found := false
			for _, env := range tenantConfig.GetEnvironments() {
				if env.GetName() == rs.Primary.ID {
					found = true
					break
				}
			}

			if !found {
				return nil // Successfully deleted
			}

			lastErr = fmt.Errorf("environment %s still exists", rs.Primary.ID)
			time.Sleep(time.Second * 2)
		}

		return fmt.Errorf("environment %s still exists after retries: %w", rs.Primary.ID, lastErr)
	}

	return nil
}

const testAccAssertsLogConfigConfig = `
resource "grafana_asserts_log_config" "test" {
  name = "test-env"
  config = <<-EOT
    name: test-env
    logConfig:
      enabled: true
      retention: "7d"
      maxLogSize: "100MB"
      compression: true
  EOT
}
`

const testAccAssertsLogConfigConfigMinimal = `
resource "grafana_asserts_log_config" "minimal" {
  name = "minimal-env"
  config = <<-EOT
    name: minimal-env
    logConfig:
      enabled: true
  EOT
}
`

const testAccAssertsLogConfigConfigUpdated = `
resource "grafana_asserts_log_config" "test" {
  name = "test-env"
  config = <<-EOT
    name: test-env
    logConfig:
      enabled: true
      retention: "30d"
      maxLogSize: "500MB"
      compression: false
      filters:
        - level: "ERROR"
        - level: "WARN"
  EOT
}
`

const testAccAssertsLogConfigConfigStress = `
resource "grafana_asserts_log_config" "stress1" {
  name = "stress-test-1"
  config = <<-EOT
    name: stress-test-1
    logConfig:
      enabled: true
      retention: "1d"
  EOT
}

resource "grafana_asserts_log_config" "stress2" {
  name = "stress-test-2"
  config = <<-EOT
    name: stress-test-2
    logConfig:
      enabled: true
      retention: "2d"
  EOT
}

resource "grafana_asserts_log_config" "stress3" {
  name = "stress-test-3"
  config = <<-EOT
    name: stress-test-3
    logConfig:
      enabled: true
      retention: "3d"
  EOT
}
`
