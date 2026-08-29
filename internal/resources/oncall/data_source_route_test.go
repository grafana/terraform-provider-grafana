package oncall_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceRoute_Basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	randomName := fmt.Sprintf("test-acc-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceRouteConfig(randomName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.grafana_oncall_route.test", "id"),
					resource.TestCheckResourceAttrPair(
						"grafana_oncall_route.test", "id",
						"data.grafana_oncall_route.test", "id",
					),
					resource.TestCheckResourceAttrPair(
						"grafana_oncall_route.test", "escalation_chain_id",
						"data.grafana_oncall_route.test", "escalation_chain_id",
					),
					resource.TestCheckResourceAttrPair(
						"grafana_oncall_route.test", "position",
						"data.grafana_oncall_route.test", "position",
					),
				),
			},
		},
	})
}

func TestAccDataSourceRoute_NotFound(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDataSourceRouteNotFoundConfig(),
				ExpectError: regexp.MustCompile(`couldn't find a route matching`),
			},
		},
	})
}

func testAccDataSourceRouteNotFoundConfig() string {
	return `
data "grafana_oncall_route" "test" {
	integration_id = "nonexistent"
	routing_regex  = ".*doesnotexist.*"
}
`
}

func testAccDataSourceRouteConfig(name string) string {
	return fmt.Sprintf(`
resource "grafana_oncall_integration" "test" {
	name = "%[1]s"
	type = "grafana"
	default_route {}
}

resource "grafana_oncall_escalation_chain" "test" {
	name = "%[1]s"
}

resource "grafana_oncall_route" "test" {
	integration_id      = grafana_oncall_integration.test.id
	escalation_chain_id = grafana_oncall_escalation_chain.test.id
	routing_regex       = ".*critical.*"
	position            = 0
}

data "grafana_oncall_route" "test" {
	integration_id = grafana_oncall_integration.test.id
	routing_regex  = grafana_oncall_route.test.routing_regex
}
`, name)
}
