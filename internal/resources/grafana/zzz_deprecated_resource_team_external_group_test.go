package grafana_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/grafana/terraform-provider-grafana/internal/resources/grafana"
	"github.com/grafana/terraform-provider-grafana/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccTeamExternalGroup_basic(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t)

	teamID := int64(-1)

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      testAccTeamExternalGroupCheckDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testAccTeamConfig_groupAdd,
				Check: resource.ComposeTestCheckFunc(
					testAccTeamExternalGroupCheckExists("grafana_team_external_group.test", &teamID),
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
					testAccTeamExternalGroupCheckExists("grafana_team_external_group.test", &teamID),
					resource.TestCheckResourceAttr(
						"grafana_team_external_group.test", "groups.#", "0",
					),
				),
			},
			{
				ResourceName:      "grafana_team_external_group.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccTeamExternalGroupCheckExists(rn string, teamID *int64) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("Resource not found: %s\n %#v", rn, s.RootModule().Resources)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("Resource id not set")
		}

		orgID, teamIDStr := grafana.SplitOrgResourceID(rs.Primary.ID)
		client := testutils.Provider.Meta().(*common.Client).GrafanaAPI.WithOrgID(orgID)

		gotTeamID, err := strconv.ParseInt(teamIDStr, 10, 64)
		if err != nil {
			return fmt.Errorf("team id is malformed")
		}

		_, err = client.TeamGroups(gotTeamID)
		if err != nil {
			return fmt.Errorf("Error getting team external groups: %s", err)
		}

		*teamID = gotTeamID

		return nil
	}
}

func testAccTeamExternalGroupCheckDestroy() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// you can't really destroy dashboard permissions so nothing to check for
		return nil
	}
}

const testAccTeamConfig_groupAdd = `
resource "grafana_team" "testTeam" {
  name  = "terraform-test-team-external-group"
}

resource "grafana_team_external_group" "test" {
  team_id    = grafana_team.testTeam.id
  groups = [
    "test-group",
  ]
}
`
const testAccTeamConfig_groupRemove = `
resource "grafana_team" "testTeam" {
  name  = "terraform-test-team-external-group"
}

resource "grafana_team_external_group" "test" {
  team_id    = grafana_team.testTeam.id
  groups = [ ]
}
`
