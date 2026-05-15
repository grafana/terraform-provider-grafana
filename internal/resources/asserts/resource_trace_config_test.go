package asserts_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// cleanupTraceConfigs deletes all test trace configs (those with names starting with "test-" or "full-").
// This ensures tests don't fail due to leftover configs from previous runs.
func cleanupTraceConfigs(t *testing.T) {
	t.Helper()

	client := testutils.Provider.Meta().(*common.Client)
	if client.AssertsAPIClient == nil {
		t.Log("Asserts API client not configured, skipping cleanup")
		return
	}

	stackID := client.GrafanaStackID
	if stackID == 0 {
		t.Log("Stack ID not configured, skipping cleanup")
		return
	}

	ctx := context.Background()

	// Get all trace configs
	request := client.AssertsAPIClient.TraceDrilldownConfigControllerAPI.GetTenantTraceConfig(ctx).
		XScopeOrgID(fmt.Sprintf("%d", stackID))

	tenantConfig, _, err := request.Execute()
	if err != nil {
		t.Logf("Failed to get trace configs for cleanup: %v", err)
		return
	}

	// Delete test configs (those starting with "test-" or "full-")
	for _, config := range tenantConfig.GetTraceDrilldownConfigs() {
		name := config.GetName()
		if strings.HasPrefix(name, "test-") || strings.HasPrefix(name, "full-") {
			t.Logf("Cleaning up leftover trace config: %s", name)
			deleteRequest := client.AssertsAPIClient.TraceDrilldownConfigControllerAPI.DeleteConfig(ctx, name).
				XScopeOrgID(fmt.Sprintf("%d", stackID))
			_, err := deleteRequest.Execute()
			if err != nil {
				t.Logf("Failed to delete trace config %s: %v", name, err)
			}
		}
	}
}

func TestAccAssertsTraceConfig_basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)
	cleanupTraceConfigs(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccAssertsTraceConfigCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAssertsTraceConfigConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_asserts_trace_config.test", "name", "test-basic"),
					resource.TestCheckResourceAttr("grafana_asserts_trace_config.test", "priority", "1000"),
					resource.TestCheckResourceAttr("grafana_asserts_trace_config.test", "default_config", "false"),
					resource.TestCheckResourceAttr("grafana_asserts_trace_config.test", "data_source_uid", "grafanacloud-tempo"),
					// match rules
					resource.TestCheckResourceAttr("grafana_asserts_trace_config.test", "match.0.property", "environment"),
					resource.TestCheckResourceAttr("grafana_asserts_trace_config.test", "match.0.op", "="),
					resource.TestCheckResourceAttr("grafana_asserts_trace_config.test", "match.0.values.0", "production"),
					// mappings
					resource.TestCheckResourceAttr("grafana_asserts_trace_config.test", "entity_property_to_trace_label_mapping.otel_namespace", "service.namespace"),
					resource.TestCheckResourceAttr("grafana_asserts_trace_config.test", "entity_property_to_trace_label_mapping.otel_service", "service.name"),
				),
			},
			{
				ResourceName:      "grafana_asserts_trace_config.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAssertsTraceConfig_update(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)
	cleanupTraceConfigs(t)

	rName := fmt.Sprintf("test-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccAssertsTraceConfigCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAssertsTraceConfigConfigNamed(rName, false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_asserts_trace_config.test", "name", rName),
					resource.TestCheckResourceAttr("grafana_asserts_trace_config.test", "priority", "1001"),
					resource.TestCheckResourceAttr("grafana_asserts_trace_config.test", "default_config", "false"),
				),
			},
			{
				Config: testAccAssertsTraceConfigConfigNamedUpdate(rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_asserts_trace_config.test", "name", rName),
					resource.TestCheckResourceAttr("grafana_asserts_trace_config.test", "priority", "1002"),
					resource.TestCheckResourceAttr("grafana_asserts_trace_config.test", "default_config", "false"),
				),
			},
		},
	})
}

func TestAccAssertsTraceConfig_fullFields(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)
	cleanupTraceConfigs(t)

	rName := fmt.Sprintf("full-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccAssertsTraceConfigCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAssertsTraceConfigConfigFullNamed(rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_asserts_trace_config.full", "name", rName),
					resource.TestCheckResourceAttr("grafana_asserts_trace_config.full", "priority", "1002"),
					resource.TestCheckResourceAttr("grafana_asserts_trace_config.full", "default_config", "false"),
					resource.TestCheckResourceAttr("grafana_asserts_trace_config.full", "data_source_uid", "grafanacloud-tempo"),
					resource.TestCheckResourceAttr("grafana_asserts_trace_config.full", "match.0.property", "cluster"),
					resource.TestCheckResourceAttr("grafana_asserts_trace_config.full", "match.0.op", "="),
					resource.TestCheckResourceAttr("grafana_asserts_trace_config.full", "match.0.values.0", "prod-cluster"),
					resource.TestCheckResourceAttr("grafana_asserts_trace_config.full", "match.1.property", "service"),
					resource.TestCheckResourceAttr("grafana_asserts_trace_config.full", "match.1.op", "="),
					resource.TestCheckResourceAttr("grafana_asserts_trace_config.full", "match.1.values.0", "api"),
					resource.TestCheckResourceAttr("grafana_asserts_trace_config.full", "match.1.values.1", "web"),
					resource.TestCheckResourceAttr("grafana_asserts_trace_config.full", "match.2.property", "environment"),
					resource.TestCheckResourceAttr("grafana_asserts_trace_config.full", "match.2.op", "CONTAINS"),
					resource.TestCheckResourceAttr("grafana_asserts_trace_config.full", "match.2.values.0", "prod"),
					// mappings
					resource.TestCheckResourceAttr("grafana_asserts_trace_config.full", "entity_property_to_trace_label_mapping.service", "service.name"),
					resource.TestCheckResourceAttr("grafana_asserts_trace_config.full", "entity_property_to_trace_label_mapping.environment", "deployment.environment"),
				),
			},
		},
	})
}

func testAccAssertsTraceConfigCheckDestroy(s *terraform.State) error {
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
		if rs.Type != "grafana_asserts_trace_config" {
			continue
		}

		name := rs.Primary.ID
		for {
			request := client.AssertsAPIClient.TraceDrilldownConfigControllerAPI.GetTenantTraceConfig(context.Background()).
				XScopeOrgID(fmt.Sprintf("%d", stackID))

			tenantConfig, _, err := request.Execute()
			if err != nil {
				if common.IsNotFoundError(err) {
					break
				}
				return fmt.Errorf("error checking trace config destruction: %s", err)
			}

			found := false
			for _, config := range tenantConfig.GetTraceDrilldownConfigs() {
				if config.GetName() == name {
					found = true
					break
				}
			}

			if !found {
				break
			}

			if time.Now().After(deadline) {
				return fmt.Errorf("trace config %s still exists", name)
			}
			time.Sleep(2 * time.Second)
		}
	}

	return nil
}

const testAccAssertsTraceConfigConfig = `
resource "grafana_asserts_trace_config" "test" {
  name            = "test-basic"
  priority        = 1000
  default_config  = false
  data_source_uid = "grafanacloud-tempo"

  match {
    property = "environment"
    op       = "="
    values   = ["production"]
  }

  entity_property_to_trace_label_mapping = {
    "otel_namespace" = "service.namespace"
    "otel_service"   = "service.name"
  }
}
`

func testAccAssertsTraceConfigConfigNamed(name string, defaultCfg bool) string {
	defaultVal := "false"
	if defaultCfg {
		defaultVal = "true"
	}
	return fmt.Sprintf(`
resource "grafana_asserts_trace_config" "test" {
  name            = "%s"
  priority        = 1001
  default_config  = %s
  data_source_uid = "grafanacloud-tempo"

  match {
    property = "namespace"
    op       = "="
    values   = ["default"]
  }

  match {
    property = "otel_service"
    op       = "IS NOT NULL"
    values   = []
  }
}
`, name, defaultVal)
}

func testAccAssertsTraceConfigConfigNamedUpdate(name string) string {
	return fmt.Sprintf(`
resource "grafana_asserts_trace_config" "test" {
  name            = "%s"
  priority        = 1002
  default_config  = false
  data_source_uid = "grafanacloud-tempo"

  match {
    property = "namespace"
    op       = "="
    values   = ["default"]
  }

  match {
    property = "otel_service"
    op       = "IS NOT NULL"
    values   = []
  }
}
`, name)
}

func testAccAssertsTraceConfigConfigFullNamed(name string) string {
	return fmt.Sprintf(`
resource "grafana_asserts_trace_config" "full" {
  name            = "%s"
  priority        = 1002
  default_config  = false
  data_source_uid = "grafanacloud-tempo"

  match {
    property = "cluster"
    op       = "="
    values   = ["prod-cluster"]
  }

  match {
    property = "service"
    op       = "="
    values   = ["api", "web"]
  }

  match {
    property = "environment"
    op       = "CONTAINS"
    values   = ["prod"]
  }

  entity_property_to_trace_label_mapping = {
    "service"     = "service.name"
    "environment" = "deployment.environment"
  }
}
`, name)
}
