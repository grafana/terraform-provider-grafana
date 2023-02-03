package oncall

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/grafana/terraform-provider-grafana/provider/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceSchedule_Basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	scheduleName := fmt.Sprintf("test-acc-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDataSourceScheduleConfig(scheduleName),
				ExpectError: regexp.MustCompile(`couldn't find a schedule`),
			},
		},
	})
}

func testAccDataSourceScheduleConfig(scheduleName string) string {
	return fmt.Sprintf(`
data "grafana_oncall_schedule" "test-acc-schedule" {
	name = "%s"
}
`, scheduleName)
}
