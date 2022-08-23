package grafana

import (
	"fmt"
	"testing"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccOnCallSchedule_basic(t *testing.T) {
	CheckCloudInstanceTestsEnabled(t)

	scheduleName := fmt.Sprintf("schedule-%s", acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckOnCallScheduleResourceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOnCallScheduleConfig(scheduleName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOnCallScheduleResourceExists("grafana_oncall_schedule.test-acc-schedule"),
				),
			},
		},
	})
}

func testAccCheckOnCallScheduleResourceDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*client).onCallAPI
	for _, r := range s.RootModule().Resources {
		if r.Type != "grafana_oncall_schedule" {
			continue
		}

		if _, _, err := client.Schedules.GetSchedule(r.Primary.ID, &onCallAPI.GetScheduleOptions{}); err == nil {
			return fmt.Errorf("Schedule still exists")
		}
	}
	return nil
}

func testAccOnCallScheduleConfig(scheduleName string) string {
	return fmt.Sprintf(`
resource "grafana_oncall_schedule" "test-acc-schedule" {
	name = "%s"
	type = "calendar"
	time_zone = "America/New_York"
}
`, scheduleName)
}

func testAccCheckOnCallScheduleResourceExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No Schedule ID is set")
		}

		client := testAccProvider.Meta().(*client).onCallAPI

		found, _, err := client.Schedules.GetSchedule(rs.Primary.ID, &onCallAPI.GetScheduleOptions{})
		if err != nil {
			return err
		}
		if found.ID != rs.Primary.ID {
			return fmt.Errorf("Schedule policy not found: %v - %v", rs.Primary.ID, found)
		}
		return nil
	}
}
