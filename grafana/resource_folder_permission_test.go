//go:build oss
// +build oss

package grafana

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccFolderPermission_basic(t *testing.T) {
	folderUID := "uninitialized"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccFolderPermissionCheckDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testAccFolderPermissionConfig_Basic,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccFolderPermissionsCheckExists("grafana_folder_permission.testPermission", &folderUID),
					resource.TestCheckResourceAttr("grafana_folder_permission.testPermission", "permissions.#", "4"),
				),
			},
			{
				Config: testAccFolderPermissionConfig_Remove,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccFolderPermissionsCheckEmpty(&folderUID),
				),
			},
		},
	})
}

func testAccFolderPermissionsCheckExists(rn string, folderUID *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("Resource not found: %s\n %#v", rn, s.RootModule().Resources)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("Resource id not set")
		}

		client := testAccProvider.Meta().(*client).gapi

		gotFolderUID := rs.Primary.ID
		_, err := client.FolderPermissions(gotFolderUID)
		if err != nil {
			return fmt.Errorf("Error getting folder permissions: %s", err)
		}

		*folderUID = gotFolderUID

		return nil
	}
}

func testAccFolderPermissionsCheckEmpty(folderUID *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*client).gapi
		permissions, err := client.FolderPermissions(*folderUID)
		if err != nil {
			return fmt.Errorf("Error getting folder permissions %s: %s", *folderUID, err)
		}
		if len(permissions) > 0 {
			return fmt.Errorf("Permissions were not empty when expected")
		}

		return nil
	}
}

func testAccFolderPermissionCheckDestroy() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// you can't really destroy folder permissions so nothing to check for
		return nil
	}
}

const testAccFolderPermissionConfig_Basic = `
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
const testAccFolderPermissionConfig_Remove = `
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
`
