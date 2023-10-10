package oncall_test

import (
	"fmt"
	"testing"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/grafana/terraform-provider-grafana/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccOnCallSchedule_basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	scheduleName := fmt.Sprintf("schedule-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      testAccCheckOnCallScheduleResourceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOnCallScheduleConfig(scheduleName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOnCallScheduleResourceExists("grafana_oncall_schedule.test-acc-schedule"),
					resource.TestCheckResourceAttr("grafana_oncall_schedule.test-acc-schedule", "enable_web_overrides", "false"),
				),
			},
			{
				Config: testAccOnCallScheduleConfigOverrides(scheduleName, true),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOnCallScheduleResourceExists("grafana_oncall_schedule.test-acc-schedule"),
					resource.TestCheckResourceAttr("grafana_oncall_schedule.test-acc-schedule", "enable_web_overrides", "true"),
				),
			},
			{
				Config: testAccOnCallScheduleConfigOverrides(scheduleName, false),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOnCallScheduleResourceExists("grafana_oncall_schedule.test-acc-schedule"),
					resource.TestCheckResourceAttr("grafana_oncall_schedule.test-acc-schedule", "enable_web_overrides", "false"),
				),
			},
		},
	})
}

func testAccCheckOnCallScheduleResourceDestroy(s *terraform.State) error {
	client := testutils.Provider.Meta().(*common.Client).OnCallClient
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

func testAccOnCallScheduleConfigOverrides(scheduleName string, enableWebOverrides bool) string {
	return fmt.Sprintf(`
resource "grafana_oncall_schedule" "test-acc-schedule" {
	name = "%s"
	type = "calendar"
	time_zone = "America/New_York"
	enable_web_overrides = "%t"
}
`, scheduleName, enableWebOverrides)
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

		client := testutils.Provider.Meta().(*common.Client).OnCallClient

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
