package grafana

import (
	"fmt"
	amixrAPI "github.com/grafana/amixr-api-go-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccAmixrOnCallShift_basic(t *testing.T) {
	CheckCloudInstanceTestsEnabled(t)

	scheduleName := fmt.Sprintf("schedule-%s", acctest.RandString(8))
	shiftName := fmt.Sprintf("shift-%s", acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAmixrOnCallShiftResourceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAmixrOnCallShiftConfig(scheduleName, shiftName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAmixrOnCallShiftResourceExists("amixr_on_call_shift.test-acc-on_call_shift"),
				),
			},
		},
	})
}

func testAccCheckAmixrOnCallShiftResourceDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*amixrAPI.Client)
	for _, r := range s.RootModule().Resources {
		if r.Type != "amixr_on_call_shift" {
			continue
		}

		if _, _, err := client.OnCallShifts.GetOnCallShift(r.Primary.ID, &amixrAPI.GetOnCallShiftOptions{}); err == nil {
			return fmt.Errorf("OnCallShift still exists")
		}

	}
	return nil
}

func testAccAmixrOnCallShiftConfig(scheduleName string, shiftName string) string {
	return fmt.Sprintf(`
resource "amixr_schedule" "test-acc-schedule" {
	type = "calendar"
	name = "%s"
	time_zone = "UTC"
}

resource "amixr_on_call_shift" "test-acc-on_call_shift" {
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

func testAccCheckAmixrOnCallShiftResourceExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No OnCallShift ID is set")
		}

		client := testAccProvider.Meta().(*amixrAPI.Client)

		found, _, err := client.OnCallShifts.GetOnCallShift(rs.Primary.ID, &amixrAPI.GetOnCallShiftOptions{})
		if err != nil {
			return err
		}
		if found.ID != rs.Primary.ID {
			return fmt.Errorf("OnCallShift policy not found: %v - %v", rs.Primary.ID, found)
		}
		return nil
	}
}
