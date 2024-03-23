package oncall_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v2/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceTeam_Basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	teamName := fmt.Sprintf("test-acc-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDataSourceTeamConfig(teamName),
				ExpectError: regexp.MustCompile(`couldn't find a team`),
			},
		},
	})
}

func testAccDataSourceTeamConfig(teamName string) string {
	return fmt.Sprintf(`
data "grafana_oncall_team" "test-acc-team" {
	name = "%s"
}
`, teamName)
}
