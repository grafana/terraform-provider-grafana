package grafana_test

import (
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDatasourceTeam_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	var team models.TeamDTO
	checks := []resource.TestCheckFunc{
		teamCheckExists.exists("grafana_team.test", &team),
		resource.TestCheckResourceAttr("data.grafana_team.from_name", "name", "test-team"),
		resource.TestMatchResourceAttr("data.grafana_team.from_name", "id", defaultOrgIDRegexp),
		resource.TestCheckResourceAttr("data.grafana_team.from_name", "email", "test-team-email@test.com"),
		resource.TestCheckResourceAttr("data.grafana_team.from_name", "members.#", "0"),
		resource.TestCheckResourceAttr("data.grafana_team.from_name", "preferences.#", "1"),
		resource.TestCheckResourceAttr("data.grafana_team.from_name", "preferences.0.theme", "dark"),
		resource.TestCheckResourceAttr("data.grafana_team.from_name", "preferences.0.timezone", "utc"),
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             teamCheckExists.destroyed(&team, nil),
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

	var team models.TeamDTO
	checks := []resource.TestCheckFunc{
		teamCheckExists.exists("grafana_team.test", &team),
		resource.TestCheckResourceAttr("data.grafana_team.from_name", "name", "test-team"),
		resource.TestMatchResourceAttr("data.grafana_team.from_name", "id", defaultOrgIDRegexp),
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
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             teamCheckExists.destroyed(&team, nil),
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "data-sources/grafana_team/with-team-sync.tf"),
				Check:  resource.ComposeTestCheckFunc(checks...),
			},
		},
	})
}
