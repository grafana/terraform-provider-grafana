package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/grafana/terraform-provider-grafana/provider/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceOnCallTeam_Basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	teamName := fmt.Sprintf("test-acc-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDataSourceOnCallTeamConfig(teamName),
				ExpectError: regexp.MustCompile(`couldn't find a team`),
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
