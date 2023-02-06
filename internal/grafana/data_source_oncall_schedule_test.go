package grafana_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/grafana/terraform-provider-grafana/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceOnCallSchedule_Basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	scheduleName := fmt.Sprintf("test-acc-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
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
