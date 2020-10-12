package grafana

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	gapi "github.com/grafana/grafana-api-golang-client"
)

func TestAccUser_basic(t *testing.T) {
	var user gapi.User
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccUserCheckDestroy(&user),
		Steps: []resource.TestStep{
			{
				Config: testAccUserConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccUserCheckExists("grafana_user.test", &user),
					resource.TestCheckResourceAttr(
						"grafana_user.test", "email", "terraform-test@localhost",
					),
					resource.TestCheckResourceAttr(
						"grafana_user.test", "name", "Terraform Test",
					),
					resource.TestCheckResourceAttr(
						"grafana_user.test", "login", "tt",
					),
					resource.TestCheckResourceAttr(
						"grafana_user.test", "password", "abc123",
					),
					resource.TestMatchResourceAttr(
						"grafana_user.test", "id", regexp.MustCompile(`\d+`),
					),
				),
			},
			{
				Config: testAccUserConfig_update,
				Check: resource.ComposeTestCheckFunc(
					testAccUserCheckExists("grafana_user.test", &user),
					resource.TestCheckResourceAttr(
						"grafana_user.test", "email", "terraform-test-update@localhost",
					),
					resource.TestCheckResourceAttr(
						"grafana_user.test", "name", "Terraform Test Update",
					),
					resource.TestCheckResourceAttr(
						"grafana_user.test", "login", "ttu",
					),
					resource.TestCheckResourceAttr(
						"grafana_user.test", "password", "zyx987",
					),
				),
			},
		},
	})
}

func testAccUserCheckExists(rn string, a *gapi.User) resource.TestCheckFunc {
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
		user, err := client.User(id)
		if err != nil {
			return fmt.Errorf("error getting data source: %s", err)
		}
		*a = user
		return nil
	}
}

func testAccUserCheckDestroy(a *gapi.User) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*gapi.Client)
		user, err := client.User(a.Id)
		if err == nil && user.Email != "" {
			return fmt.Errorf("user still exists")
		}
		return nil
	}
}

const testAccUserConfig_basic = `
resource "grafana_user" "test" {
  email    = "terraform-test@localhost"
  name     = "Terraform Test"
  login    = "tt"
  password = "abc123"
}
`

const testAccUserConfig_update = `
resource "grafana_user" "test" {
  email    = "terraform-test-update@localhost"
  name     = "Terraform Test Update"
  login    = "ttu"
  password = "zyx987"
}
`
