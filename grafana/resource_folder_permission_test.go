package grafana

import (
	"fmt"
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccFolderPermission_basic(t *testing.T) {
	folderUID := "uninitialized"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccFolderPermissionCheckDestroy(),
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

		client := testAccProvider.Meta().(*gapi.Client)

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
		client := testAccProvider.Meta().(*gapi.Client)
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
		//you can't really destroy folder permissions so nothing to check for
		return nil
	}
}

func testAccFolderPermissionsRemoval(permissions *gapi.FolderPermission) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		//since the permissions aren't deleted, let's just check if we have empty permissions
		client := testAccProvider.Meta().(*gapi.Client)
		newPermissions, err := client.FolderPermissions(permissions.FolderUID)
		if err != nil {
			return fmt.Errorf(err.Error())
		}
		if len(newPermissions) > 0 {
			return fmt.Errorf("Permissions still exist for folder")
		}
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
