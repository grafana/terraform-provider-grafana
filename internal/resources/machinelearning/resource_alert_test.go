package machinelearning_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/grafana/machine-learning-go-client/mlapi"
	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
	"github.com/grafana/terraform-provider-grafana/v3/pkg/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccResourceJobAlert(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	randomJobName := acctest.RandomWithPrefix("Test Job")
	randomAlertName := acctest.RandomWithPrefix("Test Alert")

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
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_machine_learning_alert/forecast_alert.tf", map[string]string{
					"Test Job":   randomJobName,
					"Test Alert": randomAlertName,
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccMLJobCheckExists("grafana_machine_learning_job.test_alert_job", &job),
					testAccMLJobAlertCheckExists("grafana_machine_learning_alert.test_job_alert", &job, &alert),
					resource.TestCheckResourceAttrSet("grafana_machine_learning_alert.test_job_alert", "id"),
					resource.TestCheckResourceAttr("grafana_machine_learning_alert.test_job_alert", "title", randomAlertName),
					resource.TestCheckResourceAttr("grafana_machine_learning_alert.test_job_alert", "anomaly_condition", "any"),
					resource.TestCheckResourceAttr("grafana_machine_learning_alert.test_job_alert", "threshold", ">0.8"),
					resource.TestCheckResourceAttr("grafana_machine_learning_alert.test_job_alert", "window", "15m"),
					resource.TestCheckResourceAttr("grafana_machine_learning_alert.test_job_alert", "no_data_state", "OK"),
				),
			},
			// Update the alert with a new anomaly condition.
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_machine_learning_alert/forecast_alert.tf", map[string]string{
					"Test Job":   randomJobName,
					"Test Alert": randomAlertName,
					"\"any\"":    "\"low\"",
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccMLJobCheckExists("grafana_machine_learning_job.test_alert_job", &job),
					testAccMLJobAlertCheckExists("grafana_machine_learning_alert.test_job_alert", &job, &alert),
					resource.TestCheckResourceAttrSet("grafana_machine_learning_alert.test_job_alert", "id"),
					resource.TestCheckResourceAttr("grafana_machine_learning_alert.test_job_alert", "title", randomAlertName),
					resource.TestCheckResourceAttr("grafana_machine_learning_alert.test_job_alert", "anomaly_condition", "low"),
					resource.TestCheckResourceAttr("grafana_machine_learning_alert.test_job_alert", "threshold", ">0.8"),
					resource.TestCheckResourceAttr("grafana_machine_learning_alert.test_job_alert", "window", "15m"),
					resource.TestCheckResourceAttr("grafana_machine_learning_alert.test_job_alert", "no_data_state", "OK"),
				),
			},
			{
				ResourceName: "grafana_machine_learning_alert.test_job_alert",
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return fmt.Sprintf("/jobs/%s/alerts/%s", job.ID, alert.ID), nil
				},
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccResourceOutlierAlert(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	randomOutlierName := acctest.RandomWithPrefix("Test Outlier")
	randomAlertName := acctest.RandomWithPrefix("Test Alert")

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
					resource.TestCheckResourceAttr("grafana_machine_learning_alert.test_outlier_alert", "title", randomAlertName),
					resource.TestCheckResourceAttr("grafana_machine_learning_alert.test_outlier_alert", "window", "1h"),
				),
			},
			// Test updating the window.
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_machine_learning_alert/outlier_alert.tf", map[string]string{
					"Test Outlier": randomOutlierName,
					"Test Alert":   randomAlertName,
					"\"1h\"":       "\"30m\"",
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccMLOutlierCheckExists("grafana_machine_learning_outlier_detector.test_alert_outlier_detector", &outlier),
					testAccMLOutlierAlertCheckExists("grafana_machine_learning_alert.test_outlier_alert", &outlier, &alert),
					resource.TestCheckResourceAttr("grafana_machine_learning_alert.test_outlier_alert", "title", randomAlertName),
					resource.TestCheckResourceAttr("grafana_machine_learning_alert.test_outlier_alert", "window", "30m"),
				),
			},
			{
				ResourceName: "grafana_machine_learning_alert.test_outlier_alert",
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return fmt.Sprintf("/outliers/%s/alerts/%s", outlier.ID, alert.ID), nil
				},
				ImportStateVerify: true,
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

		client := testutils.Provider.Meta().(*client.Client).MLAPI
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

		client := testutils.Provider.Meta().(*client.Client).MLAPI
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
		client := testutils.Provider.Meta().(*client.Client).MLAPI
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
		client := testutils.Provider.Meta().(*client.Client).MLAPI
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
  window = "25h"
}
`,
				ExpectError: regexp.MustCompile(".*value must be a duration less than: 1d.*"),
			},
		},
	})
}
