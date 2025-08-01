package grafana_test

import (
	"fmt"
	"testing"

	"github.com/grafana/grafana-openapi-client-go/client/folders"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccFolderPermissionItem_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.0.0") // Folder permissions only work for service accounts in Grafana 9+, so we're just not testing versions before 9.

	var (
		folder     models.Folder
		team       models.TeamDTO
		user       models.UserProfileDTO
		sa         models.ServiceAccountDTO
		randomName = acctest.RandString(6)
	)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFolderPermissionItemConfig(randomName),
				Check: resource.ComposeAggregateTestCheckFunc(
					folderCheckExists.exists("grafana_folder.testFolder", &folder),
					teamCheckExists.exists("grafana_team.testTeam", &team),
					userCheckExists.exists("grafana_user.testAdminUser", &user),
					serviceAccountCheckExists.exists("grafana_service_account.test", &sa),
					checkFolderPermissionsSet(&folder, &team, &user, &sa), // Same check as in the full folder permission test
				),
			},
			{
				ResourceName:      "grafana_folder_permission_item.role_viewer",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      "grafana_folder_permission_item.team_viewer",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      "grafana_folder_permission_item.user_admin",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      "grafana_folder_permission_item.sa_admin",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Test delete the folder and check that TF sees a difference
			{
				PreConfig: func() {
					client := grafanaTestClient()
					params := folders.NewDeleteFolderParams().WithFolderUID(folder.UID)
					_, err := client.Folders.DeleteFolder(params)
					if err != nil {
						t.Fatalf("error deleting folder: %s", err)
					}
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
			},
			// Write back the folder to check that TF can reconcile
			{
				Config: testAccFolderPermissionItemConfig(randomName),
				Check: resource.ComposeAggregateTestCheckFunc(
					folderCheckExists.exists("grafana_folder.testFolder", &folder),
				),
			},
			// Test remove permissions by removing the resources
			{
				Config: testutils.WithoutResource(t, testAccFolderPermissionItemConfig(randomName),
					"grafana_folder_permission_item.role_viewer",
					"grafana_folder_permission_item.role_editor",
					"grafana_folder_permission_item.team_viewer",
					"grafana_folder_permission_item.user_admin",
					"grafana_folder_permission_item.sa_admin",
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					folderCheckExists.exists("grafana_folder.testFolder", &folder),
					checkFolderPermissionsEmpty(&folder),
				),
			},
		},
	})
}

func testAccFolderPermissionItemConfig(name string) string {
	return fmt.Sprintf(`
resource "grafana_folder" "testFolder" {
	title = "%[1]s"
}
  
resource "grafana_team" "testTeam" {
	name = "%[1]s"
}
  
resource "grafana_user" "testAdminUser" {
	email    = "%[1]s@localhost"
	name     = "%[1]s"
	login    = "%[1]s"
	password = "zyx987"
}
  
resource "grafana_service_account" "test" {
	name        = "%[1]s"
	role        = "Editor"
	is_disabled = false
}

resource "grafana_folder_permission_item" "role_viewer" {
	folder_uid = grafana_folder.testFolder.uid
	role       = "Viewer"
	permission = "View"
}

resource "grafana_folder_permission_item" "role_editor" {
	folder_uid = grafana_folder.testFolder.uid
	role       = "Editor"
	permission = "Edit"
}

resource "grafana_folder_permission_item" "team_viewer" {
	folder_uid = grafana_folder.testFolder.uid
	team    = grafana_team.testTeam.id
	permission = "View"
}

resource "grafana_folder_permission_item" "user_admin" {
	folder_uid = grafana_folder.testFolder.uid
	user    = grafana_user.testAdminUser.id
	permission = "Admin"
}

resource "grafana_folder_permission_item" "sa_admin" {
	folder_uid = grafana_folder.testFolder.uid	
	user    = grafana_service_account.test.id
	permission = "Admin"
}`, name)
}
