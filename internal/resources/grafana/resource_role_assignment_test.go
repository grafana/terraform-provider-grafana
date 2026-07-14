package grafana_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
)

func TestAccRoleAssignments(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=9.0.0")

	testName := acctest.RandString(10)
	var role models.RoleDTO

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             roleAssignmentCheckExists.destroyed(&role, nil),
		Steps: []resource.TestStep{
			{
				Config: roleAssignmentConfig(testName),
				Check: resource.ComposeTestCheckFunc(
					roleAssignmentCheckExists.exists("grafana_role_assignment.test", &role),
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
		},
	})
}

func TestAccRoleAssignments_inOrg(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=9.0.0")

	testName := acctest.RandString(10)
	var org models.OrgDetailsDTO
	var role models.RoleDTO

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             orgCheckExists.destroyed(&org, nil),
		Steps: []resource.TestStep{
			{
				Config: roleAssignmentConfigInOrg(testName),
				Check: resource.ComposeTestCheckFunc(
					roleAssignmentCheckExists.exists("grafana_role_assignment.test", &role),
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
					orgCheckExists.exists("grafana_organization.test", &org),
					checkResourceIsInOrg("grafana_role.test", "grafana_organization.test"),
					checkResourceIsInOrg("grafana_role_assignment.test", "grafana_organization.test"),
				),
			},
			// Check destroy
			{
				Config: testutils.WithoutResource(t, roleAssignmentConfigInOrg(testName), "grafana_role_assignment.test"),
				Check: resource.ComposeTestCheckFunc(
					orgCheckExists.exists("grafana_organization.test", &org),
					roleAssignmentCheckExists.destroyed(&role, nil),
				),
			},
		},
	})
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
	global = false
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
