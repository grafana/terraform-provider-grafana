package grafana_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccResourceReport_Multiple_Dashboards(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t)

	var report models.Report
	var randomUID = acctest.RandStringFromCharSet(10, acctest.CharSetAlpha)
	var randomUID2 = acctest.RandStringFromCharSet(10, acctest.CharSetAlpha)

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      reportCheckExists.destroyed(&report, nil),
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_report/multiple-dashboards.tf", map[string]string{
					`"report-dashboard"`:   fmt.Sprintf(`"%s"`, randomUID),
					`"report-dashboard-2"`: fmt.Sprintf(`"%s"`, randomUID2),
				}),
				Check: resource.ComposeTestCheckFunc(
					reportCheckExists.exists("grafana_report.test", &report),
					resource.TestCheckResourceAttrSet("grafana_report.test", "id"),
					resource.TestCheckNoResourceAttr("grafana_report.test", "dashboard_id"),
					resource.TestCheckNoResourceAttr("grafana_report.test", "dashboard_uid"),
					resource.TestCheckResourceAttr("grafana_report.test", "time_range.#", "0"),
					resource.TestCheckResourceAttr("grafana_report.test", "name", "multiple dashboards"),
					resource.TestCheckResourceAttr("grafana_report.test", "recipients.0", "some@email.com"),
					resource.TestCheckNoResourceAttr("grafana_report.test", "recipients.1"),
					resource.TestCheckResourceAttr("grafana_report.test", "schedule.0.frequency", "monthly"),
					resource.TestCheckResourceAttrSet("grafana_report.test", "schedule.0.start_time"), // Date set to current time
					resource.TestCheckResourceAttr("grafana_report.test", "schedule.0.end_time", ""),  // No end time
					resource.TestCheckResourceAttr("grafana_report.test", "schedule.0.last_day_of_month", "true"),
					resource.TestCheckResourceAttr("grafana_report.test", "schedule.0.timezone", "America/New_York"),
					resource.TestCheckResourceAttr("grafana_report.test", "orientation", "landscape"),
					resource.TestCheckResourceAttr("grafana_report.test", "layout", "grid"),
					resource.TestCheckResourceAttr("grafana_report.test", "include_dashboard_link", "true"),
					resource.TestCheckResourceAttr("grafana_report.test", "include_table_csv", "false"),
					resource.TestCheckNoResourceAttr("grafana_report.test", "time_range.0.from"),
					resource.TestCheckNoResourceAttr("grafana_report.test", "time_range.0.to"),
					resource.TestCheckResourceAttr("grafana_report.test", "dashboards.0.uid", randomUID),
					resource.TestCheckResourceAttr("grafana_report.test", "dashboards.0.time_range.0.from", "now-1h"),
					resource.TestCheckResourceAttr("grafana_report.test", "dashboards.0.time_range.0.to", "now"),
					resource.TestCheckResourceAttr("grafana_report.test", "dashboards.1.time_range.0.from", ""),
					resource.TestCheckResourceAttr("grafana_report.test", "dashboards.1.time_range.0.to", ""),
					resource.TestCheckResourceAttr("grafana_report.test", "dashboards.1.uid", randomUID2),
				),
			},
		},
	})
}

func TestAccResourceReport_basic(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t)

	var report models.Report
	var randomUID = acctest.RandStringFromCharSet(10, acctest.CharSetAlpha)

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      reportCheckExists.destroyed(&report, nil),
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_report/resource.tf", map[string]string{
					`"report-dashboard"`: fmt.Sprintf(`"%s"`, randomUID),
				}),
				Check: resource.ComposeTestCheckFunc(
					reportCheckExists.exists("grafana_report.test", &report),
					resource.TestCheckResourceAttrSet("grafana_report.test", "id"),
					resource.TestCheckResourceAttr("grafana_report.test", "org_id", "1"),
					resource.TestCheckResourceAttrSet("grafana_report.test", "dashboard_id"),
					resource.TestCheckResourceAttr("grafana_report.test", "name", "my report"),
					resource.TestCheckResourceAttr("grafana_report.test", "dashboard_uid", randomUID),
					resource.TestCheckResourceAttr("grafana_report.test", "recipients.0", "some@email.com"),
					resource.TestCheckNoResourceAttr("grafana_report.test", "recipients.1"),
					resource.TestCheckResourceAttr("grafana_report.test", "schedule.0.frequency", "hourly"),
					resource.TestCheckResourceAttrSet("grafana_report.test", "schedule.0.start_time"), // Date set to current time
					resource.TestCheckResourceAttr("grafana_report.test", "schedule.0.end_time", ""),  // No end time
					resource.TestCheckResourceAttr("grafana_report.test", "schedule.0.timezone", "GMT"),
					resource.TestCheckResourceAttr("grafana_report.test", "orientation", "landscape"),
					resource.TestCheckResourceAttr("grafana_report.test", "layout", "grid"),
					resource.TestCheckResourceAttr("grafana_report.test", "include_dashboard_link", "true"),
					resource.TestCheckResourceAttr("grafana_report.test", "include_table_csv", "false"),
					resource.TestCheckNoResourceAttr("grafana_report.test", "time_range.0.from"),
					resource.TestCheckNoResourceAttr("grafana_report.test", "time_range.0.to"),
				),
			},
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_report/all-options.tf", map[string]string{
					`"report-dashboard"`: fmt.Sprintf(`"%s"`, randomUID),
				}),
				Check: resource.ComposeTestCheckFunc(
					func(s *terraform.State) error {
						// Check that the ID and dashboard ID are the same as the first run
						// This is a custom function to delay the report ID evaluation, because it is generated after the first run
						return resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr("grafana_report.test", "id", "1:"+strconv.FormatInt(report.ID, 10)), // <orgid>:<reportid> (1 being the default org)
							resource.TestCheckResourceAttr("grafana_report.test", "dashboard_id", strconv.FormatInt(report.Dashboards[0].Dashboard.ID, 10)),
						)(s)
					},
					resource.TestCheckResourceAttr("grafana_report.test", "dashboard_uid", randomUID),
					resource.TestCheckResourceAttr("grafana_report.test", "name", "my report updated"),
					resource.TestCheckResourceAttr("grafana_report.test", "recipients.0", "some@email.com"),
					resource.TestCheckResourceAttr("grafana_report.test", "recipients.1", "some2@email.com"),
					resource.TestCheckResourceAttr("grafana_report.test", "schedule.0.frequency", "daily"),
					resource.TestCheckResourceAttr("grafana_report.test", "schedule.0.workdays_only", "true"),
					resource.TestCheckResourceAttr("grafana_report.test", "schedule.0.start_time", "2020-01-01T07:00:00Z"), // Date transformed to UTC
					resource.TestCheckResourceAttr("grafana_report.test", "schedule.0.end_time", "2020-01-15T08:30:00Z"),   // Date transformed to UTC
					resource.TestCheckResourceAttr("grafana_report.test", "schedule.0.timezone", "GMT"),
					resource.TestCheckResourceAttr("grafana_report.test", "orientation", "portrait"),
					resource.TestCheckResourceAttr("grafana_report.test", "layout", "simple"),
					resource.TestCheckResourceAttr("grafana_report.test", "include_dashboard_link", "false"),
					resource.TestCheckResourceAttr("grafana_report.test", "include_table_csv", "true"),
					resource.TestCheckResourceAttr("grafana_report.test", "time_range.0.from", "now-1h"),
					resource.TestCheckResourceAttr("grafana_report.test", "time_range.0.to", "now"),
					resource.TestCheckResourceAttr("grafana_report.test", "formats.#", "3"),
					resource.TestCheckResourceAttr("grafana_report.test", "formats.0", "csv"),
					resource.TestCheckResourceAttr("grafana_report.test", "formats.1", "image"),
					resource.TestCheckResourceAttr("grafana_report.test", "formats.2", "pdf"),
				),
			},
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_report/monthly.tf", map[string]string{
					`"report-dashboard"`: fmt.Sprintf(`"%s"`, randomUID),
				}),
				Check: resource.ComposeTestCheckFunc(
					reportCheckExists.exists("grafana_report.test", &report),
					resource.TestCheckResourceAttrSet("grafana_report.test", "id"),
					resource.TestCheckResourceAttrSet("grafana_report.test", "dashboard_id"),
					resource.TestCheckResourceAttr("grafana_report.test", "dashboard_uid", randomUID),
					resource.TestCheckResourceAttr("grafana_report.test", "name", "my report"),
					resource.TestCheckResourceAttr("grafana_report.test", "recipients.0", "some@email.com"),
					resource.TestCheckNoResourceAttr("grafana_report.test", "recipients.1"),
					resource.TestCheckResourceAttr("grafana_report.test", "schedule.0.frequency", "monthly"),
					resource.TestCheckResourceAttrSet("grafana_report.test", "schedule.0.start_time"), // Date set to current time
					resource.TestCheckResourceAttr("grafana_report.test", "schedule.0.end_time", ""),  // No end time
					resource.TestCheckResourceAttr("grafana_report.test", "schedule.0.last_day_of_month", "true"),
					resource.TestCheckResourceAttr("grafana_report.test", "orientation", "landscape"),
					resource.TestCheckResourceAttr("grafana_report.test", "layout", "grid"),
					resource.TestCheckResourceAttr("grafana_report.test", "include_dashboard_link", "true"),
					resource.TestCheckResourceAttr("grafana_report.test", "include_table_csv", "false"),
					resource.TestCheckNoResourceAttr("grafana_report.test", "time_range.0.from"),
					resource.TestCheckNoResourceAttr("grafana_report.test", "time_range.0.to"),
				),
			},
		},
	})
}

// Testing the deprecated case of using a dashboard ID instead of a dashboard UID
// TODO: Remove in next major version
func TestAccResourceReport_CreateFromDashboardID(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t)

	var report models.Report
	var randomUID = acctest.RandStringFromCharSet(10, acctest.CharSetAlpha)

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      reportCheckExists.destroyed(&report, nil),
		Steps: []resource.TestStep{
			{
				Config: testAccReportCreateFromID(randomUID),
				Check: resource.ComposeTestCheckFunc(
					reportCheckExists.exists("grafana_report.test", &report),
					resource.TestCheckResourceAttrSet("grafana_report.test", "dashboard_id"),
					resource.TestCheckResourceAttr("grafana_report.test", "dashboard_uid", randomUID),
				),
			},
		},
	})
}

func TestAccResourceReport_InOrg(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t)

	var report models.Report
	var org models.OrgDetailsDTO
	name := acctest.RandStringFromCharSet(10, acctest.CharSetAlpha)

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      reportCheckExists.destroyed(&report, nil),
		Steps: []resource.TestStep{
			{
				Config: testAccReportCreateInOrg(name),
				Check: resource.ComposeTestCheckFunc(
					reportCheckExists.exists("grafana_report.test", &report),
					resource.TestCheckResourceAttr("grafana_report.test", "dashboard_uid", name),

					// Check that the dashboard is in the correct organization
					resource.TestMatchResourceAttr("grafana_report.test", "id", nonDefaultOrgIDRegexp),
					orgCheckExists.exists("grafana_organization.test", &org),
					checkResourceIsInOrg("grafana_report.test", "grafana_organization.test"),
				),
			},
		},
	})
}

func testAccReportCreateFromID(uid string) string {
	return fmt.Sprintf(`resource "grafana_dashboard" "test" {
	config_json = <<EOD
  {
	"title": "%[1]s",
	"uid": "%[1]s"
  }
  EOD
	message     = "initial commit."
  }
  
  resource "grafana_report" "test" {
	name         = "my report"
	dashboard_id = grafana_dashboard.test.dashboard_id
	recipients   = ["some@email.com"]
	schedule {
	  frequency = "hourly"
	}
  }
  `, uid)
}

func testAccReportCreateInOrg(name string) string {
	return fmt.Sprintf(`
resource "grafana_organization" "test" {
	name = "%s"
}

resource "grafana_dashboard" "test" {
	org_id      = grafana_organization.test.id
	config_json = <<EOD
{
	"title": "%[1]s",
	"uid": "%[1]s"
}
EOD
	message     = "initial commit."
}

resource "grafana_report" "test" {
	org_id      = grafana_organization.test.id
	name         = "my report"
	dashboard_uid = grafana_dashboard.test.uid
	recipients   = ["some@email.com"]
	schedule {
		frequency = "hourly"
	}
}`, name)
}
