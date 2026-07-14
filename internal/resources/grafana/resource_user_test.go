package grafana_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
)

func TestAccUser_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	var user models.UserProfileDTO
	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             userCheckExists.destroyed(&user, nil),
		Steps: []resource.TestStep{
			{
				Config: testAccUserConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					userCheckExists.exists("grafana_user.test", &user),
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
						"grafana_user.test", "id", common.IDRegexp,
					),
				),
			},
			{
				Config: testAccUserConfig_mixedCase,
				Check: resource.ComposeTestCheckFunc(
					userCheckExists.exists("grafana_user.test", &user),
					resource.TestCheckResourceAttr(
						"grafana_user.test", "email", "terraform-test@localhost",
					),
					resource.TestCheckResourceAttr(
						"grafana_user.test", "login", "tt",
					),
				),
			},
			{
				Config: testAccUserConfig_update,
				Check: resource.ComposeTestCheckFunc(
					userCheckExists.exists("grafana_user.test", &user),
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
					resource.TestCheckResourceAttr(
						"grafana_user.test", "is_admin", "true",
					),
				),
			},
			{
				ResourceName:            "grafana_user.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"},
			},
		},
	})
}

func TestAccUser_NeedsBasicAuth(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.0.0")
	orgScopedTest(t)

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccUserConfig_basic,
				ExpectError: regexp.MustCompile("global scope resources cannot be managed with an API key. Use basic auth instead"),
			},
		},
	})
}

const testAccUserConfig_basic = `
resource "grafana_user" "test" {
  email    = "terraform-test@localhost"
  name     = "Terraform Test"
  login    = "tt"
  password = "abc123"
  is_admin = false
}
`

const testAccUserConfig_mixedCase = `
resource "grafana_user" "test" {
  email    = "tErrAForm-TEST@localhost"
  name     = "Terraform Test"
  login    = "tT"
  password = "abc123"
  is_admin = false
}
`

const testAccUserConfig_update = `
resource "grafana_user" "test" {
  email    = "terraform-test-update@localhost"
  name     = "Terraform Test Update"
  login    = "ttu"
  password = "zyx987"
  is_admin = true
}
`
