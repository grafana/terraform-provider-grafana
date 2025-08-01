package resources_test

import (
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/grafana/terraform-provider-grafana/v4/pkg/provider"
)

// This test makes sure all resources and datasources have examples and they are all valid.
func TestAccExamples(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping long test")
	}

	// Track if all resources and datasources have been tested
	resourceMap := map[string]bool{}
	datasourceMap := map[string]bool{}

	for _, testDef := range []struct {
		category  string
		testCheck func(t *testing.T, filename string)
	}{
		{
			category: "Alerting",
			testCheck: func(t *testing.T, filename string) {
				if strings.Contains(filename, "grafana_notification_policy") {
					testutils.CheckOSSTestsEnabled(t, ">=12.1.0") // Only run on latest OSS version. The examples should be updated to reflect their latest working config.
				} else {
					testutils.CheckOSSTestsEnabled(t, ">=11.0.0") // Only run on latest OSS version. The examples should be updated to reflect their latest working config.
				}
			},
		},
		{
			category: "Grafana OSS",
			testCheck: func(t *testing.T, filename string) {
				if strings.Contains(filename, "sso_settings") {
					t.Skip() // TODO: Fix the tests to run on local instances
				} else {
					testutils.CheckOSSTestsEnabled(t, ">=11.0.0") // Only run on latest OSS version. The examples should be updated to reflect their latest working config.
				}
			},
		},
		{
			category: "Grafana Enterprise",
			testCheck: func(t *testing.T, filename string) {
				switch {
				case strings.Contains(filename, "grafana_data_source_config_lbac_rules"):
					// TODO: Fix the example to work with import.
					//
					// It looks like the test setup fails because the resource is imported but the rules don't actually exist, so the following refresh-after-apply that calls the `.Read` method fails.
					//
					// examples_test.go:148: Step 1/1 error: Error running post-apply refresh: exit status 1
					//
					//   Error: Failed to get LBAC rules
					//
					//     with grafana_data_source_config_lbac_rules.test_rule,
					//     on terraform_plugin_test.tf line 18, in resource "grafana_data_source_config_lbac_rules" "test_rule":
					//     18: resource "grafana_data_source_config_lbac_rules" "test_rule" {
					//
					//   Could not read LBAC rules for datasource "feg5jgkkikg00e": [GET
					//   /datasources/uid/{uid}/lbac/teams][500]
					//   getTeamLBACRulesApiInternalServerError {"message":"Validation error, invalid
					//   format of team LBAC rules"}
					t.Skip()
				case strings.Contains(filename, "grafana_scim_config"):
					testutils.CheckEnterpriseTestsEnabled(t, ">=12.0.0")
				default:
					testutils.CheckEnterpriseTestsEnabled(t, ">=11.0.0") // Only run on latest version
				}
			},
		},

		{
			category: "Machine Learning",
			testCheck: func(t *testing.T, filename string) {
				t.Skip() // TODO: Make all examples work
				testutils.CheckCloudInstanceTestsEnabled(t)
			},
		},
		{
			category: "SLO",
			testCheck: func(t *testing.T, filename string) {
				testutils.CheckCloudInstanceTestsEnabled(t)
			},
		},
		{
			category: "OnCall",
			testCheck: func(t *testing.T, filename string) {
				t.Skip() // TODO: Make all examples work
				testutils.CheckCloudInstanceTestsEnabled(t)
			},
		},
		{
			category: "Cloud",
			testCheck: func(t *testing.T, filename string) {
				t.Skip() // TODO: Make all examples work
				testutils.CheckCloudAPITestsEnabled(t)
			},
		},
		{
			category: "Synthetic Monitoring",
			testCheck: func(t *testing.T, filename string) {
				t.Skip() // TODO: Make all examples work
				testutils.CheckCloudInstanceTestsEnabled(t)
			},
		},
		{
			category: "k6",
			testCheck: func(t *testing.T, filename string) {
				t.Skip() // TODO: Make all examples work
				testutils.CheckCloudInstanceTestsEnabled(t)
			},
		},
		{
			category: "Cloud Provider",
			testCheck: func(t *testing.T, filename string) {
				t.Skip() // TODO: Make all examples work
				testutils.CheckCloudInstanceTestsEnabled(t)
			},
		},
		{
			category: "Connections",
			testCheck: func(t *testing.T, filename string) {
				// This satisfies the CI requirement to have this category present.
				// The examples in Connections metrics endpoint cannot be tested remotely because the metrics scrape url
				// is for demonstrative purposes only; it's not a real metrics scrape-able endpoint.
				t.Skip()
			},
		},
		{
			category: "Fleet Management",
			testCheck: func(t *testing.T, filename string) {
				t.Skip()
				testutils.CheckCloudInstanceTestsEnabled(t)
			},
		},
		{
			category: "Frontend Observability",
			testCheck: func(t *testing.T, filename string) {
				t.Skip() // TODO: Make all examples work
				testutils.CheckCloudInstanceTestsEnabled(t)
			},
		},
	} {
		// Get all the filenames for all resource examples for this category
		filenames := []string{}

		for _, r := range provider.Resources() {
			if _, ok := resourceMap[r.Name]; !ok {
				resourceMap[r.Name] = false
			}
			if string(r.Category) != testDef.category {
				continue
			}
			resourceMap[r.Name] = true
			filenames = append(filenames, filepath.Join("resources", r.Name, "resource.tf"))
		}

		for _, d := range provider.DataSources() {
			if _, ok := datasourceMap[d.Name]; !ok {
				datasourceMap[d.Name] = false
			}
			if string(d.Category) != testDef.category {
				continue
			}
			datasourceMap[d.Name] = true
			filenames = append(filenames, filepath.Join("data-sources", d.Name, "data-source.tf"))
		}
		sort.Strings(filenames)

		// Test each example in the category. We're only interested to see if it applies without errors.
		t.Run(testDef.category, func(t *testing.T) {
			for _, filename := range filenames {
				t.Run(filename, func(t *testing.T) {
					testDef.testCheck(t, filename)
					resource.Test(t, resource.TestCase{
						ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
						Steps: []resource.TestStep{{
							Config: testutils.TestAccExample(t, filename),
						}},
					})
				})
			}
		})
	}

	for name, tested := range resourceMap {
		if !tested {
			t.Errorf("Resource %s was not tested", name)
		}
	}

	for name, tested := range datasourceMap {
		if !tested {
			t.Errorf("DataSource %s was not tested", name)
		}
	}
}
