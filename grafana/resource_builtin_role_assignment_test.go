package grafana

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	gapi "github.com/grafana/grafana-api-golang-client"
)

func TestAccBuiltInRoleAssignment(t *testing.T) {
	CheckEnterpriseTestsEnabled(t)

	var br gapi.BuiltInRoleAssignment

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccBuiltInRoleAssignmentCheckDestroy(&br),
		Steps: []resource.TestStep{
			{
				Config: builtInRoleAssignmentConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccBuiltInRoleAssignmentCheckExists("grafana_builtin_role_assignment.test_assignment"),
					resource.TestCheckResourceAttr(
						"grafana_builtin_role_assignment.test_assignment", "builtin_role", "Viewer",
					),
					resource.TestCheckResourceAttr(
						"grafana_builtin_role_assignment.test_assignment", "roles.#", "2",
					),
					resource.TestCheckResourceAttr(
						"grafana_builtin_role_assignment.test_assignment", "roles.0.uid", "testroletwouid",
					),
					resource.TestCheckResourceAttr(
						"grafana_builtin_role_assignment.test_assignment", "roles.0.global", "false",
					),
					resource.TestCheckResourceAttr(
						"grafana_builtin_role_assignment.test_assignment", "roles.1.uid", "testroleuid",
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
	CheckEnterpriseTestsEnabled(t)

	var br gapi.BuiltInRoleAssignment

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccBuiltInRoleAssignmentCheckDestroy(&br),
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
					testAccBuiltInRoleAssignmentCheckExists("grafana_builtin_role_assignment.test_builtin_assignment"),
					resource.TestCheckResourceAttr(
						"grafana_builtin_role_assignment.test_builtin_assignment", "builtin_role", "Viewer",
					),
					resource.TestCheckResourceAttr(
						"grafana_builtin_role_assignment.test_builtin_assignment", "roles.#", "1",
					),
					resource.TestCheckResourceAttr(
						"grafana_builtin_role_assignment.test_builtin_assignment", "roles.0.uid", "viewer_test",
					),
					resource.TestCheckResourceAttr(
						"grafana_builtin_role_assignment.test_builtin_assignment", "roles.0.global", "true",
					),
				),
			},
			{
				Config: builtInRoleAssignmentUpdateConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccBuiltInRoleAssignmentCheckExists("grafana_builtin_role_assignment.test_builtin_assignment"),
					resource.TestCheckResourceAttr(
						"grafana_builtin_role_assignment.test_builtin_assignment", "builtin_role", "Viewer",
					),
					resource.TestCheckResourceAttr(
						"grafana_builtin_role_assignment.test_builtin_assignment", "roles.#", "2",
					),
					resource.TestCheckResourceAttr(
						"grafana_builtin_role_assignment.test_builtin_assignment", "roles.0.uid", "viewer_test",
					),
					resource.TestCheckResourceAttr(
						"grafana_builtin_role_assignment.test_builtin_assignment", "roles.0.global", "true",
					),
					resource.TestCheckResourceAttr(
						"grafana_builtin_role_assignment.test_builtin_assignment", "roles.1.uid", "viewer_test_2",
					),
					resource.TestCheckResourceAttr(
						"grafana_builtin_role_assignment.test_builtin_assignment", "roles.1.global", "true",
					),
					testAccBuiltInRoleAssignmentWereNotDestroyed(),
				),
			},
		},
	})
}

func prepareDefaultAssignments() error {
	client := testAccProvider.Meta().(*client).gapi
	r1 := gapi.Role{
		UID:     "reportviewer",
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
		UID:     "createuser",
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
		return fmt.Errorf("error creating role: %s", err)
	}
	role2, err := client.NewRole(r2)
	if err != nil {
		return fmt.Errorf("error creating role: %s", err)
	}
	_, err = client.NewBuiltInRoleAssignment(gapi.BuiltInRoleAssignment{
		BuiltinRole: "Viewer",
		RoleUID:     role1.UID,
		Global:      false,
	})
	if err != nil {
		return fmt.Errorf("error creating built-in role assigntment: %s", err)
	}
	_, err = client.NewBuiltInRoleAssignment(gapi.BuiltInRoleAssignment{
		BuiltinRole: "Viewer",
		RoleUID:     role2.UID,
		Global:      false,
	})
	if err != nil {
		return fmt.Errorf("error creating built-in role assigntment: %s", err)
	}
	return nil
}

func testAccBuiltInRoleAssignmentWereNotDestroyed() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*client).gapi
		assignments, err := client.GetBuiltInRoleAssignments()
		if err != nil || assignments["Viewer"] == nil {
			return fmt.Errorf("built-in assignments do not exist: %v", err)
		}
		roles := assignments["Viewer"]
		contains := func(roles []*gapi.Role, uid string) bool {
			for _, r := range roles {
				if r.UID == uid {
					return true
				}
			}
			return false
		}
		if !contains(roles, "reportviewer") || !contains(roles, "createuser") {
			return fmt.Errorf("built-in assignments do not exist for roles: %s and %s", "reportviewer", "createuser")
		}
		return nil
	}
}

func testAccBuiltInRoleAssignmentCheckExists(rn string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		client := testAccProvider.Meta().(*client).gapi
		assignments, err := client.GetBuiltInRoleAssignments()
		if err != nil || assignments[rs.Primary.ID] == nil {
			return fmt.Errorf("error getting built-in role assignments: %s", err)
		}

		return nil
	}
}

func testAccBuiltInRoleAssignmentCheckDestroy(br *gapi.BuiltInRoleAssignment) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*client).gapi
		bra, err := client.GetBuiltInRoleAssignments()
		if err == nil && bra[br.BuiltinRole] != nil {
			return fmt.Errorf("assignment still exists")
		}
		return nil
	}
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
  builtin_role  = "Viewer"
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
