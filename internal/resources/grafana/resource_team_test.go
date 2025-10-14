package grafana_test

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccTeam_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	var team models.TeamDTO
	teamName := acctest.RandString(5)
	teamNameUpdated := acctest.RandString(5)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             teamCheckExists.destroyed(&team, nil),
		Steps: []resource.TestStep{
			{
				Config: testAccTeamDefinition(teamName, nil, false, nil),
				Check: resource.ComposeTestCheckFunc(
					teamCheckExists.exists("grafana_team.test", &team),
					resource.TestCheckResourceAttr("grafana_team.test", "name", teamName),
					resource.TestCheckResourceAttr("grafana_team.test", "email", teamName+"@example.com"),
					resource.TestMatchResourceAttr("grafana_team.test", "id", defaultOrgIDRegexp),
					resource.TestCheckResourceAttr("grafana_team.test", "org_id", "1"),
					testutils.CheckLister("grafana_team.test"),
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

	var team models.TeamDTO
	teamName := acctest.RandString(5)
	teamNameUpdated := acctest.RandString(5)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             teamCheckExists.destroyed(&team, nil),
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
					resource.TestCheckNoResourceAttr("grafana_team.test", "preferences.0.week_start"),
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
					resource.TestCheckResourceAttr("grafana_team.test", "preferences.0.week_start", "monday"),
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
	testutils.CheckEnterpriseTestsEnabled(t, ">=9.0.0")

	var team models.TeamDTO
	teamName := acctest.RandString(5)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             teamCheckExists.destroyed(&team, nil),
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

	var team models.TeamDTO
	teamName := acctest.RandString(5)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             teamCheckExists.destroyed(&team, nil),
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
	client := grafanaTestClient()

	var team models.TeamDTO
	var userID int64 = -1
	teamName := acctest.RandString(5)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             teamCheckExists.destroyed(&team, nil),
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					// Create user
					user := models.AdminCreateUserForm{
						Email:    "user1@grafana.com",
						Login:    "user1@grafana.com",
						Name:     "user1",
						Password: "123456",
					}
					resp, err := client.AdminUsers.AdminCreateUser(&user)
					if err != nil {
						t.Fatal(err)
					}
					userID = resp.Payload.ID
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
					if _, err := client.AdminUsers.AdminDeleteUser(userID); err != nil {
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

	var team models.TeamDTO
	var org models.OrgDetailsDTO
	name := acctest.RandStringFromCharSet(10, acctest.CharSetAlpha)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             orgCheckExists.destroyed(&org, nil),
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
			// Test destroying team within org. Org keeps existing but team is gone.
			{
				Config: testutils.WithoutResource(t, testAccTeamInOrganization(name), "grafana_team.test"),
				Check: resource.ComposeTestCheckFunc(
					teamCheckExists.destroyed(&team, &org),
					orgCheckExists.exists("grafana_organization.test", &org),
				),
			},
		},
	})
}

// This tests that API keys/service account tokens cannot be used at the same time as org_id
// because API keys are already org-scoped.
func TestAccTeam_OrgScopedOnAPIKey(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.1.0")
	orgID := orgScopedTest(t)

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`resource "grafana_team" "test" {
					org_id = %d
					name = "test"
				}`, orgID),
				ExpectError: regexp.MustCompile("org_id is only supported with basic auth. API keys are already org-scoped"),
			},
			{
				Config: `resource "grafana_team" "test" {
					name = "test"
				}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_team.test", "name", "test"),
					resource.TestCheckResourceAttr("grafana_team.test", "org_id", strconv.FormatInt(orgID, 10)),
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
		week_start         = "monday"
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
