package grafana

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceSyntheticMonitoringProbe(t *testing.T) {
	CheckCloudTestsEnabled(t)

	resource.UnitTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckCloud(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccExample(t, "data-sources/grafana_synthetic_monitoring_probe/data-source.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.grafana_synthetic_monitoring_probe.atlanta", "name", "Atlanta"),
				),
			},
		},
	})
}
