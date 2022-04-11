package grafana

import (
	"fmt"
	amixrAPI "github.com/grafana/amixr-api-go-client"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccAmixrIntegration_basic(t *testing.T) {
	CheckCloudInstanceTestsEnabled(t)

	rName := fmt.Sprintf("test-acc-%s", acctest.RandString(8))
	rType := "grafana"

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAmixrIntegrationResourceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAmixrIntegrationConfig(rName, rType),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAmixrIntegrationResourceExists("grafana_amixr_integration.test-acc-integration"),
					resource.TestCheckResourceAttr("grafana_amixr_integration.test-acc-integration", "name", rName),
					resource.TestCheckResourceAttr("grafana_amixr_integration.test-acc-integration", "type", rType),
					resource.TestCheckResourceAttrSet("grafana_amixr_integration.test-acc-integration", "link"),
				),
			},
		},
	})
}

func testAccCheckAmixrIntegrationResourceDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*amixrAPI.Client)
	for _, r := range s.RootModule().Resources {
		if r.Type != "grafana_amixr_integration" {
			continue
		}

		if _, _, err := client.Integrations.GetIntegration(r.Primary.ID, &amixrAPI.GetIntegrationOptions{}); err == nil {
			return fmt.Errorf("integration still exists")
		}

	}
	return nil
}

func testAccAmixrIntegrationConfig(rName, rType string) string {
	return fmt.Sprintf(`
resource "grafana_amixr_integration" "test-acc-integration" {
	name = "%s"
	type = "%s"
}
`, rName, rType)
}

func testAccCheckAmixrIntegrationResourceExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No Integration ID is set")
		}

		client := testAccProvider.Meta().(*amixrAPI.Client)

		found, _, err := client.Integrations.GetIntegration(rs.Primary.ID, &amixrAPI.GetIntegrationOptions{})
		if err != nil {
			return err
		}
		if found.ID != rs.Primary.ID {
			return fmt.Errorf("Integration not found: %v - %v", rs.Primary.ID, found)
		}
		return nil
	}
}
