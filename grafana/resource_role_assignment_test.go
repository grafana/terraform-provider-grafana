package grafana

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	gapi "github.com/grafana/grafana-api-golang-client"
)

func TestRoleAssignments(t *testing.T) {
	CheckEnterpriseTestsEnabled(t)
	defer removeResources()

	if err := prepareResources(); err != nil {
		t.Errorf("could not prepare resources for the test: %s", err)
		return
	}
	var roleAssignment gapi.RoleAssignments

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testRoleAssignmentCheckDestroy(&roleAssignment),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(roleAssignmentConfig, roleUID, user1ID, user2ID, teamID),
				Check: resource.ComposeTestCheckFunc(
					testRoleAssignmentCheckExists("grafana_role_assignment.test", &roleAssignment),
					resource.TestCheckResourceAttr(
						"grafana_role_assignment.test", "role_uid", roleUID,
					),
					resource.TestCheckResourceAttr(
						"grafana_role_assignment.test", "users.#", "2",
					),
					resource.TestCheckResourceAttr(
						"grafana_role_assignment.test", "users.0", strconv.FormatInt(user1ID, 10),
					),
					resource.TestCheckResourceAttr(
						"grafana_role_assignment.test", "users.1", strconv.FormatInt(user2ID, 10),
					),
					resource.TestCheckResourceAttr(
						"grafana_role_assignment.test", "service_accounts.#", "0",
					),
					resource.TestCheckResourceAttr(
						"grafana_role_assignment.test", "teams.#", "1",
					),
					resource.TestCheckResourceAttr(
						"grafana_role_assignment.test", "teams.0", strconv.FormatInt(teamID, 10),
					),
				),
			},
			{
				Config:  fmt.Sprintf(roleAssignmentConfig, roleUID, user1ID, user2ID, teamID),
				Destroy: true,
			},
		},
	})
}

// TODO
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

func prepareResources() error {
	client := testAccProvider.Meta().(*client).gapi
	r := gapi.Role{
		UID:  roleUID,
		Name: "terraform_test_role",
		Permissions: []gapi.Permission{
			{
				Action: "reports:read",
				Scope:  "reports:*",
			},
		},
	}
	if _, err := client.NewRole(r); err != nil {
		return fmt.Errorf("error creating role: %w", err)
	}

	var err error
	if teamID, err = client.AddTeam("terraform_test_team", "terraform_test@team"); err != nil {
		return fmt.Errorf("error creating team: %w", err)
	}

	user := gapi.User{
		Email:    "terraform_test_user@grafana.com",
		Login:    "terraform_test_user",
		Name:     "terraform_test_user",
		Password: "123456",
	}
	if user1ID, err = client.CreateUser(user); err != nil {
		return fmt.Errorf("error creating user: %w", err)
	}

	user2 := gapi.User{
		Email:    "terraform_test_user2@grafana.com",
		Login:    "terraform_test_user2",
		Name:     "terraform_test_user2",
		Password: "123456",
	}
	if user2ID, err = client.CreateUser(user2); err != nil {
		return fmt.Errorf("error creating user: %w", err)
	}

	return nil
}

func removeResources() {
	client := testAccProvider.Meta().(*client).gapi

	if err := client.DeleteRole(roleUID, false); err != nil {
		fmt.Printf("failed to remove role with UID %s\n", roleUID)
	}
	if err := client.DeleteTeam(teamID); err != nil {
		fmt.Printf("failed to remove team with ID %d\n", teamID)
	}
	if err := client.DeleteUser(user1ID); err != nil {
		fmt.Printf("failed to remove user with ID %d\n", user1ID)
	}
	if err := client.DeleteUser(user2ID); err != nil {
		fmt.Printf("failed to remove user with ID %d\n", user2ID)
	}
}

var user1ID, user2ID, teamID int64
var roleUID = "terraform_test_role"

var roleAssignmentConfig = `
resource "grafana_role_assignment" "test" {
  role_uid = "%s"
  users = [%d,%d]
  teams = [%d]
}
`
