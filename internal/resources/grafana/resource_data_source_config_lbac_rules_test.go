package grafana_test

import (
	"fmt"
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceConfigLBACRules_basic(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=9.0.0")

	var ds models.DataSource
	name := acctest.RandString(10)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceConfigLBACRules(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					datasourceCheckExists.exists("grafana_data_source.test", &ds),
					resource.TestCheckResourceAttr("grafana_data_source_config_lbac_rules.test", "rules.%", "2"),
					resource.TestCheckResourceAttr("grafana_data_source_config_lbac_rules.test", "rules.team1.#", "2"),
					resource.TestCheckResourceAttr("grafana_data_source_config_lbac_rules.test", "rules.team1.0", "{ foo != \"bar\", foo !~ \"baz\" }"),
					resource.TestCheckResourceAttr("grafana_data_source_config_lbac_rules.test", "rules.team1.1", "{ foo = \"qux\", bar ~ \"quux\" }"),
					resource.TestCheckResourceAttr("grafana_data_source_config_lbac_rules.test", "rules.team2.#", "2"),
					resource.TestCheckResourceAttr("grafana_data_source_config_lbac_rules.test", "rules.team2.0", "{ foo != \"bar\", foo !~ \"baz\" }"),
					resource.TestCheckResourceAttr("grafana_data_source_config_lbac_rules.test", "rules.team2.1", "{ foo = \"qux\", bar ~ \"quux\" }"),
				),
			},
			{
				ResourceName:      "grafana_data_source_config_lbac_rules.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccDataSourceConfigLBACRules(name string) string {
	return fmt.Sprintf(`
resource "grafana_data_source" "test" {
	name = "%[1]s"
	type = "loki"

	json_data_encoded = jsonencode({
		defaultRegion = "us-east-1"
		authType      = "keys"
	})

	secure_json_data_encoded = jsonencode({
		accessKey = "123"
		secretKey = "456"
	})
}

resource "grafana_team" "team1" {
	name = "%[1]s-team1"
}

resource "grafana_team" "team2" {
	name = "%[1]s-team2"
}

resource "grafana_data_source_config_lbac_rules" "test" {
	datasource_uid = grafana_data_source.test.uid
	rules = {
		"${grafana_team.team1.id}" = [
			"{ foo != \"bar\", foo !~ \"baz\" }",
			"{ foo = \"qux\", bar ~ \"quux\" }"
		],
		"${grafana_team.team2.id}" = [
			"{ foo != \"bar\", foo !~ \"baz\" }",
			"{ foo = \"qux\", bar ~ \"quux\" }"
		]
	}
}`, name)
}
