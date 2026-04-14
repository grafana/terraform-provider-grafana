package grafana_test

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
)

// checkRoleVersion returns a TestCheckFunc that asserts the role version attribute.
// On Grafana <13, we no longer send version in API requests, so the server starts
// at 0 on create and increments from there (one behind Grafana 13+ values).
// On Grafana >=13, the server auto-manages version starting at 1.
func checkRoleVersion(resourceName, expected string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		versionStr := os.Getenv("GRAFANA_VERSION")
		if versionStr != "" && versionStr != "main" {
			v, err := semver.NewVersion(versionStr)
			if err == nil {
				c, _ := semver.NewConstraint("<13.0.0")
				if c.Check(v) {
					// On old Grafana, version starts at 0 instead of 1, so
					// every expected value is shifted down by 1.
					n, _ := strconv.Atoi(expected)
					expected = strconv.Itoa(n - 1)
				}
			}
		}
		return resource.TestCheckResourceAttr(resourceName, "version", expected)(s)
	}
}

func TestAccRole_basic(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=9.0.0")

	var role models.RoleDTO

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             roleCheckExists.destroyed(&role, nil),
		Steps: []resource.TestStep{
			{
				Config: roleConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					roleCheckExists.exists("grafana_role.test", &role),
					resource.TestCheckResourceAttr("grafana_role.test", "name", "terraform-acc-test"),
					resource.TestCheckResourceAttr("grafana_role.test", "description", "test desc"),
					resource.TestCheckResourceAttr("grafana_role.test", "display_name", "testdisplay"),
					resource.TestCheckResourceAttr("grafana_role.test", "group", "testgroup"),
					checkRoleVersion("grafana_role.test", "1"),
					resource.TestCheckResourceAttr("grafana_role.test", "uid", "terraform-acc-test"),
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
					checkRoleVersion("grafana_role.test", "2"),
					resource.TestCheckResourceAttr("grafana_role.test", "uid", "terraform-acc-test"),
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

func TestAccRole_NonGlobalRolesCanBeManagedWithSA(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=9.0.0")
	orgScopedTest(t)
	randomName := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: roleConfig(randomName, false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_role.test", "name", randomName),
					resource.TestCheckResourceAttr("grafana_role.test", "description", "test desc"),
					resource.TestCheckResourceAttr("grafana_role.test", "display_name", "testdisplay"),
					resource.TestCheckResourceAttr("grafana_role.test", "group", "testgroup"),
					checkRoleVersion("grafana_role.test", "1"),
					resource.TestCheckResourceAttr("grafana_role.test", "uid", randomName),
					resource.TestCheckResourceAttr("grafana_role.test", "global", "false"),
					resource.TestCheckResourceAttr("grafana_role.test", "hidden", "true"),
				),
			},
		},
	})
}

func TestAccRole_GlobalCanBeManagedInGrafanaCloud(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)
	randomName := acctest.RandStringFromCharSet(10, acctest.CharSetAlpha)
	config := roleConfig(randomName, true)
	config = strings.ReplaceAll(config, "version = 1", "auto_increment_version = true")

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_role.test", "name", randomName),
					resource.TestCheckResourceAttr("grafana_role.test", "description", "test desc"),
					resource.TestCheckResourceAttr("grafana_role.test", "display_name", "testdisplay"),
					resource.TestCheckResourceAttr("grafana_role.test", "group", "testgroup"),
					checkRoleVersion("grafana_role.test", "1"),
					resource.TestCheckResourceAttr("grafana_role.test", "uid", randomName),
					resource.TestCheckResourceAttr("grafana_role.test", "global", "true"),
					resource.TestCheckResourceAttr("grafana_role.test", "hidden", "true"),
				),
			},
			{
				Config: strings.ReplaceAll(config, "test desc", "updated desc"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_role.test", "name", randomName),
					resource.TestCheckResourceAttr("grafana_role.test", "description", "updated desc"),
					resource.TestCheckResourceAttr("grafana_role.test", "display_name", "testdisplay"),
					resource.TestCheckResourceAttr("grafana_role.test", "group", "testgroup"),
					checkRoleVersion("grafana_role.test", "2"),
					resource.TestCheckResourceAttr("grafana_role.test", "uid", randomName),
					resource.TestCheckResourceAttr("grafana_role.test", "global", "true"),
					resource.TestCheckResourceAttr("grafana_role.test", "hidden", "true"),
				),
			},
		},
	})
}

func TestAccRoleVersioning(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=9.0.0")

	var role models.RoleDTO
	name := acctest.RandomWithPrefix("versioning-")

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             roleCheckExists.destroyed(&role, nil),
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
					checkRoleVersion("grafana_role.test", "1"),
				),
			},
			{
				Config: fmt.Sprintf(`
				resource "grafana_role" "test" {
					name  = "%s"
					description = "desc 2"
					version = 2
				}`, name),
				Check: resource.ComposeTestCheckFunc(
					roleCheckExists.exists("grafana_role.test", &role),
					resource.TestCheckResourceAttr("grafana_role.test", "name", name),
					checkRoleVersion("grafana_role.test", "2"),
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
					checkRoleVersion("grafana_role.test", "3"),
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
					checkRoleVersion("grafana_role.test", "4"),
				),
			},
		},
	})
}

func TestAccRole_inOrg(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=9.0.0")

	var role models.RoleDTO
	var org models.OrgDetailsDTO
	name := acctest.RandomWithPrefix("role-")

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             orgCheckExists.destroyed(&org, nil),
		Steps: []resource.TestStep{
			{
				Config: roleInOrg(name),
				Check: resource.ComposeTestCheckFunc(
					roleCheckExists.exists("grafana_role.test", &role),
					orgCheckExists.exists("grafana_organization.test", &org),
					checkResourceIsInOrg("grafana_role.test", "grafana_organization.test"),
					resource.TestMatchResourceAttr("grafana_role.test", "id", nonDefaultOrgIDRegexp),
					resource.TestCheckResourceAttr("grafana_role.test", "name", name),
					resource.TestCheckResourceAttr("grafana_role.test", "description", name+" desc"),
					resource.TestCheckResourceAttr("grafana_role.test", "display_name", name),
					resource.TestCheckResourceAttr("grafana_role.test", "group", name),
					checkRoleVersion("grafana_role.test", "1"),
					resource.TestCheckResourceAttr("grafana_role.test", "uid", name),
					resource.TestCheckResourceAttr("grafana_role.test", "global", "false"),
					resource.TestCheckResourceAttr("grafana_role.test", "hidden", "false"),
				),
			},
			// Test destroying role within org. Org keeps existing but role is gone.
			{
				Config: testutils.WithoutResource(t, roleInOrg(name), "grafana_role.test"),
				Check: resource.ComposeTestCheckFunc(
					roleCheckExists.destroyed(&role, &org),
					orgCheckExists.exists("grafana_organization.test", &org),
				),
			},
		},
	})
}

func roleInOrg(name string) string {
	def := fmt.Sprintf(`
resource "grafana_organization" "test" {
	name = "%s"
}

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

	return def
}

var roleConfigBasic = roleConfig("terraform-acc-test", true)

func roleConfig(name string, global bool) string {
	return fmt.Sprintf(`
	resource "grafana_role" "test" {
	  name  = "%[1]s"
	  description = "test desc"
	  version = 1
	  uid = "%[1]s"
	  global = %[2]t
	  group = "testgroup"
	  display_name = "testdisplay"
	  hidden = true
	}
	`, name, global)
}

const roleConfigWithPermissions = `
resource "grafana_role" "test" {
  name  = "terraform-acc-test"
  description = "test desc"
  version = 2
  uid = "terraform-acc-test"
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
