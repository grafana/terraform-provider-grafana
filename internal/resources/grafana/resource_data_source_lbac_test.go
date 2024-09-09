package grafana_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
)

func TestAccDataSourceLBAC(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=10.0.0")

	testName := acctest.RandString(10)
	var dataSource models.DataSource

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             datasourceCheckExists.destroyed(&dataSource, nil),
		Steps: []resource.TestStep{
			{
				Config: dataSourceLBACConfig(testName),
				Check: resource.ComposeTestCheckFunc(
					datasourceCheckExists.exists("grafana_data_source.test", &dataSource),
					resource.TestCheckResourceAttr("grafana_data_source_lbac.test", "data_source_uid", testName),
					resource.TestCheckResourceAttr("grafana_data_source_lbac.test", "permission", "Query"),
				),
			},
			{
				ImportState:       true,
				ResourceName:      "grafana_data_source_lbac.test",
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccDataSourceLBAC_inOrg(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=10.0.0")

	testName := acctest.RandString(10)
	var org models.OrgDetailsDTO
	var dataSource models.DataSource

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             orgCheckExists.destroyed(&org, nil),
		Steps: []resource.TestStep{
			{
				Config: dataSourceLBACConfigInOrg(testName),
				Check: resource.ComposeTestCheckFunc(
					datasourceCheckExists.exists("grafana_data_source.test", &dataSource),
					orgCheckExists.exists("grafana_organization.test", &org),
					checkResourceIsInOrg("grafana_data_source.test", "grafana_organization.test"),
					resource.TestCheckResourceAttr("grafana_data_source_lbac_rule.test", "team_id", "1"),
				),
			},
			{
				ImportState:       true,
				ResourceName:      "grafana_data_source_lbac_rule.test",
				ImportStateVerify: true,
			},
			// Check destroy
			{
				Config: testutils.WithoutResource(t,
					dataSourceLBACConfigInOrg(testName),
					"grafana_data_source_lbac_rule.test",
				),
				Check: resource.ComposeTestCheckFunc(
					orgCheckExists.exists("grafana_organization.test", &org),
					datasourceCheckExists.exists("grafana_data_source.test", &dataSource),
					resource.TestCheckResourceAttr("grafana_data_source_lbac_rule.test", "team_id", "1"),
				),
			},
		},
	})
}

func dataSourceLBACConfig(name string) string {
	return fmt.Sprintf(`
resource "grafana_data_source" "loki" {
  type = "%[1]s"
  name = "loki"
  url  = "http://localhost:3100"
  basic_auth_enabled  = true
  basic_auth_username = "username"

  lifecycle {
    ignore_changes = [json_data_encoded, http_headers]
  }
}

resource "grafana_team" "test" {
	name = "%[1]s"
}

resource "grafana_data_source_lbac_rule" "test" {
	datasource_uid = grafana_data_source.loki.uid
	team_id         = grafana_team.test.id
	rules          = [
		"{ foo != \"bar\", foo !~ \"baz\" }",
		"{ foo = \"qux\", bar ~ \"quux\" }"
	]
}
`, name)
}

func dataSourceLBACConfigInOrg(name string) string {
	return fmt.Sprintf(`
resource "grafana_organization" "test" {
  name = "%[1]s"
}

resource "grafana_data_source" "cloudwatch" {
  org_id = grafana_organization.test.id
  type   = "cloudwatch"
  name   = "cw-example"

  json_data_encoded = jsonencode({
    defaultRegion = "us-east-1"
    authType      = "keys"
  })

  secure_json_data_encoded = jsonencode({
    accessKey = "123"
    secretKey = "456"
  })
}

resource "grafana_team" "test" {
  org_id = grafana_organization.test.id
  name   = "%[1]s"
}

resource "grafana_data_source_lbac_rule" "test" {
  org_id         = grafana_organization.test.id
  datasource_uid = grafana_data_source.cloudwatch.uid
  team_id        = grafana_team.test.id
  rules = [
    "{ foo != \"bar\", foo !~ \"baz\" }",
    "{ foo = \"qux\", bar ~ \"quux\" }"
  ]
}
`, name)
}
