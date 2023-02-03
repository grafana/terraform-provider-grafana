package provider

import (
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/provider/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDatasourceFolders(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	var folderA gapi.Folder
	var folderB gapi.Folder
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
			testAccFolderCheckDestroy(&folderA),
			testAccFolderCheckDestroy(&folderB),
		),
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "data-sources/grafana_folders/data-source.tf"),
				Check:  resource.ComposeTestCheckFunc(checks...),
			},
		},
	})
}
