package k6_test

import (
	"fmt"
	"github.com/grafana/k6-cloud-openapi-client-go/k6"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"strconv"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
)

func TestAccDataSourceK6LoadTests_basic(t *testing.T) {
	//testutils.CheckCloudInstanceTestsEnabled(t)

	var project k6.ProjectApiModel

	checkProjectIdMatch := func(value string) error {
		if value != strconv.Itoa(int(project.GetId())) {
			return fmt.Errorf("project_id does not match the expected value: %s", value)
		}
		return nil
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "data-sources/grafana_k6_load_tests/data-source.tf"),
				Check: resource.ComposeTestCheckFunc(
					projectCheckExists.exists("grafana_k6_project.load_test_project", &project),
					// 0
					resource.TestCheckResourceAttrSet("data.grafana_k6_load_tests.from_project_id", "load_tests.0.id"),
					resource.TestCheckResourceAttr("data.grafana_k6_load_tests.from_project_id", "load_tests.0.name", "Terraform Test Load Test"),
					resource.TestCheckResourceAttrWith("data.grafana_k6_load_tests.from_project_id", "load_tests.0.project_id", checkProjectIdMatch),
					resource.TestCheckResourceAttrSet("data.grafana_k6_load_tests.from_project_id", "load_tests.0.script"),
					resource.TestCheckResourceAttrSet("data.grafana_k6_load_tests.from_project_id", "load_tests.0.created"),
					resource.TestCheckResourceAttrSet("data.grafana_k6_load_tests.from_project_id", "load_tests.0.updated"),
					// 1
					resource.TestCheckResourceAttrSet("data.grafana_k6_load_tests.from_project_id", "load_tests.1.id"),
					resource.TestCheckResourceAttr("data.grafana_k6_load_tests.from_project_id", "load_tests.1.name", "Terraform Test Load Test (2)"),
					resource.TestCheckResourceAttrWith("data.grafana_k6_load_tests.from_project_id", "load_tests.1.project_id", checkProjectIdMatch),
					resource.TestCheckResourceAttrSet("data.grafana_k6_load_tests.from_project_id", "load_tests.1.script"),
					resource.TestCheckResourceAttrSet("data.grafana_k6_load_tests.from_project_id", "load_tests.1.created"),
					resource.TestCheckResourceAttrSet("data.grafana_k6_load_tests.from_project_id", "load_tests.1.updated"),
				),
			},
		},
	})
}
