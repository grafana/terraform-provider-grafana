package grafana

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"

	gapi "github.com/nytm/go-grafana-api"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDataSource_basic(t *testing.T) {
	var dataSource gapi.DataSource

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccDataSourceCheckDestroy(&dataSource),
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceCheckExists("grafana_data_source.test_influxdb", &dataSource),
					resource.TestCheckResourceAttr(
						"grafana_data_source.test_influxdb", "type", "influxdb",
					),
					resource.TestCheckResourceAttr(
						"grafana_data_source.test_influxdb", "password", "terraform_password",
					),
					resource.TestCheckResourceAttr(
						"grafana_data_source.test_influxdb", "basic_auth_password", "basic_password",
					),
					resource.TestMatchResourceAttr(
						"grafana_data_source.test_influxdb", "id", regexp.MustCompile(`\d+`),
					),
				),
			},
		},
	})
}

func TestAccDataSource_basicCloudwatch(t *testing.T) {
	var dataSource gapi.DataSource

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccDataSourceCheckDestroy(&dataSource),
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceConfig_basicCloudwatch,
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceCheckExists("grafana_data_source.test_cloudwatch", &dataSource),
					resource.TestCheckResourceAttr(
						"grafana_data_source.test_cloudwatch", "type", "cloudwatch",
					),
					resource.TestCheckResourceAttr(
						"grafana_data_source.test_cloudwatch", "json_data.0.custom_metrics_namespaces", "foo",
					),
					resource.TestCheckResourceAttr(
						"grafana_data_source.test_cloudwatch", "json_data.0.assume_role_arn", "arn:aws:sts::*:assumed-role/*/*",
					),
					resource.TestCheckResourceAttr(
						"grafana_data_source.test_cloudwatch", "json_data.0.auth_type", "keys",
					),
					resource.TestCheckResourceAttr(
						"grafana_data_source.test_cloudwatch", "json_data.0.default_region", "us-east-1",
					),
					resource.TestCheckResourceAttr(
						"grafana_data_source.test_cloudwatch", "secure_json_data.0.access_key", "123",
					),
				),
			},
		},
	})
}

func testAccDataSourceCheckExists(rn string, dataSource *gapi.DataSource) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		id, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
		if err != nil {
			return fmt.Errorf("resource id is malformed")
		}

		client := testAccProvider.Meta().(*gapi.Client)
		gotDataSource, err := client.DataSource(id)
		if err != nil {
			return fmt.Errorf("error getting data source: %s", err)
		}

		*dataSource = *gotDataSource

		return nil
	}
}

func testAccDataSourceCheckDestroy(dataSource *gapi.DataSource) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*gapi.Client)
		_, err := client.DataSource(dataSource.Id)
		if err == nil {
			return fmt.Errorf("data source still exists")
		}
		return nil
	}
}

const testAccDataSourceConfig_basic = `
resource "grafana_data_source" "test_influxdb" {
  type                = "influxdb"
  name                = "terraform-acc-test-influxdb"
  database_name       = "terraform-acc-test-influxdb"
  url                 = "http://terraform-acc-test.invalid/"
  username            = "terraform_user"
  password            = "terraform_password"
  basic_auth_password = "basic_password"
}
`
const testAccDataSourceConfig_basicCloudwatch = `
resource "grafana_data_source" "test_cloudwatch" {
  type = "cloudwatch"
  name = "terraform-acc-test-cloudwatch"

  json_data {
    default_region            = "us-east-1"
    auth_type                 = "keys"
    assume_role_arn           = "arn:aws:sts::*:assumed-role/*/*"
    custom_metrics_namespaces = "foo"
  }

  secure_json_data {
    access_key = "123"
    secret_key = "456"
  }
}
`
