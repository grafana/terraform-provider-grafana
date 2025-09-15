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

func TestAccAssertsLogConfig_basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	stackID := getTestStackID(t)
	rName := fmt.Sprintf("test-acc-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccAssertsLogConfigCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAssertsLogConfigConfig(stackID, rName),
				Check: resource.ComposeTestCheckFunc(
					testAccAssertsLogConfigCheckExists("grafana_asserts_log_config.test", stackID, rName),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.test", "name", rName),
					testutils.CheckLister("grafana_asserts_log_config.test"),
				),
			},
			{
				// Test import
				ResourceName:            "grafana_asserts_log_config.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"config"},
			},
			{
				// Test update
				Config: testAccAssertsLogConfigConfigUpdated(stackID, rName),
				Check: resource.ComposeTestCheckFunc(
					testAccAssertsLogConfigCheckExists("grafana_asserts_log_config.test", stackID, rName),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.test", "name", rName),
				),
			},
		},
	})
}

func TestAccAssertsLogConfig_minimal(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	stackID := getTestStackID(t)
	rName := fmt.Sprintf("test-minimal-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccAssertsLogConfigCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAssertsLogConfigConfigMinimal(stackID, rName),
				Check: resource.ComposeTestCheckFunc(
					testAccAssertsLogConfigCheckExists("grafana_asserts_log_config.test", stackID, rName),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.test", "name", rName),
				),
			},
		},
	})
}

func testAccAssertsLogConfigCheckExists(rn string, stackID int64, name string) resource.TestCheckFunc {
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

		// Get tenant log config and find our environment
		request := client.LogConfigControllerAPI.GetTenantEnvConfig(ctx).
			XScopeOrgID(fmt.Sprintf("%d", stackID))

		tenantConfig, _, err := request.Execute()
		if err != nil {
			return fmt.Errorf("error getting tenant log config: %s", err)
		}

		// Find our specific environment
		for _, env := range tenantConfig.GetEnvironments() {
			if env.GetName() == name {
				return nil // Found it
			}
		}

		return fmt.Errorf("log config environment with name %s not found", name)
	}
}

func testAccAssertsLogConfigCheckDestroy(s *terraform.State) error {
	client := testutils.Provider.Meta().(*common.Client).AssertsAPIClient
	ctx := context.Background()

	deadline := time.Now().Add(60 * time.Second)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "grafana_asserts_log_config" {
			continue
		}

		// Resource ID is just the name now
		name := rs.Primary.ID
		stackID := fmt.Sprintf("%d", testutils.Provider.Meta().(*common.Client).GrafanaStackID)

		for {
			// Get tenant log config
			request := client.LogConfigControllerAPI.GetTenantEnvConfig(ctx).
				XScopeOrgID(stackID)

			tenantConfig, _, err := request.Execute()
			if err != nil {
				// If we can't get config, assume it's because they don't exist
				if common.IsNotFoundError(err) {
					break
				}
				return fmt.Errorf("error checking log config destruction: %s", err)
			}

			// Check if our environment still exists
			stillExists := false
			for _, env := range tenantConfig.GetEnvironments() {
				if env.GetName() == name {
					stillExists = true
					break
				}
			}

			if !stillExists {
				break
			}

			if time.Now().After(deadline) {
				return fmt.Errorf("log config environment %s still exists", name)
			}
			time.Sleep(2 * time.Second)
		}
	}

	return nil
}

func testAccAssertsLogConfigConfig(stackID int64, name string) string {
	return fmt.Sprintf(`
resource "grafana_asserts_log_config" "test" {
  name = "%s"
  
  config = <<-EOT
    name: "%s"
    envsForLog:
      - "production"
      - "staging"
    sitesForLog:
      - "us-east-1"
    logConfig:
      tool: "loki"
      url: "https://logs.example.com"
      dateFormat: "RFC3339"
      correlationLabels: "trace_id,span_id"
      defaultSearchText: "error"
      errorFilter: "level=error"
      columns:
        - "timestamp"
        - "level"
        - "message"
      index: "logs-*"
      interval: "1h"
      query:
        job: "app"
        level: "error"
      sort:
        - "timestamp desc"
      httpResponseCodeField: "status_code"
      orgId: "1"
      dataSource: "loki"
    defaultConfig: false
  EOT
}
`, name, name)
}

func testAccAssertsLogConfigConfigUpdated(stackID int64, name string) string {
	return fmt.Sprintf(`
resource "grafana_asserts_log_config" "test" {
  name = "%s"
  
  config = <<-EOT
    name: "%s"
    envsForLog:
      - "production"
      - "staging"
      - "development"
    sitesForLog:
      - "us-east-1"
      - "us-west-2"
    logConfig:
      tool: "elasticsearch"
      url: "https://elastic.example.com"
      dateFormat: "ISO8601"
      correlationLabels: "trace_id,span_id,request_id"
      defaultSearchText: "warning"
      errorFilter: "level=error OR level=warning"
      columns:
        - "timestamp"
        - "level"
        - "message"
        - "service"
      index: "app-logs-*"
      interval: "30m"
      query:
        job: "app"
        level: "error"
        service: "api"
      sort:
        - "timestamp desc"
        - "level asc"
      httpResponseCodeField: "status_code"
      orgId: "1"
      dataSource: "elasticsearch"
    defaultConfig: true
  EOT
}
`, name, name)
}

func testAccAssertsLogConfigConfigMinimal(stackID int64, name string) string {
	return fmt.Sprintf(`
resource "grafana_asserts_log_config" "test" {
  name = "%s"
  
  config = <<-EOT
    name: "%s"
    logConfig:
      tool: "loki"
      url: "https://logs.example.com"
    defaultConfig: false
  EOT
}
`, name, name)
}