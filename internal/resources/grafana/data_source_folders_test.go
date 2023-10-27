package grafana_test

import (
	"testing"

	goapi "github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDatasourceFolders(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	var folderA goapi.Folder
	var folderB goapi.Folder
	titleBase := "test-folder-"
	uidBase := "test-ds-folder-uid-"
	checks := []resource.TestCheckFunc{
		testAccFolderCheckExists("grafana_folder.test_a", &folderA),
		testAccFolderCheckExists("grafana_folder.test_b", &folderB),
		resource.TestCheckResourceAttr(
			"data.grafana_folders.test", "folders.#", "2",
		),
		resource.TestCheckTypeSetElemNestedAttrs("data.grafana_folders.test", "folders.*", map[string]string{
			"uid":   uidBase + "a",
			"title": titleBase + "a",
		}),
		resource.TestCheckTypeSetElemNestedAttrs("data.grafana_folders.test", "folders.*", map[string]string{
			"uid":   uidBase + "b",
			"title": titleBase + "b",
		}),
	}

	// TODO: Make parallelizable
	resource.Test(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccFolderCheckDestroy(&folderA, 0),
			testAccFolderCheckDestroy(&folderB, 0),
		),
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "data-sources/grafana_folders/data-source.tf"),
				Check:  resource.ComposeTestCheckFunc(checks...),
			},
		},
	})
}
