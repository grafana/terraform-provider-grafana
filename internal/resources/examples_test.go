package resources_test

import (
	"encoding/json"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/grafana/terraform-provider-grafana/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// This test makes sure all resources and datasources have examples and they are all valid.
func TestAccExamples(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping long test")
	}

	// Read the subcategories.json file
	// The subcategories file is a map expected to contain all resources and datasources
	var resourceCategories map[string]string
	categoriesFile, err := os.Open("../../tools/subcategories.json")
	if err != nil {
		t.Fatal(err)
	}
	defer categoriesFile.Close()
	err = json.NewDecoder(categoriesFile).Decode(&resourceCategories)
	if err != nil {
		t.Fatal(err)
	}

	testedResources := map[string]struct{}{}
	for _, testDef := range []struct {
		category  string
		testCheck func(*testing.T)
	}{
		{
			category: "Alerting",
			testCheck: func(t *testing.T) {
				testutils.CheckOSSTestsEnabled(t, ">=10.2.0") // Only run on latest OSS version. The examples should be updated to reflect their latest working config.
			},
		},
		{
			category: "Grafana OSS",
			testCheck: func(t *testing.T) {
				testutils.CheckOSSTestsEnabled(t, ">=10.2.0") // Only run on latest OSS version. The examples should be updated to reflect their latest working config.
			},
		},
		{
			category: "Grafana Enterprise",
			testCheck: func(t *testing.T) {
				testutils.CheckEnterpriseTestsEnabled(t)
			},
		},

		{
			category: "Machine Learning",
			testCheck: func(t *testing.T) {
				t.Skip() // TODO: Make all examples work
				testutils.CheckCloudInstanceTestsEnabled(t)
			},
		},
		{
			category: "SLO",
			testCheck: func(t *testing.T) {
				testutils.CheckCloudInstanceTestsEnabled(t)
			},
		},
		{
			category: "OnCall",
			testCheck: func(t *testing.T) {
				t.Skip() // TODO: Make all examples work
				testutils.CheckCloudInstanceTestsEnabled(t)
			},
		},
		{
			category: "Cloud",
			testCheck: func(t *testing.T) {
				t.Skip() // TODO: Make all examples work
				testutils.CheckCloudAPITestsEnabled(t)
			},
		},
		{
			category: "Synthetic Monitoring",
			testCheck: func(t *testing.T) {
				t.Skip() // TODO: Make all examples work
				testutils.CheckCloudInstanceTestsEnabled(t)
			},
		},
	} {
		// Get all the filenames for all resource examples for this category
		filenames := []string{}
		for rName, category := range resourceCategories {
			filename := rName
			// grafana_ is omitted in the resource names but we need it in the example file names
			filename = strings.Replace(filename, "data-sources/", "data-sources/grafana_", 1)
			filename = strings.Replace(filename, "resources/", "resources/grafana_", 1)
			// The file name is (data-source|resource).tf
			if strings.HasPrefix(filename, "data-sources") {
				filename += "/data-source.tf"
			}
			if strings.HasPrefix(filename, "resources") {
				filename += "/resource.tf"
			}
			if category == testDef.category {
				filenames = append(filenames, filename)
				testedResources[rName] = struct{}{}
			}
		}
		sort.Strings(filenames)

		// Test each example in the category. We're only interested to see if it applies without errors.
		t.Run(testDef.category, func(t *testing.T) {
			testDef.testCheck(t)

			for _, filename := range filenames {
				t.Run(filename, func(t *testing.T) {
					resource.Test(t, resource.TestCase{
						ProviderFactories: testutils.ProviderFactories,
						Steps: []resource.TestStep{{
							Config: testutils.TestAccExample(t, filename),
						}},
					})
				})
			}
		})
	}

	// Sanity check that we have all resources and datasources have been tested
	// remove this check because the SSO Settings API is not available yet in Grafana
	//for rName := range testutils.Provider.ResourcesMap {
	//	if _, ok := testedResources["resources/"+strings.TrimPrefix(rName, "grafana_")]; !ok {
	//		t.Errorf("Resource %s was not tested", rName)
	//	}
	//}
	for rName := range testutils.Provider.DataSourcesMap {
		if _, ok := testedResources["data-sources/"+strings.TrimPrefix(rName, "grafana_")]; !ok {
			t.Errorf("Datasource %s was not tested", rName)
		}
	}

	// Additional nice to have test. Check that there are no extras in the subcategories file
	for rName := range testedResources {
		if strings.HasPrefix(rName, "resources/") {
			rName = "grafana_" + strings.TrimPrefix(rName, "resources/")
			if _, ok := testutils.Provider.ResourcesMap[rName]; !ok {
				t.Errorf("Resource %s was tested but is not declared by the provider", rName)
			}
		}

		if strings.HasPrefix(rName, "data-sources/") {
			rName = "grafana_" + strings.TrimPrefix(rName, "data-sources/")
			if _, ok := testutils.Provider.DataSourcesMap[rName]; !ok {
				t.Errorf("Datasource %s was tested but is not declared by the provider", rName)
			}
		}
	}
}
