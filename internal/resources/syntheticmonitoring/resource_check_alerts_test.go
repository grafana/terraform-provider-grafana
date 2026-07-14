package syntheticmonitoring_test

import (
	"regexp"
	"strconv"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
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
					resource.TestCheckTypeSetElemNestedAttrs("grafana_synthetic_monitoring_check_alerts.main", "alerts.*", map[string]string{
						"name":        "ProbeFailedExecutionsTooHigh",
						"threshold":   "1",
						"period":      "15m",
						"runbook_url": "",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("grafana_synthetic_monitoring_check_alerts.main", "alerts.*", map[string]string{
						"name":        "TLSTargetCertificateCloseToExpiring",
						"threshold":   "14",
						"period":      "",
						"runbook_url": "",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("grafana_synthetic_monitoring_check_alerts.main", "alerts.*", map[string]string{
						"name":        "HTTPRequestDurationTooHighAvg",
						"threshold":   "5000",
						"period":      "10m",
						"runbook_url": "https://wiki.company.com/runbooks/http-duration",
					}),
				),
			},
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_synthetic_monitoring_check_alerts/resource_update.tf", map[string]string{
					`"Check Alert Test Updated"`: strconv.Quote(jobName),
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check_alerts.main", "id"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_check_alerts.main", "check_id"),
					resource.TestCheckTypeSetElemNestedAttrs("grafana_synthetic_monitoring_check_alerts.main", "alerts.*", map[string]string{
						"name":        "ProbeFailedExecutionsTooHigh",
						"threshold":   "2",
						"period":      "10m",
						"runbook_url": "",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("grafana_synthetic_monitoring_check_alerts.main", "alerts.*", map[string]string{
						"name":        "TLSTargetCertificateCloseToExpiring",
						"threshold":   "7",
						"period":      "",
						"runbook_url": "",
					}),
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
				ExpectError: regexp.MustCompile(`expected alerts\.0\.name to be one of \["ProbeFailedExecutionsTooHigh" "TLSTargetCertificateCloseToExpiring" "HTTPRequestDurationTooHighAvg" "PingRequestDurationTooHighAvg" "DNSRequestDurationTooHighAvg"\], got InvalidAlertName`),
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
		runbook_url = ""
	}]
}`
