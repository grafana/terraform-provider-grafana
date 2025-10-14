package syntheticmonitoring_test

import (
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceProbes(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	// TODO: Make parallelizable
	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "data-sources/grafana_synthetic_monitoring_probes/data-source.tf"),
				Check:  resource.TestCheckResourceAttrSet("data.grafana_synthetic_monitoring_probes.main", "probes.Ohio"),
			},
			{
				Config: testutils.TestAccExample(t, "data-sources/grafana_synthetic_monitoring_probes/with-deprecated.tf"),
				// We're not checking for deprecated probes here because there may not be any, causing tests to fail.
				Check: resource.TestCheckResourceAttrSet("data.grafana_synthetic_monitoring_probes.main", "probes.Ohio"),
			},
		},
	})
}
