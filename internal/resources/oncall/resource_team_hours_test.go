package oncall_test

import (
	"fmt"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccTeamHours_basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	teamName := fmt.Sprintf("test-acc-%s", acctest.RandString(8))
	const rn = "grafana_oncall_team_hours.test"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTeamHoursConfig(teamName, "17:00"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(rn, "id"),
					resource.TestCheckResourceAttr(rn, "interval.#", "2"),
					resource.TestCheckResourceAttr(rn, "interval.0.name", "Business hours"),
					resource.TestCheckResourceAttr(rn, "interval.0.monday.0.end", "17:00"),
					resource.TestCheckResourceAttr("grafana_oncall_escalation.test", "type", "pause_until_team_hours"),
				),
			},
			{
				Config: testAccTeamHoursConfig(teamName, "18:00"),
				Check:  resource.TestCheckResourceAttr(rn, "interval.0.monday.0.end", "18:00"),
			},
			{
				ResourceName:      rn,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccTeamHoursConfig(teamName, end string) string {
	return fmt.Sprintf(`
resource "grafana_team" "test" {
  name = "%[1]s"
}

data "grafana_oncall_team" "test" {
  name       = grafana_team.test.name
  depends_on = [grafana_team.test]
}

resource "grafana_oncall_team_hours" "test" {
  team_id = data.grafana_oncall_team.test.id

  interval {
    name   = "Business hours"
    monday {
      start = "09:00"
      end   = "%[2]s"
    }
  }

  interval {
    name     = "Weekend coverage"
    saturday {
      start = "10:00"
      end   = "14:00"
    }
  }
}

resource "grafana_oncall_escalation_chain" "test" {
  name = "%[1]s"
}

resource "grafana_oncall_escalation" "test" {
  escalation_chain_id    = grafana_oncall_escalation_chain.test.id
  type                   = "pause_until_team_hours"
  notify_to_team_members = data.grafana_oncall_team.test.id
  position               = 0
  depends_on             = [grafana_oncall_team_hours.test]
}
`, teamName, end)
}
