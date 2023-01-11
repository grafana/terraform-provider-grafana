package grafana

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/grafana/machine-learning-go-client/mlapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccResourceMachineLearningJob(t *testing.T) {
	CheckCloudInstanceTestsEnabled(t)

	var job mlapi.Job
	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccMLJobCheckDestroy(&job),
		Steps: []resource.TestStep{
			{
				Config: testAccExample(t, "resources/grafana_machine_learning_job/job.tf"),
				Check: resource.ComposeTestCheckFunc(
					testAccMLJobCheckExists("grafana_machine_learning_job.test_job", &job),
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
				Config: testAccExample(t, "resources/grafana_machine_learning_job/datasource_uid_job.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_machine_learning_job.test_job", "id"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "name", "Test Job"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "metric", "tf_test_job"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "datasource_type", "prometheus"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "datasource_uid", "grafanacloud-usage"),
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
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "datasource_uid", "grafanacloud-usage"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "query_params.expr", "grafanacloud_grafana_instance_active_user_count"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "interval", "300"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "training_window", "7776000"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "hyper_params.daily_seasonality", "15"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "hyper_params.weekly_seasonality", "10"),
				),
			},
			{
				Config: testAccExample(t, "resources/grafana_machine_learning_job/holidays_job.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_machine_learning_job.test_job", "id"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "name", "Test Job"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "metric", "tf_test_job"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "datasource_type", "prometheus"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "datasource_id", "10"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "query_params.expr", "grafanacloud_grafana_instance_active_user_count"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "interval", "300"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "training_window", "7776000"),
					resource.TestCheckResourceAttrSet("grafana_machine_learning_job.test_job", "holidays.0"),
				),
			},
		},
	})
}

func testAccMLJobCheckExists(rn string, job *mlapi.Job) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s\n %#v", rn, s.RootModule().Resources)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		client := testAccProvider.Meta().(*client).mlapi
		gotJob, err := client.Job(context.Background(), rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("error getting job: %s", err)
		}

		*job = gotJob

		return nil
	}
}

func testAccMLJobCheckDestroy(job *mlapi.Job) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// This check is to make sure that no pointer conversions are incorrect
		// while mutating job.
		if job.ID == "" {
			return fmt.Errorf("checking deletion of empty id")
		}
		client := testAccProvider.Meta().(*client).mlapi
		_, err := client.Job(context.Background(), job.ID)
		if err == nil {
			return fmt.Errorf("job still exists on server")
		}
		return nil
	}
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

const machineLearningJobMissingDatasourceIDOrUID = `
resource "grafana_machine_learning_job" "invalid" {
  name            = "Test Job"
  metric          = "tf_test_job"
  datasource_type = "prometheus"
  query_params = {
    expr = "grafanacloud_grafana_instance_active_user_count"
  }
}
`

func TestAccResourceInvalidMachineLearningJob(t *testing.T) {
	CheckCloudInstanceTestsEnabled(t)

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      machineLearningJobInvalid,
				ExpectError: regexp.MustCompile(".*datasourceType.*"),
			},
			{
				Config:      machineLearningJobMissingDatasourceIDOrUID,
				ExpectError: regexp.MustCompile(".*datasource_id or datasource_uid.*"),
			},
		},
	})
}
