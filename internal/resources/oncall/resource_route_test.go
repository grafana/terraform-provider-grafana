package oncall_test

import (
	"fmt"
	"testing"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
	"github.com/grafana/terraform-provider-grafana/v3/pkg/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccOnCallRoute_basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	riName := fmt.Sprintf("integration-%s", acctest.RandString(8))
	rrRegex := fmt.Sprintf("regex-%s", acctest.RandString(8))

	// TODO: Make parallelizable
	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckOnCallRouteResourceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOnCallRouteConfig(riName, rrRegex),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOnCallRouteResourceExists("grafana_oncall_route.test-acc-route"),
				),
			},
		},
	})
}

func testAccCheckOnCallRouteResourceDestroy(s *terraform.State) error {
	client := testutils.Provider.Meta().(*client.Client).OnCallClient
	for _, r := range s.RootModule().Resources {
		if r.Type != "grafana_oncall_route" {
			continue
		}

		if _, _, err := client.Routes.GetRoute(r.Primary.ID, &onCallAPI.GetRouteOptions{}); err == nil {
			return fmt.Errorf("Route still exists")
		}
	}
	return nil
}

func testAccOnCallRouteConfig(riName string, rrRegex string) string {
	return fmt.Sprintf(`
resource "grafana_oncall_integration" "test-acc-integration" {
	name = "%s"
	type = "grafana"
	default_route {
	    slack {
	        enabled = false
	    }
	    telegram {
	        enabled = false
	    }
	}
}

resource "grafana_oncall_escalation_chain" "test-acc-escalation-chain"{
	name = "acc-test-%s"
}

resource "grafana_oncall_route" "test-acc-route" {
	integration_id = grafana_oncall_integration.test-acc-integration.id
	escalation_chain_id = grafana_oncall_escalation_chain.test-acc-escalation-chain.id
	routing_regex = "%s"
	position = 0
    slack {
        enabled = false
    }
    telegram {
        enabled = false
    }
}
`, riName, riName, rrRegex)
}

func testAccCheckOnCallRouteResourceExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No Route ID is set")
		}

		client := testutils.Provider.Meta().(*client.Client).OnCallClient

		found, _, err := client.Routes.GetRoute(rs.Primary.ID, &onCallAPI.GetRouteOptions{})
		if err != nil {
			return err
		}
		if found.ID != rs.Primary.ID {
			return fmt.Errorf("Route policy not found: %v - %v", rs.Primary.ID, found)
		}
		return nil
	}
}
