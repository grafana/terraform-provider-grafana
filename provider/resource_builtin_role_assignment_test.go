package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/provider/common"
	"github.com/grafana/terraform-provider-grafana/provider/testutils"
)

const (
	roleUID1 = "reportviewer"
	roleUID2 = "createuser"

	roleUID3 = "testroletwouid"
	roleUID4 = "testroleuid"

	roleUID5 = "viewer_test"
	roleUID6 = "viewer_test_2"
)

func TestAccBuiltInRoleAssignment(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t)

	var assignments map[string][]*gapi.Role

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccBuiltInRoleAssignmentCheckDestroy(&assignments, []string{roleUID3, roleUID4}, nil),
		Steps: []resource.TestStep{
			{
				Config: builtInRoleAssignmentConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccBuiltInRoleAssignmentCheckExists("grafana_builtin_role_assignment.test_assignment", &assignments),
					resource.TestCheckResourceAttr(
						"grafana_builtin_role_assignment.test_assignment", "builtin_role", "Editor",
					),
					resource.TestCheckResourceAttr(
						"grafana_builtin_role_assignment.test_assignment", "roles.#", "2",
					),
					resource.TestCheckResourceAttr(
						"grafana_builtin_role_assignment.test_assignment", "roles.0.uid", roleUID3,
					),
					resource.TestCheckResourceAttr(
						"grafana_builtin_role_assignment.test_assignment", "roles.0.global", "false",
					),
					resource.TestCheckResourceAttr(
						"grafana_builtin_role_assignment.test_assignment", "roles.1.uid", roleUID4,
					),
					resource.TestCheckResourceAttr(
						"grafana_builtin_role_assignment.test_assignment", "roles.1.global", "true",
					),
				),
			},
		},
	})
}

func TestAccBuiltInRoleAssignmentUpdate(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t)

	var assignments map[string][]*gapi.Role

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccBuiltInRoleAssignmentCheckDestroy(&assignments, []string{roleUID5, roleUID6}, []string{roleUID1, roleUID2}),
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					err := prepareDefaultAssignments()
					if err != nil {
						t.Errorf("error when creating built-in role ssignments %s", err)
					}
				},
				Config: builtInRoleAssignmentUpdatePreConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccBuiltInRoleAssignmentCheckExists("grafana_builtin_role_assignment.test_builtin_assignment", &assignments),
					resource.TestCheckResourceAttr(
						"grafana_builtin_role_assignment.test_builtin_assignment", "builtin_role", "Viewer",
					),
					resource.TestCheckResourceAttr(
						"grafana_builtin_role_assignment.test_builtin_assignment", "roles.#", "1",
					),
					resource.TestCheckResourceAttr(
						"grafana_builtin_role_assignment.test_builtin_assignment", "roles.0.uid", roleUID5,
					),
					resource.TestCheckResourceAttr(
						"grafana_builtin_role_assignment.test_builtin_assignment", "roles.0.global", "true",
					),
				),
			},
			{
				Config: builtInRoleAssignmentUpdateConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccBuiltInRoleAssignmentCheckExists("grafana_builtin_role_assignment.test_builtin_assignment", &assignments),
					resource.TestCheckResourceAttr(
						"grafana_builtin_role_assignment.test_builtin_assignment", "builtin_role", "Viewer",
					),
					resource.TestCheckResourceAttr(
						"grafana_builtin_role_assignment.test_builtin_assignment", "roles.#", "2",
					),
					resource.TestCheckResourceAttr(
						"grafana_builtin_role_assignment.test_builtin_assignment", "roles.0.uid", roleUID5,
					),
					resource.TestCheckResourceAttr(
						"grafana_builtin_role_assignment.test_builtin_assignment", "roles.0.global", "true",
					),
					resource.TestCheckResourceAttr(
						"grafana_builtin_role_assignment.test_builtin_assignment", "roles.1.uid", roleUID6,
					),
					resource.TestCheckResourceAttr(
						"grafana_builtin_role_assignment.test_builtin_assignment", "roles.1.global", "true",
					),
					testAccBuiltInRoleAssignmentWereNotDestroyed("Viewer", roleUID1, roleUID2),
				),
			},
		},
	})
}

func prepareDefaultAssignments() error {
	client := testAccProvider.Meta().(*common.Client).GrafanaAPI
	r1 := gapi.Role{
		UID:     roleUID1,
		Version: 1,
		Name:    "Test Report Viewer",
		Global:  true,
		Permissions: []gapi.Permission{
			{
				Action: "reports:read",
				Scope:  "reports:*",
			},
			{
				Action: "users:create",
			},
		},
	}
	r2 := gapi.Role{
		UID:     roleUID2,
		Version: 1,
		Name:    "Test Create User",
		Global:  true,
		Permissions: []gapi.Permission{
			{
				Action: "users:create",
			},
		},
	}
	role1, err := client.NewRole(r1)
	if err != nil {
		return fmt.Errorf("error creating role: %w", err)
	}
	role2, err := client.NewRole(r2)
	if err != nil {
		return fmt.Errorf("error creating role: %w", err)
	}
	_, err = client.NewBuiltInRoleAssignment(gapi.BuiltInRoleAssignment{
		BuiltinRole: "Viewer",
		RoleUID:     role1.UID,
		Global:      false,
	})
	if err != nil {
		return fmt.Errorf("error creating built-in role assigntment: %w", err)
	}
	_, err = client.NewBuiltInRoleAssignment(gapi.BuiltInRoleAssignment{
		BuiltinRole: "Viewer",
		RoleUID:     role2.UID,
		Global:      false,
	})
	if err != nil {
		return fmt.Errorf("error creating built-in role assigntment: %w", err)
	}
	return nil
}

func testAccBuiltInRoleAssignmentWereNotDestroyed(brName string, roleUIDs ...string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*common.Client).GrafanaAPI
		assignments, err := client.GetBuiltInRoleAssignments()
		if err != nil || assignments[brName] == nil {
			return fmt.Errorf("built-in assignments were destroyed, but expected to exist: %v", err)
		}
		return checkAssignmentsExists(assignments[brName], roleUIDs...)
	}
}

func testAccBuiltInRoleAssignmentCheckExists(rn string, brAssignments *map[string][]*gapi.Role) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		client := testAccProvider.Meta().(*common.Client).GrafanaAPI
		assignments, err := client.GetBuiltInRoleAssignments()
		if err != nil || assignments[rs.Primary.ID] == nil {
			return fmt.Errorf("error getting built-in role assignments: %s", err)
		}

		*brAssignments = map[string][]*gapi.Role{
			rs.Primary.ID: assignments[rs.Primary.ID],
		}
		return nil
	}
}

func testAccBuiltInRoleAssignmentCheckDestroy(brAssignments *map[string][]*gapi.Role, destroyedUIDs []string, preservedRoleUIDs []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*common.Client).GrafanaAPI
		bra, err := client.GetBuiltInRoleAssignments()
		if err != nil {
			return fmt.Errorf("error getting built-in role assignments: %s", err)
		}

		if preservedRoleUIDs != nil {
			for br := range *brAssignments {
				err := checkAssignmentsExists(bra[br], preservedRoleUIDs...)
				if err != nil {
					return fmt.Errorf("assignments were entirely destroyed, but expected to have roles with UID %v assigned to %s built-in role", preservedRoleUIDs, br)
				}
			}
		}

		if destroyedUIDs != nil {
			for br := range *brAssignments {
				err := checkAssignmentsDoNotExist(bra[br], destroyedUIDs...)
				if err != nil {
					return fmt.Errorf("assignments were supped to destroyed, but have roles with UID %v assigned to %s built-in role", destroyedUIDs, br)
				}
			}
		}

		return nil
	}
}

func checkAssignmentsDoNotExist(roles []*gapi.Role, roleUIDs ...string) error {
	for _, uid := range roleUIDs {
		if contains(roles, uid) {
			return fmt.Errorf("built-in assignments still exists for a role UID: %s", uid)
		}
	}
	return nil
}

func checkAssignmentsExists(roles []*gapi.Role, roleUIDs ...string) error {
	for _, uid := range roleUIDs {
		if !contains(roles, uid) {
			return fmt.Errorf("built-in assignments do not exist for a role UID: %s", uid)
		}
	}
	return nil
}

func contains(roles []*gapi.Role, uid string) bool {
	for _, r := range roles {
		if r.UID == uid {
			return true
		}
	}
	return false
}

const builtInRoleAssignmentConfig = `
resource "grafana_role" "test_role" {
  name  = "test_role"
  description = "test desc"
  version = 1
  uid = "testroleuid"
  global = true
  permissions {
	action = "users:read"
    scope = "global:users:*"
  }
  permissions {
	action = "users:create"
  }
}

resource "grafana_role" "test_role_two" {
  name  = "test_role_two"
  description = "test desc"
  version = 1
  uid = "testroletwouid"
  global = true
  permissions {
	action = "users:read"
    scope = "global:users:*"
  }
  permissions {
	action = "users:create"
  }
}

resource "grafana_builtin_role_assignment" "test_assignment" {
  builtin_role  = "Editor"
  roles {
	uid = grafana_role.test_role.id
	global = true
  }
  roles {
	uid = grafana_role.test_role_two.id
	global = false
  }
}
`

const builtInRoleAssignmentUpdatePreConfig = `
resource "grafana_role" "viewer_test" {
  name  = "viewer_test"
  description = "test desc"
  version = 1
  uid = "viewer_test"
  global = true 
  permissions {
	action = "users:create"
  }
}

resource "grafana_builtin_role_assignment" "test_builtin_assignment" {
  builtin_role  = "Viewer"
  roles {
	uid = grafana_role.viewer_test.id
	global = true
  } 
}
`

const builtInRoleAssignmentUpdateConfig = `
resource "grafana_role" "viewer_test" {
  name  = "viewer_test"
  description = "test desc"
  version = 1
  uid = "viewer_test"
  global = true 
  permissions {
	action = "users:create"
  }
}

resource "grafana_role" "viewer_test_2" {
  name  = "viewer_test_2"
  description = "test desc"
  version = 1
  uid = "viewer_test_2"
  global = true 
  permissions {
	action = "users:create"
  }
}

resource "grafana_builtin_role_assignment" "test_builtin_assignment" {
  builtin_role  = "Viewer"
  roles {
	uid = grafana_role.viewer_test.id
	global = true
  } 
  roles {
	uid = grafana_role.viewer_test_2.id
	global = true
  } 
}
`
