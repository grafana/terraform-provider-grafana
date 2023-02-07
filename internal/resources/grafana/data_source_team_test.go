package grafana_test

import (
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/grafana/terraform-provider-grafana/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDatasourceTeam(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	var team gapi.Team
	checks := []resource.TestCheckFunc{
		testAccTeamCheckExists("grafana_team.test", &team),
		resource.TestCheckResourceAttr(
			"data.grafana_team.from_name", "name", "test-team",
		),
		resource.TestMatchResourceAttr(
			"data.grafana_team.from_name", "id", common.IDRegexp,
		),
	}

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      testAccTeamCheckDestroy(&team),
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "data-sources/grafana_team/data-source.tf"),
				Check:  resource.ComposeTestCheckFunc(checks...),
			},
		},
	})
}
