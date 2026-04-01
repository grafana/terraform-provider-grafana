package k6_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
)

func TestAccDataSourceK6ProjectLimits_basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	projectName := "Terraform Project Test Limits " + acctest.RandString(8)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExampleWithReplace(t, "data-sources/grafana_k6_project_limits/data-source.tf", map[string]string{
					"Terraform Project Test Limits": projectName,
				}),
				Check: resource.ComposeTestCheckFunc(
					// Verify datasource values match what was set by the resource.
					resource.TestCheckResourceAttrPair("data.grafana_k6_project_limits.from_project_id", "vuh_max_per_month", "grafana_k6_project_limits.test_limits", "vuh_max_per_month"),
					resource.TestCheckResourceAttrPair("data.grafana_k6_project_limits.from_project_id", "vu_max_per_test", "grafana_k6_project_limits.test_limits", "vu_max_per_test"),
					resource.TestCheckResourceAttrPair("data.grafana_k6_project_limits.from_project_id", "vu_browser_max_per_test", "grafana_k6_project_limits.test_limits", "vu_browser_max_per_test"),
					resource.TestCheckResourceAttrPair("data.grafana_k6_project_limits.from_project_id", "duration_max_per_test", "grafana_k6_project_limits.test_limits", "duration_max_per_test"),
				),
			},
		},
	})
}
