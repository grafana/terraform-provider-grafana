package provider

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/provider/common"
	"github.com/grafana/terraform-provider-grafana/provider/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccTeam_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	var team gapi.Team
	teamName := acctest.RandString(5)
	teamNameUpdated := acctest.RandString(5)

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.GetProviderFactories(),
		CheckDestroy:      testAccTeamCheckDestroy(&team),
		Steps: []resource.TestStep{
			{
				Config: testAccTeamDefinition(teamName, nil),
				Check: resource.ComposeTestCheckFunc(
					testAccTeamCheckExists("grafana_team.test", &team),
					resource.TestCheckResourceAttr("grafana_team.test", "name", teamName),
					resource.TestCheckResourceAttr("grafana_team.test", "email", teamName+"@example.com"),
					resource.TestMatchResourceAttr("grafana_team.test", "id", idRegexp),
				),
			},
			{
				Config: testAccTeamDefinition(teamNameUpdated, nil),
				Check: resource.ComposeTestCheckFunc(
					testAccTeamCheckExists("grafana_team.test", &team),
					resource.TestCheckResourceAttr("grafana_team.test", "name", teamNameUpdated),
					resource.TestCheckResourceAttr("grafana_team.test", "email", teamNameUpdated+"@example.com"),
					resource.TestMatchResourceAttr("grafana_team.test", "id", idRegexp),
				),
			},
			{
				ResourceName:            "grafana_team.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"ignore_externally_synced_members"},
			},
		},
	})
}

func TestAccTeam_Members(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	var team gapi.Team
	teamName := acctest.RandString(5)

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.GetProviderFactories(),
		CheckDestroy:      testAccTeamCheckDestroy(&team),
		Steps: []resource.TestStep{
			{
				Config: testAccTeamDefinition(teamName, []string{
					"grafana_user.users.0.email",
					"grafana_user.users.1.email",
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccTeamCheckExists("grafana_team.test", &team),
					resource.TestCheckResourceAttr("grafana_team.test", "name", teamName),
					resource.TestCheckResourceAttr("grafana_team.test", "members.#", "2"),
					resource.TestCheckResourceAttr("grafana_team.test", "members.0", teamName+"-user-0@example.com"),
					resource.TestCheckResourceAttr("grafana_team.test", "members.1", teamName+"-user-1@example.com"),
				),
			},
			// Reorder members but only plan changes. There should be no changes.
			{
				Config: testAccTeamDefinition(teamName, []string{
					"grafana_user.users.1.email",
					"grafana_user.users.0.email",
				}),
				PlanOnly: true,
			},
			// When adding a new member, the state should be updated and re-sorted.
			{
				Config: testAccTeamDefinition(teamName, []string{
					"grafana_user.users.1.email",
					"grafana_user.users.0.email",
					"grafana_user.users.2.email",
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccTeamCheckExists("grafana_team.test", &team),
					resource.TestCheckResourceAttr("grafana_team.test", "name", teamName),
					resource.TestCheckResourceAttr("grafana_team.test", "members.#", "3"),
					resource.TestCheckResourceAttr("grafana_team.test", "members.0", teamName+"-user-0@example.com"),
					resource.TestCheckResourceAttr("grafana_team.test", "members.1", teamName+"-user-1@example.com"),
					resource.TestCheckResourceAttr("grafana_team.test", "members.2", teamName+"-user-2@example.com"),
				),
			},
			// Test the import with members
			{
				ResourceName:            "grafana_team.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"ignore_externally_synced_members"},
			},
			{
				Config: testAccTeamDefinition(teamName, nil),
				Check: resource.ComposeTestCheckFunc(
					testAccTeamCheckExists("grafana_team.test", &team),
					resource.TestCheckResourceAttr("grafana_team.test", "name", teamName),
					resource.TestCheckResourceAttr("grafana_team.test", "members.#", "0"),
				),
			},
		},
	})
}

// Test that deleted users can still be removed as members of a team
func TestAccTeam_RemoveUnexistingMember(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)
	client := testutils.GetProvider().Meta().(*common.Client).GrafanaAPI

	var team gapi.Team
	var userID int64 = -1
	teamName := acctest.RandString(5)

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.GetProviderFactories(),
		CheckDestroy:      testAccTeamCheckDestroy(&team),
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					// Create user
					user := gapi.User{
						Email:    "user1@grafana.com",
						Login:    "user1@grafana.com",
						Name:     "user1",
						Password: "123456",
					}
					var err error
					userID, err = client.CreateUser(user)
					if err != nil {
						t.Fatal(err)
					}
				},
				Config: testAccTeamDefinition(teamName, []string{`"user1@grafana.com"`}),
				Check: resource.ComposeTestCheckFunc(
					testAccTeamCheckExists("grafana_team.test", &team),
					resource.TestCheckResourceAttr("grafana_team.test", "members.#", "1"),
					resource.TestCheckResourceAttr("grafana_team.test", "members.0", "user1@grafana.com"),
				),
			},
			{
				PreConfig: func() {
					// Delete the user
					if err := client.DeleteUser(userID); err != nil {
						t.Fatal(err)
					}
				},
				Config: testAccTeamDefinition(teamName, nil),
				Check: resource.ComposeTestCheckFunc(
					testAccTeamCheckExists("grafana_team.test", &team),
					resource.TestCheckResourceAttr("grafana_team.test", "members.#", "0"),
				),
			},
		},
	})
}

func testAccTeamCheckExists(rn string, a *gapi.Team) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}
		id, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
		if err != nil {
			return fmt.Errorf("resource id is malformed")
		}

		client := testutils.GetProvider().Meta().(*common.Client).GrafanaAPI
		team, err := client.Team(id)
		if err != nil {
			return fmt.Errorf("error getting data source: %s", err)
		}

		*a = *team

		return nil
	}
}

func testAccTeamCheckDestroy(a *gapi.Team) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testutils.GetProvider().Meta().(*common.Client).GrafanaAPI
		team, err := client.Team(a.ID)
		if err == nil && team.Name != "" {
			return fmt.Errorf("team still exists")
		}
		return nil
	}
}

func testAccTeamDefinition(name string, teamMembers []string) string {
	definition := fmt.Sprintf(`
resource "grafana_team" "test" {
	name    = "%[1]s"
	email   = "%[1]s@example.com"
	members = [ %[2]s ]
}
`, name, strings.Join(teamMembers, `, `))

	// If we're referencing a grafana_user resource, we need to create those users
	if len(teamMembers) > 0 && strings.Contains(teamMembers[0], "grafana_user") {
		definition += fmt.Sprintf(`
resource "grafana_user" "users" {
	count = 3

	email    = "%[1]s-user-${count.index}@example.com"
	name     = "%[1]s-user-${count.index}"
	login    = "%[1]s-user-${count.index}"
	password = "my-password"
	is_admin = false
}
`, name)
	}

	return definition
}
