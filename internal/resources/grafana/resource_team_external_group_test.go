package grafana_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/grafana/terraform-provider-grafana/internal/resources/grafana"
	"github.com/grafana/terraform-provider-grafana/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
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
		},
	})
}

func TestAccExampleThing_basic(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t)

	ctx := context.Background()
	d := &schema.ResourceData{}
	teamName := "testTeam"
	groupName := "test"
	d.Set("name", teamName)
	d.Set("email", "test-team@team.com")
	grafana.CreateTeam(ctx, d, "")
	d.Set("groups", []string{groupName})
	grafana.CreateTeamExternalGroup(ctx, d, "")

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		/* ... existing TestCase functions ... */
		Steps: []resource.TestStep{
			/* ... existing TestStep ... */
			{
				ResourceName:      fmt.Sprintf("grafana_team.%s", teamName),
				ImportState:       true,
				ImportStateId:     d.Get("team_id").(string),
				ImportStateVerify: true,
			},
			{
				ResourceName:      fmt.Sprintf("grafana_team_external_group.%s", groupName),
				ImportState:       true,
				ImportStateId:     d.Get("team_id").(string),
				ImportStateVerify: true,
			},
			{
				Config: testAccTeamConfig_groupRemove,
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

		client := testutils.Provider.Meta().(*common.Client).GrafanaAPI

		gotTeamID, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
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
