package grafana

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/grafana/machine-learning-go-client/mlapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccResourceMachineLearningHoliday(t *testing.T) {
	CheckCloudInstanceTestsEnabled(t)

	var holiday mlapi.Holiday
	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccMLHolidayCheckDestroy(&holiday),
		Steps: []resource.TestStep{
			{
				Config: testAccExample(t, "resources/grafana_machine_learning_holiday/ical_holiday.tf"),
				Check: resource.ComposeTestCheckFunc(
					testAccMLHolidayCheckExists("grafana_machine_learning_holiday.ical", &holiday),
					resource.TestCheckResourceAttrSet("grafana_machine_learning_holiday.ical", "id"),
					resource.TestCheckResourceAttr("grafana_machine_learning_holiday.ical", "name", "My iCal holiday"),
				),
			},
			{
				Config: testAccExample(t, "resources/grafana_machine_learning_holiday/custom_periods_holiday.tf"),
				Check: resource.ComposeTestCheckFunc(
					testAccMLHolidayCheckExists("grafana_machine_learning_holiday.custom_periods", &holiday),
					resource.TestCheckResourceAttrSet("grafana_machine_learning_holiday.custom_periods", "id"),
					resource.TestCheckResourceAttr("grafana_machine_learning_holiday.custom_periods", "name", "My custom periods holiday"),
				),
			},
		},
	})
}

func testAccMLHolidayCheckExists(rn string, holiday *mlapi.Holiday) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s\n %#v", rn, s.RootModule().Resources)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		client := testAccProvider.Meta().(*client).mlapi
		gotHoliday, err := client.Holiday(context.Background(), rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("error getting holiday: %s", err)
		}

		*holiday = gotHoliday

		return nil
	}
}

func testAccMLHolidayCheckDestroy(holiday *mlapi.Holiday) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// This check is to make sure that no pointer conversions are incorrect
		// while mutating holiday.
		if holiday.ID == "" {
			return fmt.Errorf("checking deletion of empty id")
		}
		client := testAccProvider.Meta().(*client).mlapi
		_, err := client.Holiday(context.Background(), holiday.ID)
		if err == nil {
			return fmt.Errorf("holiday still exists on server")
		}
		return nil
	}
}

const machineLearningHolidayInvalid = `
resource "grafana_machine_learning_holiday" "invalid" {
  name            = "Test Holiday"
}
`

const machineLearningHolidayInvalidTimeZone = `
resource "grafana_machine_learning_holiday" "invalid" {
  name          = "Test Holiday"
	ical_url      = "https://calendar.google.com/calendar/ical/en.uk%23holiday%40group.v.calendar.google.com/public/basic.ics"
	ical_timezone = "invalid"
}
`

const machineLearningHolidayInvalidCustomPeriodTimes = `
resource "grafana_machine_learning_holiday" "invalid" {
  name           = "Test Holiday"
	custom_periods {
		start_time = "not a time"
		end_time = "not a time"
	}
}
`

func TestAccResourceInvalidMachineLearningHoliday(t *testing.T) {
	CheckCloudInstanceTestsEnabled(t)

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      machineLearningHolidayInvalid,
				ExpectError: regexp.MustCompile(".*one of `custom_periods,ical_url` must be specified"),
			},
			{
				Config:      machineLearningHolidayInvalidTimeZone,
				ExpectError: regexp.MustCompile(".*IANA.*"),
			},
			{
				Config:      machineLearningHolidayInvalidCustomPeriodTimes,
				ExpectError: regexp.MustCompile(".*valid RFC3339 date"),
			},
		},
	})
}
