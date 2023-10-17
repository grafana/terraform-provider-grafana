package grafana_test

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/grafana/terraform-provider-grafana/internal/resources/grafana"
	"github.com/grafana/terraform-provider-grafana/internal/testutils"
)

func TestAccServiceAccount_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)
	testutils.CheckOSSTestsSemver(t, ">=9.1.0")

	name := acctest.RandString(10)

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      testAccServiceAccountCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testServiceAccountConfig(name, "Editor"),
				Check: resource.ComposeTestCheckFunc(
					testAccServiceAccountCheckExists,
					resource.TestCheckResourceAttr("grafana_service_account.test", "name", name),
					resource.TestCheckResourceAttr("grafana_service_account.test", "org_id", "1"),
					resource.TestCheckResourceAttr("grafana_service_account.test", "role", "Editor"),
					resource.TestCheckResourceAttr("grafana_service_account.test", "is_disabled", "false"),
					resource.TestMatchResourceAttr("grafana_service_account.test", "id", defaultOrgIDRegexp),
				),
			},
		},
	})
}

func TestAccServiceAccount_many(t *testing.T) {
	testutils.CheckOSSLongRunningTestsEnabled(t)
	testutils.CheckOSSTestsSemver(t, ">=9.1.0")

	name := acctest.RandString(10)

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      testAccServiceAccountCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testManyServiceAccountsConfig(name, 60),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_service_account.test_1", "name", name+"-1"),
					resource.TestCheckResourceAttr("grafana_service_account.test_2", "name", name+"-2"),
				),
			},
		},
	})
}

func TestAccServiceAccount_invalid_role(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      testAccServiceAccountCheckDestroy,
		Steps: []resource.TestStep{
			{
				ExpectError: regexp.MustCompile(`.*expected role to be one of \[.+\], got InvalidRole`),
				Config:      testServiceAccountConfig("any", "InvalidRole"),
			},
		},
	})
}

func testManyServiceAccountsConfig(prefix string, count int) string {
	config := ``

	for i := 0; i < count; i++ {
		config += fmt.Sprintf(`
		resource "grafana_service_account" "test_%[2]d" {
			name        = "%[1]s-%[2]d"
			is_disabled = false
			role        = "Viewer"
		}
`, prefix, i)
	}

	return config
}

func testAccServiceAccountCheckExists(s *terraform.State) error {
	return testAccServiceAccountCheckExistsBool(s, true)
}

func testAccServiceAccountCheckDestroy(s *terraform.State) error {
	return testAccServiceAccountCheckExistsBool(s, false)
}

func testAccServiceAccountCheckExistsBool(s *terraform.State, shouldExist bool) error {
	c := testutils.Provider.Meta().(*common.Client).GrafanaAPI

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "grafana_service_account" {
			continue
		}

		orgID, idStr := grafana.SplitOrgResourceID(rs.Primary.ID)
		id, err := strconv.ParseInt(idStr, 10, 32)
		if err != nil {
			return err
		}

		// If orgID > 1, always check that the SA doesn't exist in the default org
		if orgID > 1 {
			sas, err := c.GetServiceAccounts()
			if err != nil {
				return err
			}

			for _, sa := range sas {
				if sa.ID == id {
					return errors.New("Service account exists in the default org")
				}
			}

			c = c.WithOrgID(orgID)
		}

		sas, err := c.GetServiceAccounts()
		if err != nil {
			return err
		}

		for _, sa := range sas {
			if sa.ID == id {
				if shouldExist {
					return nil
				} else {
					return errors.New("Service account still exists")
				}
			}
		}

		if shouldExist {
			return errors.New("Service account was not found")
		}
	}

	return nil
}

func testServiceAccountConfig(name, role string) string {
	return fmt.Sprintf(`
resource "grafana_service_account" "test" {
	name        = "%[1]s"
	role        = "%[2]s"
	is_disabled = false
}`, name, role)
}
