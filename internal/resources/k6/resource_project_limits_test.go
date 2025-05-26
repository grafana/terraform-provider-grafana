package k6_test

import (
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/grafana/k6-cloud-openapi-client-go/k6"

	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
)

func TestAccProjectLimits_basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	var project k6.ProjectApiModel
	var projectLimits k6.ProjectLimitsApiModel

	projectName := "Terraform Project Test Limits " + acctest.RandString(8)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			projectCheckExists.destroyed(&project),
			projectLimitsCheckExists.destroyed(&projectLimits),
		),
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_k6_project_limits/resource.tf", map[string]string{
					"Terraform Project Test Limits": projectName,
				}),
				Check: resource.ComposeTestCheckFunc(
					projectCheckExists.exists("grafana_k6_project.test_project_limits", &project),
					projectLimitsCheckExists.exists("grafana_k6_project_limits.test_limits", &projectLimits),
					resource.TestCheckResourceAttr("grafana_k6_project_limits.test_limits", "vuh_max_per_month", "1000"),
					resource.TestCheckResourceAttr("grafana_k6_project_limits.test_limits", "vu_max_per_test", "100"),
					resource.TestCheckResourceAttr("grafana_k6_project_limits.test_limits", "vu_browser_max_per_test", "10"),
					resource.TestCheckResourceAttr("grafana_k6_project_limits.test_limits", "duration_max_per_test", "3600"),
				),
			},
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_k6_project_limits/resource.tf", map[string]string{
					"Terraform Project Test Limits":  projectName,
					"vuh_max_per_month       = 1000": "vuh_max_per_month       = 2000",
				}),
				Check: resource.ComposeTestCheckFunc(
					projectCheckExists.exists("grafana_k6_project.test_project_limits", &project),
					projectLimitsCheckExists.exists("grafana_k6_project_limits.test_limits", &projectLimits),
					resource.TestCheckResourceAttr("grafana_k6_project_limits.test_limits", "vuh_max_per_month", "2000"),
					resource.TestCheckResourceAttr("grafana_k6_project_limits.test_limits", "vu_max_per_test", "100"),
					resource.TestCheckResourceAttr("grafana_k6_project_limits.test_limits", "vu_browser_max_per_test", "10"),
					resource.TestCheckResourceAttr("grafana_k6_project_limits.test_limits", "duration_max_per_test", "3600"),
				),
			},
			{
				ResourceName:      "grafana_k6_project_limits.test_limits",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return strconv.Itoa(int(project.GetId())), nil
				},
			},
		},
	})
}
