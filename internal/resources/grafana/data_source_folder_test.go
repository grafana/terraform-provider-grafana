package grafana_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDatasourceFolder_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	var folder models.Folder
	checks := []resource.TestCheckFunc{
		folderCheckExists.exists("grafana_folder.test", &folder),
		resource.TestCheckResourceAttr("data.grafana_folder.from_title", "title", "test-folder"),
		resource.TestMatchResourceAttr("data.grafana_folder.from_title", "id", defaultOrgIDRegexp),
		resource.TestCheckResourceAttr("data.grafana_folder.from_title", "uid", "test-ds-folder-uid"),
		resource.TestCheckResourceAttr("data.grafana_folder.from_title", "url", strings.TrimRight(os.Getenv("GRAFANA_URL"), "/")+"/dashboards/f/test-ds-folder-uid/test-folder"),
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             folderCheckExists.destroyed(&folder, nil),
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "data-sources/grafana_folder/data-source.tf"),
				Check:  resource.ComposeTestCheckFunc(checks...),
			},
		},
	})
}

func TestAccDatasourceFolder_nested(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=10.3.0")

	var parent models.Folder
	var child models.Folder
	randomName := acctest.RandStringFromCharSet(6, acctest.CharSetAlpha)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			folderCheckExists.destroyed(&parent, nil),
			folderCheckExists.destroyed(&child, nil),
		),
		Steps: []resource.TestStep{
			{
				Config: testNestedFolderData(randomName),
				Check: resource.ComposeTestCheckFunc(
					folderCheckExists.exists("grafana_folder.parent", &parent),
					folderCheckExists.exists("grafana_folder.child", &child),
					resource.TestCheckResourceAttr("data.grafana_folder.parent", "title", randomName),
					resource.TestCheckResourceAttr("data.grafana_folder.parent", "uid", randomName),
					resource.TestMatchResourceAttr("data.grafana_folder.parent", "id", defaultOrgIDRegexp),
					resource.TestCheckResourceAttr("data.grafana_folder.parent", "parent_folder_uid", ""),

					resource.TestCheckResourceAttr("data.grafana_folder.child", "title", randomName+"-child"),
					resource.TestCheckResourceAttr("data.grafana_folder.child", "uid", randomName+"-child"),
					resource.TestMatchResourceAttr("data.grafana_folder.child", "id", defaultOrgIDRegexp),
					resource.TestCheckResourceAttr("data.grafana_folder.child", "parent_folder_uid", randomName),
				),
			},
		},
	})
}

func testNestedFolderData(name string) string {
	return fmt.Sprintf(`
resource "grafana_folder" "parent" {
	title = "%[1]s"
	uid   = "%[1]s"
}
	
resource "grafana_folder" "child" {
	title = "%[1]s-child"
	uid   = "%[1]s-child"
	parent_folder_uid = grafana_folder.parent.uid
}
	
data "grafana_folder" "parent" {
	title = grafana_folder.parent.title
}
	
data "grafana_folder" "child" {
	title = grafana_folder.child.title
}
`, name)
}
