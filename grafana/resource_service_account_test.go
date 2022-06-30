package grafana

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	gapi "github.com/grafana/grafana-api-golang-client"
)

func TestAccServiceAccount_basic(t *testing.T) {
	CheckOSSTestsEnabled(t)

	user := gapi.ServiceAccountDTO{}
	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccServiceAccountCheckDestroy(&user),
		Steps: []resource.TestStep{
			{
				Config: testAccServiceAccountConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccServiceAccountCheckExists("grafana_service_account.test", &user),
					resource.TestCheckResourceAttr(
						"grafana_service_account.test", "name", "sa-terraform-test",
					),
					resource.TestCheckResourceAttr(
						"grafana_service_account.test", "role", "Editor",
					),
					resource.TestMatchResourceAttr(
						"grafana_service_account.test", "id", idRegexp,
					),
				),
			},
			{
				Config: testAccServiceAccountConfig_update,
				Check: resource.ComposeTestCheckFunc(
					testAccServiceAccountCheckExists("grafana_service_account.test_disabled", &user),
					resource.TestCheckResourceAttr(
						"grafana_service_account.test_disabled", "name", "sa-terraform-test-disabled",
					),
					resource.TestCheckResourceAttr(
						"grafana_service_account.test_disabled", "role", "Viewer",
					),
					resource.TestCheckResourceAttr(
						"grafana_service_account.test_disabled", "is_disabled", "true",
					),
					resource.TestMatchResourceAttr(
						"grafana_service_account.test", "id", idRegexp,
					),
				),
			},
			{
				ResourceName:            "grafana_service_account.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"},
			},
		},
	})
}

func testAccServiceAccountCheckExists(rn string, a *gapi.ServiceAccountDTO) resource.TestCheckFunc {
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
		sas, err := client.GetServiceAccounts()
		for _, sa := range sas {
			if sa.ID == id {
				*a = sa
				return nil
			}
		}

		return fmt.Errorf("error getting service account: %s", err)
	}
}

func testAccServiceAccountCheckDestroy(a *gapi.ServiceAccountDTO) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*client).gapi
		sas, err := client.GetServiceAccounts()
		if err != nil {
			return err
		}

		for _, sa := range sas {
			if sa.ID == a.ID {
				return fmt.Errorf("service account still exists")
			}
		}

		return nil
	}
}

const testAccServiceAccountConfig_basic = `
resource "grafana_service_account" "test" {
  name     = "sa-terraform-test"
  role     = "Editor"
}
`

const testAccServiceAccountConfig_update = `
resource "grafana_service_account" "test_disabled" {
  name       = "sa-terraform-test-disabled"
  is_disabled = true
}
`
