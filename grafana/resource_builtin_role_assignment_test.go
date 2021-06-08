// +build enterprise

package grafana

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	gapi "github.com/grafana/grafana-api-golang-client"
)

func TestAccBuiltInRoleAssignment(t *testing.T) {
	var br gapi.BuiltInRoleAssignment

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccBuiltInRoleAssignmentCheckDestroy(&br),
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

func testAccBuiltInRoleAssignmentCheckExists(rn string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		client := testAccProvider.Meta().(*gapi.Client)
		assignments, err := client.GetBuiltInRoleAssignments()
		if err != nil || assignments[rs.Primary.ID] == nil {
			return fmt.Errorf("error getting built-in role assignments: %s", err)
		}

		return nil
	}
}

func testAccBuiltInRoleAssignmentCheckDestroy(br *gapi.BuiltInRoleAssignment) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*gapi.Client)
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
