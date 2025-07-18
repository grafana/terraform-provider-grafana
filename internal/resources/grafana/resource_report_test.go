package grafana_test

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v3/internal/resources/grafana"
	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestCheckTimezoneFormatDate(t *testing.T) {
	tests := []struct {
		name         string
		date         string
		timezone     string
		shouldError  bool
		expectedTime string // Expected time in the target timezone
	}{
		{
			name:         "UTC to America/New_York",
			date:         "2024-01-15T15:00:00Z",
			timezone:     "America/New_York",
			shouldError:  false,
			expectedTime: "2024-01-15T10:00:00-05:00", // EST offset
		},
		{
			name:         "UTC to UTC",
			date:         "2024-01-15T15:00:00Z",
			timezone:     "UTC",
			shouldError:  false,
			expectedTime: "2024-01-15T15:00:00Z",
		},
		{
			name:         "America/New_York to UTC",
			date:         "2024-01-15T10:00:00-05:00",
			timezone:     "UTC",
			shouldError:  false,
			expectedTime: "2024-01-15T15:00:00Z",
		},
		{
			name:         "America/New_York to Europe/London",
			date:         "2024-01-15T10:00:00-05:00",
			timezone:     "Europe/London",
			shouldError:  false,
			expectedTime: "2024-01-15T15:00:00Z", // London is UTC in January
		},
		{
			name:        "Invalid RFC3339 date",
			date:        "invalid-date",
			timezone:    "UTC",
			shouldError: true,
		},
		{
			name:        "Invalid timezone",
			date:        "2024-01-15T15:00:00Z",
			timezone:    "Invalid/Timezone",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Load the target timezone
			tz, err := time.LoadLocation(tt.timezone)
			if err != nil && !tt.shouldError {
				t.Fatalf("Failed to load timezone %s: %v", tt.timezone, err)
			}
			if err != nil && tt.shouldError {
				return // Expected error for invalid timezone
			}

			// Call the exported function for testing
			result, err := grafana.CheckTimezoneFormatDate(tt.date, tz)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Errorf("Expected result but got nil")
				return
			}

			// Parse the expected time for comparison
			expectedTime, err := time.Parse(time.RFC3339, tt.expectedTime)
			if err != nil {
				t.Fatalf("Failed to parse expected time: %v", err)
			}

			// Convert result back to time for comparison
			resultTime := time.Time(*result)

			// Compare times (they should represent the same instant)
			if !resultTime.Equal(expectedTime) {
				t.Errorf("Expected %s, got %s", expectedTime.Format(time.RFC3339), resultTime.Format(time.RFC3339))
			}
		})
	}
}

func TestAccResourceReport_Multiple_Dashboards(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=9.0.0")

	var report models.Report
	var randomUID = acctest.RandStringFromCharSet(10, acctest.CharSetAlpha)
	var randomUID2 = acctest.RandStringFromCharSet(10, acctest.CharSetAlpha)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             reportCheckExists.destroyed(&report, nil),
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_report/multiple-dashboards.tf", map[string]string{
					`"report-dashboard"`:   fmt.Sprintf(`"%s"`, randomUID),
					`"report-dashboard-2"`: fmt.Sprintf(`"%s"`, randomUID2),
				}),
				Check: resource.ComposeTestCheckFunc(
					reportCheckExists.exists("grafana_report.test", &report),
					resource.TestCheckResourceAttrSet("grafana_report.test", "id"),
					resource.TestCheckResourceAttr("grafana_report.test", "time_range.#", "0"),
					resource.TestCheckResourceAttr("grafana_report.test", "name", "multiple dashboards"),
					resource.TestCheckResourceAttr("grafana_report.test", "recipients.0", "some@email.com"),
					resource.TestCheckNoResourceAttr("grafana_report.test", "recipients.1"),
					resource.TestCheckResourceAttr("grafana_report.test", "schedule.0.frequency", "monthly"),
					resource.TestCheckResourceAttr("grafana_report.test", "schedule.0.start_time", "2024-02-10T15:00:00-05:00"),
					resource.TestCheckResourceAttr("grafana_report.test", "schedule.0.end_time", "2024-02-15T10:00:00-05:00"),
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
					resource.TestCheckResourceAttr("grafana_report.test", "dashboards.0.report_variables.query0", "a,b"),
					resource.TestCheckResourceAttr("grafana_report.test", "dashboards.0.report_variables.query1", "c,d"),
					resource.TestCheckResourceAttr("grafana_report.test", "dashboards.1.time_range.0.from", ""),
					resource.TestCheckResourceAttr("grafana_report.test", "dashboards.1.time_range.0.to", ""),
					resource.TestCheckResourceAttr("grafana_report.test", "dashboards.1.uid", randomUID2),
					testutils.CheckLister("grafana_report.test"),
				),
			},
		},
	})
}

func TestAccResourceReport_basic(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=9.0.0")

	var report models.Report
	var randomUID = acctest.RandStringFromCharSet(10, acctest.CharSetAlpha)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             reportCheckExists.destroyed(&report, nil),
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_report/resource.tf", map[string]string{
					`"report-dashboard"`: fmt.Sprintf(`"%s"`, randomUID),
				}),
				Check: resource.ComposeTestCheckFunc(
					reportCheckExists.exists("grafana_report.test", &report),
					resource.TestCheckResourceAttrSet("grafana_report.test", "id"),
					resource.TestCheckResourceAttr("grafana_report.test", "org_id", "1"),
					resource.TestCheckResourceAttr("grafana_report.test", "name", "my report"),
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
					resource.TestCheckResourceAttr("grafana_report.test", "dashboards.0.uid", randomUID),
				),
			},
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_report/all-options.tf", map[string]string{
					`"report-dashboard"`: fmt.Sprintf(`"%s"`, randomUID),
				}),
				Check: resource.ComposeTestCheckFunc(
					func(s *terraform.State) error {
						// Check that the ID is the same as the first run
						// This is a custom function to delay the report ID evaluation, because it is generated after the first run
						return resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr("grafana_report.test", "id", "1:"+strconv.FormatInt(report.ID, 10)), // <orgid>:<reportid> (1 being the default org)
						)(s)
					},
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
					resource.TestCheckResourceAttr("grafana_report.test", "formats.#", "3"),
					resource.TestCheckResourceAttr("grafana_report.test", "formats.0", "csv"),
					resource.TestCheckResourceAttr("grafana_report.test", "formats.1", "image"),
					resource.TestCheckResourceAttr("grafana_report.test", "formats.2", "pdf"),
					resource.TestCheckResourceAttr("grafana_report.test", "dashboards.0.uid", randomUID),
					resource.TestCheckResourceAttr("grafana_report.test", "dashboards.0.time_range.0.from", "now-1h"),
					resource.TestCheckResourceAttr("grafana_report.test", "dashboards.0.time_range.0.to", "now"),
				),
			},
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_report/monthly.tf", map[string]string{
					`"report-dashboard"`: fmt.Sprintf(`"%s"`, randomUID),
				}),
				Check: resource.ComposeTestCheckFunc(
					reportCheckExists.exists("grafana_report.test", &report),
					resource.TestCheckResourceAttrSet("grafana_report.test", "id"),
					resource.TestCheckResourceAttr("grafana_report.test", "name", "my report"),
					resource.TestCheckResourceAttr("grafana_report.test", "recipients.0", "some@email.com"),
					resource.TestCheckNoResourceAttr("grafana_report.test", "recipients.1"),
					resource.TestCheckResourceAttr("grafana_report.test", "schedule.0.frequency", "monthly"),
					resource.TestCheckResourceAttrSet("grafana_report.test", "schedule.0.start_time"), // Date set to current time
					resource.TestCheckResourceAttr("grafana_report.test", "schedule.0.end_time", ""),  // No end time
					resource.TestCheckResourceAttr("grafana_report.test", "schedule.0.timezone", "GMT"),
					resource.TestCheckResourceAttr("grafana_report.test", "schedule.0.last_day_of_month", "true"),
					resource.TestCheckResourceAttr("grafana_report.test", "orientation", "landscape"),
					resource.TestCheckResourceAttr("grafana_report.test", "layout", "grid"),
					resource.TestCheckResourceAttr("grafana_report.test", "include_dashboard_link", "true"),
					resource.TestCheckResourceAttr("grafana_report.test", "include_table_csv", "false"),
					resource.TestCheckResourceAttr("grafana_report.test", "dashboards.0.uid", randomUID),
				),
			},
		},
	})
}

func TestAccResourceReport_InOrg(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=9.0.0")

	var report models.Report
	var org models.OrgDetailsDTO
	name := acctest.RandStringFromCharSet(10, acctest.CharSetAlpha)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             reportCheckExists.destroyed(&report, nil),
		Steps: []resource.TestStep{
			{
				Config: testAccReportCreateInOrg(name),
				Check: resource.ComposeTestCheckFunc(
					reportCheckExists.exists("grafana_report.test", &report),
					resource.TestCheckResourceAttr("grafana_report.test", "dashboards.0.uid", name),

					// Check that the dashboard is in the correct organization
					resource.TestMatchResourceAttr("grafana_report.test", "id", nonDefaultOrgIDRegexp),
					orgCheckExists.exists("grafana_organization.test", &org),
					checkResourceIsInOrg("grafana_report.test", "grafana_organization.test"),
				),
			},
		},
	})
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
	recipients   = ["some@email.com"]
	schedule {
		frequency = "hourly"
	}
	dashboards {
		uid = grafana_dashboard.test.uid
	}
}`, name)
}

func TestAccResourceReport_DashboardUIDChange_WithTimezone(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=9.0.0")

	var report models.Report
	var randomUID1 = acctest.RandStringFromCharSet(10, acctest.CharSetAlpha)
	var randomUID2 = acctest.RandStringFromCharSet(10, acctest.CharSetAlpha)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             reportCheckExists.destroyed(&report, nil),
		Steps: []resource.TestStep{
			{
				// Create report with non-GMT timezone and explicit start/end times
				Config: testAccReportWithTimezoneStep1(randomUID1, randomUID2),
				Check: resource.ComposeTestCheckFunc(
					reportCheckExists.exists("grafana_report.test", &report),
					resource.TestCheckResourceAttr("grafana_report.test", "name", "timezone test report"),
					resource.TestCheckResourceAttr("grafana_report.test", "schedule.0.timezone", "America/New_York"),
					resource.TestCheckResourceAttr("grafana_report.test", "dashboards.0.uid", randomUID1),
				),
			},
			{
				// Update dashboard UID - this was triggering the timezone error before the fix
				Config: testAccReportWithTimezoneStep2(randomUID1, randomUID2),
				Check: resource.ComposeTestCheckFunc(
					reportCheckExists.exists("grafana_report.test", &report),
					resource.TestCheckResourceAttr("grafana_report.test", "name", "timezone test report"),
					resource.TestCheckResourceAttr("grafana_report.test", "schedule.0.timezone", "America/New_York"),
					// Dashboard UID should be updated
					resource.TestCheckResourceAttr("grafana_report.test", "dashboards.0.uid", randomUID2),
				),
			},
		},
	})
}

func testAccReportWithTimezoneStep1(dashboardUID1, dashboardUID2 string) string {
	return fmt.Sprintf(`
resource "grafana_dashboard" "test1" {
	config_json = <<EOD
{
	"title": "Test Dashboard %[1]s",
	"uid": "%[1]s"
}
EOD
}

resource "grafana_dashboard" "test2" {
	config_json = <<EOD
{
	"title": "Test Dashboard %[2]s",
	"uid": "%[2]s"
}
EOD
}

resource "grafana_report" "test" {
	name         = "timezone test report"
	recipients   = ["test@example.com"]
	schedule {
		frequency  = "monthly"
		start_time = "2024-02-10T15:00:00"  # Short format, no timezone
		end_time   = "2024-02-15T10:00:00"  # Short format, no timezone  
		timezone   = "America/New_York"     # Non-GMT timezone
	}
	dashboards {
		uid = grafana_dashboard.test1.uid
	}
}`, dashboardUID1, dashboardUID2)
}

func testAccReportWithTimezoneStep2(dashboardUID1, dashboardUID2 string) string {
	return fmt.Sprintf(`
resource "grafana_dashboard" "test1" {
	config_json = <<EOD
{
	"title": "Test Dashboard %[1]s",
	"uid": "%[1]s"
}
EOD
}

resource "grafana_dashboard" "test2" {
	config_json = <<EOD
{
	"title": "Test Dashboard %[2]s",
	"uid": "%[2]s"
}
EOD
}

resource "grafana_report" "test" {
	name         = "timezone test report"
	recipients   = ["test@example.com"]
	schedule {
		frequency  = "monthly"
		start_time = "2024-02-10T15:00:00"  # Short format, no timezone
		end_time   = "2024-02-15T10:00:00"  # Short format, no timezone  
		timezone   = "America/New_York"     # Non-GMT timezone
	}
	dashboards {
		uid = grafana_dashboard.test2.uid
	}
}`, dashboardUID1, dashboardUID2)
}
