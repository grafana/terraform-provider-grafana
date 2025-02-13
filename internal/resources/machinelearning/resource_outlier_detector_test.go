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

func TestAccResourceOutlierDetector(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	randomName := acctest.RandomWithPrefix("Outlier Detector")

	var outlier mlapi.OutlierDetector
	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccMLOutlierCheckDestroy(&outlier),
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_machine_learning_outlier_detector/mad.tf", map[string]string{
					"My MAD outlier detector": "MAD " + randomName,
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccMLOutlierCheckExists("grafana_machine_learning_outlier_detector.my_mad_outlier_detector", &outlier),
					resource.TestCheckResourceAttrSet("grafana_machine_learning_outlier_detector.my_mad_outlier_detector", "id"),
					resource.TestCheckResourceAttr("grafana_machine_learning_outlier_detector.my_mad_outlier_detector", "name", "MAD "+randomName),
					resource.TestCheckResourceAttr("grafana_machine_learning_outlier_detector.my_mad_outlier_detector", "metric", "tf_test_mad_job"),
					resource.TestCheckResourceAttr("grafana_machine_learning_outlier_detector.my_mad_outlier_detector", "datasource_type", "prometheus"),
					resource.TestCheckResourceAttr("grafana_machine_learning_outlier_detector.my_mad_outlier_detector", "datasource_uid", "AbCd12345"),
					resource.TestCheckResourceAttr("grafana_machine_learning_outlier_detector.my_mad_outlier_detector", "query_params.expr", "grafanacloud_grafana_instance_active_user_count"),
					resource.TestCheckResourceAttr("grafana_machine_learning_outlier_detector.my_mad_outlier_detector", "interval", "300"),
					resource.TestCheckResourceAttr("grafana_machine_learning_outlier_detector.my_mad_outlier_detector", "algorithm.0.name", "mad"),
					resource.TestCheckResourceAttr("grafana_machine_learning_outlier_detector.my_mad_outlier_detector", "algorithm.0.sensitivity", "0.7"),
					testutils.CheckLister("grafana_machine_learning_outlier_detector.my_mad_outlier_detector"),
				),
			},
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_machine_learning_outlier_detector/dbscan.tf", map[string]string{
					"My DBSCAN outlier detector": "DBSCAN " + randomName,
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_machine_learning_outlier_detector.my_dbscan_outlier_detector", "id"),
					resource.TestCheckResourceAttr("grafana_machine_learning_outlier_detector.my_dbscan_outlier_detector", "name", "DBSCAN "+randomName),
					resource.TestCheckResourceAttr("grafana_machine_learning_outlier_detector.my_dbscan_outlier_detector", "metric", "tf_test_dbscan_job"),
					resource.TestCheckResourceAttr("grafana_machine_learning_outlier_detector.my_dbscan_outlier_detector", "datasource_type", "prometheus"),
					resource.TestCheckResourceAttr("grafana_machine_learning_outlier_detector.my_dbscan_outlier_detector", "datasource_uid", "AbCd12345"),
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

		client := testutils.Provider.Meta().(*client.Client).MLAPI
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
		client := testutils.Provider.Meta().(*client.Client).MLAPI
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
  datasource_uid   = "bla"
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
  datasource_uid   = "bla"
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
	testutils.CheckCloudInstanceTestsEnabled(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      machineLearningOutlierDetectorInvalid,
				ExpectError: regexp.MustCompile(".*datasource_type.*"),
			},

			{
				Config:      machineLearningOutlierDetectorMultipleAlgorithm,
				ExpectError: regexp.MustCompile(".*No more than 1 \"algorithm\" blocks are allowed.*"),
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
