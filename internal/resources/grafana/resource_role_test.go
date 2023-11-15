package grafana_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/internal/testutils"
)

func TestAccRole_basic(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t)

	var role models.RoleDTO

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      roleCheckExists.destroyed(&role, nil),
		Steps: []resource.TestStep{
			{
				Config: roleConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					roleCheckExists.exists("grafana_role.test", &role),
					resource.TestCheckResourceAttr("grafana_role.test", "name", "terraform-acc-test"),
					resource.TestCheckResourceAttr("grafana_role.test", "description", "test desc"),
					resource.TestCheckResourceAttr("grafana_role.test", "display_name", "testdisplay"),
					resource.TestCheckResourceAttr("grafana_role.test", "group", "testgroup"),
					resource.TestCheckResourceAttr("grafana_role.test", "version", "1"),
					resource.TestCheckResourceAttr("grafana_role.test", "uid", "testuid"),
					resource.TestCheckResourceAttr("grafana_role.test", "global", "true"),
					resource.TestCheckResourceAttr("grafana_role.test", "hidden", "true"),
				),
			},
			{
				Config: roleConfigWithPermissions,
				Check: resource.ComposeTestCheckFunc(
					roleCheckExists.exists("grafana_role.test", &role),
					resource.TestCheckResourceAttr("grafana_role.test", "name", "terraform-acc-test"),
					resource.TestCheckResourceAttr("grafana_role.test", "description", "test desc"),
					resource.TestCheckResourceAttr("grafana_role.test", "display_name", "testdisplay"),
					resource.TestCheckResourceAttr("grafana_role.test", "group", "testgroup"),
					resource.TestCheckResourceAttr("grafana_role.test", "version", "2"),
					resource.TestCheckResourceAttr("grafana_role.test", "uid", "testuid"),
					resource.TestCheckResourceAttr("grafana_role.test", "global", "true"),
					resource.TestCheckResourceAttr("grafana_role.test", "hidden", "true"),
					resource.TestCheckResourceAttr("grafana_role.test", "permissions.#", "2"),
					resource.TestCheckResourceAttr("grafana_role.test", "permissions.0.action", "users:create"),
					resource.TestCheckResourceAttr("grafana_role.test", "permissions.1.scope", "global.users:*"),
					resource.TestCheckResourceAttr("grafana_role.test", "permissions.1.action", "users:read"),
				),
			},
		},
	})
}

func TestAccRoleVersioning(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t)

	var role models.RoleDTO
	name := acctest.RandomWithPrefix("versioning-")

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      roleCheckExists.destroyed(&role, nil),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "grafana_role" "test" {
					name  = "%s"
					description = "desc 1"
					auto_increment_version = true
				}`, name),
				Check: resource.ComposeTestCheckFunc(
					roleCheckExists.exists("grafana_role.test", &role),
					resource.TestCheckResourceAttr("grafana_role.test", "name", name),
					resource.TestCheckResourceAttr("grafana_role.test", "version", "1"),
				),
			},
			{
				Config: fmt.Sprintf(`
				resource "grafana_role" "test" {
					name  = "%s"
					description = "desc 2"
					version = 5
				}`, name),
				Check: resource.ComposeTestCheckFunc(
					roleCheckExists.exists("grafana_role.test", &role),
					resource.TestCheckResourceAttr("grafana_role.test", "name", name),
					resource.TestCheckResourceAttr("grafana_role.test", "version", "5"),
				),
			},
			{
				Config: fmt.Sprintf(`
				resource "grafana_role" "test" {
					name  = "%s"
					description = "desc 3"
					auto_increment_version = true
				}`, name),
				Check: resource.ComposeTestCheckFunc(
					roleCheckExists.exists("grafana_role.test", &role),
					resource.TestCheckResourceAttr("grafana_role.test", "name", name),
					resource.TestCheckResourceAttr("grafana_role.test", "version", "6"),
				),
			},
			{
				Config: fmt.Sprintf(`
				resource "grafana_role" "test" {
					name  = "%s"
					description = "desc 4"
					auto_increment_version = true
				}`, name),
				Check: resource.ComposeTestCheckFunc(
					roleCheckExists.exists("grafana_role.test", &role),
					resource.TestCheckResourceAttr("grafana_role.test", "name", name),
					resource.TestCheckResourceAttr("grafana_role.test", "version", "7"),
				),
			},
		},
	})
}

func TestAccRole_inOrg(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t)

	var role models.RoleDTO
	var org gapi.Org
	name := acctest.RandomWithPrefix("role-")

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			roleCheckExists.destroyed(&role, &org),
			testAccOrganizationCheckDestroy(&org),
		),
		Steps: []resource.TestStep{
			{
				Config: roleInOrg(name, true),
				Check: resource.ComposeTestCheckFunc(
					roleCheckExists.exists("grafana_role.test", &role),
					testAccOrganizationCheckExists("grafana_organization.test", &org),
					checkResourceIsInOrg("grafana_role.test", "grafana_organization.test"),
					resource.TestMatchResourceAttr("grafana_role.test", "id", nonDefaultOrgIDRegexp),
					resource.TestCheckResourceAttr("grafana_role.test", "name", name),
					resource.TestCheckResourceAttr("grafana_role.test", "description", name+" desc"),
					resource.TestCheckResourceAttr("grafana_role.test", "display_name", name),
					resource.TestCheckResourceAttr("grafana_role.test", "group", name),
					resource.TestCheckResourceAttr("grafana_role.test", "version", "1"),
					resource.TestCheckResourceAttr("grafana_role.test", "uid", name),
					resource.TestCheckResourceAttr("grafana_role.test", "global", "false"),
					resource.TestCheckResourceAttr("grafana_role.test", "hidden", "false"),
				),
			},
			// Test destroying role within org
			{
				Config: roleInOrg(name, false),
				Check: resource.ComposeTestCheckFunc(
					roleCheckExists.destroyed(&role, &org),
					testAccOrganizationCheckExists("grafana_organization.test", &org),
				),
			},
		},
	})
}

func roleInOrg(name string, roleExists bool) string {
	def := fmt.Sprintf(`
resource "grafana_organization" "test" {
	name = "%s"
}`, name)

	if roleExists {
		def += fmt.Sprintf(`
resource "grafana_role" "test" {
	org_id = grafana_organization.test.id
	name  = "%[1]s"
	description = "%[1]s desc"
	uid = "%[1]s"
	global = false
	group = "%[1]s"
	display_name = "%[1]s"
	hidden = false
	auto_increment_version = true
}`, name)
	}

	return def
}

const roleConfigBasic = `
resource "grafana_role" "test" {
  name  = "terraform-acc-test"
  description = "test desc"
  version = 1
  uid = "testuid"
  global = true
  group = "testgroup"
  display_name = "testdisplay"
  hidden = true
}
`

const roleConfigWithPermissions = `
resource "grafana_role" "test" {
  name  = "terraform-acc-test"
  description = "test desc"
  version = 2
  uid = "testuid"
  global = true
  group = "testgroup"
  display_name = "testdisplay"
  hidden = true
  permissions {
	action = "users:read"
    scope = "global.users:*"
  }
  permissions {
	action = "users:create"
  }
}
`
