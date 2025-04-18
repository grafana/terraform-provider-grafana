package k6_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
)

func TestAccDataSourceK6LoadTest_basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "data-sources/grafana_k6_load_test/data-source.tf"),
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
