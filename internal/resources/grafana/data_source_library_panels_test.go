package grafana_test

import (
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDatasourceLibraryPanels_basic(t *testing.T) {
	randomName := acctest.RandString(10)

	testCases := []struct {
		versionConstraint string
		replacements      map[string]string
	}{
		{
			versionConstraint: ">=8.0.0,<=11.0.0",
			replacements: map[string]string{
				"panelname": randomName,
				`"general"`: "null",
			},
		},
		{
			versionConstraint: ">11.0.0",
			replacements: map[string]string{
				"panelname": randomName,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.versionConstraint, func(t *testing.T) {
			testutils.CheckOSSTestsEnabled(t, tc.versionConstraint)

			resource.ParallelTest(t, resource.TestCase{
				ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: testutils.TestAccExampleWithReplace(t,
							"data-sources/grafana_library_panels/data-source.tf",
							tc.replacements,
						),
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckTypeSetElemNestedAttrs("data.grafana_library_panels.all", "panels.*", map[string]string{
								"description":   "test description",
								"folder_uid":    "",
								"panels.0.name": randomName,
							}),
							resource.TestCheckTypeSetElemNestedAttrs("data.grafana_library_panels.all", "panels.*", map[string]string{
								"description":   "",
								"folder_uid":    randomName + "-folder",
								"panels.0.name": randomName + " In Folder",
							}),
						),
					},
				},
			})
		})
	}
}
