package grafana

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	gapi "github.com/nytm/go-grafana-api"
)

func TestAccOrganization_basic(t *testing.T) {
	var org gapi.Org

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccOrganizationCheckDestroy(&org),
		Steps: []resource.TestStep{
			{
				Config: testAccOrganizationConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccOrganizationCheckExists("grafana_organization.test", &org),
					resource.TestCheckResourceAttr(
						"grafana_organization.test", "name", "terraform-acc-test",
					),
					resource.TestMatchResourceAttr(
						"grafana_organization.test", "id", regexp.MustCompile(`\d+`),
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
		},
	})
}

func TestAccOrganization_users(t *testing.T) {
	var org gapi.Org

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccOrganizationCheckDestroy(&org),
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
					resource.TestCheckResourceAttr(
						"grafana_organization.test", "admins.0", "john.doe@example.com",
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
					resource.TestCheckResourceAttr(
						"grafana_organization.test", "admins.#", "0",
					),
					resource.TestCheckResourceAttr(
						"grafana_organization.test", "editors.#", "1",
					),
					resource.TestCheckResourceAttr(
						"grafana_organization.test", "editors.0", "john.doe@example.com",
					),
					resource.TestCheckResourceAttr(
						"grafana_organization.test", "viewers.#", "0",
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

func TestAccOrganization_defaultAdmin(t *testing.T) {
	var org gapi.Org

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccOrganizationCheckDestroy(&org),
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
					resource.TestCheckResourceAttr(
						"grafana_organization.test", "admins.0", "john.doe@example.com",
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
					resource.TestCheckResourceAttr(
						"grafana_organization.test", "admins.0", "admin@localhost",
					),
					resource.TestCheckResourceAttr(
						"grafana_organization.test", "admins.1", "john.doe@example.com",
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

func testAccOrganizationCheckExists(rn string, a *gapi.Org) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}
		tmp, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
		id := int64(tmp)
		if err != nil {
			return fmt.Errorf("resource id is malformed")
		}

		client := testAccProvider.Meta().(*gapi.Client)
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
		client := testAccProvider.Meta().(*gapi.Client)
		org, err := client.Org(a.Id)
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
