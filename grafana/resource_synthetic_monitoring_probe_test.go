//go:build cloud
// +build cloud

package grafana

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceSyntheticMonitoringProbe(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckCloud(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccExample(t, "resources/grafana_synthetic_monitoring_probe/resource.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_probe.main", "id"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_probe.main", "auth_token"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_probe.main", "name", "Mount Everest"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_probe.main", "latitude", "27.986059188842773"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_probe.main", "longitude", "86.92262268066406"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_probe.main", "region", "APAC"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_probe.main", "public", "false"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_probe.main", "labels.type", "mountain"),
				),
			},
			{
				Config: testAccExample(t, "resources/grafana_synthetic_monitoring_probe/resource_update.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_probe.main", "id"),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_probe.main", "auth_token"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_probe.main", "name", "Mauna Loa"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_probe.main", "latitude", "19.479480743408203"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_probe.main", "longitude", "-155.60281372070312"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_probe.main", "region", "AMER"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_probe.main", "public", "false"),
					resource.TestCheckResourceAttr("grafana_synthetic_monitoring_probe.main", "labels.type", "volcano"),
				),
			},
		},
	})
}
