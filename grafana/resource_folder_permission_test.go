package grafana

import (
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccFolderPermission_basic(t *testing.T) {
	var folderPermission gapi.FolderPermission

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccFolderPermissionCheckDestroy(&folderPermission),
		Steps: []resource.TestStep{
			{
				Config: testAccFolderPermissionConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_folder_permission.testPermission", "permissions.#", "4"),
				),
			},
		},
	})
}

func testAccFolderPermissionCheckDestroy(a *gapi.FolderPermission) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		//you can't really destroy folder permissions so nothing to check for
		return nil
	}
}

const testAccFolderPermissionConfig = `
resource "grafana_folder" "testFolder" {
  title = "terraform-test-folder-permissions"
}

resource "grafana_team" "testTeam" {
  name = "terraform-test-team-permissions"
}
  
resource "grafana_user" "testAdminUser" {
  email    = "terraform-test-permissions@localhost"
  name     = "Terraform Test Permissions"
  login    = "ttp"
  password = "zyx987"
}
  
resource "grafana_folder_permission" "testPermission" {
  folder_uid = grafana_folder.testFolder.uid
  permissions {
    role       = "Viewer"
    permission = "View"
  }
  permissions {
    role       = "Editor"
    permission = "Edit"
  }
  permissions {
    team_id    = grafana_team.testTeam.id
    permission = "View"
  }
  permissions {
    user_id    = grafana_user.testAdminUser.id
    permission = "Admin"
  }
}
`
