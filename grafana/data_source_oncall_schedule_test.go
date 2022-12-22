package grafana

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceOnCallSchedule_Basic(t *testing.T) {
	CheckCloudInstanceTestsEnabled(t)

	scheduleName := fmt.Sprintf("test-acc-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDataSourceOnCallScheduleConfig(scheduleName),
				ExpectError: regexp.MustCompile(`couldn't find a schedule`),
			},
		},
	})
}

func testAccDataSourceOnCallScheduleConfig(scheduleName string) string {
	return fmt.Sprintf(`
data "grafana_oncall_schedule" "test-acc-schedule" {
	name = "%s"
}
`, scheduleName)
}
