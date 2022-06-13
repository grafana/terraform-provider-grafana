package grafana

import (
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDatasourceTeam(t *testing.T) {
	CheckOSSTestsEnabled(t)

	var team gapi.Team
	checks := []resource.TestCheckFunc{
		testAccTeamCheckExists("grafana_team.test", &team),
		resource.TestCheckResourceAttr(
			"data.grafana_team.from_name", "name", "test-team",
		),
		resource.TestMatchResourceAttr(
			"data.grafana_team.from_name", "id", idRegexp,
		),
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccTeamCheckDestroy(&team),
		Steps: []resource.TestStep{
			{
				Config: testAccExample(t, "data-sources/grafana_team/data-source.tf"),
				Check:  resource.ComposeTestCheckFunc(checks...),
			},
		},
	})
}
