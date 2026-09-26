package grafana_test

import (
	"fmt"
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDatasourceFolder_permissions(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=10.3.0")

	var test models.Folder
	randomName := acctest.RandStringFromCharSet(6, acctest.CharSetAlpha)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			folderCheckExists.destroyed(&test, nil),
		),
		Steps: []resource.TestStep{
			{
				Config: testFolderPermissionData(randomName),
				Check: resource.ComposeTestCheckFunc(
					folderCheckExists.exists("grafana_folder.test", &test),
					resource.TestCheckResourceAttr("data.grafana_folder_permission.test", "folder_uid", randomName),
					resource.TestMatchResourceAttr("data.grafana_folder_permission.test", "id", defaultOrgIDRegexp),
					resource.TestCheckResourceAttr("data.grafana_folder_permission.test", "permissions.#", "3"),
					resource.TestCheckResourceAttr("data.grafana_folder_permission.test", "permissions.0.permission", "Admin"),
					resource.TestCheckResourceAttr("data.grafana_folder_permission.test", "permissions.1.role", "Editor"),
					resource.TestCheckResourceAttr("data.grafana_folder_permission.test", "permissions.1.permission", "Edit"),
					resource.TestCheckResourceAttr("data.grafana_folder_permission.test", "permissions.2.role", "Viewer"),
					resource.TestCheckResourceAttr("data.grafana_folder_permission.test", "permissions.2.permission", "View"),
				),
			},
		},
	})
}

func testFolderPermissionData(name string) string {
	return fmt.Sprintf(`
resource "grafana_folder" "test" {
	title = "%[1]s"
	uid   = "%[1]s"
}
	
data "grafana_folder_permission" "test" {
	folder_uid = grafana_folder.test.uid
}
	
`, name)
}
