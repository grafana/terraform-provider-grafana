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
					resource.TestCheckResourceAttr("grafana_asserts_log_config.test", "name", "test-basic"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.test", "default_config", "false"),
					resource.TestCheckResourceAttrSet("grafana_asserts_log_config.test", "data_source_uid"),
				),
			},
			{
				ResourceName:      "grafana_asserts_log_config.test",
				ImportState:       true,
				ImportStateVerify: true,
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

func TestAccAssertsLogConfig_fullFields(t *testing.T) {
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
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "name", rName),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "default_config", "true"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "data_source_uid", "loki-uid-456"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "error_label", "error"),
					// match rules
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "match.0.property", "service"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "match.0.op", "equals"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "match.0.values.0", "api"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "match.0.values.1", "web"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "match.1.property", "environment"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "match.1.op", "contains"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "match.1.values.0", "prod"),
					// mappings
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "entity_property_to_log_label_mapping.service", "service_name"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "entity_property_to_log_label_mapping.environment", "env"),
					// filters
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "filter_by_span_id", "true"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "filter_by_trace_id", "true"),
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
			request := client.AssertsAPIClient.LogDrilldownConfigControllerAPI.GetTenantLogConfig(context.Background()).
				XScopeOrgID(fmt.Sprintf("%d", stackID))

			tenantConfig, _, err := request.Execute()
			if err != nil {
				if common.IsNotFoundError(err) {
					break
				}
				return fmt.Errorf("error checking log config destruction: %s", err)
			}

			found := false
			for _, config := range tenantConfig.GetLogDrilldownConfigs() {
				if config.GetName() == name {
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
  name            = "test-basic"
  default_config  = false
  data_source_uid = "loki-uid-123"
}
`

func testAccAssertsLogConfigConfigNamed(name string, defaultCfg bool) string {
	defaultVal := "false"
	if defaultCfg {
		defaultVal = "true"
	}
	return fmt.Sprintf(`
resource "grafana_asserts_log_config" "test" {
  name            = "%s"
  default_config  = %s
  data_source_uid = "loki-uid-123"
}
`, name, defaultVal)
}

func testAccAssertsLogConfigConfigFullNamed(name string) string {
	return fmt.Sprintf(`
resource "grafana_asserts_log_config" "full" {
  name            = "%s"
  default_config  = true
  data_source_uid = "loki-uid-456"
  error_label     = "error"
  
  match {
    property = "service"
    op       = "equals"
    values   = ["api", "web"]
  }
  
  match {
    property = "environment"
    op       = "contains"
    values   = ["prod"]
  }
  
  entity_property_to_log_label_mapping = {
    "service"     = "service_name"
    "environment" = "env"
  }
  
  filter_by_span_id  = true
  filter_by_trace_id = true
}
`, name)
}

func testAccAssertsLogConfigStressConfig(baseName string) string {
	return fmt.Sprintf(`
resource "grafana_asserts_log_config" "stress1" {
  name            = "%s-1"
  default_config  = false
  data_source_uid = "loki-uid-stress1"
}

resource "grafana_asserts_log_config" "stress2" {
  name            = "%s-2"
  default_config  = false
  data_source_uid = "loki-uid-stress2"
}
`, baseName, baseName)
}
