package grafana_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceConfigLBACRules_basic(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=11.0.0")

	var ds models.DataSource
	name := acctest.RandString(10)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceConfigLBACRules(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					datasourceCheckExists.exists("grafana_data_source.test", &ds),
					resource.TestCheckResourceAttrSet("grafana_data_source_config_lbac_rules.test", "rules"),
					resource.TestCheckResourceAttrWith("grafana_data_source_config_lbac_rules.test", "rules", func(value string) error {
						var rulesMap map[string][]string
						err := json.Unmarshal([]byte(value), &rulesMap)
						if err != nil {
							return fmt.Errorf("failed to parse rules JSON: %v", err)
						}

						expectedRules := []string{
							"{ foo != \"bar\", foo !~ \"baz\" }",
							"{ foo = \"qux\" }",
						}

						if len(rulesMap) != 2 {
							return fmt.Errorf("expected 2 team rules, got %d", len(rulesMap))
						}

						for teamID, teamRules := range rulesMap {
							if !strings.HasPrefix(teamID, "1:") {
								return fmt.Errorf("unexpected team ID format: %s", teamID)
							}
							if !reflect.DeepEqual(teamRules, expectedRules) {
								return fmt.Errorf("for team %s, expected rules %v, got %v", teamID, expectedRules, teamRules)
							}
						}

						return nil
					}),
					resource.TestCheckResourceAttrWith("grafana_data_source.test", "json_data_encoded", func(value string) error {
						var jsonData map[string]interface{}
						err := json.Unmarshal([]byte(value), &jsonData)
						if err != nil {
							return fmt.Errorf("failed to parse json_data_encoded: %v", err)
						}
						return nil
					}),
				),
			},
		},
	})
}

func testAccDataSourceConfigLBACRules(name string) string {
	return fmt.Sprintf(`
resource "grafana_data_source" "test" {
	name = "%[1]s"
	type = "loki"

	basic_auth_enabled = true
	basic_auth_username = "admin"

	lifecycle {
    	ignore_changes = [json_data_encoded]
  }
}

resource "grafana_team" "team1" {
	name = "team1"
}

resource "grafana_team" "team2" {
	name = "team2"
}

resource "grafana_data_source_config_lbac_rules" "test" {
    datasource_uid = grafana_data_source.test.uid
    rules = {
        "${grafana_team.team1.id}" = jsonencode([
            "{ foo != \"bar\", foo !~ \"baz\" }",
            "{ foo = \"qux\" }"
        ])
    }
}
`, name)
}
