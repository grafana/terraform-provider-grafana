package k6_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
)

func TestAccDataSourceK6ProjectLimits_basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "data-sources/grafana_k6_project_limits/data-source.tf"),
				Check: resource.ComposeTestCheckFunc(
					// project_id
					resource.TestCheckResourceAttr("data.grafana_k6_project_limits.from_project_id", "vuh_max_per_month", "10000"),
					resource.TestCheckResourceAttr("data.grafana_k6_project_limits.from_project_id", "vu_max_per_test", "10000"),
					resource.TestCheckResourceAttr("data.grafana_k6_project_limits.from_project_id", "vu_browser_max_per_test", "1000"),
					resource.TestCheckResourceAttr("data.grafana_k6_project_limits.from_project_id", "duration_max_per_test", "3600"),
				),
			},
		},
	})
}
