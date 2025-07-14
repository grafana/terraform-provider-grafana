package oncall_test

import (
	"fmt"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceOnCallShift_Basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	shiftName := fmt.Sprintf("test-acc-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceOnCallShiftConfig(shiftName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.grafana_oncall_shift.test-acc-shift", "id"),
					resource.TestCheckResourceAttr("data.grafana_oncall_shift.test-acc-shift", "name", shiftName),
				),
			},
		},
	})
}

func testAccDataSourceOnCallShiftConfig(shiftName string) string {
	return fmt.Sprintf(`
data "grafana_oncall_shift" "test-acc-shift" {
	name = "%s"
}
`, shiftName)
}
