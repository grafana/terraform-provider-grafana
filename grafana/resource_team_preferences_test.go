package grafana

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccTeamPreferences_basic(t *testing.T) {
	CheckOSSTestsEnabled(t)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccTeamPreferencesCheckDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testAccTeamPreferencesConfig_Basic,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_team_preferences.testTeamPreferences", "theme", "dark"),
					resource.TestCheckResourceAttr("grafana_team_preferences.testTeamPreferences", "timezone", "utc"),
				),
			},
			{
				Config: testAccTeamPreferencesConfig_Update,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_team_preferences.testTeamPreferences", "theme", "light"),
					resource.TestCheckResourceAttr("grafana_team_preferences.testTeamPreferences", "timezone", "browser"),
				),
			},
		},
	})
}

func testAccTeamPreferencesCheckDestroy() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// you can't really destroy team preferences so nothing to check for
		return nil
	}
}

const testAccTeamPreferencesConfig_Basic = `
resource "grafana_team" "testTeam" {
  name = "terraform-test-team-preferences"
}

resource "grafana_dashboard" "test" {
  config_json = <<EOT
{
  "title": "Terraform Team Preferences Acceptance Test",
  "id": 13,
  "version": "43",
  "uid": "someuid"
}
EOT
}

resource "grafana_team_preferences" "testTeamPreferences" {
  team_id           = grafana_team.testTeam.id
  theme             = "dark"
  home_dashboard_id = grafana_dashboard.test.dashboard_id
  timezone          = "utc"
}
`
const testAccTeamPreferencesConfig_Update = `
resource "grafana_team" "testTeam" {
  name = "terraform-test-team-preferences"
}

resource "grafana_dashboard" "test" {
  config_json = <<EOT
{
  "title": "Terraform Team Preferences Acceptance Test",
  "id": 13,
  "version": "43",
  "uid": "someuid"
}
EOT
}
resource "grafana_team_preferences" "testTeamPreferences" {
  team_id           = grafana_team.testTeam.id
  theme             = "light"
  home_dashboard_id = grafana_dashboard.test.dashboard_id
  timezone          = "browser"
}
`
