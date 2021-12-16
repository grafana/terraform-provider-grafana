//go:build oss
// +build oss

package grafana

import (
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDatasourceFolder(t *testing.T) {
	var folder gapi.Folder
	checks := []resource.TestCheckFunc{
		testAccFolderCheckExists("grafana_folder.test", &folder),
		resource.TestCheckResourceAttr(
			"data.grafana_folder.from_title", "title", "test-folder",
		),
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccFolderCheckDestroy(&folder),
		Steps: []resource.TestStep{
			{
				Config: testAccExample(t, "data-sources/grafana_folder/data-source.tf"),
				Check:  resource.ComposeTestCheckFunc(checks...),
			},
		},
	})
}
