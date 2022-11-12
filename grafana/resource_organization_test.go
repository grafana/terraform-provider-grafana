package grafana

import (
	"fmt"
	"strconv"
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccOrganization_basic(t *testing.T) {
	CheckOSSTestsEnabled(t)

	var org gapi.Org

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccOrganizationCheckDestroy(&org),
		Steps: []resource.TestStep{
			{
				Config: testAccOrganizationConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccOrganizationCheckExists("grafana_organization.test", &org),
					resource.TestCheckResourceAttr(
						"grafana_organization.test", "name", "terraform-acc-test",
					),
					resource.TestMatchResourceAttr(
						"grafana_organization.test", "org_id", idRegexp,
					),
					resource.TestMatchResourceAttr(
						"grafana_organization.test", "id", idRegexp,
					),
				),
			},
			{
				Config: testAccOrganizationConfig_updateName,
				Check: resource.ComposeTestCheckFunc(
					testAccOrganizationCheckExists("grafana_organization.test", &org),
					resource.TestCheckResourceAttr(
						"grafana_organization.test", "name", "terraform-acc-test-update",
					),
				),
			},
			{
				ResourceName:            "grafana_organization.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"admins", "admin_user", "create_users"}, // Users are imported explicitly (with create_users == false)
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					if len(s) != 1 {
						return fmt.Errorf("expected 1 state: %#v", s)
					}
					admin := s[0].Attributes["admins.0"]
					if admin != "admin@localhost" {
						return fmt.Errorf("expected admin@localhost: %s", admin)
					}
					return nil
				},
			},
		},
	})
}

func TestAccOrganization_users(t *testing.T) {
	CheckOSSTestsEnabled(t)

	var org gapi.Org

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccOrganizationCheckDestroy(&org),
		Steps: []resource.TestStep{
			{
				Config: testAccOrganizationConfig_usersCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccOrganizationCheckExists("grafana_organization.test", &org),
					resource.TestCheckResourceAttr(
						"grafana_organization.test", "name", "terraform-acc-test",
					),
					resource.TestCheckResourceAttr(
						"grafana_organization.test", "admins.#", "1",
					),
					resource.TestCheckNoResourceAttr(
						"grafana_organization.test", "editors.#",
					),
					resource.TestCheckNoResourceAttr(
						"grafana_organization.test", "viewers.#",
					),
				),
			},
			{
				Config: testAccOrganizationConfig_usersUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccOrganizationCheckExists("grafana_organization.test", &org),
					resource.TestCheckResourceAttr(
						"grafana_organization.test", "name", "terraform-acc-test",
					),
					resource.TestCheckNoResourceAttr(
						"grafana_organization.test", "admins.#",
					),
					resource.TestCheckResourceAttr(
						"grafana_organization.test", "editors.#", "1",
					),
					resource.TestCheckNoResourceAttr(
						"grafana_organization.test", "viewers.#",
					),
				),
			},
			{
				Config: testAccOrganizationConfig_usersRemove,
				Check: resource.ComposeTestCheckFunc(
					testAccOrganizationCheckExists("grafana_organization.test", &org),
					resource.TestCheckResourceAttr(
						"grafana_organization.test", "name", "terraform-acc-test",
					),
					resource.TestCheckResourceAttr(
						"grafana_organization.test", "admins.#", "0",
					),
					resource.TestCheckResourceAttr(
						"grafana_organization.test", "editors.#", "0",
					),
					resource.TestCheckResourceAttr(
						"grafana_organization.test", "viewers.#", "0",
					),
				),
			},
		},
	})
}

func TestAccOrganization_createManyUsers(t *testing.T) {
	CheckOSSTestsEnabled(t)

	var org gapi.Org

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccOrganizationCheckDestroy(&org),
		Steps: []resource.TestStep{
			{
				Config: testAccOrganizationConfig_usersCreateMany,
				Check: resource.ComposeTestCheckFunc(
					testAccOrganizationCheckExists("grafana_organization.test", &org),
					resource.TestCheckResourceAttr(
						"grafana_organization.test", "name", "terraform-acc-test",
					),
					resource.TestCheckResourceAttr(
						"grafana_organization.test", "admins.#", "1024",
					),
				),
			},
		},
	})
}

func TestAccOrganization_defaultAdmin(t *testing.T) {
	CheckOSSTestsEnabled(t)

	var org gapi.Org

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccOrganizationCheckDestroy(&org),
		Steps: []resource.TestStep{
			{
				Config: testAccOrganizationConfig_defaultAdminNormal,
				Check: resource.ComposeTestCheckFunc(
					testAccOrganizationCheckExists("grafana_organization.test", &org),
					resource.TestCheckResourceAttr(
						"grafana_organization.test", "name", "terraform-acc-test",
					),
					resource.TestCheckResourceAttr(
						"grafana_organization.test", "admin_user", "admin",
					),
					resource.TestCheckResourceAttr(
						"grafana_organization.test", "admins.#", "1",
					),
					resource.TestCheckNoResourceAttr(
						"grafana_organization.test", "editors.#",
					),
					resource.TestCheckNoResourceAttr(
						"grafana_organization.test", "viewers.#",
					),
				),
			},
			{
				Config: testAccOrganizationConfig_defaultAdminChange,
				Check: resource.ComposeTestCheckFunc(
					testAccOrganizationCheckExists("grafana_organization.test", &org),
					resource.TestCheckResourceAttr(
						"grafana_organization.test", "name", "terraform-acc-test",
					),
					resource.TestCheckResourceAttr(
						"grafana_organization.test", "admin_user", "nobody",
					),
					resource.TestCheckResourceAttr(
						"grafana_organization.test", "admins.#", "2",
					),
					resource.TestCheckNoResourceAttr(
						"grafana_organization.test", "editors.#",
					),
					resource.TestCheckNoResourceAttr(
						"grafana_organization.test", "viewers.#",
					),
				),
			},
			{
				ResourceName:            "grafana_organization.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"admin_user", "create_user"}, // These are provider-side attributes and aren't returned by the API
			},
		},
	})
}

func TestAccOrganization_externalUser(t *testing.T) {
	CheckOSSTestsEnabled(t)

	var org gapi.Org

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccOrganizationCheckDestroy(&org),
		Steps: []resource.TestStep{
			{
				Config: testAccOrganizationConfig_externalUser,
				Check: resource.ComposeTestCheckFunc(
					testAccOrganizationCheckExists("grafana_organization.test", &org),
					resource.TestCheckResourceAttr("grafana_organization.test", "name", "terraform-acc-test-external-user"),
					resource.TestCheckResourceAttr("grafana_organization.test", "admins.#", "1"),
					resource.TestCheckResourceAttr("grafana_organization.test", "admins.0", "external-user@example.com"),
					resource.TestCheckNoResourceAttr("grafana_organization.test", "editors.#"),
					resource.TestCheckNoResourceAttr("grafana_organization.test", "viewers.#"),
				),
			},
			// Removing the external user from Grafana and the organization should succeed (bugfix)
			// Both operations are done from state, so Terraform would try to delete the user reference in the organization
			//   after the user no longer existed. This would fail, so the org user update is now skipped in that case
			{
				Config: testAccOrganizationConfig_externalUserRemoved,
				Check: resource.ComposeTestCheckFunc(
					testAccOrganizationCheckExists("grafana_organization.test", &org),
					resource.TestCheckResourceAttr("grafana_organization.test", "name", "terraform-acc-test-external-user"),
					resource.TestCheckNoResourceAttr("grafana_organization.test", "admins.#"),
					resource.TestCheckNoResourceAttr("grafana_organization.test", "editors.#"),
					resource.TestCheckNoResourceAttr("grafana_organization.test", "viewers.#"),
				),
			},
		},
	})
}

//nolint:unparam // `rn` always receives `"grafana_organization.test"`
func testAccOrganizationCheckExists(rn string, a *gapi.Org) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}
		id, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
		if err != nil {
			return fmt.Errorf("resource id is malformed")
		}

		client := testAccProvider.Meta().(*client).gapi
		org, err := client.Org(id)
		if err != nil {
			return fmt.Errorf("error getting data source: %s", err)
		}

		*a = org

		return nil
	}
}

func testAccOrganizationCheckDestroy(a *gapi.Org) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*client).gapi
		org, err := client.Org(a.ID)
		if err == nil && org.Name != "" {
			return fmt.Errorf("organization still exists")
		}
		return nil
	}
}

const testAccOrganizationConfig_basic = `
resource "grafana_organization" "test" {
    name = "terraform-acc-test"
}
`
const testAccOrganizationConfig_updateName = `
resource "grafana_organization" "test" {
    name = "terraform-acc-test-update"
}
`

const testAccOrganizationConfig_usersCreate = `
resource "grafana_organization" "test" {
    name = "terraform-acc-test"
    admin_user = "admin"
    create_users = true
    admins = [
        "john.doe@example.com",
    ]
}
`
const testAccOrganizationConfig_usersUpdate = `
resource "grafana_organization" "test" {
    name = "terraform-acc-test"
    admin_user = "admin"
    create_users = false
    editors = [
        "john.doe@example.com",
    ]
}
`
const testAccOrganizationConfig_usersRemove = `
resource "grafana_organization" "test" {
    name = "terraform-acc-test"
    admin_user = "admin"
    create_users = false
}
`

const testAccOrganizationConfig_defaultAdminNormal = `
resource "grafana_organization" "test" {
    name = "terraform-acc-test"
    admin_user = "admin"
    create_users = false
    admins = [
        "john.doe@example.com"
    ]
}
`
const testAccOrganizationConfig_defaultAdminChange = `
resource "grafana_organization" "test" {
    name = "terraform-acc-test"
    admin_user = "nobody"
    create_users = false
    admins = [
        "admin@localhost",
        "john.doe@example.com"
    ]
}
`

const testAccOrganizationConfig_externalUser = `
resource "grafana_user" "external" {
	name     = "external"
	email    = "external-user@example.com"
	login    = "external-user"
	password = "password"

}

resource "grafana_organization" "test" {
    name = "terraform-acc-test-external-user"
    create_users = false
    admins = [
        grafana_user.external.email
    ]
}
`

const testAccOrganizationConfig_externalUserRemoved = `
resource "grafana_organization" "test" {
    name = "terraform-acc-test-external-user"
    create_users = false
    admins = []
}
`

const testAccOrganizationConfig_usersCreateMany = `
resource "grafana_organization" "test" {
    name = "terraform-acc-test"
    admin_user = "admin"
    create_users = true
    admins = [for i in range(1024): "user-${i}@example.com"]
}
`
