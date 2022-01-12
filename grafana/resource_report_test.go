package grafana

import (
	"fmt"
	"strconv"
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccResourceReport(t *testing.T) {
	CheckCloudTestsEnabled(t)

	var report gapi.Report

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckCloud(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccReportCheckDestroy(&report),
		Steps: []resource.TestStep{
			{
				Config: testAccExample(t, "resources/grafana_report/resource.tf"),
				Check: resource.ComposeTestCheckFunc(
					testAccReportCheckExists("grafana_report.test", &report),
					resource.TestCheckResourceAttrSet("grafana_report.test", "id"),
					resource.TestCheckResourceAttrSet("grafana_report.test", "dashboard_id"),
					resource.TestCheckResourceAttr("grafana_report.test", "name", "my report"),
					resource.TestCheckResourceAttr("grafana_report.test", "recipients.0", "some@email.com"),
					resource.TestCheckNoResourceAttr("grafana_report.test", "recipients.1"),
					resource.TestCheckResourceAttr("grafana_report.test", "schedule.0.frequency", "hourly"),
					resource.TestCheckResourceAttrSet("grafana_report.test", "schedule.0.start_time"), // Date set to current time
					resource.TestCheckResourceAttr("grafana_report.test", "schedule.0.end_time", ""),  // No end time
					resource.TestCheckResourceAttr("grafana_report.test", "orientation", "landscape"),
					resource.TestCheckResourceAttr("grafana_report.test", "layout", "grid"),
					resource.TestCheckResourceAttr("grafana_report.test", "include_dashboard_link", "true"),
					resource.TestCheckResourceAttr("grafana_report.test", "include_table_csv", "false"),
					resource.TestCheckNoResourceAttr("grafana_report.test", "time_range.0.from"),
					resource.TestCheckNoResourceAttr("grafana_report.test", "time_range.0.to"),
				),
			},
			{
				Config: testAccExample(t, "resources/grafana_report/all-options.tf"),
				Check: resource.ComposeTestCheckFunc(
					func(s *terraform.State) error {
						// Check that the ID and dashboard ID are the same as the first run
						// This is a custom function to delay the report ID evaluation, because it is generated after the first run
						return resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr("grafana_report.test", "id", strconv.FormatInt(report.ID, 10)),
							resource.TestCheckResourceAttr("grafana_report.test", "dashboard_id", strconv.FormatInt(report.DashboardID, 10)),
						)(s)
					},
					resource.TestCheckResourceAttr("grafana_report.test", "name", "my report updated"),
					resource.TestCheckResourceAttr("grafana_report.test", "recipients.0", "some@email.com"),
					resource.TestCheckResourceAttr("grafana_report.test", "recipients.1", "some2@email.com"),
					resource.TestCheckResourceAttr("grafana_report.test", "schedule.0.frequency", "daily"),
					resource.TestCheckResourceAttr("grafana_report.test", "schedule.0.workdays_only", "true"),
					resource.TestCheckResourceAttr("grafana_report.test", "schedule.0.start_time", "2020-01-01T07:00:00Z"), // Date transformed to UTC
					resource.TestCheckResourceAttr("grafana_report.test", "schedule.0.end_time", "2020-01-15T08:30:00Z"),   // Date transformed to UTC
					resource.TestCheckResourceAttr("grafana_report.test", "orientation", "portrait"),
					resource.TestCheckResourceAttr("grafana_report.test", "layout", "simple"),
					resource.TestCheckResourceAttr("grafana_report.test", "include_dashboard_link", "false"),
					resource.TestCheckResourceAttr("grafana_report.test", "include_table_csv", "true"),
					resource.TestCheckResourceAttr("grafana_report.test", "time_range.0.from", "now-1h"),
					resource.TestCheckResourceAttr("grafana_report.test", "time_range.0.to", "now"),
				),
			},
		},
	})
}

func testAccReportCheckExists(rn string, report *gapi.Report) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s\n %#v", rn, s.RootModule().Resources)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		client := testAccProvider.Meta().(*client).gapi
		id, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
		if err != nil {
			return err
		}

		if id == 0 {
			return fmt.Errorf("got a report id of 0")
		}
		gotReport, err := client.Report(id)
		if err != nil {
			return fmt.Errorf("error getting report: %s", err)
		}

		*report = *gotReport

		return nil
	}
}

func testAccReportCheckDestroy(report *gapi.Report) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*client).gapi
		_, err := client.Report(report.ID)
		if err == nil {
			return fmt.Errorf("report still exists")
		}
		return nil
	}
}
