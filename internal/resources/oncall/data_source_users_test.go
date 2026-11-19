package oncall_test

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccDataSourceUsers_Basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "grafana_oncall_users" "test-acc-users" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.grafana_oncall_users.test-acc-users", "id", "oncall_users"),
					resource.TestCheckResourceAttrSet("data.grafana_oncall_users.test-acc-users", "users.0.id"),
					resource.TestCheckResourceAttrSet("data.grafana_oncall_users.test-acc-users", "users.0.username"),
					// teams is set on every user, and is empty rather than null for a user
					// on no team. A null list is left out of state entirely, so a missing
					// count here means the nil normalisation regressed.
					checkEveryUserHasTeams("data.grafana_oncall_users.test-acc-users"),
				),
			},
		},
	})
}

// TestAccDataSourceUsers_TeamFilter checks what the filter does without depending
// on team membership. OnCall syncs membership from Grafana in the background, so a
// member added by a test is not reliably visible on the next read; that the right
// people come back for a team is verified manually against a stack instead.
func TestAccDataSourceUsers_TeamFilter(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	teamName := fmt.Sprintf("test-acc-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDataSourceUsersTeamFilterConfig(teamName, `team_id = ""`),
				ExpectError: regexp.MustCompile(`Attribute team_id must be a team ID`),
			},
			{
				Config:      testAccDataSourceUsersTeamFilterConfig(teamName, `team_id = " "`),
				ExpectError: regexp.MustCompile(`Attribute team_id must be a team ID`),
			},
			{
				Config:      testAccDataSourceUsersTeamFilterConfig(teamName, `team_id = "null"`),
				ExpectError: regexp.MustCompile(`team_id does not accept`),
			},
			{
				Config: testAccDataSourceUsersTeamFilterConfig(teamName, `team_id = data.grafana_oncall_team.test-acc-team.id`),
				Check: resource.ComposeAggregateTestCheckFunc(
					// The filter is the team the data source resolved by name.
					resource.TestCheckResourceAttrPair(
						"data.grafana_oncall_users.test-acc-team-users", "team_id",
						"data.grafana_oncall_team.test-acc-team", "id",
					),
					// A filtered read gets its own ID, so two instances can coexist.
					resource.TestMatchResourceAttr("data.grafana_oncall_users.test-acc-team-users", "id", regexp.MustCompile(`^oncall_users:.+`)),
					checkFilteredUsersID(
						"data.grafana_oncall_users.test-acc-team-users",
						"data.grafana_oncall_team.test-acc-team",
					),
					// A team ID that does not exist returns no users instead of an error.
					resource.TestCheckResourceAttr("data.grafana_oncall_users.test-acc-unknown-team", "users.#", "0"),
				),
			},
		},
	})
}

func testAccDataSourceUsersTeamFilterConfig(teamName, teamIDAttr string) string {
	return fmt.Sprintf(`
resource "grafana_team" "test-acc-team" {
	name = "%[1]s"
}

data "grafana_oncall_team" "test-acc-team" {
	depends_on = [grafana_team.test-acc-team]
	name = "%[1]s"
}

data "grafana_oncall_users" "test-acc-team-users" {
	%[2]s
}

data "grafana_oncall_users" "test-acc-unknown-team" {
	team_id = "TDOESNOTEXIST"
}
`, teamName, teamIDAttr)
}

// checkFilteredUsersID asserts the data source ID is derived from the filter, so
// that two instances filtered differently do not collide.
func checkFilteredUsersID(usersAddr, teamAddr string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		users, ok := s.RootModule().Resources[usersAddr]
		if !ok {
			return fmt.Errorf("%s not found in state", usersAddr)
		}
		team, ok := s.RootModule().Resources[teamAddr]
		if !ok {
			return fmt.Errorf("%s not found in state", teamAddr)
		}

		want := "oncall_users:" + team.Primary.ID
		if got := users.Primary.ID; got != want {
			return fmt.Errorf("ID is %q, want %q", got, want)
		}
		return nil
	}
}

// checkEveryUserHasTeams asserts teams is an empty list rather than null for
// every user, not only the first. A null list is absent from state entirely.
func checkEveryUserHasTeams(addr string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[addr]
		if !ok {
			return fmt.Errorf("%s not found in state", addr)
		}

		count, err := strconv.Atoi(rs.Primary.Attributes["users.#"])
		if err != nil {
			return fmt.Errorf("users.# is %q: %w", rs.Primary.Attributes["users.#"], err)
		}
		if count == 0 {
			return fmt.Errorf("%s returned no users, so teams was never checked", addr)
		}

		for i := range count {
			key := fmt.Sprintf("users.%d.teams.#", i)
			if _, ok := rs.Primary.Attributes[key]; !ok {
				return fmt.Errorf("%s is missing, so teams is null for user %d", key, i)
			}
		}
		return nil
	}
}
