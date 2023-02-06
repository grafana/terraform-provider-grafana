package syntheticmonitoring

import (
	"testing"

	"github.com/grafana/terraform-provider-grafana/provider/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceSyntheticMonitoringProbes(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	// TODO: Make parallelizable
	resource.Test(t, resource.TestCase{
		ProviderFactories: testutils.GetProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "data-sources/grafana_synthetic_monitoring_probes/data-source.tf"),
				Check:  resource.TestCheckResourceAttr("data.grafana_synthetic_monitoring_probes.main", "probes.Atlanta", "1"),
			},
			{
				Config: testutils.TestAccExample(t, "data-sources/grafana_synthetic_monitoring_probes/with-deprecated.tf"),
				// We're not checking for deprecated probes here because there may not be any, causing tests to fail.
				Check: resource.TestCheckResourceAttr("data.grafana_synthetic_monitoring_probes.main", "probes.Atlanta", "1"),
			},
		},
	})
}
