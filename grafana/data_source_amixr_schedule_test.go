package grafana

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceAmixrSchedule_Basic(t *testing.T) {
	CheckCloudInstanceTestsEnabled(t)

	scheduleName := fmt.Sprintf("test-acc-%s", acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDataSourceAmixrScheduleConfig(scheduleName),
				ExpectError: regexp.MustCompile(`couldn't find a schedule`),
			},
		},
	})
}

func testAccDataSourceAmixrScheduleConfig(scheduleName string) string {
	return fmt.Sprintf(`
data "grafana_amixr_schedule" "test-acc-schedule" {
	name = "%s"
}
`, scheduleName)
}
