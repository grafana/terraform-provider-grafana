package grafana_test

import (
	"fmt"
	"strings"
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/grafana-openapi-client-go/models"
	goapi "github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/grafana/terraform-provider-grafana/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccTeam_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	var team goapi.TeamDTO
	teamName := acctest.RandString(5)
	teamNameUpdated := acctest.RandString(5)

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      teamCheckExists.destroyed(&team, nil),
		Steps: []resource.TestStep{
			{
				Config: testAccTeamDefinition(teamName, nil, false, nil),
				Check: resource.ComposeTestCheckFunc(
					teamCheckExists.exists("grafana_team.test", &team),
					resource.TestCheckResourceAttr("grafana_team.test", "name", teamName),
					resource.TestCheckResourceAttr("grafana_team.test", "email", teamName+"@example.com"),
					resource.TestMatchResourceAttr("grafana_team.test", "id", defaultOrgIDRegexp),
					resource.TestCheckResourceAttr("grafana_team.test", "org_id", "1"),
				),
			},
			{
				Config: testAccTeamDefinition(teamNameUpdated, nil, false, nil),
				Check: resource.ComposeTestCheckFunc(
					teamCheckExists.exists("grafana_team.test", &team),
					resource.TestCheckResourceAttr("grafana_team.test", "name", teamNameUpdated),
					resource.TestCheckResourceAttr("grafana_team.test", "email", teamNameUpdated+"@example.com"),
					resource.TestMatchResourceAttr("grafana_team.test", "id", defaultOrgIDRegexp),
					resource.TestCheckResourceAttr("grafana_team.test", "org_id", "1"),
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

func TestAccTeam_preferences(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">= 9.0.0") // Dashboard UID is only available in Grafana 9.0.0+

	var team goapi.TeamDTO
	teamName := acctest.RandString(5)
	teamNameUpdated := acctest.RandString(5)

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      teamCheckExists.destroyed(&team, nil),
		Steps: []resource.TestStep{
			{
				Config: testAccTeamDefinition(teamName, nil, false, nil),
				Check: resource.ComposeTestCheckFunc(
					teamCheckExists.exists("grafana_team.test", &team),
					resource.TestCheckResourceAttr("grafana_team.test", "name", teamName),
					resource.TestCheckResourceAttr("grafana_team.test", "email", teamName+"@example.com"),
					resource.TestMatchResourceAttr("grafana_team.test", "id", defaultOrgIDRegexp),
					resource.TestCheckResourceAttr("grafana_team.test", "org_id", "1"),
					resource.TestCheckNoResourceAttr("grafana_team.test", "preferences.0.home_dashboard_uid"),
					resource.TestCheckNoResourceAttr("grafana_team.test", "preferences.0.theme"),
					resource.TestCheckNoResourceAttr("grafana_team.test", "preferences.0.timezone"),
				),
			},
			{
				Config: testAccTeamDefinition(teamNameUpdated, nil, true, nil),
				Check: resource.ComposeTestCheckFunc(
					teamCheckExists.exists("grafana_team.test", &team),
					resource.TestCheckResourceAttr("grafana_team.test", "name", teamNameUpdated),
					resource.TestCheckResourceAttr("grafana_team.test", "email", teamNameUpdated+"@example.com"),
					resource.TestMatchResourceAttr("grafana_team.test", "id", defaultOrgIDRegexp),
					resource.TestCheckResourceAttr("grafana_team.test", "org_id", "1"),
					resource.TestMatchResourceAttr("grafana_team.test", "preferences.0.home_dashboard_uid", common.UIDRegexp),
					resource.TestCheckResourceAttr("grafana_team.test", "preferences.0.theme", "dark"),
					resource.TestCheckResourceAttr("grafana_team.test", "preferences.0.timezone", "utc"),
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

func TestAccTeam_teamSync(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">= 8.0.0")

	var team goapi.TeamDTO
	teamName := acctest.RandString(5)

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      teamCheckExists.destroyed(&team, nil),
		Steps: []resource.TestStep{
			// Test without team sync
			{
				Config: testAccTeamDefinition(teamName, nil, false, nil),
				Check: resource.ComposeTestCheckFunc(
					teamCheckExists.exists("grafana_team.test", &team),
					resource.TestCheckResourceAttr("grafana_team.test", "name", teamName),
					resource.TestCheckResourceAttr("grafana_team.test", "email", teamName+"@example.com"),
					resource.TestMatchResourceAttr("grafana_team.test", "id", defaultOrgIDRegexp),
					resource.TestCheckResourceAttr("grafana_team.test", "team_sync.0.groups.#", "0"),
				),
			},
			// Add some groups
			{
				Config: testAccTeamDefinition(teamName, nil, false, []string{"group1", "group2"}),
				Check: resource.ComposeTestCheckFunc(
					teamCheckExists.exists("grafana_team.test", &team),
					resource.TestCheckResourceAttr("grafana_team.test", "name", teamName),
					resource.TestCheckResourceAttr("grafana_team.test", "email", teamName+"@example.com"),
					resource.TestMatchResourceAttr("grafana_team.test", "id", defaultOrgIDRegexp),
					resource.TestCheckResourceAttr("grafana_team.test", "team_sync.0.groups.#", "2"),
					resource.TestCheckResourceAttr("grafana_team.test", "team_sync.0.groups.0", "group1"),
					resource.TestCheckResourceAttr("grafana_team.test", "team_sync.0.groups.1", "group2"),
				),
			},
			// Add some, remove some
			{
				Config: testAccTeamDefinition(teamName, nil, false, []string{"group3", "group2"}),
				Check: resource.ComposeTestCheckFunc(
					teamCheckExists.exists("grafana_team.test", &team),
					resource.TestCheckResourceAttr("grafana_team.test", "name", teamName),
					resource.TestCheckResourceAttr("grafana_team.test", "email", teamName+"@example.com"),
					resource.TestMatchResourceAttr("grafana_team.test", "id", defaultOrgIDRegexp),
					resource.TestCheckResourceAttr("grafana_team.test", "team_sync.0.groups.#", "2"),
					resource.TestCheckResourceAttr("grafana_team.test", "team_sync.0.groups.0", "group2"),
					resource.TestCheckResourceAttr("grafana_team.test", "team_sync.0.groups.1", "group3"),
				),
			},
			// Remove all groups
			{
				Config: testAccTeamDefinition(teamName, nil, false, []string{}),
				Check: resource.ComposeTestCheckFunc(
					teamCheckExists.exists("grafana_team.test", &team),
					resource.TestCheckResourceAttr("grafana_team.test", "name", teamName),
					resource.TestCheckResourceAttr("grafana_team.test", "email", teamName+"@example.com"),
					resource.TestMatchResourceAttr("grafana_team.test", "id", defaultOrgIDRegexp),
					resource.TestCheckResourceAttr("grafana_team.test", "team_sync.0.groups.#", "0"),
				),
			},
		},
	})
}

func TestAccTeam_Members(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	var team goapi.TeamDTO
	teamName := acctest.RandString(5)

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      teamCheckExists.destroyed(&team, nil),
		Steps: []resource.TestStep{
			{
				Config: testAccTeamDefinition(teamName, []string{
					"grafana_user.users.0.email",
					"grafana_user.users.1.email",
				}, false, nil),
				Check: resource.ComposeTestCheckFunc(
					teamCheckExists.exists("grafana_team.test", &team),
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
				}, false, nil),
				PlanOnly: true,
			},
			// When adding a new member, the state should be updated and re-sorted.
			{
				Config: testAccTeamDefinition(teamName, []string{
					"grafana_user.users.1.email",
					"grafana_user.users.0.email",
					"grafana_user.users.2.email",
				}, false, nil),
				Check: resource.ComposeTestCheckFunc(
					teamCheckExists.exists("grafana_team.test", &team),
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
				Config: testAccTeamDefinition(teamName, nil, false, nil),
				Check: resource.ComposeTestCheckFunc(
					teamCheckExists.exists("grafana_team.test", &team),
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
	client := testutils.Provider.Meta().(*common.Client).GrafanaAPI

	var team goapi.TeamDTO
	var userID int64 = -1
	teamName := acctest.RandString(5)

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      teamCheckExists.destroyed(&team, nil),
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
				Config: testAccTeamDefinition(teamName, []string{`"user1@grafana.com"`}, false, nil),
				Check: resource.ComposeTestCheckFunc(
					teamCheckExists.exists("grafana_team.test", &team),
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
				Config: testAccTeamDefinition(teamName, nil, false, nil),
				Check: resource.ComposeTestCheckFunc(
					teamCheckExists.exists("grafana_team.test", &team),
					resource.TestCheckResourceAttr("grafana_team.test", "members.#", "0"),
				),
			},
		},
	})
}

func TestAccResourceTeam_InOrg(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	var team goapi.TeamDTO
	var org models.OrgDetailsDTO
	name := acctest.RandStringFromCharSet(10, acctest.CharSetAlpha)

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      teamCheckExists.destroyed(&team, &org),
		Steps: []resource.TestStep{
			{
				Config: testAccTeamInOrganization(name),
				Check: resource.ComposeTestCheckFunc(
					teamCheckExists.exists("grafana_team.test", &team),

					// Check that the team is in the correct organization
					resource.TestMatchResourceAttr("grafana_team.test", "id", nonDefaultOrgIDRegexp),
					orgCheckExists.exists("grafana_organization.test", &org),
					checkResourceIsInOrg("grafana_team.test", "grafana_organization.test"),
				),
			},
		},
	})
}

func testAccTeamDefinition(name string, teamMembers []string, withPreferences bool, externalGroups []string) string {
	withPreferencesBlock := ""
	if withPreferences {
		withPreferencesBlock = `
	preferences {
		theme              = "dark"
		timezone           = "utc"
		home_dashboard_uid = grafana_dashboard.test.uid
	}
`
	}

	teamSyncBlock := ""
	if externalGroups != nil {
		groups := ""
		if len(externalGroups) > 0 {
			groups = fmt.Sprintf(`"%s"`, strings.Join(externalGroups, `", "`))
		}
		teamSyncBlock = fmt.Sprintf(`
	team_sync {
		groups = [ %s ]
	}
`, groups)
	}

	definition := fmt.Sprintf(`
resource "grafana_dashboard" "test" {
	config_json = jsonencode({
		title = "dashboard-%[1]s"
	})
}

resource "grafana_team" "test" {
	name    = "%[1]s"
	email   = "%[1]s@example.com"
	members = [ %[2]s ]

	%[3]s // withPreferencesBlock
	%[4]s // teamSyncBlock
}
`, name, strings.Join(teamMembers, `, `), withPreferencesBlock, teamSyncBlock)

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

func testAccTeamInOrganization(orgName string) string {
	return fmt.Sprintf(`
resource "grafana_organization" "test" {
	name = "%[1]s"
}

resource "grafana_team" "test" {
	org_id  = grafana_organization.test.id
	name    = "%[1]s"
	email   = "%[1]s@example.com"
	members = [ ]
}`, orgName)
}
