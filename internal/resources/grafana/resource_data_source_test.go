package grafana_test

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/grafana/terraform-provider-grafana/internal/testutils"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccDataSource_Loki(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	var dataSource gapi.DataSource
	dsName := acctest.RandString(10)

	config := fmt.Sprintf(`
	resource "grafana_data_source" "tempo" {
		name = "%[1]s-tempo"
		type = "tempo"
	}

	resource "grafana_data_source" "loki" {
		type                = "loki"
		name                = "%[1]s"
		url                 = "http://acc-test.invalid/"

		json_data_encoded = jsonencode({
			maxLines = 2022
			derivedFields = [
				{
					name = "WithoutDatasource"
					matcherRegex = "(?:traceID|trace_id)=(\\w+)"
					url = "example.com/$${__value.raw}"
				},
				{
					name = "WithDatasource"
					matcherRegex = "(?:traceID|trace_id)=(\\w+)"
					url = "$${__value.raw}"
					datasourceUid = grafana_data_source.tempo.uid
				}
			]
		})
	}
	`, dsName)
	checks := resource.ComposeTestCheckFunc(
		testAccDataSourceCheckExists("grafana_data_source.loki", &dataSource),
		resource.TestMatchResourceAttr("grafana_data_source.loki", "id", common.IDRegexp),
		resource.TestMatchResourceAttr("grafana_data_source.loki", "uid", common.UIDRegexp),
		resource.TestCheckResourceAttr("grafana_data_source.loki", "name", dsName),
		resource.TestCheckResourceAttr("grafana_data_source.loki", "type", "loki"),
		resource.TestCheckResourceAttr("grafana_data_source.loki", "url", "http://acc-test.invalid/"),
		func(s *terraform.State) error {
			if dataSource.JSONData["derivedFields"] == nil {
				return fmt.Errorf("expected derived fields")
			}
			// Check datasource IDs
			derivedFields := dataSource.JSONData["derivedFields"].([]interface{})
			if len(derivedFields) != 2 {
				return fmt.Errorf("expected 2 derived fields, got %d", len(derivedFields))
			}
			firstDerivedField := derivedFields[0].(map[string]interface{})
			if _, ok := firstDerivedField["datasourceUid"]; ok {
				return fmt.Errorf("expected empty datasource_uid")
			}
			secondDerivedField := derivedFields[1].(map[string]interface{})
			if !common.UIDRegexp.MatchString(secondDerivedField["datasourceUid"].(string)) {
				return fmt.Errorf("expected valid datasource_uid")
			}
			return nil
		},
	)

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      testAccDataSourceCheckDestroy(&dataSource),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  checks,
			},
			// Test import using ID
			{
				ResourceName:      "grafana_data_source.loki",
				ImportState:       true,
				ImportStateVerify: true,
				// Ignore sensitive attributes, we mostly only care about "json_data_encoded"
				ImportStateVerifyIgnore: []string{"secure_json_data_encoded", "http_headers."},
			},
			// Test import using UID
			{
				ResourceName:      "grafana_data_source.loki",
				ImportState:       true,
				ImportStateVerify: true,
				// Ignore sensitive attributes, we mostly only care about "json_data_encoded"
				ImportStateVerifyIgnore: []string{"secure_json_data_encoded", "http_headers."},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["grafana_data_source.loki"]
					if !ok {
						return "", fmt.Errorf("resource not found: %s", "grafana_data_source.loki")
					}

					if rs.Primary.ID == "" {
						return "", fmt.Errorf("resource id not set")
					}
					return rs.Primary.Attributes["uid"], nil
				},
			},
		},
	})
}

func TestAccDataSource_TestData(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	var dataSource gapi.DataSource

	dsName := acctest.RandString(10)
	config := fmt.Sprintf(`
	resource "grafana_data_source" "testdata" {
		type                = "testdata"
		name                = "%s"
		access_mode					= "direct"
		basic_auth_enabled  = true
		basic_auth_username = "ba_username"
		database_name       = "db_name"
		is_default					= true
		url                 = "http://acc-test.invalid/"
		username            = "user"
		secure_json_data_encoded = jsonencode({
			password = "ba_password"
		})
	}`, dsName)

	checks := resource.ComposeTestCheckFunc(
		testAccDataSourceCheckExists("grafana_data_source.testdata", &dataSource),
		resource.TestMatchResourceAttr("grafana_data_source.testdata", "id", common.IDRegexp),
		resource.TestMatchResourceAttr("grafana_data_source.testdata", "uid", common.UIDRegexp),
		resource.TestCheckResourceAttr("grafana_data_source.testdata", "name", dsName),
		resource.TestCheckResourceAttr("grafana_data_source.testdata", "type", "testdata"),
		resource.TestCheckResourceAttr("grafana_data_source.testdata", "access_mode", "direct"),
		resource.TestCheckResourceAttr("grafana_data_source.testdata", "basic_auth_enabled", "true"),
		resource.TestCheckResourceAttr("grafana_data_source.testdata", "basic_auth_username", "ba_username"),
		resource.TestCheckResourceAttr("grafana_data_source.testdata", "database_name", "db_name"),
		resource.TestCheckResourceAttr("grafana_data_source.testdata", "is_default", "true"),
		resource.TestCheckResourceAttr("grafana_data_source.testdata", "url", "http://acc-test.invalid/"),
		resource.TestCheckResourceAttr("grafana_data_source.testdata", "username", "user"),
	)

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      testAccDataSourceCheckDestroy(&dataSource),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  checks,
			},
		},
	})
}

func TestAccDataSource_Influx(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	var dataSource gapi.DataSource

	dsName := acctest.RandString(10)
	config := fmt.Sprintf(`
	resource "grafana_data_source" "influx" {
		type         = "influxdb"
		name         = "%s"
		url          = "http://acc-test.invalid/"
		http_headers = {
			Authorization = "Token sdkfjsdjflkdsjflksjdklfjslkdfjdksljfldksjsflkj"
		}
		json_data_encoded = jsonencode({
			defaultBucket       = "telegraf"
			organization        = "organization"
			tlsAuth             = false
			tlsAuthWithCACert   = false
			version             = "Flux"
		})
	}`, dsName)

	checks := resource.ComposeTestCheckFunc(
		testAccDataSourceCheckExists("grafana_data_source.influx", &dataSource),
		resource.TestMatchResourceAttr("grafana_data_source.influx", "id", common.IDRegexp),
		resource.TestMatchResourceAttr("grafana_data_source.influx", "uid", common.UIDRegexp),
		resource.TestCheckResourceAttr("grafana_data_source.influx", "name", dsName),
		resource.TestCheckResourceAttr("grafana_data_source.influx", "type", "influxdb"),
		resource.TestCheckResourceAttr("grafana_data_source.influx", "url", "http://acc-test.invalid/"),
		func(s *terraform.State) error {
			expected := map[string]interface{}{
				"defaultBucket":     "telegraf",
				"organization":      "organization",
				"tlsAuth":           false,
				"tlsAuthWithCACert": false,
				"version":           "Flux",
				"httpHeaderName1":   "Authorization",
			}
			if !reflect.DeepEqual(dataSource.JSONData, expected) {
				return fmt.Errorf("bad json_data_encoded: %#v. Expected: %+v", dataSource.JSONData, expected)
			}
			if v, ok := dataSource.JSONData["httpHeaderName1"]; !ok && v != "Authorization" {
				return fmt.Errorf("http header Authorization not found")
			}
			return nil
		},
	)

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      testAccDataSourceCheckDestroy(&dataSource),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  checks,
			},
		},
	})
}

func TestAccDataSource_changeUID(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	var dataSource gapi.DataSource

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      testAccDataSourceCheckDestroy(&dataSource),
		Steps: []resource.TestStep{
			{
				Config: `
	resource "grafana_data_source" "test" {
		name = "test-change-uid"
		type = "prometheus"
		url  = "http://localhost:9090"
		uid  = "initial-uid"
	}`,
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceCheckExists("grafana_data_source.test", &dataSource),
					resource.TestCheckResourceAttr("grafana_data_source.test", "uid", "initial-uid"),
				),
			},
			{
				Config: `
	resource "grafana_data_source" "test" {
		name = "test-change-uid"
		type = "prometheus"
		url  = "http://localhost:9090"
		uid  = "changed-uid"
	}`,
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceCheckExists("grafana_data_source.test", &dataSource),
					resource.TestCheckResourceAttr("grafana_data_source.test", "uid", "changed-uid"),
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

		client := testutils.Provider.Meta().(*common.Client).GrafanaAPI
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
		client := testutils.Provider.Meta().(*common.Client).GrafanaAPI
		_, err := client.DataSource(dataSource.ID)
		if err == nil {
			return fmt.Errorf("data source still exists")
		}
		return nil
	}
}
