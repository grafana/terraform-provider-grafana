package grafana_test

import (
	"errors"
	"fmt"
	"strconv"
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/grafana/terraform-provider-grafana/internal/resources/grafana"
	"github.com/grafana/terraform-provider-grafana/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccGrafanaAuthKey_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	testName := acctest.RandString(10)

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      testAccGrafanaAuthKeyCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGrafanaAuthKeyConfig(testName, "Admin", 0, false),
				Check: resource.ComposeTestCheckFunc(
					testAccGrafanaAuthKeyCheckExists,
					resource.TestMatchResourceAttr("grafana_api_key.foo", "id", defaultOrgIDRegexp),
					resource.TestCheckResourceAttr("grafana_api_key.foo", "org_id", "1"),
					resource.TestCheckResourceAttrSet("grafana_api_key.foo", "key"),
					resource.TestCheckResourceAttr("grafana_api_key.foo", "name", testName),
					resource.TestCheckResourceAttr("grafana_api_key.foo", "role", "Admin"),
					resource.TestCheckNoResourceAttr("grafana_api_key.foo", "expiration"),
				),
			},
			{
				Config: testAccGrafanaAuthKeyConfig(testName+"-modified", "Viewer", 300, false),
				Check: resource.ComposeTestCheckFunc(
					testAccGrafanaAuthKeyCheckExists,
					resource.TestMatchResourceAttr("grafana_api_key.foo", "id", defaultOrgIDRegexp),
					resource.TestCheckResourceAttrSet("grafana_api_key.foo", "key"),
					resource.TestCheckResourceAttr("grafana_api_key.foo", "name", testName+"-modified"),
					resource.TestCheckResourceAttr("grafana_api_key.foo", "role", "Viewer"),
					resource.TestCheckResourceAttrSet("grafana_api_key.foo", "expiration"),
				),
			},
		},
	})
}

func TestAccGrafanaAuthKey_inOrg(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	var org models.OrgDetailsDTO
	testName := acctest.RandString(10)

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      testAccGrafanaAuthKeyCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGrafanaAuthKeyConfig(testName, "Admin", 0, true),
				Check: resource.ComposeTestCheckFunc(
					testAccGrafanaAuthKeyCheckExists,
					resource.TestCheckResourceAttrSet("grafana_api_key.foo", "key"),
					resource.TestCheckResourceAttr("grafana_api_key.foo", "name", testName),
					resource.TestCheckResourceAttr("grafana_api_key.foo", "role", "Admin"),
					resource.TestCheckNoResourceAttr("grafana_api_key.foo", "expiration"),

					// Check that the API key is in the correct organization
					resource.TestMatchResourceAttr("grafana_api_key.foo", "id", nonDefaultOrgIDRegexp),
					orgCheckExists.exists("grafana_organization.test", &org),
					checkResourceIsInOrg("grafana_api_key.foo", "grafana_organization.test"),
				),
			},
		},
	})
}

func testAccGrafanaAuthKeyCheckExists(s *terraform.State) error {
	return testAccGrafanaAuthKeyCheckExistsBool(s, true)
}

func testAccGrafanaAuthKeyCheckDestroy(s *terraform.State) error {
	return testAccGrafanaAuthKeyCheckExistsBool(s, false)
}

func testAccGrafanaAuthKeyCheckExistsBool(s *terraform.State, shouldExist bool) error {
	c := testutils.Provider.Meta().(*common.Client).GrafanaAPI

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "grafana_api_key" {
			continue
		}

		orgID, idStr := grafana.SplitOrgResourceID(rs.Primary.ID)
		id, err := strconv.ParseInt(idStr, 10, 32)
		if err != nil {
			return err
		}

		// If orgID > 1, always check that they key doesn't exist in the default org
		if orgID > 1 {
			keys, err := c.GetAPIKeys(false)
			if err != nil {
				return err
			}

			for _, key := range keys {
				if key.ID == id {
					return errors.New("API key exists in the default org")
				}
			}

			c = c.WithOrgID(orgID)
		}

		keys, err := c.GetAPIKeys(false)
		if err != nil {
			return err
		}

		for _, key := range keys {
			if key.ID == id {
				if shouldExist {
					return nil
				} else {
					return errors.New("API key still exists")
				}
			}
		}

		if shouldExist {
			return errors.New("API key was not found")
		}
	}

	return nil
}

func testAccGrafanaAuthKeyConfig(name, role string, secondsToLive int, inOrg bool) string {
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
}`, name)
		orgIDAttr = "org_id = grafana_organization.test.id"
	}

	config += fmt.Sprintf(`
	resource "grafana_api_key" "foo" {
		name = "%s"
		role = "%s"
		%s
		%s
	}
	`, name, role, secondsToLiveAttr, orgIDAttr)

	return config
}
