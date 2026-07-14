package oncall_test

import (
	"fmt"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceTeam_Basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	teamName := fmt.Sprintf("test-acc-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceTeamConfig(teamName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.grafana_oncall_team.test-acc-team", "id"),
					resource.TestCheckResourceAttr("data.grafana_oncall_team.test-acc-team", "name", teamName),
				),
			},
		},
	})
}

func testAccDataSourceTeamConfig(teamName string) string {
	return fmt.Sprintf(`
resource "grafana_team" "test-acc-team" {
	name = "%[1]s"
}

data "grafana_oncall_team" "test-acc-team" {
	depends_on = [grafana_team.test-acc-team]
	name = "%[1]s"
}
`, teamName)
}
