package k6_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
)

func TestAccDataSourceK6Project_basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "data-sources/grafana_k6_project/data-source.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.grafana_k6_project.from_id", "name", "Terraform Test Project"),
					resource.TestCheckResourceAttrSet("data.grafana_k6_project.from_id", "grafana_folder_uid"),
					resource.TestCheckResourceAttrSet("data.grafana_k6_project.from_id", "created"),
					resource.TestCheckResourceAttrSet("data.grafana_k6_project.from_id", "updated"),
				),
			},
		},
	})
}
