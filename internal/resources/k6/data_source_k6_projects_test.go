package k6_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/grafana/k6-cloud-openapi-client-go/k6"

	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
)

func TestAccDataSourceK6Projects_basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	var (
		project  k6.ProjectApiModel
		project2 k6.ProjectApiModel
	)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "data-sources/grafana_k6_projects/data-source.tf"),
				Check: resource.ComposeTestCheckFunc(
					projectCheckExists.exists("grafana_k6_project.project", &project),
					projectCheckExists.exists("grafana_k6_project.project_2", &project2),
					// from_name.0
					resource.TestCheckResourceAttrSet("data.grafana_k6_projects.from_name", "projects.0.id"),
					resource.TestCheckResourceAttr("data.grafana_k6_projects.from_name", "projects.0.name", "Terraform Test Project"),
					resource.TestCheckResourceAttrSet("data.grafana_k6_projects.from_name", "projects.0.is_default"),
					resource.TestCheckResourceAttrSet("data.grafana_k6_projects.from_name", "projects.0.created"),
					resource.TestCheckResourceAttrSet("data.grafana_k6_projects.from_name", "projects.0.updated"),
					// from_name.1
					resource.TestCheckResourceAttrSet("data.grafana_k6_projects.from_name", "projects.1.id"),
					resource.TestCheckResourceAttr("data.grafana_k6_projects.from_name", "projects.1.name", "Terraform Test Project"),
					resource.TestCheckResourceAttrSet("data.grafana_k6_projects.from_name", "projects.1.is_default"),
					resource.TestCheckResourceAttrSet("data.grafana_k6_projects.from_name", "projects.1.created"),
					resource.TestCheckResourceAttrSet("data.grafana_k6_projects.from_name", "projects.1.updated"),
				),
			},
		},
	})
}
