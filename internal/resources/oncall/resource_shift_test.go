package oncall_test

import (
	"fmt"
	"regexp"
	"testing"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccOnCallOnCallShift_basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	scheduleName := fmt.Sprintf("schedule-%s", acctest.RandString(8))
	shiftName := fmt.Sprintf("shift-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckOnCallOnCallShiftResourceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOnCallOnCallShiftConfigWeekly(scheduleName, shiftName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOnCallOnCallShiftResourceExists("grafana_oncall_on_call_shift.test-acc-on_call_shift"),
					resource.TestCheckResourceAttr("grafana_oncall_on_call_shift.test-acc-on_call_shift", "name", shiftName),
					resource.TestCheckResourceAttr("grafana_oncall_on_call_shift.test-acc-on_call_shift", "type", "recurrent_event"),
					resource.TestCheckResourceAttr("grafana_oncall_on_call_shift.test-acc-on_call_shift", "start", "2020-09-04T16:00:00"),
					resource.TestCheckResourceAttr("grafana_oncall_on_call_shift.test-acc-on_call_shift", "duration", "3600"),
					resource.TestCheckResourceAttr("grafana_oncall_on_call_shift.test-acc-on_call_shift", "level", "1"),
					resource.TestCheckResourceAttr("grafana_oncall_on_call_shift.test-acc-on_call_shift", "frequency", "weekly"),
					resource.TestCheckResourceAttr("grafana_oncall_on_call_shift.test-acc-on_call_shift", "week_start", "SU"),
					resource.TestCheckResourceAttr("grafana_oncall_on_call_shift.test-acc-on_call_shift", "interval", "2"),
					resource.TestCheckResourceAttr("grafana_oncall_on_call_shift.test-acc-on_call_shift", "by_day.#", "2"),
					resource.TestCheckResourceAttr("grafana_oncall_on_call_shift.test-acc-on_call_shift", "by_day.0", "FR"),
					resource.TestCheckResourceAttr("grafana_oncall_on_call_shift.test-acc-on_call_shift", "by_day.1", "MO"),
					resource.TestCheckResourceAttr("grafana_oncall_on_call_shift.test-acc-on_call_shift", "until", ""),
				),
			},
			{
				Config: testAccOnCallOnCallShiftConfigHourly(scheduleName, shiftName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOnCallOnCallShiftResourceExists("grafana_oncall_on_call_shift.test-acc-on_call_shift"),
					resource.TestCheckResourceAttr("grafana_oncall_on_call_shift.test-acc-on_call_shift", "frequency", "hourly"),
				),
			},
			{
				Config: testAccOnCallOnCallShiftEmptyRollingUsers(scheduleName, shiftName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOnCallOnCallShiftResourceExists("grafana_oncall_on_call_shift.test-acc-on_call_shift"),
					resource.TestCheckResourceAttr("grafana_oncall_on_call_shift.test-acc-on_call_shift", "rolling_users.#", "0"),
				),
			},
			{
				Config:      testAccOnCallOnCallShiftRollingUsersEmptyGroup(scheduleName, shiftName),
				ExpectError: regexp.MustCompile("Error: `rolling_users` can not include an empty group"),
			},
			{
				Config: testAccOnCallOnCallShiftConfigSingle(scheduleName, shiftName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOnCallOnCallShiftResourceExists("grafana_oncall_on_call_shift.test-acc-on_call_shift"),
					resource.TestCheckResourceAttr("grafana_oncall_on_call_shift.test-acc-on_call_shift", "name", shiftName),
				),
			},
			{
				Config: testAccOnCallOnCallShiftConfigUntil(scheduleName, shiftName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOnCallOnCallShiftResourceExists("grafana_oncall_on_call_shift.test-acc-on_call_shift"),
					resource.TestCheckResourceAttr("grafana_oncall_on_call_shift.test-acc-on_call_shift", "until", "2020-10-04T16:00:00"),
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

func testAccOnCallOnCallShiftConfigWeekly(scheduleName, shiftName string) string {
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

func testAccOnCallOnCallShiftConfigHourly(scheduleName, shiftName string) string {
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
	duration = 60
	level = 1
	frequency = "hourly"
	interval = 2
}
`, scheduleName, shiftName)
}

func testAccOnCallOnCallShiftEmptyRollingUsers(scheduleName, shiftName string) string {
	return fmt.Sprintf(`
resource "grafana_oncall_schedule" "test-acc-schedule" {
	type = "calendar"
	name = "%s"
	time_zone = "UTC"
}

resource "grafana_oncall_on_call_shift" "test-acc-on_call_shift" {
	name = "%s"
	type = "rolling_users"
	start = "2020-09-04T16:00:00"
	duration = 3600
	level = 1
	frequency = "weekly"
	week_start = "SU"
	interval = 2
	by_day = ["MO", "FR"]
	rolling_users = []
}
`, scheduleName, shiftName)
}

func testAccOnCallOnCallShiftRollingUsersEmptyGroup(scheduleName, shiftName string) string {
	return fmt.Sprintf(`
resource "grafana_oncall_schedule" "test-acc-schedule" {
	type = "calendar"
	name = "%s"
	time_zone = "UTC"
}

resource "grafana_oncall_on_call_shift" "test-acc-on_call_shift" {
	name = "%s"
	type = "rolling_users"
	start = "2020-09-04T16:00:00"
	duration = 3600
	level = 1
	frequency = "weekly"
	week_start = "SU"
	interval = 2
	by_day = ["MO", "FR"]
	rolling_users = [[]]
}
`, scheduleName, shiftName)
}

func testAccOnCallOnCallShiftConfigSingle(scheduleName, shiftName string) string {
	return fmt.Sprintf(`
resource "grafana_oncall_schedule" "test-acc-schedule" {
	type = "calendar"
	name = "%s"
	time_zone = "UTC"
}

resource "grafana_oncall_on_call_shift" "test-acc-on_call_shift" {
	name = "%s"
	type = "single_event"
	start = "2020-09-04T16:00:00"
	duration = 60
}
`, scheduleName, shiftName)
}

func testAccOnCallOnCallShiftConfigUntil(scheduleName, shiftName string) string {
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
	until = "2020-10-04T16:00:00"
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
