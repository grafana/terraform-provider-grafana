package grafana_test

import (
	"fmt"
	"strconv"
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/grafana/terraform-provider-grafana/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccServiceAccountToken_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)
	testutils.CheckOSSTestsSemver(t, ">=9.1.0")

	name := acctest.RandString(10)
	var sa gapi.ServiceAccountDTO

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      testAccServiceAccountTokenCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccServiceAccountTokenConfig(name, "Editor", 0, false),
				Check: resource.ComposeTestCheckFunc(
					testAccServiceAccountCheckExists(&sa),
					resource.TestCheckResourceAttr("grafana_service_account.test", "name", name),
					resource.TestCheckResourceAttr("grafana_service_account.test", "role", "Editor"),
					resource.TestCheckResourceAttr("grafana_service_account_token.test", "name", name),
					resource.TestCheckNoResourceAttr("grafana_service_account_token.test", "expiration"),
				),
			},
			{
				Config: testAccServiceAccountTokenConfig(name+"-updated", "Viewer", 300, false),
				Check: resource.ComposeTestCheckFunc(
					testAccServiceAccountCheckExists(&sa),
					resource.TestCheckResourceAttr("grafana_service_account.test", "name", name+"-updated"),
					resource.TestCheckResourceAttr("grafana_service_account.test", "role", "Viewer"),
					resource.TestCheckResourceAttr("grafana_service_account_token.test", "name", name+"-updated"),
					resource.TestCheckResourceAttrSet("grafana_service_account_token.test", "expiration"),
				),
			},
		},
	})
}

func TestAccServiceAccountToken_inOrg(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)
	testutils.CheckOSSTestsSemver(t, ">=9.1.0")

	name := acctest.RandString(10)
	var org gapi.Org
	var sa gapi.ServiceAccountDTO

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      testAccServiceAccountTokenCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccServiceAccountTokenConfig(name, "Editor", 0, true),
				Check: resource.ComposeTestCheckFunc(
					testAccServiceAccountCheckExists(&sa),
					resource.TestCheckResourceAttr("grafana_service_account.test", "name", name),
					resource.TestCheckResourceAttr("grafana_service_account.test", "role", "Editor"),
					resource.TestCheckResourceAttr("grafana_service_account_token.test", "name", name),
					resource.TestCheckNoResourceAttr("grafana_service_account_token.test", "expiration"),

					// Check that the service account is in the correct organization
					resource.TestMatchResourceAttr("grafana_service_account.test", "id", nonDefaultOrgIDRegexp),
					testAccOrganizationCheckExists("grafana_organization.test", &org),
					checkResourceIsInOrg("grafana_service_account.test", "grafana_organization.test"),
				),
			},
			{
				Config: testAccServiceAccountTokenConfig(name+"-updated", "Viewer", 300, true),
				Check: resource.ComposeTestCheckFunc(
					testAccServiceAccountCheckExists(&sa),
					resource.TestCheckResourceAttr("grafana_service_account.test", "name", name+"-updated"),
					resource.TestCheckResourceAttr("grafana_service_account.test", "role", "Viewer"),
					resource.TestCheckResourceAttr("grafana_service_account_token.test", "name", name+"-updated"),
					resource.TestCheckResourceAttrSet("grafana_service_account_token.test", "expiration"),

					// Check that the service account is in the correct organization
					resource.TestMatchResourceAttr("grafana_service_account.test", "id", nonDefaultOrgIDRegexp),
					testAccOrganizationCheckExists("grafana_organization.test", &org),
					checkResourceIsInOrg("grafana_service_account.test", "grafana_organization.test"),
				),
			},
		},
	})
}

func testAccServiceAccountTokenCheckDestroy(s *terraform.State) error {
	c := testutils.Provider.Meta().(*common.Client).GrafanaAPI

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "grafana_service_account_token" {
			continue
		}

		idStr := rs.Primary.ID
		id, err := strconv.ParseInt(idStr, 10, 32)
		if err != nil {
			return err
		}

		keys, err := c.GetServiceAccountTokens(1)
		if err != nil {
			return err
		}

		for _, key := range keys {
			if key.ID == id {
				return fmt.Errorf("API key still exists")
			}
		}
	}

	return nil
}

func testAccServiceAccountTokenConfig(name, role string, secondsToLive int, inOrg bool) string {
	config := ""

	secondsToLiveAttr := ""
	if secondsToLive > 0 {
		secondsToLiveAttr = fmt.Sprintf("seconds_to_live = %d", secondsToLive)
	}

	orgIDAttr := ""
	if inOrg {
		config = fmt.Sprintf(`
resource "grafana_organization" "test" {
	name = "%s"
}
`, name)
		orgIDAttr = "org_id = grafana_organization.test.id"
	}

	return config + fmt.Sprintf(`
resource "grafana_service_account" "test" {
	name     = "%[1]s"
	role     = "%[2]s"
	%[4]s
}

resource "grafana_service_account_token" "test" {
	name = "%[1]s"
	service_account_id = grafana_service_account.test.id
	%[3]s
}
`, name, role, secondsToLiveAttr, orgIDAttr)
}
