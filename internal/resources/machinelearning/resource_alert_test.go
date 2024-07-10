package machinelearning_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/grafana/machine-learning-go-client/mlapi"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccResourceJobAlert(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	randomJobName := acctest.RandomWithPrefix("Test Job")
	randomAlertName := acctest.RandomWithPrefix("Test Job Alert")

	var job mlapi.Job
	var alert mlapi.Alert
	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccMLJobAlertCheckDestroy(&job, &alert),
			testAccMLJobCheckDestroy(&job),
		),
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_machine_learning_alert/resource.tf", map[string]string{
					"Test Job":   randomJobName,
					"Test Alert": randomAlertName,
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccMLJobCheckExists("grafana_machine_learning_job.test_alert_job", &job),
					testAccMLJobAlertCheckExists("grafana_machine_learning_alert.test_job_alert", &job, &alert),
				),
			},
		},
	})
}

func TestAccResourceOutlierAlert(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	randomOutlierName := acctest.RandomWithPrefix("Test Job")
	randomAlertName := acctest.RandomWithPrefix("Test Outlier Alert")

	var outlier mlapi.OutlierDetector
	var alert mlapi.Alert
	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccMLOutlierAlertCheckDestroy(&outlier, &alert),
			testAccMLOutlierCheckDestroy(&outlier),
		),
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_machine_learning_alert/outlier_alert.tf", map[string]string{
					"Test Outlier": randomOutlierName,
					"Test Alert":   randomAlertName,
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccMLOutlierCheckExists("grafana_machine_learning_outlier_detector.test_alert_outlier_detector", &outlier),
					testAccMLOutlierAlertCheckExists("grafana_machine_learning_alert.test_outlier_alert", &outlier, &alert),
				),
			},
		},
	})
}

func testAccMLJobAlertCheckExists(rn string, job *mlapi.Job, alert *mlapi.Alert) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s\n %#v", rn, s.RootModule().Resources)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		client := testutils.Provider.Meta().(*common.Client).MLAPI
		gotAlert, err := client.JobAlert(context.Background(), job.ID, rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("error getting job: %s", err)
		}

		*alert = gotAlert

		return nil
	}
}

func testAccMLOutlierAlertCheckExists(rn string, outlier *mlapi.OutlierDetector, alert *mlapi.Alert) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s\n %#v", rn, s.RootModule().Resources)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		client := testutils.Provider.Meta().(*common.Client).MLAPI
		gotAlert, err := client.OutlierAlert(context.Background(), outlier.ID, rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("error getting job: %s", err)
		}

		*alert = gotAlert

		return nil
	}
}

func testAccMLJobAlertCheckDestroy(job *mlapi.Job, alert *mlapi.Alert) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// This check is to make sure that no pointer conversions are incorrect
		// while mutating alert.
		if job.ID == "" {
			return fmt.Errorf("checking deletion of empty job id")
		}
		if alert.ID == "" {
			return fmt.Errorf("checking deletion of empty alert id")
		}
		client := testutils.Provider.Meta().(*common.Client).MLAPI
		_, err := client.JobAlert(context.Background(), job.ID, alert.ID)
		if err == nil {
			return fmt.Errorf("job still exists on server")
		}
		return nil
	}
}

func testAccMLOutlierAlertCheckDestroy(outlier *mlapi.OutlierDetector, alert *mlapi.Alert) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// This check is to make sure that no pointer conversions are incorrect
		// while mutating alert.
		if outlier.ID == "" {
			return fmt.Errorf("checking deletion of empty outlier id")
		}
		if alert.ID == "" {
			return fmt.Errorf("checking deletion of empty alert id")
		}
		client := testutils.Provider.Meta().(*common.Client).MLAPI
		_, err := client.OutlierAlert(context.Background(), outlier.ID, alert.ID)
		if err == nil {
			return fmt.Errorf("job still exists on server")
		}
		return nil
	}
}

func TestAccResourceInvalidMachineLearningAlert(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "grafana_machine_learning_alert" "invalid" {
  job_id = "xyz"
  title  = "Test Job"
  for    = "foo"
}
`,
				ExpectError: regexp.MustCompile(".*value must be a duration.*"),
			},
			{
				Config: `
resource "grafana_machine_learning_alert" "invalid" {
  job_id = "xyz"
  title  = "Test Job"
  window = "24h"
}
`,
				ExpectError: regexp.MustCompile(".*value must be a duration less than: 12h.*"),
			},
		},
	})
}
