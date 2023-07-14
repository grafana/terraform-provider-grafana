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
		resource.TestCheckResourceAttr("data.grafana_team.from_name", "name", "test-team"),
		resource.TestMatchResourceAttr("data.grafana_team.from_name", "id", common.IDRegexp),
		resource.TestCheckResourceAttr("data.grafana_team.from_name", "email", "test-team-email@test.com"),
		resource.TestCheckResourceAttr("data.grafana_team.from_name", "members.#", "0"),
		resource.TestCheckResourceAttr("data.grafana_team.from_name", "preferences.#", "1"),
		resource.TestCheckResourceAttr("data.grafana_team.from_name", "preferences.0.theme", "dark"),
		resource.TestCheckResourceAttr("data.grafana_team.from_name", "preferences.0.timezone", "utc"),
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

func TestAccDatasourceTeam_teamSync(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t)

	var team gapi.Team
	checks := []resource.TestCheckFunc{
		testAccTeamCheckExists("grafana_team.test", &team),
		resource.TestCheckResourceAttr("data.grafana_team.from_name", "name", "test-team"),
		resource.TestMatchResourceAttr("data.grafana_team.from_name", "id", common.IDRegexp),
		resource.TestCheckResourceAttr("data.grafana_team.from_name", "email", "test-team-email@test.com"),
		resource.TestCheckResourceAttr("data.grafana_team.from_name", "members.#", "0"),
		resource.TestCheckResourceAttr("data.grafana_team.from_name", "preferences.#", "1"),
		resource.TestCheckResourceAttr("data.grafana_team.from_name", "preferences.0.theme", "dark"),
		resource.TestCheckResourceAttr("data.grafana_team.from_name", "preferences.0.timezone", "utc"),
		resource.TestCheckResourceAttr("data.grafana_team.from_name", "team_sync.0.groups.#", "2"),
		resource.TestCheckResourceAttr("data.grafana_team.from_name", "team_sync.0.groups.0", "group1"),
		resource.TestCheckResourceAttr("data.grafana_team.from_name", "team_sync.0.groups.1", "group2"),
	}

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      testAccTeamCheckDestroy(&team),
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "data-sources/grafana_team/with-team-sync.tf"),
				Check:  resource.ComposeTestCheckFunc(checks...),
			},
		},
	})
}
