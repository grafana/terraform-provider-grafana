package grafana

import (
	"fmt"
	"testing"

	amixrAPI "github.com/grafana/amixr-api-go-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccAmixrRoute_basic(t *testing.T) {
	CheckCloudInstanceTestsEnabled(t)

	riName := fmt.Sprintf("integration-%s", acctest.RandString(8))
	rrRegex := fmt.Sprintf("regex-%s", acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAmixrRouteResourceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAmixrRouteConfig(riName, rrRegex),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAmixrRouteResourceExists("grafana_amixr_route.test-acc-route"),
				),
			},
		},
	})
}

func testAccCheckAmixrRouteResourceDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*client).amixrAPI
	for _, r := range s.RootModule().Resources {
		if r.Type != "grafana_amixr_route" {
			continue
		}

		if _, _, err := client.Routes.GetRoute(r.Primary.ID, &amixrAPI.GetRouteOptions{}); err == nil {
			return fmt.Errorf("Route still exists")
		}
	}
	return nil
}

func testAccAmixrRouteConfig(riName string, rrRegex string) string {
	return fmt.Sprintf(`
resource "grafana_amixr_integration" "test-acc-integration" {
	name = "%s"
	type = "grafana"
	default_route {
	}
}

resource "grafana_amixr_escalation_chain" "test-acc-escalation-chain"{
	name = "acc-test"
}

resource "grafana_amixr_route" "test-acc-route" {
	integration_id = grafana_amixr_integration.test-acc-integration.id
	escalation_chain_id = grafana_amixr_escalation_chain.test-acc-escalation-chain.id
	routing_regex = "%s"
	position = 0
}
`, riName, rrRegex)
}

func testAccCheckAmixrRouteResourceExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No Route ID is set")
		}

		client := testAccProvider.Meta().(*client).amixrAPI

		found, _, err := client.Routes.GetRoute(rs.Primary.ID, &amixrAPI.GetRouteOptions{})
		if err != nil {
			return err
		}
		if found.ID != rs.Primary.ID {
			return fmt.Errorf("Route policy not found: %v - %v", rs.Primary.ID, found)
		}
		return nil
	}
}
