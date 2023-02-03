package oncall

import (
	"fmt"
	"testing"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/grafana/terraform-provider-grafana/provider/common"
	"github.com/grafana/terraform-provider-grafana/provider/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccOnCallOnCallShift_basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	scheduleName := fmt.Sprintf("schedule-%s", acctest.RandString(8))
	shiftName := fmt.Sprintf("shift-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      testAccCheckOnCallOnCallShiftResourceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOnCallOnCallShiftConfig(scheduleName, shiftName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOnCallOnCallShiftResourceExists("grafana_oncall_on_call_shift.test-acc-on_call_shift"),
				),
			},
		},
	})
}

func testAccCheckOnCallOnCallShiftResourceDestroy(s *terraform.State) error {
	client := testutils.Provider.Meta().(*common.Client).OnCallClient
	for _, r := range s.RootModule().Resources {
		if r.Type != "grafana_oncall_on_call_shift" {
			continue
		}

		if _, _, err := client.OnCallShifts.GetOnCallShift(r.Primary.ID, &onCallAPI.GetOnCallShiftOptions{}); err == nil {
			return fmt.Errorf("OnCallShift still exists")
		}
	}
	return nil
}

func testAccOnCallOnCallShiftConfig(scheduleName string, shiftName string) string {
	return fmt.Sprintf(`
resource "grafana_oncall_schedule" "test-acc-schedule" {
	type = "calendar"
	name = "%s"
	time_zone = "UTC"
}

resource "grafana_oncall_on_call_shift" "test-acc-on_call_shift" {
	name = "%s"
	type = "recurrent_event"
	start = "2020-09-04T16:00:00"
	duration = 3600
	level = 1
	frequency = "weekly"
	week_start = "SU"
	interval = 2
	by_day = ["MO", "FR"]
}
`, scheduleName, shiftName)
}

func testAccCheckOnCallOnCallShiftResourceExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No OnCallShift ID is set")
		}

		client := testutils.Provider.Meta().(*common.Client).OnCallClient

		found, _, err := client.OnCallShifts.GetOnCallShift(rs.Primary.ID, &onCallAPI.GetOnCallShiftOptions{})
		if err != nil {
			return err
		}
		if found.ID != rs.Primary.ID {
			return fmt.Errorf("OnCallShift policy not found: %v - %v", rs.Primary.ID, found)
		}
		return nil
	}
}
