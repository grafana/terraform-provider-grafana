package k6_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/grafana/k6-cloud-openapi-client-go/k6"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
)

func TestAccDataSourceK6Projects_basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	var project k6.ProjectApiModel

	projectName := "Terraform Test Project " + acctest.RandString(8)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExampleWithReplace(t, "data-sources/grafana_k6_projects/data-source.tf", map[string]string{
					"Terraform Test Project": projectName,
				}),
				Check: resource.ComposeTestCheckFunc(
					projectCheckExists.exists("grafana_k6_project.project", &project),
					resource.TestCheckResourceAttrSet("data.grafana_k6_projects.from_name", "projects.0.id"),
					resource.TestCheckResourceAttr("data.grafana_k6_projects.from_name", "projects.0.name", projectName),
					resource.TestCheckResourceAttrSet("data.grafana_k6_projects.from_name", "projects.0.is_default"),
					resource.TestCheckResourceAttrSet("data.grafana_k6_projects.from_name", "projects.0.grafana_folder_uid"),
					resource.TestCheckResourceAttrSet("data.grafana_k6_projects.from_name", "projects.0.created"),
					resource.TestCheckResourceAttrSet("data.grafana_k6_projects.from_name", "projects.0.updated"),
				),
			},
		},
	})
}
