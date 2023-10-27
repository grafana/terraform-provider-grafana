package grafana_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/grafana/terraform-provider-grafana/internal/testutils"
)

func TestAccRoleAssignments(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t)
	var roleAssignment gapi.RoleAssignments

	testName := acctest.RandString(10)

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      testRoleAssignmentCheckDestroy(&roleAssignment),
		Steps: []resource.TestStep{
			{
				Config: roleAssignmentConfig(testName),
				Check: resource.ComposeTestCheckFunc(
					testRoleAssignmentCheckExists("grafana_role_assignment.test", &roleAssignment),
					resource.TestCheckResourceAttr(
						"grafana_role_assignment.test", "role_uid", testName,
					),
					resource.TestCheckResourceAttr(
						"grafana_role_assignment.test", "users.#", "2",
					),
					resource.TestCheckResourceAttr(
						"grafana_role_assignment.test", "service_accounts.#", "1",
					),
					resource.TestCheckResourceAttr(
						"grafana_role_assignment.test", "teams.#", "1",
					),
				),
			},
			{
				Config:  roleAssignmentConfig(testName),
				Destroy: true,
			},
		},
	})
}

func TestAccRoleAssignments_inOrg(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t)
	var roleAssignment gapi.RoleAssignments
	var org gapi.Org

	testName := acctest.RandString(10)

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      testRoleAssignmentCheckDestroy(&roleAssignment),
		Steps: []resource.TestStep{
			{
				Config: roleAssignmentConfigInOrg(testName),
				Check: resource.ComposeTestCheckFunc(
					testRoleAssignmentCheckExists("grafana_role_assignment.test", &roleAssignment),
					resource.TestCheckResourceAttr(
						"grafana_role_assignment.test", "role_uid", testName,
					),
					resource.TestCheckResourceAttr(
						"grafana_role_assignment.test", "users.#", "2",
					),
					resource.TestCheckResourceAttr(
						"grafana_role_assignment.test", "service_accounts.#", "1",
					),
					resource.TestCheckResourceAttr(
						"grafana_role_assignment.test", "teams.#", "1",
					),

					// Check that the role is in the correct organization
					resource.TestMatchResourceAttr("grafana_role.test", "id", nonDefaultOrgIDRegexp),
					resource.TestMatchResourceAttr("grafana_role_assignment.test", "id", nonDefaultOrgIDRegexp),
					testAccOrganizationCheckExists("grafana_organization.test", &org),
					checkResourceIsInOrg("grafana_role.test", "grafana_organization.test"),
					checkResourceIsInOrg("grafana_role_assignment.test", "grafana_organization.test"),
				),
			},
			{
				Config:  roleAssignmentConfig(testName),
				Destroy: true,
			},
		},
	})
}

func testRoleAssignmentCheckExists(rn string, ra *gapi.RoleAssignments) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		uid, ok := rs.Primary.Attributes["role_uid"]
		if !ok {
			return fmt.Errorf("resource UID not set")
		}

		client := testutils.Provider.Meta().(*common.Client).GrafanaAPI
		role, err := client.GetRoleAssignments(uid)
		if err != nil {
			return fmt.Errorf("error getting role assignments: %s", err)
		}

		*ra = *role

		return nil
	}
}

func testRoleAssignmentCheckDestroy(ra *gapi.RoleAssignments) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testutils.Provider.Meta().(*common.Client).GrafanaAPI
		role, err := client.GetRoleAssignments(ra.RoleUID)
		if err == nil && (len(role.Users) > 0 || len(role.ServiceAccounts) > 0 || len(role.Teams) > 0) {
			return fmt.Errorf("role is still assigned")
		}
		return nil
	}
}

func roleAssignmentConfig(name string) string {
	return fmt.Sprintf(`
resource "grafana_role" "test" {
	name  = "%[1]s"
	description = "test desc"
	version = 1
	uid = "%[1]s"
	global = true
	group = "testgroup"
	display_name = "testdisplay"
	hidden = true
  }

resource "grafana_team" "test_team" {
	name = "%[1]s"
}

resource "grafana_user" "test_user" {
	email = "%[1]s-1@test.com"
	login    = "%[1]s-1@test.com"
	password = "12345"
}

resource "grafana_user" "test_user2" {
	email = "%[1]s-2@test.com"
	login    = "%[1]s-2@test.com"
	password = "12345"
}

resource "grafana_service_account" "test" {
	name        = "%[1]s-terraform-test"
	role        = "Editor"
	is_disabled = false
  }

resource "grafana_role_assignment" "test" {
  role_uid = grafana_role.test.uid
  users = [grafana_user.test_user.id, grafana_user.test_user2.id]
  teams = [grafana_team.test_team.id]
  service_accounts = [grafana_service_account.test.id]
}
`, name)
}

func roleAssignmentConfigInOrg(name string) string {
	return fmt.Sprintf(`
resource "grafana_organization" "test" {
	name = "%[1]s"
}

resource "grafana_role" "test" {
	org_id = grafana_organization.test.id

	name  = "%[1]s"
	description = "test desc"
	version = 1
	uid = "%[1]s"
	global = true
	group = "testgroup"
	display_name = "testdisplay"
	hidden = true
  }

resource "grafana_team" "test_team" {
	org_id = grafana_organization.test.id
	name = "%[1]s"
}

resource "grafana_user" "test_user" {
	email = "%[1]s-1@test.com"
	login    = "%[1]s-1@test.com"
	password = "12345"
}

resource "grafana_user" "test_user2" {
	email = "%[1]s-2@test.com"
	login    = "%[1]s-2@test.com"
	password = "12345"
}

resource "grafana_service_account" "test" {
	org_id = grafana_organization.test.id

	name        = "%[1]s-terraform-test"
	role        = "Editor"
	is_disabled = false
  }

resource "grafana_role_assignment" "test" {
	org_id = grafana_organization.test.id

  	role_uid = grafana_role.test.uid
  	users = [grafana_user.test_user.id, grafana_user.test_user2.id]
  	teams = [grafana_team.test_team.id]
  	service_accounts = [grafana_service_account.test.id]
}
`, name)
}
