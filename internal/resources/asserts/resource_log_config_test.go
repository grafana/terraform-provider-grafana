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

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccAssertsLogConfigCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAssertsLogConfigConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_asserts_log_config.test", "name", "test-env"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.test", "envs_for_log.0", "test-env"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.test", "default_config", "false"),
				),
			},
			{
				ResourceName:            "grafana_asserts_log_config.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"log_config"},
			},
		},
	})
}

func TestAccAssertsLogConfig_update(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	rName := fmt.Sprintf("test-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccAssertsLogConfigCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAssertsLogConfigConfigNamed(rName, false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_asserts_log_config.test", "name", rName),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.test", "envs_for_log.0", rName),
				),
			},
			{
				Config: testAccAssertsLogConfigConfigNamed(rName, true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_asserts_log_config.test", "name", rName),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.test", "default_config", "true"),
				),
			},
		},
	})
}

func TestAccAssertsLogConfig_logConfigFull(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)
	testutils.CheckStressTestsEnabled(t)

	rName := fmt.Sprintf("full-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccAssertsLogConfigCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAssertsLogConfigConfigFullNamed(rName),
				Check: resource.ComposeTestCheckFunc(
					// top-level
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "name", rName),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "envs_for_log.0", rName),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "sites_for_log.0", "us-east-1"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "sites_for_log.1", "us-west-2"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "default_config", "true"),
					// log_config nested
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "log_config.0.tool", "loki"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "log_config.0.url", "https://logs.example.com"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "log_config.0.date_format", "RFC3339"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "log_config.0.correlation_labels", "trace_id,span_id"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "log_config.0.default_search_text", "error"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "log_config.0.error_filter", "level=error"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "log_config.0.columns.0", "timestamp"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "log_config.0.columns.1", "level"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "log_config.0.columns.2", "message"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "log_config.0.index", "logs-*"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "log_config.0.interval", "1h"),
					// map flattens to keys
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "log_config.0.query.job", "app"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "log_config.0.query.level", "error"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "log_config.0.sort.0", "timestamp desc"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "log_config.0.http_response_code_field", "status_code"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "log_config.0.org_id", "1"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "log_config.0.data_source", "loki"),
				),
			},
		},
	})
}

func TestAccAssertsLogConfig_eventualConsistencyStress(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)
	testutils.CheckStressTestsEnabled(t)

	baseName := fmt.Sprintf("stress-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccAssertsLogConfigCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAssertsLogConfigStressConfig(baseName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_asserts_log_config.stress1", "name", baseName+"-1"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.stress2", "name", baseName+"-2"),
				),
			},
		},
	})
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

	deadline := time.Now().Add(60 * time.Second)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "grafana_asserts_log_config" {
			continue
		}

		name := rs.Primary.ID
		for {
			request := client.AssertsAPIClient.LogConfigControllerAPI.GetTenantEnvConfig(context.Background()).
				XScopeOrgID(fmt.Sprintf("%d", stackID))

			tenantConfig, _, err := request.Execute()
			if err != nil {
				if common.IsNotFoundError(err) {
					break
				}
				return fmt.Errorf("error checking log config destruction: %s", err)
			}

			found := false
			for _, env := range tenantConfig.GetEnvironments() {
				if env.GetName() == name {
					found = true
					break
				}
			}

			if !found {
				break
			}

			if time.Now().After(deadline) {
				return fmt.Errorf("log config %s still exists", name)
			}
			time.Sleep(2 * time.Second)
		}
	}

	return nil
}

const testAccAssertsLogConfigConfig = `
resource "grafana_asserts_log_config" "test" {
  name           = "test-env"
  envs_for_log   = ["test-env"]
  default_config = false
}
`

func testAccAssertsLogConfigConfigNamed(name string, defaultCfg bool) string {
	defaultVal := "false"
	if defaultCfg {
		defaultVal = "true"
	}
	return fmt.Sprintf(`
resource "grafana_asserts_log_config" "test" {
  name           = "%s"
  envs_for_log   = ["%s"]
  default_config = %s
}
`, name, name, defaultVal)
}

func testAccAssertsLogConfigConfigFullNamed(name string) string {
	return fmt.Sprintf(`
resource "grafana_asserts_log_config" "full" {
  name           = "%s"
  envs_for_log   = ["%s"]
  sites_for_log  = ["us-east-1", "us-west-2"]
  default_config = true
  log_config {
    tool                 = "loki"
    url                  = "https://logs.example.com"
    date_format          = "RFC3339"
    correlation_labels   = "trace_id,span_id"
    default_search_text  = "error"
    error_filter         = "level=error"
    columns              = ["timestamp", "level", "message"]
    index                = "logs-*"
    interval             = "1h"
    query = {
      job   = "app"
      level = "error"
    }
    sort                   = ["timestamp desc"]
    http_response_code_field = "status_code"
    org_id                = "1"
    data_source           = "loki"
  }
}
`, name, name)
}

func testAccAssertsLogConfigStressConfig(baseName string) string {
	return fmt.Sprintf(`
resource "grafana_asserts_log_config" "stress1" {
  name           = "%s-1"
  envs_for_log   = ["%s-1"]
  default_config = false
}

resource "grafana_asserts_log_config" "stress2" {
  name           = "%s-2"
  envs_for_log   = ["%s-2"]
  default_config = false
}
`, baseName, baseName, baseName, baseName)
}
