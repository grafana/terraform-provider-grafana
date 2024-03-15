package grafana_test

import (
	"fmt"
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/grafana/terraform-provider-grafana/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccOrganization_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	var org models.OrgDetailsDTO

	// TODO: Make parallelizable
	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             orgCheckExists.destroyed(&org, &org),
		Steps: []resource.TestStep{
			{
				Config: testAccOrganizationConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					orgCheckExists.exists("grafana_organization.test", &org),
					resource.TestCheckResourceAttr(
						"grafana_organization.test", "name", "terraform-acc-test",
					),
					resource.TestMatchResourceAttr(
						"grafana_organization.test", "org_id", common.IDRegexp,
					),
					resource.TestMatchResourceAttr(
						"grafana_organization.test", "id", common.IDRegexp,
					),
				),
			},
			{
				Config: testAccOrganizationConfig_updateName,
				Check: resource.ComposeTestCheckFunc(
					orgCheckExists.exists("grafana_organization.test", &org),
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
	testutils.CheckOSSTestsEnabled(t)

	var org models.OrgDetailsDTO

	// TODO: Make parallelizable
	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             orgCheckExists.destroyed(&org, &org),
		Steps: []resource.TestStep{
			{
				Config: testAccOrganizationConfig_usersCreate,
				Check: resource.ComposeTestCheckFunc(
					orgCheckExists.exists("grafana_organization.test", &org),
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
					resource.TestCheckNoResourceAttr(
						"grafana_organization.test", "users_without_access.#",
					),
				),
			},
			{
				Config: testAccOrganizationConfig_usersUpdate,
				Check: resource.ComposeTestCheckFunc(
					orgCheckExists.exists("grafana_organization.test", &org),
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
					resource.TestCheckNoResourceAttr(
						"grafana_organization.test", "users_without_access.#",
					),
				),
			},
			{
				Config: testAccOrganizationConfig_usersRemove,
				Check: resource.ComposeTestCheckFunc(
					orgCheckExists.exists("grafana_organization.test", &org),
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
					resource.TestCheckResourceAttr(
						"grafana_organization.test", "users_without_access.#", "0",
					),
				),
			},
		},
	})
}

func TestAccOrganization_roleNoneUser(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=10.2.0")

	var org models.OrgDetailsDTO

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             orgCheckExists.destroyed(&org, &org),
		Steps: []resource.TestStep{
			{
				Config: testAccOrganizationConfig_usersCreate,
				Check: resource.ComposeTestCheckFunc(
					orgCheckExists.exists("grafana_organization.test", &org),
					resource.TestCheckResourceAttr(
						"grafana_organization.test", "admins.#", "1",
					),
				),
			},
			{
				Config: testAccOrganization_roleNoneUsersUpdate,
				Check: resource.ComposeTestCheckFunc(
					orgCheckExists.exists("grafana_organization.test", &org),
					resource.TestCheckResourceAttr(
						"grafana_organization.test", "name", "terraform-acc-test",
					),
					resource.TestCheckNoResourceAttr(
						"grafana_organization.test", "admins.#",
					),
					resource.TestCheckNoResourceAttr(
						"grafana_organization.test", "editors.#",
					),
					resource.TestCheckNoResourceAttr(
						"grafana_organization.test", "viewers.#",
					),
					resource.TestCheckResourceAttr(
						"grafana_organization.test", "users_without_access.#", "1",
					),
				),
			},
			{
				Config: testAccOrganizationConfig_usersRemove,
				Check: resource.ComposeTestCheckFunc(
					orgCheckExists.exists("grafana_organization.test", &org),
					resource.TestCheckResourceAttr(
						"grafana_organization.test", "admins.#", "0",
					),
					resource.TestCheckResourceAttr(
						"grafana_organization.test", "users_without_access.#", "0",
					),
				),
			},
		},
	})
}

func TestAccOrganization_createManyUsers_longtest(t *testing.T) {
	if testing.Short() { // Also named "longtest" to allow targeting with -run=.*longtest
		t.Skip("skipping test in short mode")
	}
	testutils.CheckOSSTestsEnabled(t)

	var org models.OrgDetailsDTO

	// Don't make this test parallel, it's already creating 1000+ users
	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             orgCheckExists.destroyed(&org, &org),
		Steps: []resource.TestStep{
			{Config: testAccOrganizationConfig_usersCreateMany_1},
			{
				Config: testAccOrganizationConfig_usersCreateMany_2,
				Check: resource.ComposeTestCheckFunc(
					orgCheckExists.exists("grafana_organization.test", &org),
					resource.TestCheckResourceAttr(
						"grafana_organization.test", "name", "terraform-acc-test",
					),
					resource.TestCheckResourceAttr(
						"grafana_organization.test", "viewers.#", "125",
					),
				),
			},
		},
	})
}

func TestAccOrganization_defaultAdmin(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	var org models.OrgDetailsDTO

	// TODO: Make parallelizable
	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             orgCheckExists.destroyed(&org, &org),
		Steps: []resource.TestStep{
			{
				Config: testAccOrganizationConfig_defaultAdminNormal,
				Check: resource.ComposeTestCheckFunc(
					orgCheckExists.exists("grafana_organization.test", &org),
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
					orgCheckExists.exists("grafana_organization.test", &org),
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
	testutils.CheckOSSTestsEnabled(t)

	var org models.OrgDetailsDTO

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             orgCheckExists.destroyed(&org, &org),
		Steps: []resource.TestStep{
			{
				Config: testAccOrganizationConfig_externalUser,
				Check: resource.ComposeTestCheckFunc(
					orgCheckExists.exists("grafana_organization.test", &org),
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
					orgCheckExists.exists("grafana_organization.test", &org),
					resource.TestCheckResourceAttr("grafana_organization.test", "name", "terraform-acc-test-external-user"),
					resource.TestCheckNoResourceAttr("grafana_organization.test", "admins.#"),
					resource.TestCheckNoResourceAttr("grafana_organization.test", "editors.#"),
					resource.TestCheckNoResourceAttr("grafana_organization.test", "viewers.#"),
				),
			},
		},
	})
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
const testAccOrganization_roleNoneUsersUpdate = `
resource "grafana_organization" "test" {
    name = "terraform-acc-test"
    admin_user = "admin"
    create_users = false
		editors = []
    users_without_access = [
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

const testAccOrganizationConfig_usersCreateMany_1 = `
resource "grafana_user" "users" {
	count = 125

	name     = "user-${count.index}"
	email    = "user-${count.index}@example.com"
	login    = "user-${count.index}@example.com"
	password = "password"
}
`

const testAccOrganizationConfig_usersCreateMany_2 = `
resource "grafana_user" "users" {
	count = 125

	name     = "user-${count.index}"
	email    = "user-${count.index}@example.com"
	login    = "user-${count.index}@example.com"
	password = "password"
}

resource "grafana_organization" "test" {
    name         = "terraform-acc-test"
    admin_user   = "admin"
    create_users = false
    viewers      = [ for user in grafana_user.users : user.email ]
}
`
