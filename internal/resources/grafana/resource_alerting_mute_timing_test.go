package grafana_test

import (
	"fmt"
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/grafana/terraform-provider-grafana/internal/testutils"
)

func TestAccMuteTiming_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">9.0.0")

	var mt models.MuteTimeInterval

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		// Implicitly tests deletion.
		CheckDestroy: alertingMuteTimingCheckExists.destroyed(&mt, nil),
		Steps: []resource.TestStep{
			// Test creation.
			{
				Config: testutils.TestAccExample(t, "resources/grafana_mute_timing/resource.tf"),
				Check: resource.ComposeTestCheckFunc(
					alertingMuteTimingCheckExists.exists("grafana_mute_timing.my_mute_timing", &mt),
					resource.TestCheckResourceAttr("grafana_mute_timing.my_mute_timing", "name", "My Mute Timing"),
					resource.TestCheckResourceAttr("grafana_mute_timing.my_mute_timing", "intervals.0.times.0.start", "04:56"),
					resource.TestCheckResourceAttr("grafana_mute_timing.my_mute_timing", "intervals.0.times.0.end", "14:17"),
					resource.TestCheckResourceAttr("grafana_mute_timing.my_mute_timing", "intervals.0.weekdays.0", "monday"),
					resource.TestCheckResourceAttr("grafana_mute_timing.my_mute_timing", "intervals.0.weekdays.1", "tuesday:thursday"),
					resource.TestCheckResourceAttr("grafana_mute_timing.my_mute_timing", "intervals.0.days_of_month.0", "1:7"),
					resource.TestCheckResourceAttr("grafana_mute_timing.my_mute_timing", "intervals.0.days_of_month.1", "-1"),
					resource.TestCheckResourceAttr("grafana_mute_timing.my_mute_timing", "intervals.0.months.0", "1:3"),
					resource.TestCheckResourceAttr("grafana_mute_timing.my_mute_timing", "intervals.0.months.1", "12"),
					resource.TestCheckResourceAttr("grafana_mute_timing.my_mute_timing", "intervals.0.years.0", "2030"),
					resource.TestCheckResourceAttr("grafana_mute_timing.my_mute_timing", "intervals.0.years.1", "2025:2026"),
					resource.TestCheckResourceAttr("grafana_mute_timing.my_mute_timing", "intervals.0.location", "America/New_York"),
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
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_mute_timing/resource.tf", map[string]string{
					"monday": "friday",
				}),
				Check: resource.ComposeTestCheckFunc(
					alertingMuteTimingCheckExists.exists("grafana_mute_timing.my_mute_timing", &mt),
					resource.TestCheckResourceAttr("grafana_mute_timing.my_mute_timing", "intervals.0.weekdays.0", "friday"),
					resource.TestCheckResourceAttr("grafana_mute_timing.my_mute_timing", "intervals.0.weekdays.1", "tuesday:thursday"),
				),
			},
			// Test rename.
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_mute_timing/resource.tf", map[string]string{
					"My Mute Timing": "A Different Mute Timing",
				}),
				Check: resource.ComposeTestCheckFunc(
					alertingMuteTimingCheckExists.exists("grafana_mute_timing.my_mute_timing", &mt),
					resource.TestCheckResourceAttr("grafana_mute_timing.my_mute_timing", "name", "A Different Mute Timing"),
					alertingMuteTimingCheckExists.destroyed(&models.MuteTimeInterval{Name: "My Mute Timing"}, nil),
				),
			},
		},
	})
}

func TestAccMuteTiming_AllTime(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">9.0.0")

	var mt models.MuteTimeInterval
	name := "My Mute Timing"

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      alertingMuteTimingCheckExists.destroyed(&mt, nil),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "grafana_mute_timing" "my_mute_timing" {
	  name = "%s"
	  intervals {}
}`, name),
				Check: resource.ComposeTestCheckFunc(
					alertingMuteTimingCheckExists.exists("grafana_mute_timing.my_mute_timing", &mt),
					resource.TestCheckResourceAttr("grafana_mute_timing.my_mute_timing", "name", name),
					resource.TestCheckResourceAttr("grafana_mute_timing.my_mute_timing", "intervals.0.times.#", "0"),
					resource.TestCheckResourceAttr("grafana_mute_timing.my_mute_timing", "intervals.0.weekdays.#", "0"),
					resource.TestCheckResourceAttr("grafana_mute_timing.my_mute_timing", "intervals.0.days_of_month.#", "0"),
					resource.TestCheckResourceAttr("grafana_mute_timing.my_mute_timing", "intervals.0.months.#", "0"),
					resource.TestCheckResourceAttr("grafana_mute_timing.my_mute_timing", "intervals.0.years.#", "0"),
					resource.TestCheckNoResourceAttr("grafana_mute_timing.my_mute_timing", "intervals.0.location"),
				),
			},
		},
	})
}
