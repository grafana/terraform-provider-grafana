package grafana

import (
	"fmt"
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccMuteTiming_basic(t *testing.T) {
	CheckOSSTestsEnabled(t)
	CheckOSSTestsSemver(t, ">9.0.0")

	var mt gapi.MuteTiming

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		// Implicitly tests deletion.
		CheckDestroy: testMuteTimingCheckDestroy(&mt),
		Steps: []resource.TestStep{
			// Test creation.
			{
				Config: testAccExample(t, "resources/grafana_mute_timing/resource.tf"),
				Check: resource.ComposeTestCheckFunc(
					testMuteTimingCheckExists("grafana_mute_timing.my_mute_timing", &mt),
					resource.TestCheckResourceAttr("grafana_mute_timing.my_mute_timing", "intervals.0.weekdays.0", "monday"),
					resource.TestCheckResourceAttr("grafana_mute_timing.my_mute_timing", "intervals.0.weekdays.1", "tuesday:thursday"),
					resource.TestCheckResourceAttr("grafana_mute_timing.my_mute_timing", "intervals.0.days_of_month.0", "1:7"),
					resource.TestCheckResourceAttr("grafana_mute_timing.my_mute_timing", "intervals.0.days_of_month.1", "-1"),
					resource.TestCheckResourceAttr("grafana_mute_timing.my_mute_timing", "intervals.0.months.0", "1:3"),
					resource.TestCheckResourceAttr("grafana_mute_timing.my_mute_timing", "intervals.0.months.1", "12"),
					resource.TestCheckResourceAttr("grafana_mute_timing.my_mute_timing", "intervals.0.years.0", "2030"),
					resource.TestCheckResourceAttr("grafana_mute_timing.my_mute_timing", "intervals.0.years.1", "2025:2026"),
				),
			},
			// Test import.
			{
				ResourceName:      "grafana_mute_timing.my_mute_timing",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Test update content.
			{
				Config: testAccExampleWithReplace(t, "resources/grafana_mute_timing/resource.tf", map[string]string{
					"monday": "friday",
				}),
				Check: resource.ComposeTestCheckFunc(
					testMuteTimingCheckExists("grafana_mute_timing.my_mute_timing", &mt),
					resource.TestCheckResourceAttr("grafana_mute_timing.my_mute_timing", "intervals.0.weekdays.0", "friday"),
					resource.TestCheckResourceAttr("grafana_mute_timing.my_mute_timing", "intervals.0.weekdays.1", "tuesday:thursday"),
				),
			},
			// Test rename.
			{
				Config: testAccExampleWithReplace(t, "resources/grafana_mute_timing/resource.tf", map[string]string{
					"My Mute Timing": "A Different Mute Timing",
				}),
				Check: resource.ComposeTestCheckFunc(
					testMuteTimingCheckExists("grafana_mute_timing.my_mute_timing", &mt),
					resource.TestCheckResourceAttr("grafana_mute_timing.my_mute_timing", "name", "A Different Mute Timing"),
					testMuteTimingCheckDestroy(&gapi.MuteTiming{Name: "My Mute Timing"}),
				),
			},
		},
	})
}

func testMuteTimingCheckExists(rname string, timing *gapi.MuteTiming) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resource, ok := s.RootModule().Resources[rname]
		if !ok {
			return fmt.Errorf("resource not found: %s, resources: %#v", rname, s.RootModule().Resources)
		}

		if resource.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		client := testAccProvider.Meta().(*client).gapi
		mt, err := client.MuteTiming(resource.Primary.ID)
		if err != nil {
			return fmt.Errorf("error getting resource: %w", err)
		}
		*timing = mt
		return nil
	}
}

func testMuteTimingCheckDestroy(timing *gapi.MuteTiming) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*client).gapi
		mt, err := client.MuteTiming(timing.Name)
		if err == nil && mt.Name != "" {
			return fmt.Errorf("mute timing still exists on the server")
		}
		return nil
	}
}
