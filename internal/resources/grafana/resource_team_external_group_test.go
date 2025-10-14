package grafana_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccTeamExternalGroup_basic(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=9.0.0")

	name := acctest.RandString(10)
	var team models.TeamDTO

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             teamCheckExists.destroyed(&team, nil),
		Steps: []resource.TestStep{
			// Add groups and test import
			{
				Config: testAccTeamExternalGroupConfig(name, []string{"test-group1", "test-group2"}),
				Check: resource.ComposeTestCheckFunc(
					teamCheckExists.exists("grafana_team.test", &team),
					testAccTeamExternalGroupCheck(&team, []string{"test-group1", "test-group2"}),
					resource.TestCheckResourceAttr("grafana_team_external_group.test", "groups.#", "2"),
					resource.TestCheckResourceAttr("grafana_team_external_group.test", "groups.0", "test-group1"),
					resource.TestCheckResourceAttr("grafana_team_external_group.test", "groups.1", "test-group2"),
				),
			},
			{
				ResourceName:      "grafana_team_external_group.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Remove groups
			{
				Config: testAccTeamExternalGroupConfig(name, nil),
				Check: resource.ComposeTestCheckFunc(
					teamCheckExists.exists("grafana_team.test", &team),
					testAccTeamExternalGroupCheck(&team, nil),
					resource.TestCheckResourceAttr("grafana_team_external_group.test", "groups.#", "0"),
				),
			},
			// Add groups again
			{
				Config: testAccTeamExternalGroupConfig(name, []string{"test-group3", "test-group4"}),
				Check: resource.ComposeTestCheckFunc(
					teamCheckExists.exists("grafana_team.test", &team),
					testAccTeamExternalGroupCheck(&team, []string{"test-group3", "test-group4"}),
					resource.TestCheckResourceAttr("grafana_team_external_group.test", "groups.#", "2"),
					resource.TestCheckResourceAttr("grafana_team_external_group.test", "groups.0", "test-group3"),
					resource.TestCheckResourceAttr("grafana_team_external_group.test", "groups.1", "test-group4"),
				),
			},
			// Delete resource and check groups are removed
			{
				Config: testutils.WithoutResource(t, testAccTeamExternalGroupConfig(name, []string{"test-group3", "test-group4"}), "grafana_team_external_group.test"),
				Check: resource.ComposeTestCheckFunc(
					teamCheckExists.exists("grafana_team.test", &team),
					testAccTeamExternalGroupCheck(&team, nil),
				),
			},
		},
	})
}

func testAccTeamExternalGroupCheck(team *models.TeamDTO, expectedGroups []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := grafanaTestClient()

		resp, err := client.SyncTeamGroups.GetTeamGroupsAPI(team.ID)
		if err != nil {
			return fmt.Errorf("Error getting team external groups: %s", err)
		}

		expectedGroupsMap := map[string]struct{}{}
		for _, group := range expectedGroups {
			expectedGroupsMap[group] = struct{}{}
		}

		if len(resp.Payload) != len(expectedGroups) {
			return fmt.Errorf("Expected %d groups, got %d", len(expectedGroups), len(resp.Payload))
		}

		for _, group := range resp.Payload {
			if _, ok := expectedGroupsMap[group.GroupID]; !ok {
				return fmt.Errorf("Unexpected group %s", group.GroupID)
			}
		}

		return nil
	}
}

func testAccTeamExternalGroupConfig(name string, groups []string) string {
	groupsString := ""
	if len(groups) > 0 {
		groupsString = fmt.Sprintf(`"%s"`, strings.Join(groups, `", "`))
	}

	return fmt.Sprintf(`
	resource "grafana_team" "test" {
		name  = "%s"
	}
	
	resource "grafana_team_external_group" "test" {
		team_id    = grafana_team.test.id
		groups = [ %s ]
	}`, name, groupsString)
}
