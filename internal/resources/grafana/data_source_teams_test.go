package grafana_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDatasourceTeams_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	var team models.TeamDTO

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             teamCheckExists.destroyed(&team, nil),
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "data-sources/grafana_teams/_acc_basic.tf"),
				Check: resource.ComposeTestCheckFunc(
					teamCheckExists.exists("grafana_team.dev_alpha", &team),
					resource.TestCheckResourceAttr("data.grafana_teams.by_query", "teams.#", "2"),
				),
			},
		},
	})
}

func TestAccDatasourceTeams_all(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	var team models.TeamDTO

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             teamCheckExists.destroyed(&team, nil),
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "data-sources/grafana_teams/_acc_all.tf"),
				Check: resource.ComposeTestCheckFunc(
					teamCheckExists.exists("grafana_team.test_one", &team),
					resource.TestCheckResourceAttrWith("data.grafana_teams.all", "teams.#", func(value string) error {
						count, err := strconv.Atoi(value)
						if err != nil {
							return fmt.Errorf("teams.# is not a number: %s", value)
						}
						if count < 3 {
							return fmt.Errorf("expected at least 3 teams, got %d", count)
						}
						return nil
					}),
				),
			},
		},
	})
}