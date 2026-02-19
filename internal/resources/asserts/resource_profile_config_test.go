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

func TestAccAssertsProfileConfig_basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccAssertsProfileConfigCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAssertsProfileConfigConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_asserts_profile_config.test", "name", "test-basic"),
					resource.TestCheckResourceAttr("grafana_asserts_profile_config.test", "priority", "1000"),
					resource.TestCheckResourceAttr("grafana_asserts_profile_config.test", "default_config", "false"),
					resource.TestCheckResourceAttr("grafana_asserts_profile_config.test", "data_source_uid", "grafanacloud-profiles"),
					// match rules
					resource.TestCheckResourceAttr("grafana_asserts_profile_config.test", "match.0.property", "environment"),
					resource.TestCheckResourceAttr("grafana_asserts_profile_config.test", "match.0.op", "="),
					resource.TestCheckResourceAttr("grafana_asserts_profile_config.test", "match.0.values.0", "production"),
					// mappings
					resource.TestCheckResourceAttr("grafana_asserts_profile_config.test", "entity_property_to_profile_label_mapping.otel_namespace", "service_namespace"),
					resource.TestCheckResourceAttr("grafana_asserts_profile_config.test", "entity_property_to_profile_label_mapping.otel_service", "service_name"),
				),
			},
			{
				ResourceName:      "grafana_asserts_profile_config.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAssertsProfileConfig_update(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	rName := fmt.Sprintf("test-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccAssertsProfileConfigCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAssertsProfileConfigConfigNamed(rName, false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_asserts_profile_config.test", "name", rName),
					resource.TestCheckResourceAttr("grafana_asserts_profile_config.test", "priority", "1001"),
					resource.TestCheckResourceAttr("grafana_asserts_profile_config.test", "default_config", "false"),
				),
			},
			{
				Config: testAccAssertsProfileConfigConfigNamedUpdate(rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_asserts_profile_config.test", "name", rName),
					resource.TestCheckResourceAttr("grafana_asserts_profile_config.test", "priority", "1002"),
					resource.TestCheckResourceAttr("grafana_asserts_profile_config.test", "default_config", "false"),
				),
			},
		},
	})
}

func TestAccAssertsProfileConfig_fullFields(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	rName := fmt.Sprintf("full-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccAssertsProfileConfigCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAssertsProfileConfigConfigFullNamed(rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_asserts_profile_config.full", "name", rName),
					resource.TestCheckResourceAttr("grafana_asserts_profile_config.full", "priority", "1002"),
					resource.TestCheckResourceAttr("grafana_asserts_profile_config.full", "default_config", "false"),
					resource.TestCheckResourceAttr("grafana_asserts_profile_config.full", "data_source_uid", "grafanacloud-profiles"),
					resource.TestCheckResourceAttr("grafana_asserts_profile_config.full", "match.0.property", "cluster"),
					resource.TestCheckResourceAttr("grafana_asserts_profile_config.full", "match.0.op", "="),
					resource.TestCheckResourceAttr("grafana_asserts_profile_config.full", "match.0.values.0", "prod-cluster"),
					resource.TestCheckResourceAttr("grafana_asserts_profile_config.full", "match.1.property", "service"),
					resource.TestCheckResourceAttr("grafana_asserts_profile_config.full", "match.1.op", "="),
					resource.TestCheckResourceAttr("grafana_asserts_profile_config.full", "match.1.values.0", "api"),
					resource.TestCheckResourceAttr("grafana_asserts_profile_config.full", "match.1.values.1", "web"),
					resource.TestCheckResourceAttr("grafana_asserts_profile_config.full", "match.2.property", "environment"),
					resource.TestCheckResourceAttr("grafana_asserts_profile_config.full", "match.2.op", "CONTAINS"),
					resource.TestCheckResourceAttr("grafana_asserts_profile_config.full", "match.2.values.0", "prod"),
					// mappings
					resource.TestCheckResourceAttr("grafana_asserts_profile_config.full", "entity_property_to_profile_label_mapping.service", "service_name"),
					resource.TestCheckResourceAttr("grafana_asserts_profile_config.full", "entity_property_to_profile_label_mapping.environment", "env"),
				),
			},
		},
	})
}

func testAccAssertsProfileConfigCheckDestroy(s *terraform.State) error {
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
		if rs.Type != "grafana_asserts_profile_config" {
			continue
		}

		name := rs.Primary.ID
		for {
			request := client.AssertsAPIClient.ProfileDrilldownConfigControllerAPI.GetTenantProfileConfig(context.Background()).
				XScopeOrgID(fmt.Sprintf("%d", stackID))

			tenantConfig, _, err := request.Execute()
			if err != nil {
				if common.IsNotFoundError(err) {
					break
				}
				return fmt.Errorf("error checking profile config destruction: %s", err)
			}

			found := false
			for _, config := range tenantConfig.GetProfileDrilldownConfigs() {
				if config.GetName() == name {
					found = true
					break
				}
			}

			if !found {
				break
			}

			if time.Now().After(deadline) {
				return fmt.Errorf("profile config %s still exists", name)
			}
			time.Sleep(2 * time.Second)
		}
	}

	return nil
}

const testAccAssertsProfileConfigConfig = `
resource "grafana_asserts_profile_config" "test" {
  name            = "test-basic"
  priority        = 1000
  default_config  = false
  data_source_uid = "grafanacloud-profiles"

  match {
    property = "environment"
    op       = "="
    values   = ["production"]
  }

  entity_property_to_profile_label_mapping = {
    "otel_namespace" = "service_namespace"
    "otel_service"   = "service_name"
  }
}
`

func testAccAssertsProfileConfigConfigNamed(name string, defaultCfg bool) string {
	defaultVal := "false"
	if defaultCfg {
		defaultVal = "true"
	}
	return fmt.Sprintf(`
resource "grafana_asserts_profile_config" "test" {
  name            = "%s"
  priority        = 1001
  default_config  = %s
  data_source_uid = "grafanacloud-profiles"

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

func testAccAssertsProfileConfigConfigNamedUpdate(name string) string {
	return fmt.Sprintf(`
resource "grafana_asserts_profile_config" "test" {
  name            = "%s"
  priority        = 1002
  default_config  = false
  data_source_uid = "grafanacloud-profiles"

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

func testAccAssertsProfileConfigConfigFullNamed(name string) string {
	return fmt.Sprintf(`
resource "grafana_asserts_profile_config" "full" {
  name            = "%s"
  priority        = 1002
  default_config  = false
  data_source_uid = "grafanacloud-profiles"

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

  entity_property_to_profile_label_mapping = {
    "service"     = "service_name"
    "environment" = "env"
  }
}
`, name)
}
