package grafana

import (
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccTeam_External_Groups(t *testing.T) {
	var team gapi.Team

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTeamConfig_groupAdd,
				Check: resource.ComposeTestCheckFunc(
					testAccTeamCheckExists("grafana_team_external_group.test", &team),
					resource.TestCheckResourceAttr(
						"grafana_team_external_group.test", "team_id", "1",
					),
					resource.TestCheckResourceAttr(
						"grafana_team_external_group.test", "groups.#", "1",
					),
					resource.TestCheckResourceAttr(
						"grafana_team_external_group.test", "groups.0", "test-group",
					),
				),
			},
			{
				Config: testAccTeamConfig_groupRemove,
				Check: resource.ComposeTestCheckFunc(
					testAccTeamCheckExists("grafana_team_external_group.test", &team),
					resource.TestCheckResourceAttr(
						"grafana_team_external_group.test", "team_id", "1",
					),
					resource.TestCheckResourceAttr(
						"grafana_team_external_group.test", "groups.#", "0",
					),
				),
			},
		},
	})
}

const testAccTeamConfig_groupAdd = `
resource "grafana_team_external_group" "test" {
  team_id    = 1
  groups = [
    "test-group",
  ]
}
`
const testAccTeamConfig_groupRemove = `
resource "grafana_team_external_group" "test" {
  team_id    = 1
  groups = [ ]
}
`
