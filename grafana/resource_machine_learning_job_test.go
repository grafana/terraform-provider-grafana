package grafana

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceMachineLearningJob(t *testing.T) {
	CheckCloudTestsEnabled(t)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckCloud(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccExample(t, "resources/grafana_machine_learning_job/job.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_machine_learning_job.test_job", "id"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "name", "Test Job"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "metric", "tf_test_job"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "datasource_type", "prometheus"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "datasource_id", "10"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "query_params.expr", "grafanacloud_grafana_instance_active_user_count"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "interval", "300"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "training_window", "7776000"),
				),
			},
			{
				Config: testAccExample(t, "resources/grafana_machine_learning_job/tuned_job.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_machine_learning_job.test_job", "id"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "name", "Test Job"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "metric", "tf_test_job"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "datasource_type", "prometheus"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "datasource_id", "10"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "query_params.expr", "grafanacloud_grafana_instance_active_user_count"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "interval", "300"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "training_window", "7776000"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "hyper_params.daily_seasonality", "15"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "hyper_params.weekly_seasonality", "10"),
				),
			},
		},
	})
}

const machineLearningJobInvalid = `
resource "grafana_machine_learning_job" "invalid" {
  name            = "Test Job"
  metric          = "tf_test_job"
  datasource_type = "fake"
  datasource_id   = 10
  query_params = {
    expr = "grafanacloud_grafana_instance_active_user_count"
  }
}
`

func TestAccResourceInvalidMachineLearningJob(t *testing.T) {
	CheckCloudTestsEnabled(t)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckCloud(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      machineLearningJobInvalid,
				ExpectError: regexp.MustCompile(".*datasourceType.*"),
			},
		},
	})
}
