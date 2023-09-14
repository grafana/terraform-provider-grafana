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

func TestRoleAssignments(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t)
	var roleAssignment gapi.RoleAssignments

	testName := acctest.RandomWithPrefix("role-assignment")

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
						"grafana_role_assignment.test", "service_accounts.#", "0",
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

resource "grafana_role_assignment" "test" {
  role_uid = grafana_role.test.uid
  users = [grafana_user.test_user.id, grafana_user.test_user2.id]
  teams = [grafana_team.test_team.id]
}
`, name)
}
