package oncall_test

import (
	"fmt"
	"testing"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/grafana/terraform-provider-grafana/v2/internal/common"
	"github.com/grafana/terraform-provider-grafana/v2/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccDataSourceIntegration_Basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	integrationID := "test_integration"
	randomName := fmt.Sprintf("test-name-%s", acctest.RandString(10))

	integrationPath := fmt.Sprintf("grafana_oncall_integration.%s", integrationID)
	dataSourcePath := "data.grafana_oncall_integration.test_integration_ds"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy: testAccCheckDataSourceIntegrationResourceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceIntegration(integrationID, randomName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIntegrationResourceExists(fmt.Sprintf("grafana_oncall_integration.%s", integrationID)),
					resource.TestCheckResourceAttrPair(
						integrationPath, "id",
						dataSourcePath, "id",
					),
				),
			},
		},
	})
}

func testAccCheckDataSourceIntegrationResourceDestroy(s *terraform.State) error {
	client := testutils.Provider.Meta().(*common.Client).OnCallClient
	for _, r := range s.RootModule().Resources {
		if r.Type != "grafana_oncall_integration" {
			continue
		}

		if _, _, err := client.Integrations.GetIntegration(r.Primary.ID, &onCallAPI.GetIntegrationOptions{}); err == nil {
			return fmt.Errorf("expected a 404 but found an integration")
		}
	}
	return nil
}

func testAccDataSourceIntegration(integrationID string, randomName string) string {
	return fmt.Sprintf(`
resource "grafana_oncall_integration" "%[1]s" {
	name = "%[2]s"
	type = "grafana"
	default_route {
	}
}

data "grafana_oncall_integration" "test_integration_ds" {
	id = grafana_oncall_integration.%[1]s.id
}
`, integrationID, randomName)
}

func testAccCheckIntegrationResourceExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No integration ID is set")
		}

		client := testutils.Provider.Meta().(*common.Client).OnCallClient

		found, _, err := client.Integrations.GetIntegration(rs.Primary.ID, &onCallAPI.GetIntegrationOptions{})
		if err != nil {
			return err
		}
		if found.ID != rs.Primary.ID {
			return fmt.Errorf("Integration not found: %v - %v", rs.Primary.ID, found)
		}
		return nil
	}
}
