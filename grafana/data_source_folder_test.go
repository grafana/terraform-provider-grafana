package grafana

import (
	"os"
	"strings"
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDatasourceFolder(t *testing.T) {
	CheckOSSTestsEnabled(t)

	var folder gapi.Folder
	checks := []resource.TestCheckFunc{
		testAccFolderCheckExists("grafana_folder.test", &folder),
		resource.TestCheckResourceAttr(
			"data.grafana_folder.from_title", "title", "test-folder",
		),
		resource.TestMatchResourceAttr(
			"data.grafana_folder.from_title", "id", idRegexp,
		),
		resource.TestCheckResourceAttr(
			"data.grafana_folder.from_title", "uid", "test-ds-folder-uid",
		),
		resource.TestCheckResourceAttr(
			"data.grafana_folder.from_title", "url", strings.TrimRight(os.Getenv("GRAFANA_URL"), "/")+"/dashboards/f/test-ds-folder-uid/test-folder",
		),
	}

	resource.ParallelTest(t, resource.TestCase{
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
