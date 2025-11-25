package grafana_test

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccDataSource_Loki(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	var dataSource models.DataSource
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
		datasourceCheckExists.exists("grafana_data_source.loki", &dataSource),
		resource.TestMatchResourceAttr("grafana_data_source.loki", "id", defaultOrgIDRegexp),
		resource.TestCheckResourceAttr("grafana_data_source.loki", "org_id", "1"), // default org
		resource.TestMatchResourceAttr("grafana_data_source.loki", "uid", common.UIDRegexp),
		resource.TestCheckResourceAttr("grafana_data_source.loki", "name", dsName),
		resource.TestCheckResourceAttr("grafana_data_source.loki", "type", "loki"),
		resource.TestCheckResourceAttr("grafana_data_source.loki", "url", "http://acc-test.invalid/"),
		testutils.CheckLister("grafana_data_source.loki"),
		func(s *terraform.State) error {
			jsonData := dataSource.JSONData.(map[string]any)
			if jsonData["derivedFields"] == nil {
				return fmt.Errorf("expected derived fields")
			}
			// Check datasource IDs
			derivedFields := jsonData["derivedFields"].([]any)
			if len(derivedFields) != 2 {
				return fmt.Errorf("expected 2 derived fields, got %d", len(derivedFields))
			}
			firstDerivedField := derivedFields[0].(map[string]any)
			if _, ok := firstDerivedField["datasourceUid"]; ok {
				return fmt.Errorf("expected empty datasource_uid")
			}
			secondDerivedField := derivedFields[1].(map[string]any)
			if !common.UIDRegexp.MatchString(secondDerivedField["datasourceUid"].(string)) {
				return fmt.Errorf("expected valid datasource_uid")
			}
			return nil
		},
	)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             datasourceCheckExists.destroyed(&dataSource, nil),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  checks,
			},
			// Test import using UID
			{
				ResourceName:      "grafana_data_source.loki",
				ImportState:       true,
				ImportStateVerify: true,
				// Ignore sensitive attributes, we mostly only care about "json_data_encoded"
				ImportStateVerifyIgnore: []string{"secure_json_data_encoded", "http_headers."},
			},
		},
	})
}

func TestAccDataSource_TestData(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	var dataSource models.DataSource

	dsName := acctest.RandString(10)
	config := fmt.Sprintf(`
	resource "grafana_data_source" "testdata" {
		type                = "grafana-testdata-datasource"
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
		datasourceCheckExists.exists("grafana_data_source.testdata", &dataSource),
		resource.TestMatchResourceAttr("grafana_data_source.testdata", "id", defaultOrgIDRegexp),
		resource.TestCheckResourceAttr("grafana_data_source.testdata", "org_id", "1"), // default org
		resource.TestMatchResourceAttr("grafana_data_source.testdata", "uid", common.UIDRegexp),
		resource.TestCheckResourceAttr("grafana_data_source.testdata", "name", dsName),
		resource.TestCheckResourceAttr("grafana_data_source.testdata", "type", "grafana-testdata-datasource"),
		resource.TestCheckResourceAttr("grafana_data_source.testdata", "access_mode", "direct"),
		resource.TestCheckResourceAttr("grafana_data_source.testdata", "basic_auth_enabled", "true"),
		resource.TestCheckResourceAttr("grafana_data_source.testdata", "basic_auth_username", "ba_username"),
		resource.TestCheckResourceAttr("grafana_data_source.testdata", "database_name", "db_name"),
		resource.TestCheckResourceAttr("grafana_data_source.testdata", "is_default", "true"),
		resource.TestCheckResourceAttr("grafana_data_source.testdata", "url", "http://acc-test.invalid/"),
		resource.TestCheckResourceAttr("grafana_data_source.testdata", "username", "user"),
	)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             datasourceCheckExists.destroyed(&dataSource, nil),
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

	var dataSource models.DataSource

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
		datasourceCheckExists.exists("grafana_data_source.influx", &dataSource),
		resource.TestMatchResourceAttr("grafana_data_source.influx", "id", defaultOrgIDRegexp),
		resource.TestCheckResourceAttr("grafana_data_source.influx", "org_id", "1"), // default org
		resource.TestMatchResourceAttr("grafana_data_source.influx", "uid", common.UIDRegexp),
		resource.TestCheckResourceAttr("grafana_data_source.influx", "name", dsName),
		resource.TestCheckResourceAttr("grafana_data_source.influx", "type", "influxdb"),
		resource.TestCheckResourceAttr("grafana_data_source.influx", "url", "http://acc-test.invalid/"),
		func(s *terraform.State) error {
			expected := map[string]any{
				"defaultBucket":     "telegraf",
				"organization":      "organization",
				"tlsAuth":           false,
				"tlsAuthWithCACert": false,
				"version":           "Flux",
				"httpHeaderName1":   "Authorization",
			}
			jsonData := dataSource.JSONData.(map[string]any)
			if !reflect.DeepEqual(jsonData, expected) {
				return fmt.Errorf("bad json_data_encoded: %#v. Expected: %+v", dataSource.JSONData, expected)
			}
			if v, ok := jsonData["httpHeaderName1"]; !ok && v != "Authorization" {
				return fmt.Errorf("http header Authorization not found")
			}
			return nil
		},
	)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             datasourceCheckExists.destroyed(&dataSource, nil),
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

	var dataSource models.DataSource

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             datasourceCheckExists.destroyed(&dataSource, nil),
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
					datasourceCheckExists.exists("grafana_data_source.test", &dataSource),
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
					datasourceCheckExists.exists("grafana_data_source.test", &dataSource),
					resource.TestCheckResourceAttr("grafana_data_source.test", "uid", "changed-uid"),
				),
			},
		},
	})
}

func TestAccDataSource_numericalUID(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	var dataSource models.DataSource
	uid := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             datasourceCheckExists.destroyed(&dataSource, nil),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
	resource "grafana_data_source" "test" {
		name = "numerical-%[1]d"
		type = "prometheus"
		url  = "http://localhost:9090"
		uid  = "%[1]d"
	}`, uid),
				Check: resource.ComposeTestCheckFunc(
					datasourceCheckExists.exists("grafana_data_source.test", &dataSource),
					resource.TestCheckResourceAttr("grafana_data_source.test", "uid", strconv.Itoa(uid)),
				),
			},
		},
	})
}

func TestAccDatasource_inOrg(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	var dataSource models.DataSource
	var org models.OrgDetailsDTO

	orgName := acctest.RandString(10)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             datasourceCheckExists.destroyed(&dataSource, &org),
		Steps: []resource.TestStep{
			{
				Config: testAccDatasourceInOrganization(orgName),
				Check: resource.ComposeTestCheckFunc(
					orgCheckExists.exists("grafana_organization.test", &org),

					// Check that the datasource is in the correct organization
					datasourceCheckExists.exists("grafana_data_source.test", &dataSource),
					resource.TestMatchResourceAttr("grafana_data_source.test", "id", nonDefaultOrgIDRegexp),
					resource.TestCheckResourceAttr("grafana_data_source.test", "uid", "test-in-org"),
					resource.TestCheckResourceAttr("grafana_data_source.test", "name", "test-in-org"),
					resource.TestMatchResourceAttr("grafana_data_source.test", "org_id", regexp.MustCompile(`([^0-1]\d*|1\d+)`)), // > 1
					checkResourceIsInOrg("grafana_data_source.test", "grafana_organization.test"),
				),
			},
		},
	})
}

func TestAccDataSource_ValidateHttpHeaders(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
				resource "grafana_data_source" "influx" {
					type         = "influxdb"
					name         = "anything"
					url          = "http://acc-test.invalid/"
					json_data_encoded = jsonencode({
						httpHeaderName1     = "Authorization"
						defaultBucket       = "telegraf"
						organization        = "organization"
						tlsAuth             = false
						tlsAuthWithCACert   = false
						version             = "Flux"
					})
					secure_json_data_encoded = jsonencode({
						httpHeaderValue1 = "Token sdkfjsdjflkdsjflksjdklfjslkdfjdksljfldksjsflkj"
					})
				}`,
				ExpectError: regexp.MustCompile(`httpHeaderName{num} is a reserved key and cannot be used in JSON data. Use the http_headers attribute instead`),
			},
		},
	})
}

func TestAccDataSource_SeparateConfig(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=v9.0.0")

	var dataSource models.DataSource

	dsName := acctest.RandString(10)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             datasourceCheckExists.destroyed(&dataSource, nil),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "grafana_data_source" "influx" {
					type         = "influxdb"
					name         = "%s"
					url          = "http://acc-test.invalid/"

					lifecycle {
						ignore_changes = [json_data_encoded, http_headers]
					}
				}
				
				resource "grafana_data_source_config" "influx" {
					uid = grafana_data_source.influx.uid
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
					secure_json_data_encoded = jsonencode({
						password = "password"
					})
				}`, dsName),
				Check: resource.ComposeTestCheckFunc(
					datasourceCheckExists.exists("grafana_data_source.influx", &dataSource),
					resource.TestMatchResourceAttr("grafana_data_source.influx", "id", defaultOrgIDRegexp),
					resource.TestCheckResourceAttr("grafana_data_source.influx", "org_id", "1"), // default org
					resource.TestMatchResourceAttr("grafana_data_source.influx", "uid", common.UIDRegexp),
					resource.TestCheckResourceAttr("grafana_data_source.influx", "name", dsName),
					resource.TestCheckResourceAttr("grafana_data_source.influx", "type", "influxdb"),
					resource.TestCheckResourceAttr("grafana_data_source.influx", "url", "http://acc-test.invalid/"),

					resource.TestCheckResourceAttr("grafana_data_source_config.influx", "json_data_encoded", `{"defaultBucket":"telegraf","organization":"organization","tlsAuth":false,"tlsAuthWithCACert":false,"version":"Flux"}`),
					resource.TestCheckResourceAttr("grafana_data_source_config.influx", "secure_json_data_encoded", `{"password":"password"}`),
					resource.TestCheckResourceAttr("grafana_data_source_config.influx", "http_headers.Authorization", "Token sdkfjsdjflkdsjflksjdklfjslkdfjdksljfldksjsflkj"),

					func(s *terraform.State) error {
						expected := map[string]any{
							"defaultBucket":     "telegraf",
							"organization":      "organization",
							"tlsAuth":           false,
							"tlsAuthWithCACert": false,
							"version":           "Flux",
							"httpHeaderName1":   "Authorization",
						}
						jsonData := dataSource.JSONData.(map[string]any)
						if !reflect.DeepEqual(jsonData, expected) {
							return fmt.Errorf("bad json_data_encoded: %#v. Expected: %+v", dataSource.JSONData, expected)
						}
						if v, ok := jsonData["httpHeaderName1"]; !ok && v != "Authorization" {
							return fmt.Errorf("http header Authorization not found")
						}
						if v := dataSource.SecureJSONFields["password"]; !v {
							return fmt.Errorf("password not set")
						}
						return nil
					},
				),
			},
			{
				ImportState:             true,
				ResourceName:            "grafana_data_source_config.influx",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"secure_json_data_encoded", "http_headers"},
			},
		},
	})
}

func TestAccDataSource_ImportReadOnly(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:            `resource "grafana_data_source" "prometheus" {}`,
				ImportState:       true,
				ResourceName:      "grafana_data_source.prometheus",
				ImportStateVerify: true,
				ImportStateId:     "grafanacloud-prom",
				ExpectError:       regexp.MustCompile("this Grafana data source is read-only. It cannot be imported as a resource. Use the `data_grafana_data_source` data source instead"),
			},
		},
	})
}

func testAccDatasourceInOrganization(orgName string) string {
	return fmt.Sprintf(`
resource "grafana_organization" "test" {
	name = "%[1]s"
}

resource "grafana_data_source" "test" {
	org_id = grafana_organization.test.id
	name   = "test-in-org"
	uid    = "test-in-org"
	type   = "prometheus"
	url    = "http://localhost:9090"
}`, orgName)
}
