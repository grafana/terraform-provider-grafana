package grafana_test

import (
	"fmt"
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccGrafanaAuthKey_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	var apiKey models.APIKeyDTO
	testName := acctest.RandString(10)

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      apiKeyCheckExists.destroyed(&apiKey, nil),
		Steps: []resource.TestStep{
			{
				Config: testAccGrafanaAuthKeyConfig(testName, "Admin", 0, false),
				Check: resource.ComposeTestCheckFunc(
					apiKeyCheckExists.exists("grafana_api_key.foo", &apiKey),
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
					apiKeyCheckExists.exists("grafana_api_key.foo", &apiKey),
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
	var apiKey models.APIKeyDTO
	testName := acctest.RandString(10)

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      orgCheckExists.destroyed(&org, nil),
		Steps: []resource.TestStep{
			{
				Config: testAccGrafanaAuthKeyConfig(testName, "Admin", 0, true),
				Check: resource.ComposeTestCheckFunc(
					apiKeyCheckExists.exists("grafana_api_key.foo", &apiKey),
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
			// Check API key deletion within an organization
			{
				Config: testutils.WithoutResource(t, testAccGrafanaAuthKeyConfig(testName, "Admin", 0, true), "grafana_api_key.foo"),
				Check: resource.ComposeTestCheckFunc(
					orgCheckExists.exists("grafana_organization.test", &org),
					apiKeyCheckExists.destroyed(&apiKey, &org),
				),
			},
		},
	})
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
