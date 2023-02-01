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

func TestAccResourceMachineLearningOutlierDetector(t *testing.T) {
	CheckCloudInstanceTestsEnabled(t)

	var outlier mlapi.OutlierDetector
	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccMLOutlierCheckDestroy(&outlier),
		Steps: []resource.TestStep{
			{
				Config: testAccExample(t, "resources/grafana_machine_learning_outlier_detector/mad.tf"),
				Check: resource.ComposeTestCheckFunc(
					testAccMLOutlierCheckExists("grafana_machine_learning_outlier_detector.my_mad_outlier_detector", &outlier),
					resource.TestCheckResourceAttrSet("grafana_machine_learning_outlier_detector.my_mad_outlier_detector", "id"),
					resource.TestCheckResourceAttr("grafana_machine_learning_outlier_detector.my_mad_outlier_detector", "name", "My MAD outlier detector"),
					resource.TestCheckResourceAttr("grafana_machine_learning_outlier_detector.my_mad_outlier_detector", "metric", "tf_test_mad_job"),
					resource.TestCheckResourceAttr("grafana_machine_learning_outlier_detector.my_mad_outlier_detector", "datasource_type", "prometheus"),
					resource.TestCheckResourceAttr("grafana_machine_learning_outlier_detector.my_mad_outlier_detector", "datasource_uid", "AbCd12345"),
					resource.TestCheckResourceAttr("grafana_machine_learning_outlier_detector.my_mad_outlier_detector", "query_params.expr", "grafanacloud_grafana_instance_active_user_count"),
					resource.TestCheckResourceAttr("grafana_machine_learning_outlier_detector.my_mad_outlier_detector", "interval", "300"),
					resource.TestCheckResourceAttr("grafana_machine_learning_outlier_detector.my_mad_outlier_detector", "algorithm.0.name", "mad"),
					resource.TestCheckResourceAttr("grafana_machine_learning_outlier_detector.my_mad_outlier_detector", "algorithm.0.sensitivity", "0.7"),
				),
			},
			{
				Config: testAccExample(t, "resources/grafana_machine_learning_outlier_detector/dbscan.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_machine_learning_outlier_detector.my_dbscan_outlier_detector", "id"),
					resource.TestCheckResourceAttr("grafana_machine_learning_outlier_detector.my_dbscan_outlier_detector", "name", "My DBSCAN outlier detector"),
					resource.TestCheckResourceAttr("grafana_machine_learning_outlier_detector.my_dbscan_outlier_detector", "metric", "tf_test_dbscan_job"),
					resource.TestCheckResourceAttr("grafana_machine_learning_outlier_detector.my_dbscan_outlier_detector", "datasource_type", "prometheus"),
					resource.TestCheckResourceAttr("grafana_machine_learning_outlier_detector.my_dbscan_outlier_detector", "datasource_id", "12"),
					resource.TestCheckResourceAttr("grafana_machine_learning_outlier_detector.my_dbscan_outlier_detector", "query_params.expr", "grafanacloud_grafana_instance_active_user_count"),
					resource.TestCheckResourceAttr("grafana_machine_learning_outlier_detector.my_dbscan_outlier_detector", "interval", "300"),
					resource.TestCheckResourceAttr("grafana_machine_learning_outlier_detector.my_dbscan_outlier_detector", "algorithm.0.name", "dbscan"),
					resource.TestCheckResourceAttr("grafana_machine_learning_outlier_detector.my_dbscan_outlier_detector", "algorithm.0.sensitivity", "0.5"),
					resource.TestCheckResourceAttr("grafana_machine_learning_outlier_detector.my_dbscan_outlier_detector", "algorithm.0.config.0.epsilon", "1"),
				),
			},
		},
	})
}

func testAccMLOutlierCheckExists(rn string, outlier *mlapi.OutlierDetector) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s\n %#v", rn, s.RootModule().Resources)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		client := testAccProvider.Meta().(*client).mlapi
		gotOutlier, err := client.OutlierDetector(context.Background(), rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("error getting outlier: %s", err)
		}

		*outlier = gotOutlier

		return nil
	}
}

func testAccMLOutlierCheckDestroy(outlier *mlapi.OutlierDetector) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// This check is to make sure that no pointer conversions are incorrect
		// while mutating the outlier.
		if outlier.ID == "" {
			return fmt.Errorf("checking deletion of empty id")
		}
		client := testAccProvider.Meta().(*client).mlapi
		_, err := client.OutlierDetector(context.Background(), outlier.ID)
		if err == nil {
			return fmt.Errorf("outlier still exists on server")
		}
		return nil
	}
}

const machineLearningOutlierDetectorInvalid = `
resource "grafana_machine_learning_outlier_detector" "invalid" {
  name            = "Test Outlier Detector"
  metric          = "tf_my_mad_outlier_detector"
  datasource_type = "fake"
  datasource_id   = 10
  query_params = {
    expr = "grafanacloud_grafana_instance_active_user_count"
  }
  algorithm {
    name = "mad"
    sensitivity = 0.5
  }
}
`

const machineLearningOutlierDetectorMissingDatasourceIDOrUID = `
resource "grafana_machine_learning_outlier_detector" "invalid" {
  name            = "Test Outlier Detector"
  metric          = "tf_my_mad_outlier_detector"
  datasource_type = "prometheus"
  query_params = {
    expr = "grafanacloud_grafana_instance_active_user_count"
  }
  algorithm {
    name = "mad"
    sensitivity = 0.5
  }
}
`
const machineLearningOutlierDetectorMultipleAlgorithm = `
resource "grafana_machine_learning_outlier_detector" "invalid" {
  name            = "Test Outlier Detector"
  metric          = "tf_my_mad_outlier_detector"
  datasource_type = "datadog"
  datasource_uid   = 100000
  query_params = {
    expr = "grafanacloud_grafana_instance_active_user_count"
  }
  algorithm {
    name = "mad"
    sensitivity = 0.5
  }
  algorithm {
    name = "dbscan"
    sensitivity = 0.5
  }
}
`
const machineLearningOutlierDetectorDBSCANMissingConfig = `
resource "grafana_machine_learning_outlier_detector" "invalid" {
  name            = "Test Outlier Detector"
  metric          = "tf_my_mad_outlier_detector"
  datasource_type = "loki"
  datasource_id   = 4
  query_params = {
    expr = "grafanacloud_grafana_instance_active_user_count"
  }
  algorithm {
    name = "dbscan"
    sensitivity = 0.5
  }
}
`
const machineLearningOutlierDetectorDBSCANEmptyConfig = `
resource "grafana_machine_learning_outlier_detector" "invalid" {
  name            = "Test Outlier Detector"
  metric          = "tf_my_mad_outlier_detector"
  datasource_type = "graphite"
  datasource_uid  = "abcdefgh"
  query_params = {
    expr = "grafanacloud_grafana_instance_active_user_count"
  }
  algorithm {
    name = "dbscan"
    sensitivity = 0.5
    config {}
  }
}
`

func TestAccResourceInvalidMachineLearningOutlierDetector(t *testing.T) {
	CheckCloudInstanceTestsEnabled(t)

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      machineLearningOutlierDetectorInvalid,
				ExpectError: regexp.MustCompile(".*datasource_type.*"),
			},
			{
				Config:      machineLearningOutlierDetectorMissingDatasourceIDOrUID,
				ExpectError: regexp.MustCompile(".*datasource_id or datasource_uid.*"),
			},
			{
				Config:      machineLearningOutlierDetectorMultipleAlgorithm,
				ExpectError: regexp.MustCompile(".*most one \"algorithm\" block.*"),
			},
			{
				Config:      machineLearningOutlierDetectorDBSCANMissingConfig,
				ExpectError: regexp.MustCompile(".*requires a single \"config\" block.*"),
			},
			{
				Config:      machineLearningOutlierDetectorDBSCANEmptyConfig,
				ExpectError: regexp.MustCompile(".*argument \"epsilon\" is required.*"),
			},
		},
	})
}
