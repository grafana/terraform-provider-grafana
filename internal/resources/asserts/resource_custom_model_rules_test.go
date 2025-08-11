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
				ResourceName:      "grafana_asserts_custom_model_rules.test",
				ImportState:       true,
				ImportStateVerify: true,
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
			if strings.Contains(err.Error(), "not found") {
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
  rules = <<-EOT
    entities:
      - name: "Service"
        type: "Service"
        definedBy:
          - query: "up{job!=''}"
  EOT
}
`, name)
}

func testAccAssertsCustomModelRulesConfigUpdated(stackID int64, name string) string {
	return fmt.Sprintf(`
resource "grafana_asserts_custom_model_rules" "test" {
  name = "%s"
  rules = <<-EOT
    entities:
      - name: "Service"
        type: "Service"
        definedBy:
          - query: "up{job!=''}"
      - name: "Pod"
        type: "Pod"
        definedBy:
          - query: "up{pod!=''}"
  EOT
}
`, name)
}
