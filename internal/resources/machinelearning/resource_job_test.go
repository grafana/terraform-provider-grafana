package machinelearning_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/grafana/machine-learning-go-client/mlapi"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccResourceJob(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	t.Skip("skipping test because it errors with addDataSourceConflict {'message':'data source with the same name already exists'}'}")

	randomName := acctest.RandomWithPrefix("Test Job")

	var job mlapi.Job
	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccMLJobCheckDestroy(&job),
			testAccDatasourceCheckDestroy(),
		),
		Steps: []resource.TestStep{
			{
				// Note for the reader: these tests construct a datasource & a job, where the Job's datasource id is set by terraform.
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_machine_learning_job/job.tf", map[string]string{
					"Test Job": randomName,
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccMLJobCheckExists("grafana_machine_learning_job.test_job", &job),
					resource.TestCheckResourceAttrSet("grafana_machine_learning_job.test_job", "id"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "name", randomName),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "metric", "tf_test_job"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "datasource_type", "prometheus"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "datasource_uid", "prometheus-ds-test-uid"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "query_params.expr", "grafanacloud_grafana_instance_active_user_count"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "interval", "300"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "training_window", "7776000"),
					testutils.CheckLister("grafana_machine_learning_job.test_job"),
				),
			},
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_machine_learning_job/tuned_job.tf", map[string]string{
					"Test Job": randomName,
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_machine_learning_job.test_job", "id"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "name", randomName),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "metric", "tf_test_job"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "datasource_type", "prometheus"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "datasource_uid", "prometheus-ds-test-uid"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "query_params.expr", "grafanacloud_grafana_instance_active_user_count"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "interval", "300"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "training_window", "7776000"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "hyper_params.daily_seasonality", "15"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "hyper_params.weekly_seasonality", "10"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "custom_labels.example_label", "example_value"),
				),
			},
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_machine_learning_job/holidays_job.tf", map[string]string{
					"Test Job": randomName,
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_machine_learning_job.test_job", "id"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "name", randomName),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "metric", "tf_test_job"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "datasource_type", "prometheus"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "datasource_uid", "prometheus-ds-test-uid"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "query_params.expr", "grafanacloud_grafana_instance_active_user_count"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "interval", "300"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "training_window", "7776000"),
					resource.TestCheckResourceAttrSet("grafana_machine_learning_job.test_job", "holidays.0"),
				),
			},
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_machine_learning_job/transformed_job.tf", map[string]string{
					"Test Job": randomName,
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_machine_learning_job.test_job", "id"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "name", randomName),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "metric", "tf_test_job"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "datasource_type", "prometheus"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "datasource_uid", "prometheus-ds-test-uid"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "query_params.expr", "grafanacloud_grafana_instance_active_user_count"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "interval", "300"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "training_window", "7776000"),
					resource.TestCheckResourceAttr("grafana_machine_learning_job.test_job", "hyper_params.transformation_id", "power"),
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

		client := testutils.Provider.Meta().(*common.Client).MLAPI
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
		client := testutils.Provider.Meta().(*common.Client).MLAPI
		_, err := client.Job(context.Background(), job.ID)
		if err == nil {
			return fmt.Errorf("job still exists on server")
		}
		return nil
	}
}

func testAccDatasourceCheckDestroy() resource.TestCheckFunc {
	// Check the `machinelearningDatasource` has been destroyed
	return func(s *terraform.State) error {
		var orgID int64 = 1
		client := testutils.Provider.Meta().(*common.Client).GrafanaAPI.WithOrgID(orgID)
		ds, err := client.Datasources.GetDataSourceByName("prometheus-ds-test")
		if err == nil {
			return fmt.Errorf("Datasource `%s` still exists after destroy", ds.Payload.Name)
		}
		return nil
	}
}

const machineLearningJobInvalid = `
resource "grafana_machine_learning_job" "invalid" {
  name            = "Test Job"
  metric          = "tf_test_job"
  datasource_type = "fake"
  datasource_uid   = "bla"
  query_params = {
    expr = "grafanacloud_grafana_instance_active_user_count"
  }
}
`

func TestAccResourceInvalidMachineLearningJob(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      machineLearningJobInvalid,
				ExpectError: regexp.MustCompile(".*datasourceType.*"),
			},
		},
	})
}
