package grafana_test

import (
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDatasourceTeams_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	checks := []resource.TestCheckFunc{
		resource.TestCheckResourceAttrSet("data.grafana_teams.all", "teams.#"),
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "grafana_team" "test" {
	name  = "test-teams-ds"
	email = "test-teams-ds@example.com"
}
data "grafana_teams" "all" {
    depends_on = [grafana_team.test]
}
				`,
				Check: resource.ComposeTestCheckFunc(checks...),
			},
		},
	})
}
