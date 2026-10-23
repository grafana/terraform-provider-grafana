package oncall_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceIntegration_Basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	randomName := fmt.Sprintf("test-name-%s", acctest.RandString(10))

	integrationPath := "grafana_oncall_integration.test_integration"
	dataSourcePath := "data.grafana_oncall_integration.test_integration_ds"

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckOnCallIntegrationResourceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceIntegration(randomName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOnCallIntegrationResourceExists(integrationPath),
					resource.TestCheckResourceAttr(integrationPath, "name", randomName),
					resource.TestCheckResourceAttrSet(integrationPath, "link"),
					resource.TestCheckResourceAttrPair(
						integrationPath, "id",
						dataSourcePath, "id",
					),
				),
			},
		},
	})
}

func testAccDataSourceIntegration(randomName string) string {
	return fmt.Sprintf(`
resource "grafana_oncall_integration" "test_integration" {
	name = "%[1]s"
	type = "grafana"
	default_route {
	}
}

data "grafana_oncall_integration" "test_integration_ds" {
	id = grafana_oncall_integration.test_integration.id
}
`, randomName)
}

func TestAccDataSourceIntegration_Lookup(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	randomName := fmt.Sprintf("test-acc-%s", acctest.RandString(8))

	integrationPath := "grafana_oncall_integration.test_integration"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckOnCallIntegrationResourceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceIntegrationLookup(randomName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(integrationPath, "id", "data.grafana_oncall_integration.by_name", "id"),
					resource.TestCheckResourceAttrPair(integrationPath, "id", "data.grafana_oncall_integration.by_team_and_type", "id"),
					resource.TestCheckResourceAttr("data.grafana_oncall_integration.by_id", "name", randomName),
					resource.TestCheckResourceAttr("data.grafana_oncall_integration.by_id", "type", "webhook"),
					resource.TestCheckResourceAttrPair(integrationPath, "team_id", "data.grafana_oncall_integration.by_id", "team_id"),
					resource.TestCheckResourceAttr("data.grafana_oncall_integration.direct_paging", "type", "direct_paging"),
					resource.TestCheckResourceAttrPair("data.grafana_oncall_team.test", "id", "data.grafana_oncall_integration.direct_paging", "team_id"),
				),
			},
			{
				Config: testAccDataSourceIntegrationLookup(randomName) + `
data "grafana_oncall_integration" "ambiguous" {
	team_id    = data.grafana_oncall_team.test.id
	depends_on = [grafana_oncall_integration.test_integration]
}
`,
				ExpectError: regexp.MustCompile(`2 integrations match team_id`),
			},
			{
				Config: testAccDataSourceIntegrationNoTeam(randomName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair("grafana_oncall_integration.no_team", "id", "data.grafana_oncall_integration.no_team", "id"),
					resource.TestCheckResourceAttr("data.grafana_oncall_integration.no_team", "team_id", ""),
				),
			},
			{
				Config: `
data "grafana_oncall_integration" "missing" {
	name = "test-acc-no-such-integration"
}
`,
				ExpectError: regexp.MustCompile(`couldn't find an integration matching name "test-acc-no-such-integration"`),
			},
		},
	})
}

func testAccDataSourceIntegrationLookup(randomName string) string {
	return fmt.Sprintf(`
resource "grafana_team" "test" {
	name = "%[1]s"
}

data "grafana_oncall_team" "test" {
	depends_on = [grafana_team.test]
	name       = grafana_team.test.name
}

resource "grafana_oncall_integration" "test_integration" {
	name    = "%[1]s"
	type    = "webhook"
	team_id = data.grafana_oncall_team.test.id
	default_route {
	}
}

data "grafana_oncall_integration" "by_name" {
	name = grafana_oncall_integration.test_integration.name
}

data "grafana_oncall_integration" "by_team_and_type" {
	team_id    = data.grafana_oncall_team.test.id
	type       = "webhook"
	depends_on = [grafana_oncall_integration.test_integration]
}

data "grafana_oncall_integration" "by_id" {
	id = grafana_oncall_integration.test_integration.id
}

data "grafana_oncall_integration" "direct_paging" {
	team_id = data.grafana_oncall_team.test.id
	type    = "direct_paging"
}
`, randomName)
}

// testAccDataSourceIntegrationNoTeam creates a team integration and a no-team
// integration with the same name, which only team_id = "null" tells apart.
func testAccDataSourceIntegrationNoTeam(randomName string) string {
	return fmt.Sprintf(`
resource "grafana_team" "test" {
	name = "%[1]s"
}

data "grafana_oncall_team" "test" {
	depends_on = [grafana_team.test]
	name       = grafana_team.test.name
}

resource "grafana_oncall_integration" "test_integration" {
	name    = "%[1]s"
	type    = "webhook"
	team_id = data.grafana_oncall_team.test.id
	default_route {
	}
}

resource "grafana_oncall_integration" "no_team" {
	name = "%[1]s"
	type = "webhook"
	default_route {
	}
}

data "grafana_oncall_integration" "no_team" {
	name       = "%[1]s"
	team_id    = "null"
	depends_on = [grafana_oncall_integration.no_team, grafana_oncall_integration.test_integration]
}
`, randomName)
}
