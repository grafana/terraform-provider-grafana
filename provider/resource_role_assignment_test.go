package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/provider/common"
)

func TestRoleAssignments(t *testing.T) {
	CheckEnterpriseTestsEnabled(t)
	var roleAssignment gapi.RoleAssignments

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testRoleAssignmentCheckDestroy(&roleAssignment),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(roleAssignmentConfig, roleUID),
				Check: resource.ComposeTestCheckFunc(
					testRoleAssignmentCheckExists("grafana_role_assignment.test", &roleAssignment),
					resource.TestCheckResourceAttr(
						"grafana_role_assignment.test", "role_uid", roleUID,
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
				Config:  fmt.Sprintf(roleAssignmentConfig, roleUID),
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

		client := testAccProvider.Meta().(*common.Client).GrafanaAPI
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
		client := testAccProvider.Meta().(*common.Client).GrafanaAPI
		role, err := client.GetRoleAssignments(ra.RoleUID)
		if err == nil && (len(role.Users) > 0 || len(role.ServiceAccounts) > 0 || len(role.Teams) > 0) {
			return fmt.Errorf("role is still assigned")
		}
		return nil
	}
}

var roleUID = "terraform_test_role"

var roleAssignmentConfig = `
resource "grafana_team" "test_team" {
	name = "terraform_test_team"
}

resource "grafana_user" "test_user" {
	email = "terraform_user@test.com"
	login    = "terraform_user@test.com"
	password = "12345"
}

resource "grafana_user" "test_user2" {
	email = "terraform_user2@test.com"
	login    = "terraform_user2@test.com"
	password = "12345"
}

resource "grafana_role_assignment" "test" {
  role_uid = "%s"
  users = [grafana_user.test_user.id, grafana_user.test_user2.id]
  teams = [grafana_team.test_team.id]
}
`
