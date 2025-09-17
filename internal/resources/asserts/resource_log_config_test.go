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
					resource.TestCheckResourceAttr("grafana_asserts_log_config.test", "priority", "1000"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.test", "default_config", "false"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.test", "data_source_uid", "grafanacloud-logs"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.test", "error_label", "error"),
					// match rules
					resource.TestCheckResourceAttr("grafana_asserts_log_config.test", "match.0.property", "asserts_entity_type"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.test", "match.0.op", "EQUALS"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.test", "match.0.values.0", "Service"),
					// mappings
					resource.TestCheckResourceAttr("grafana_asserts_log_config.test", "entity_property_to_log_label_mapping.otel_namespace", "service_namespace"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.test", "entity_property_to_log_label_mapping.otel_service", "service_name"),
					// filters
					resource.TestCheckResourceAttr("grafana_asserts_log_config.test", "filter_by_span_id", "true"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.test", "filter_by_trace_id", "true"),
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
					resource.TestCheckResourceAttr("grafana_asserts_log_config.test", "priority", "1001"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.test", "default_config", "false"),
				),
			},
			{
				Config: testAccAssertsLogConfigConfigNamedUpdate(rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_asserts_log_config.test", "name", rName),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.test", "priority", "1002"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.test", "default_config", "false"),
				),
			},
		},
	})
}

func TestAccAssertsLogConfig_fullFields(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	rName := fmt.Sprintf("full-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccAssertsLogConfigCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAssertsLogConfigConfigFullNamed(rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "name", rName),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "priority", "1002"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "default_config", "false"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "data_source_uid", "loki-uid-456"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "error_label", "error"),
					// match rules
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "match.0.property", "service"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "match.0.op", "EQUALS"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "match.0.values.0", "api"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "match.0.values.1", "web"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "match.1.property", "environment"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.full", "match.1.op", "CONTAINS"),
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

func TestAccAssertsLogConfig_multiple(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	baseName := fmt.Sprintf("multi-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccAssertsLogConfigCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAssertsLogConfigMultipleConfig(baseName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_asserts_log_config.multi1", "name", baseName+"-1"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.multi1", "priority", "2001"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.multi1", "default_config", "false"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.multi1", "data_source_uid", "loki-uid-multi1"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.multi2", "name", baseName+"-2"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.multi2", "priority", "2002"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.multi2", "default_config", "false"),
					resource.TestCheckResourceAttr("grafana_asserts_log_config.multi2", "data_source_uid", "loki-uid-multi2"),
				),
			},
			{
				ResourceName:      "grafana_asserts_log_config.multi1",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      "grafana_asserts_log_config.multi2",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAssertsLogConfig_optimisticLocking(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	baseName := fmt.Sprintf("lock-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccAssertsLogConfigCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAssertsLogConfigOptimisticLockingConfig(baseName),
				// This test expects optimistic locking conflicts when creating multiple
				// resources simultaneously. The error validates that the locking mechanism works.
				ExpectError: regexp.MustCompile(`failed to create log configuration.*giving up after.*attempt`),
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
  priority        = 1000
  default_config  = false
  data_source_uid = "grafanacloud-logs"
  error_label     = "error"
  
  match {
    property = "asserts_entity_type"
    op       = "EQUALS"
    values   = ["Service"]
  }
  
  entity_property_to_log_label_mapping = {
    "otel_namespace" = "service_namespace"
    "otel_service"   = "service_name"
  }
  
  filter_by_span_id  = true
  filter_by_trace_id = true
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
  priority        = 1001
  default_config  = %s
  data_source_uid = "grafanacloud-logs"
  
  match {
    property = "otel_service"
    op       = "IS_NOT_NULL"
    values   = []
  }
}
`, name, defaultVal)
}

func testAccAssertsLogConfigConfigNamedUpdate(name string) string {
	return fmt.Sprintf(`
resource "grafana_asserts_log_config" "test" {
  name            = "%s"
  priority        = 1002
  default_config  = false
  data_source_uid = "grafanacloud-logs"
  
  match {
    property = "otel_service"
    op       = "IS_NOT_NULL"
    values   = []
  }
}
`, name)
}

func testAccAssertsLogConfigConfigFullNamed(name string) string {
	return fmt.Sprintf(`
resource "grafana_asserts_log_config" "full" {
  name            = "%s"
  priority        = 1002
  default_config  = false
  data_source_uid = "loki-uid-456"
  error_label     = "error"
  
  match {
    property = "service"
    op       = "EQUALS"
    values   = ["api", "web"]
  }
  
  match {
    property = "environment"
    op       = "CONTAINS"
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

func testAccAssertsLogConfigMultipleConfig(baseName string) string {
	return fmt.Sprintf(`
resource "grafana_asserts_log_config" "multi1" {
  name            = "%s-1"
  priority        = 2001
  default_config  = false
  data_source_uid = "loki-uid-multi1"
}

resource "grafana_asserts_log_config" "multi2" {
  name            = "%s-2"
  priority        = 2002
  default_config  = false
  data_source_uid = "loki-uid-multi2"
  
  depends_on = [grafana_asserts_log_config.multi1]
}
`, baseName, baseName)
}

func testAccAssertsLogConfigOptimisticLockingConfig(baseName string) string {
	return fmt.Sprintf(`
resource "grafana_asserts_log_config" "lock1" {
  name            = "%s-1"
  priority        = 3001
  default_config  = false
  data_source_uid = "loki-uid-lock1"
}

resource "grafana_asserts_log_config" "lock2" {
  name            = "%s-2"
  priority        = 3002
  default_config  = false
  data_source_uid = "loki-uid-lock2"
}
`, baseName, baseName)
}
