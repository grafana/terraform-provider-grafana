package grafana

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceSyntheticMonitoringProbes(t *testing.T) {
	CheckCloudInstanceTestsEnabled(t)

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccExample(t, "data-sources/grafana_synthetic_monitoring_probes/data-source.tf"),
				Check:  resource.TestCheckResourceAttr("data.grafana_synthetic_monitoring_probes.main", "probes.Atlanta", "1"),
			},
			{
				Config: testAccExample(t, "data-sources/grafana_synthetic_monitoring_probes/with-deprecated.tf"),
				// We're not checking for deprecated probes here because there may not be any, causing tests to fail.
				Check: resource.TestCheckResourceAttr("data.grafana_synthetic_monitoring_probes.main", "probes.Atlanta", "1"),
			},
			// Test with a custom probe
			{
				Config: testProbeDatasourceWithCustom,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.grafana_synthetic_monitoring_probes.all", "probes.Everest"),
					resource.TestCheckResourceAttr("data.grafana_synthetic_monitoring_probe.one", "labels.type", "mountain"),
				),
			},
		},
	})
}

const testProbeDatasourceWithCustom = `
resource "grafana_synthetic_monitoring_probe" "main" {
	name      = "Everest"
	latitude  = 27.98606
	longitude = 86.92262
	region    = "APAC"
	labels = {
	  type = "mountain"
	}
  }

data "grafana_synthetic_monitoring_probes" "all" {
	depends_on = [grafana_synthetic_monitoring_probe.main]
}

data "grafana_synthetic_monitoring_probe" "one" {
	name = grafana_synthetic_monitoring_probe.main.name
}
  `
