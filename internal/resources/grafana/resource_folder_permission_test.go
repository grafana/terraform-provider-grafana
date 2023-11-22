package grafana_test

import (
	"fmt"
	"testing"

	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/grafana/terraform-provider-grafana/internal/resources/grafana"
	"github.com/grafana/terraform-provider-grafana/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccFolderPermission_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.0.0") // Folder permissions only work for service accounts in Grafana 9+, so we're just not testing versions before 9.

	folderUID := "uninitialized"

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      testAccFolderPermissionCheckDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testAccFolderPermissionConfig_Basic,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccFolderPermissionsCheckExists("grafana_folder_permission.testPermission", &folderUID),
					resource.TestCheckResourceAttr("grafana_folder_permission.testPermission", "permissions.#", "5"),
				),
			},
			{
				ResourceName:      "grafana_folder_permission.testPermission",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Test remove permissions by not setting any permissions
			{
				Config: testAccFolderPermissionConfig_NoPermissions,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccFolderPermissionsCheckEmpty(&folderUID),
				),
			},
			// Reapply permissions
			{
				Config: testAccFolderPermissionConfig_Basic,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccFolderPermissionsCheckExists("grafana_folder_permission.testPermission", &folderUID),
					resource.TestCheckResourceAttr("grafana_folder_permission.testPermission", "permissions.#", "5"),
				),
			},
			// Test remove permissions by removing the resource
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

		orgID, gotFolderUID := grafana.SplitOrgResourceID(rs.Primary.ID)
		client := testutils.Provider.Meta().(*common.Client).GrafanaAPI.WithOrgID(orgID)

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
		client := testutils.Provider.Meta().(*common.Client).GrafanaAPI
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

resource "grafana_service_account" "test" {
	name        = "terraform-test-service-account-folder-perms"
	role        = "Editor"
	is_disabled = false
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
  permissions {
	user_id    = grafana_service_account.test.id
	permission = "Admin"
  }
}
`

const testAccFolderPermissionConfig_NoPermissions = `
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

resource "grafana_service_account" "test" {
	name        = "terraform-test-service-account-folder-perms"
	role        = "Editor"
	is_disabled = false
}

resource "grafana_folder_permission" "testPermission" {
  folder_uid = grafana_folder.testFolder.uid
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
