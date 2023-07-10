package grafana_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/grafana/terraform-provider-grafana/internal/testutils"
)

func TestAccRole(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t)

	var role gapi.Role

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      testAccRoleCheckDestroy(&role),
		Steps: []resource.TestStep{
			{
				Config: roleConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccRoleCheckExists("grafana_role.test", &role),
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
					testAccRoleCheckExists("grafana_role.test", &role),
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

func testAccRoleCheckExists(rn string, r *gapi.Role) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		client := testutils.Provider.Meta().(*common.Client).GrafanaAPI
		role, err := client.GetRole(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("error getting role: %s", err)
		}

		*r = *role

		return nil
	}
}

func testAccRoleCheckDestroy(r *gapi.Role) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testutils.Provider.Meta().(*common.Client).GrafanaAPI
		role, err := client.GetRole(r.UID)
		if err == nil && role.Name != "" {
			return fmt.Errorf("role still exists")
		}
		return nil
	}
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
