package oncall_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceOnCallShift_Basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	randomName := fmt.Sprintf("test-acc-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceOnCallShiftConfig(randomName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.grafana_oncall_on_call_shift.test", "id"),
					resource.TestCheckResourceAttrPair(
						"grafana_oncall_on_call_shift.test", "id",
						"data.grafana_oncall_on_call_shift.test", "id",
					),
					resource.TestCheckResourceAttrPair(
						"grafana_oncall_on_call_shift.test", "type",
						"data.grafana_oncall_on_call_shift.test", "type",
					),
				),
			},
		},
	})
}

func TestAccDataSourceOnCallShift_NotFound(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	shiftName := fmt.Sprintf("test-acc-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDataSourceOnCallShiftNotFoundConfig(shiftName),
				ExpectError: regexp.MustCompile(`couldn't find an on-call shift matching`),
			},
		},
	})
}

func testAccDataSourceOnCallShiftNotFoundConfig(name string) string {
	return fmt.Sprintf(`
data "grafana_oncall_on_call_shift" "test" {
	name = "%s"
}
`, name)
}

func testAccDataSourceOnCallShiftConfig(name string) string {
	return fmt.Sprintf(`
resource "grafana_oncall_on_call_shift" "test" {
	name       = "%[1]s"
	type       = "single_event"
	start      = "2020-09-04T16:00:00"
	duration   = 3600
}

data "grafana_oncall_on_call_shift" "test" {
	name = grafana_oncall_on_call_shift.test.name
}
`, name)
}
