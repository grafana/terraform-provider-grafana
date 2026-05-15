package grafana_test

import (
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDatasourceLibraryPanels_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=8.0.0, <11.6.0") // TODO: library panels data source broken in Grafana 11.6+

	randomName := acctest.RandString(10)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExampleWithReplace(t, "data-sources/grafana_library_panels/data-source.tf", map[string]string{
					"panelname": randomName,
				}),
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
}
