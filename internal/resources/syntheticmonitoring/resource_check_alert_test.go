package syntheticmonitoring_test

import (
	"regexp"
	"strconv"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceCheckAlerts(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	// Create a random job name to avoid conflicts
	jobName := acctest.RandomWithPrefix("check-alert")

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_synthetic_monitoring_check_alerts/resource.tf", map[string]string{
					`"Check Alert Test"`: strconv.Quote(jobName),
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check_alerts.main", "id"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check_alerts.main", "check_id"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check_alerts.main", "alerts.0.name", "ProbeFailedExecutionsTooHigh"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check_alerts.main", "alerts.0.threshold", "1"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check_alerts.main", "alerts.0.period", "15m"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check_alerts.main", "alerts.1.name", "TLSTargetCertificateCloseToExpiring"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check_alerts.main", "alerts.1.threshold", "14"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check_alerts.main", "alerts.1.period", ""),
				),
			},
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_synthetic_monitoring_check_alerts/resource_update.tf", map[string]string{
					`"Check Alert Test Updated"`: strconv.Quote(jobName),
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check_alerts.main", "id"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check_alerts.main", "check_id"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check_alerts.main", "alerts.0.name", "ProbeFailedExecutionsTooHigh"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check_alerts.main", "alerts.0.threshold", "2"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check_alerts.main", "alerts.0.period", "10m"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check_alerts.main", "alerts.1.name", "TLSTargetCertificateCloseToExpiring"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check_alerts.main", "alerts.1.threshold", "7"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_check_alerts.main", "alerts.1.period", ""),
				),
			},
		},
	})
}

func TestAccResourceCheckAlert_InvalidAlertName(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccResourceCheckAlert_InvalidAlertName,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`expected alerts\.0\.name to be one of \["ProbeFailedExecutionsTooHigh" "TLSTargetCertificateCloseToExpiring"\], got InvalidAlertName`),
			},
		},
	})
}

func TestAccResourceCheckAlert_Import(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_synthetic_monitoring_check_alerts/resource.tf", map[string]string{
					`"Check Alert Test"`: strconv.Quote(acctest.RandomWithPrefix("check-alert")),
				}),
			},
			{
				ResourceName:      "grafana_synthetic_monitoring_check_alerts.main",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

const testAccResourceCheckAlert_InvalidAlertName = `
resource "grafana_synthetic_monitoring_check_alerts" "main" {
	check_id = 1
	alerts = [{
		name = "InvalidAlertName"
		threshold = 0.5
		period = ""
	}]
}`
