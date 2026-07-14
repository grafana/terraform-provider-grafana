package k6_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
)

func TestAccDataSourceK6LoadTest_basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	projectName := "Terraform Load Test Project " + acctest.RandString(8)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExampleWithReplace(t, "data-sources/grafana_k6_load_test/data-source.tf", map[string]string{
					"Terraform Load Test Project": projectName,
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.grafana_k6_load_test.from_id", "id"),
					resource.TestCheckResourceAttr("data.grafana_k6_load_test.from_id", "name", "Terraform Test Load Test"),
					resource.TestCheckResourceAttrSet("data.grafana_k6_load_test.from_id", "script"),
					resource.TestCheckResourceAttrSet("data.grafana_k6_load_test.from_id", "created"),
					resource.TestCheckResourceAttrSet("data.grafana_k6_load_test.from_id", "updated"),
				),
			},
		},
	})
}
