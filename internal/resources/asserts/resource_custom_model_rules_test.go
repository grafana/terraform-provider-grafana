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

func TestAccAssertsCustomModelRules_basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	stackID := testutils.Provider.Meta().(*common.Client).GrafanaStackID
	rName := fmt.Sprintf("test-acc-cmr-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccAssertsCustomModelRulesCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAssertsCustomModelRulesConfig(stackID, rName),
				Check: resource.ComposeTestCheckFunc(
					testAccAssertsCustomModelRulesCheckExists("grafana_asserts_custom_model_rules.test", stackID, rName),
					resource.TestCheckResourceAttr("grafana_asserts_custom_model_rules.test", "name", rName),
				),
			},
			{
				// Test import
				ResourceName:            "grafana_asserts_custom_model_rules.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"rules"},
			},
			{
				// Test update
				Config: testAccAssertsCustomModelRulesConfigUpdated(stackID, rName),
				Check: resource.ComposeTestCheckFunc(
					testAccAssertsCustomModelRulesCheckExists("grafana_asserts_custom_model_rules.test", stackID, rName),
					resource.TestCheckResourceAttr("grafana_asserts_custom_model_rules.test", "name", rName),
				),
			},
		},
	})
}

func testAccAssertsCustomModelRulesCheckExists(rn string, stackID int64, name string) resource.TestCheckFunc {
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

		_, _, err := client.CustomModelRulesControllerAPI.GetModelRules(ctx, name).XScopeOrgID(fmt.Sprintf("%d", stackID)).Execute()
		if err != nil {
			return fmt.Errorf("error getting custom model rules: %s", err)
		}
		return nil
	}
}

func testAccAssertsCustomModelRulesCheckDestroy(s *terraform.State) error {
	client := testutils.Provider.Meta().(*common.Client).AssertsAPIClient
	ctx := context.Background()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "grafana_asserts_custom_model_rules" {
			continue
		}

		name := rs.Primary.ID
		stackID := fmt.Sprintf("%d", testutils.Provider.Meta().(*common.Client).GrafanaStackID)

		_, _, err := client.CustomModelRulesControllerAPI.GetModelRules(ctx, name).XScopeOrgID(stackID).Execute()
		if err != nil {
			if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "404") {
				continue
			}
			return fmt.Errorf("error checking custom model rules destruction: %s", err)
		}
		return fmt.Errorf("custom model rules %s still exists", name)
	}

	return nil
}

func testAccAssertsCustomModelRulesConfig(stackID int64, name string) string {
	return fmt.Sprintf(`
resource "grafana_asserts_custom_model_rules" "test" {
  name = "%s"
  rules {
    entity {
      type = "Service"
      name = "Service"
      defined_by {
        query = "up{job!=''}"
      }
    }
  }
}
`, name)
}

func testAccAssertsCustomModelRulesConfigUpdated(stackID int64, name string) string {
	return fmt.Sprintf(`
resource "grafana_asserts_custom_model_rules" "test" {
  name = "%s"
  rules {
    entity {
      type = "Service"
      name = "Service"
      defined_by {
        query = "up{job!=''}"
      }
    }
    entity {
      type = "Pod"
      name = "Pod"
      defined_by {
        query = "up{pod!=''}"
      }
    }
  }
}
`, name)
}

// TestAccAssertsCustomModelRules_eventualConsistencyStress tests multiple resources created simultaneously
// to verify the retry logic handles eventual consistency properly
func TestAccAssertsCustomModelRules_eventualConsistencyStress(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)
	testutils.CheckStressTestsEnabled(t)

	stackID := testutils.Provider.Meta().(*common.Client).GrafanaStackID
	baseName := fmt.Sprintf("stress-cmr-%s", acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccAssertsCustomModelRulesCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAssertsCustomModelRulesStressConfig(stackID, baseName),
				Check: resource.ComposeTestCheckFunc(
					// Check that all resources were created successfully
					testAccAssertsCustomModelRulesCheckExists("grafana_asserts_custom_model_rules.test1", stackID, baseName+"-1"),
					testAccAssertsCustomModelRulesCheckExists("grafana_asserts_custom_model_rules.test2", stackID, baseName+"-2"),
					testAccAssertsCustomModelRulesCheckExists("grafana_asserts_custom_model_rules.test3", stackID, baseName+"-3"),
					resource.TestCheckResourceAttr("grafana_asserts_custom_model_rules.test1", "name", baseName+"-1"),
					resource.TestCheckResourceAttr("grafana_asserts_custom_model_rules.test2", "name", baseName+"-2"),
					resource.TestCheckResourceAttr("grafana_asserts_custom_model_rules.test3", "name", baseName+"-3"),
				),
			},
		},
	})
}

func testAccAssertsCustomModelRulesStressConfig(stackID int64, baseName string) string {
	return fmt.Sprintf(`
resource "grafana_asserts_custom_model_rules" "test1" {
  name = "%s-1"
  rules {
    entity {
      type = "Service"
      name = "Service"
      defined_by {
        query = "up{job!=''}"
      }
    }
  }
}

resource "grafana_asserts_custom_model_rules" "test2" {
  name = "%s-2"
  rules {
    entity {
      type = "Pod"
      name = "Pod"
      defined_by {
        query = "up{pod!=''}"
      }
    }
  }
}

resource "grafana_asserts_custom_model_rules" "test3" {
  name = "%s-3"
  rules {
    entity {
      type = "Namespace"
      name = "Namespace"
      defined_by {
        query = "up{namespace!=''}"
      }
    }
  }
}

`, baseName, baseName, baseName)
}
