package grafana_test

import (
	"os"
	"strings"
	"testing"

	goapi "github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/grafana/terraform-provider-grafana/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDatasourceFolder(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	var folder goapi.Folder
	checks := []resource.TestCheckFunc{
		testAccFolderCheckExists("grafana_folder.test", &folder),
		resource.TestCheckResourceAttr(
			"data.grafana_folder.from_title", "title", "test-folder",
		),
		resource.TestMatchResourceAttr(
			"data.grafana_folder.from_title", "id", common.IDRegexp,
		),
		resource.TestCheckResourceAttr(
			"data.grafana_folder.from_title", "uid", "test-ds-folder-uid",
		),
		resource.TestCheckResourceAttr(
			"data.grafana_folder.from_title", "url", strings.TrimRight(os.Getenv("GRAFANA_URL"), "/")+"/dashboards/f/test-ds-folder-uid/test-folder",
		),
	}

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      testAccFolderCheckDestroy(&folder, 0),
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "data-sources/grafana_folder/data-source.tf"),
				Check:  resource.ComposeTestCheckFunc(checks...),
			},
		},
	})
}
