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
					// Don't assert exact numbers; they vary by org/plan and can change over time.
					resource.TestCheckResourceAttrSet("data.grafana_k6_project_limits.from_project_id", "vuh_max_per_month"),
					resource.TestCheckResourceAttrSet("data.grafana_k6_project_limits.from_project_id", "vu_max_per_test"),
					resource.TestCheckResourceAttrSet("data.grafana_k6_project_limits.from_project_id", "vu_browser_max_per_test"),
					resource.TestCheckResourceAttrSet("data.grafana_k6_project_limits.from_project_id", "duration_max_per_test"),
				),
			},
		},
	})
}
