package grafana

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	gapi "github.com/grafana/grafana-api-golang-client"
)

func TestRoleAssignments(t *testing.T) {
	CheckEnterpriseTestsEnabled(t)

	var roleAssignment gapi.RoleAssignments

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testRoleAssignmentCheckDestroy(&roleAssignment),
		Steps: []resource.TestStep{
			{
				Config: roleAssignmentConfig,
				Check: resource.ComposeTestCheckFunc(
					testRoleAssignmentCheckExists("grafana_role_assignment.test", &roleAssignment),
					resource.TestCheckResourceAttr(
						"grafana_role_assignment.test", "role_uid", "test_uid",
					),
					resource.TestCheckResourceAttr(
						"grafana_role_assignment.test", "users.#", "2",
					),
					resource.TestCheckResourceAttr(
						"grafana_role_assignment.test", "users.0", "1",
					),
					resource.TestCheckResourceAttr(
						"grafana_role_assignment.test", "users.1", "3",
					),
					resource.TestCheckResourceAttr(
						"grafana_role_assignment.test", "service_accounts.#", "0",
					),
					resource.TestCheckResourceAttr(
						"grafana_role_assignment.test", "teams.#", "1",
					),
					resource.TestCheckResourceAttr(
						"grafana_role_assignment.test", "teams.0", "5",
					),
				),
			},
		},
	})
}

// TODO
func testRoleAssignmentCheckExists(rUID string, ra *gapi.RoleAssignments) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rUID]
		if !ok {
			return fmt.Errorf("resource not found: %s", rUID)
		}

		uid, ok := rs.Primary.Attributes["roleUID"]
		if !ok {
			return fmt.Errorf("resource UID not set")
		}

		client := testAccProvider.Meta().(*client).gapi
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
		client := testAccProvider.Meta().(*client).gapi
		role, err := client.GetRoleAssignments(ra.RoleUID)
		if err == nil && (len(role.Users) > 0 || len(role.ServiceAccounts) > 0 || len(role.Teams) > 0) {
			return fmt.Errorf("role is still assigned")
		}
		return nil
	}
}

const roleAssignmentConfig = `
resource "grafana_role_assignment" "test" {
  role_uid = "test_uid"
  users = [1,3]
  teams = [5]
}
`
