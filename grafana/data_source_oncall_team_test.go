package grafana

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccDataSourceOnCallTeam_Basic(t *testing.T) {
	CheckCloudInstanceTestsEnabled(t)

	teamName := fmt.Sprintf("test-acc-%s", acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDataSourceOnCallTeamConfig(teamName),
				ExpectError: regexp.MustCompile(`couldn't find a team`),
			},
			{
				Config: testAccExample(t, "data-sources/grafana_oncall_team/data-source.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.grafana_oncall_team.example_team", "name", "Example Team"),
					resource.TestCheckResourceAttr("data.grafana_oncall_team.example_team", "email", "Example Team"),
					resource.TestMatchResourceAttr("data.grafana_oncall_team.example_team", "id", regexp.MustCompile("T[A-Z0-9]+")),
					// Check that the OnCall team ID is different from the Grafana team ID
					func(s *terraform.State) error {
						onCallTeamID := s.RootModule().Resources["data.grafana_oncall_team.example_team"].Primary.ID
						grafanaTeam := s.RootModule().Resources["grafana_team.example_team"].Primary
						if grafanaTeam.ID == onCallTeamID {
							return fmt.Errorf("expected grafana_team.example_team.id to not equal data.grafana_oncall_team.example_team.id")
						}
						if grafanaTeam.Attributes["uid"] == onCallTeamID {
							return fmt.Errorf("expected grafana_team.example_team.uid to not equal data.grafana_oncall_team.example_team.id")
						}
						return nil
					},
				),
			},
		},
	})
}

func testAccDataSourceOnCallTeamConfig(teamName string) string {
	return fmt.Sprintf(`
data "grafana_oncall_team" "test-acc-team" {
	name = "%s"
}
`, teamName)
}
