package provider

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/provider/common"
	"github.com/grafana/terraform-provider-grafana/provider/testutils"
)

func TestAccServiceAccount_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)
	testutils.CheckOSSTestsSemver(t, ">=9.1.0")

	sa := gapi.ServiceAccountDTO{}
	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccServiceAccountCheckDestroy(&sa),
		Steps: []resource.TestStep{
			{
				Config: testServiceAccountConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccServiceAccountCheckExists("grafana_service_account.test", &sa),
					resource.TestCheckResourceAttr(
						"grafana_service_account.test", "name", "sa-terraform-test",
					),
					resource.TestCheckResourceAttr(
						"grafana_service_account.test", "role", "Editor",
					),
					resource.TestCheckResourceAttr(
						"grafana_service_account.test", "is_disabled", "false",
					),
					resource.TestMatchResourceAttr(
						"grafana_service_account.test", "id", idRegexp,
					),
				),
			},
		},
	})
}

func TestAccServiceAccount_invalid_role(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	sa := gapi.ServiceAccountDTO{}

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccServiceAccountCheckDestroy(&sa),
		Steps: []resource.TestStep{
			{
				ExpectError: regexp.MustCompile(`expected role to be one of \[Viewer Editor Admin], got InvalidRole`),
				Config:      testServiceAccountConfigInvalidRole,
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
		client := testAccProvider.Meta().(*common.Client).GrafanaAPI
		sas, err := client.GetServiceAccounts()
		for _, sa := range sas {
			if sa.ID == id {
				*a = sa
				a.Name = rs.Primary.Attributes["name"]
				a.Role = rs.Primary.Attributes["role"]
				d, err := strconv.ParseBool(rs.Primary.Attributes["is_disabled"])
				if err != nil {
					return fmt.Errorf("error parsing is_disabled field: %s", err)
				}
				a.IsDisabled = d
				return nil
			}
		}

		return fmt.Errorf("error getting service account: %s", err)
	}
}

func testAccServiceAccountCheckDestroy(a *gapi.ServiceAccountDTO) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*common.Client).GrafanaAPI
		sas, err := client.GetServiceAccounts()
		if err != nil {
			return err
		}

		for _, sa := range sas {
			if a.ID == sa.ID {
				return fmt.Errorf("service account still exists")
			}
		}

		return nil
	}
}

const testServiceAccountConfigBasic = `
resource "grafana_service_account" "test" {
  name        = "sa-terraform-test"
  role        = "Editor"
  is_disabled = false
}`

const testServiceAccountConfigInvalidRole = `
resource "grafana_service_account" "test" {
  name        = "sa-terraform-test"
  role        = "InvalidRole"
  is_disabled = false
}`
