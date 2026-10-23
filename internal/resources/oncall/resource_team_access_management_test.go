package oncall_test

import (
	"fmt"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccTeamAccessManagement_basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	teamName := fmt.Sprintf("test-acc-%s", acctest.RandString(8))
	const rn = "grafana_oncall_team_access_management.test"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTeamAccessManagementConfig(teamName, false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(rn, "id"),
					resource.TestCheckResourceAttr(rn, "is_sharing_resources_to_all", "false"),
					resource.TestCheckResourceAttr("data.grafana_oncall_team.test", "is_sharing_resources_to_all", "false"),
				),
			},
			{
				Config: testAccTeamAccessManagementConfig(teamName, true),
				Check:  resource.TestCheckResourceAttr(rn, "is_sharing_resources_to_all", "true"),
			},
			{
				ResourceName:      rn,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccTeamAccessManagementConfig(teamName string, shared bool) string {
	return fmt.Sprintf(`
resource "grafana_team" "test" {
  name = "%[1]s"
}

data "grafana_oncall_team" "test" {
  name       = grafana_team.test.name
  depends_on = [grafana_oncall_team_access_management.test]
}

resource "grafana_oncall_team_access_management" "test" {
  team_id                     = data.grafana_oncall_team.pre.id
  is_sharing_resources_to_all = %[2]t
}

data "grafana_oncall_team" "pre" {
  name       = grafana_team.test.name
  depends_on = [grafana_team.test]
}
`, teamName, shared)
}
