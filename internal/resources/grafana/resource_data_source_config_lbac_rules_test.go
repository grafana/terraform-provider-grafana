package grafana_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceConfigLBACRules_basic(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=11.5.0")

	name := acctest.RandString(10)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceConfigLBACRules(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_data_source_config_lbac_rules.test", "rules"),
					resource.TestCheckResourceAttrWith("grafana_data_source_config_lbac_rules.test", "rules", func(value string) error {
						var rulesMap map[string][]string
						err := json.Unmarshal([]byte(value), &rulesMap)
						if err != nil {
							return fmt.Errorf("failed to parse rules JSON: %v", err)
						}

						expectedRules := []string{
							"{ cluster = \"dev-us-central-0\", namespace = \"hosted-grafana\" }",
							"{ foo = \"qux\" }",
						}

						if len(rulesMap) != 1 {
							return fmt.Errorf("expected 1 team id of rules, got %d", len(rulesMap))
						}

						for teamUID, teamRules := range rulesMap {
							if !reflect.DeepEqual(teamRules, expectedRules) {
								return fmt.Errorf("for team %s, expected rules %v, got %v", teamUID, expectedRules, teamRules)
							}
						}

						return nil
					}),
					resource.TestCheckResourceAttrWith("grafana_data_source.test", "json_data_encoded", func(value string) error {
						var jsonData map[string]any
						err := json.Unmarshal([]byte(value), &jsonData)
						if err != nil {
							return fmt.Errorf("failed to parse json_data_encoded: %v", err)
						}
						return nil
					}),
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
resource "grafana_team" "team" {
	name = "%[1]s-team"
}

resource "grafana_data_source" "test" {
	name = "%[1]s"
	type = "loki"

	basic_auth_enabled = true
	basic_auth_username = "admin"

	lifecycle {
		ignore_changes = [json_data_encoded]
	}
}

resource "grafana_data_source_config_lbac_rules" "test" {
    datasource_uid = grafana_data_source.test.uid
	rules = jsonencode({
		"${grafana_team.team.team_uid}" = [
		"{ cluster = \"dev-us-central-0\", namespace = \"hosted-grafana\" }",
		"{ foo = \"qux\" }"
		]
	})

    depends_on = [
        grafana_team.team,
        grafana_data_source.test
    ]
}
`, name)
}
