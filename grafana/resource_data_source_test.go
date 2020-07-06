package grafana

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"

	gapi "github.com/nytm/go-grafana-api"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

var resourceTests = []struct {
	resource   string
	config     string
	attrChecks map[string]string
}{
	{
		"grafana_data_source.testdata",
		`
resource "grafana_data_source" "testdata" {
	type                = "testdata"
	name                = "testdata"
	access_mode					= "direct"
	basic_auth_enabled  = true
	basic_auth_password = "ba_password"
	basic_auth_username = "ba_username"
	database_name       = "db_name"
	is_default					= true
	url                 = "http://acc-test.invalid/"
	username            = "user"
	password            = "pass"
}
`,
		map[string]string{
			"type":                "testdata",
			"name":                "testdata",
			"access_mode":         "direct",
			"basic_auth_enabled":  "true",
			"basic_auth_password": "ba_password",
			"basic_auth_username": "ba_username",
			"database_name":       "db_name",
			"is_default":          "true",
			"url":                 "http://acc-test.invalid/",
			"username":            "user",
			"password":            "pass",
		},
	},
	{
		"grafana_data_source.graphite",
		`
	resource "grafana_data_source" "graphite" {
		type = "graphite"
		name = "graphite"
		url  = "http://acc-test.invalid/"
		json_data {
			graphite_version = "1.1"
		}
	}
	`,
		map[string]string{
			"type":                         "graphite",
			"name":                         "graphite",
			"url":                          "http://acc-test.invalid/",
			"json_data.0.graphite_version": "1.1",
		},
	},
	{
		"grafana_data_source.influx",
		`
	resource "grafana_data_source" "influx" {
		type          = "influx"
		name          = "influx"
		database_name = "db_name"
		username      = "user"
		password      = "pass"
		url           = "http://acc-test.invalid/"
		json_data {
			time_interval = "60s"
		}
	}
	`,
		map[string]string{
			"type":                      "influx",
			"name":                      "influx",
			"database_name":             "db_name",
			"username":                  "user",
			"password":                  "pass",
			"url":                       "http://acc-test.invalid/",
			"json_data.0.time_interval": "60s",
		},
	},
	{
		"grafana_data_source.elasticsearch",
		`
	resource "grafana_data_source" "elasticsearch" {
		type          = "elasticsearch"
		name          = "elasticsearch"
		database_name = "[filebeat-]YYYY.MM.DD"
		url 	        = "http://acc-test.invalid/"
		json_data {
			es_version        = 70
			interval          = "Daily"
			time_field        = "@timestamp"
			log_message_field = "message"
			log_level_field   = "fields.level"
		}
	}
	`,
		map[string]string{
			"type":                          "elasticsearch",
			"name":                          "elasticsearch",
			"database_name":                 "[filebeat-]YYYY.MM.DD",
			"url":                           "http://acc-test.invalid/",
			"json_data.0.es_version":        "70",
			"json_data.0.interval":          "Daily",
			"json_data.0.time_field":        "@timestamp",
			"json_data.0.log_message_field": "message",
			"json_data.0.log_level_field":   "fields.level",
		},
	},
	{
		"grafana_data_source.opentsdb",
		`
	resource "grafana_data_source" "opentsdb" {
		type = "opentsdb"
		name = "opentsdb"
		url	 = "http://acc-test.invalid/"
		json_data {
			tsdb_resolution = 1
			tsdb_version    = 1
		}
	}
	`,
		map[string]string{
			"type":                        "opentsdb",
			"name":                        "opentsdb",
			"url":                         "http://acc-test.invalid/",
			"json_data.0.tsdb_resolution": "1",
			"json_data.0.tsdb_version":    "1",
		},
	},
	{
		"grafana_data_source.cloudwatch",
		`
	resource "grafana_data_source" "cloudwatch" {
		type = "cloudwatch"
		name = "cloudwatch"
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
	`,
		map[string]string{
			"type":                                  "cloudwatch",
			"name":                                  "cloudwatch",
			"json_data.0.default_region":            "us-east-1",
			"json_data.0.auth_type":                 "keys",
			"json_data.0.assume_role_arn":           "arn:aws:sts::*:assumed-role/*/*",
			"json_data.0.custom_metrics_namespaces": "foo",
			"secure_json_data.0.access_key":         "123",
			"secure_json_data.0.secret_key":         "456",
		},
	},
	{
		"grafana_data_source.mssql",
		`
		resource "grafana_data_source" "mssql" {
			type          = "mssql"
			name          = "mssql"
			database_name = "db"
			url 	        = "acc-test.invalid:1433"
			json_data {
				max_open_conns    = 0
				max_idle_conns    = 2
				conn_max_lifetime = 14400
				encrypt           = "yes"
			}
			secure_json_data {
				password = "pass"
			}
		}
		`,
		map[string]string{
			"type":                          "mssql",
			"name":                          "mssql",
			"database_name":                 "db",
			"url":                           "acc-test.invalid:1433",
			"json_data.0.max_open_conns":    "0",
			"json_data.0.max_idle_conns":    "2",
			"json_data.0.conn_max_lifetime": "14400",
			"json_data.0.encrypt":           "yes",
			"secure_json_data.0.password":   "pass",
		},
	},
	{
		"grafana_data_source.postgres",
		`
		resource "grafana_data_source" "postgres" {
			type          = "postgres"
			name          = "postgres"
			database_name = "db"
			url 	        = "acc-test.invalid:5432"
			username      = "user"
			json_data {
				max_open_conns    = 0
				max_idle_conns    = 2
				conn_max_lifetime = 14400
				postgres_version  = 905
				timescaledb 			= false
			}
			secure_json_data {
				password = "pass"
			}
		}
		`,
		map[string]string{
			"type":                          "postgres",
			"name":                          "postgres",
			"database_name":                 "db",
			"url":                           "acc-test.invalid:5432",
			"json_data.0.max_open_conns":    "0",
			"json_data.0.max_idle_conns":    "2",
			"json_data.0.conn_max_lifetime": "14400",
			"json_data.0.postgres_version":  "905",
			"json_data.0.timescaledb":       "false",
			"secure_json_data.0.password":   "pass",
		},
	},
	{
		"grafana_data_source.prometheus",
		`
		resource "grafana_data_source" "prometheus" {
			type = "prometheus"
			name = "prometheus"
			url  = "http://acc-test.invalid:9090"
			json_data {
				http_method = "GET"
				query_timeout = "1"
			}
		}
		`,
		map[string]string{
			"type":                      "prometheus",
			"name":                      "prometheus",
			"url":                       "http://acc-test.invalid:9090",
			"json_data.0.http_method":   "GET",
			"json_data.0.query_timeout": "1",
		},
	},
	{
		"grafana_data_source.stackdriver",
		`
		resource "grafana_data_source" "stackdriver" {
			type = "stackdriver"
			name = "stackdriver"
			secure_json_data {
				private_key = "-----BEGIN PRIVATE KEY-----\nprivate-key\n-----END PRIVATE KEY-----\n"
			}
		}
		`,
		map[string]string{
			"type":                           "stackdriver",
			"name":                           "stackdriver",
			"secure_json_data.0.private_key": "-----BEGIN PRIVATE KEY-----\nprivate-key\n-----END PRIVATE KEY-----\n",
		},
	},
}

func TestAccDataSource_basic(t *testing.T) {
	var dataSource gapi.DataSource

	// Iterate over the provided configurations for datasources
	for _, test := range resourceTests {

		// Always check that the resource was created and that `id` is a number
		checks := []resource.TestCheckFunc{
			testAccDataSourceCheckExists(test.resource, &dataSource),
			resource.TestMatchResourceAttr(
				test.resource,
				"id",
				regexp.MustCompile(`\d+`),
			),
		}

		// Add custom checks for specified attribute values
		for attr, value := range test.attrChecks {
			checks = append(checks, resource.TestCheckResourceAttr(
				test.resource,
				attr,
				value,
			))
		}

		resource.Test(t, resource.TestCase{
			PreCheck:     func() { testAccPreCheck(t) },
			Providers:    testAccProviders,
			CheckDestroy: testAccDataSourceCheckDestroy(&dataSource),
			Steps: []resource.TestStep{
				{
					Config: test.config,
					Check: resource.ComposeAggregateTestCheckFunc(
						checks...,
					),
				},
			},
		})
	}
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
