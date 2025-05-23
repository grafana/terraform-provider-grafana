package k6_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
)

func TestAccDataSourceK6Project_basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	projectName := "Terraform Test Project " + acctest.RandString(8)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExampleWithReplace(t, "data-sources/grafana_k6_project/data-source.tf", map[string]string{
					"Terraform Test Project": projectName,
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.grafana_k6_project.from_id", "id"),
					resource.TestCheckResourceAttr("data.grafana_k6_project.from_id", "name", "Terraform Test Project"),
					resource.TestCheckResourceAttrSet("data.grafana_k6_project.from_id", "is_default"),
					resource.TestCheckResourceAttrSet("data.grafana_k6_project.from_id", "grafana_folder_uid"),
					resource.TestCheckResourceAttrSet("data.grafana_k6_project.from_id", "created"),
					resource.TestCheckResourceAttrSet("data.grafana_k6_project.from_id", "updated"),
				),
			},
		},
	})
}
