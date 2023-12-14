package grafana_test

import (
	"fmt"
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/internal/resources/grafana"
	"github.com/grafana/terraform-provider-grafana/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccFolderPermission_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.0.0") // Folder permissions only work for service accounts in Grafana 9+, so we're just not testing versions before 9.

	var (
		folder models.Folder
		team   models.TeamDTO
		user   models.UserProfileDTO
		sa     models.ServiceAccountDTO
	)

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFolderPermissionConfig_Basic,
				Check: resource.ComposeAggregateTestCheckFunc(
					folderCheckExists.exists("grafana_folder.testFolder", &folder),
					teamCheckExists.exists("grafana_team.testTeam", &team),
					userCheckExists.exists("grafana_user.testAdminUser", &user),
					serviceAccountCheckExists.exists("grafana_service_account.test", &sa),
					resource.TestCheckResourceAttr("grafana_folder_permission.testPermission", "permissions.#", "5"),
					checkFolderPermissionsSet(&folder, &team, &user, &sa),
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
					folderCheckExists.exists("grafana_folder.testFolder", &folder),
					resource.TestCheckResourceAttr("grafana_folder_permission.testPermission", "permissions.#", "0"),
					checkFolderPermissionsEmpty(&folder),
				),
			},
			// Reapply permissions
			{
				Config: testAccFolderPermissionConfig_Basic,
				Check: resource.ComposeAggregateTestCheckFunc(
					folderCheckExists.exists("grafana_folder.testFolder", &folder),
					teamCheckExists.exists("grafana_team.testTeam", &team),
					userCheckExists.exists("grafana_user.testAdminUser", &user),
					serviceAccountCheckExists.exists("grafana_service_account.test", &sa),
					resource.TestCheckResourceAttr("grafana_folder_permission.testPermission", "permissions.#", "5"),
					checkFolderPermissionsSet(&folder, &team, &user, &sa),
				),
			},
			// Test remove permissions by removing the resource
			{
				Config: testutils.WithoutResource(t, testAccFolderPermissionConfig_Basic, "grafana_folder_permission.testPermission"),
				Check: resource.ComposeAggregateTestCheckFunc(
					folderCheckExists.exists("grafana_folder.testFolder", &folder),
					checkFolderPermissionsEmpty(&folder),
				),
			},
		},
	})
}

func checkFolderPermissionsSet(folder *models.Folder, team *models.TeamDTO, user *models.UserProfileDTO, sa *models.ServiceAccountDTO) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		expectedPerms := []*models.DashboardACLInfoDTO{
			{
				Role:           "Viewer",
				PermissionName: "View",
			},
			{
				Role:           "Editor",
				PermissionName: "Edit",
			},
			{
				TeamID:         team.ID,
				PermissionName: "View",
			},
			{
				UserID:         user.ID,
				PermissionName: "Admin",
			},
			{
				UserID:         sa.ID,
				PermissionName: "Admin",
			},
		}

		return checkFolderPermissions(folder, expectedPerms)
	}
}

func checkFolderPermissionsEmpty(folder *models.Folder) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		return checkFolderPermissions(folder, []*models.DashboardACLInfoDTO{})
	}
}

func checkFolderPermissions(folder *models.Folder, expectedPerms []*models.DashboardACLInfoDTO) error {
	client := grafana.OAPIGlobalClient(testutils.Provider.Meta())
	resp, err := client.FolderPermissions.GetFolderPermissionList(folder.UID)
	if err != nil {
		return fmt.Errorf("error getting folder permissions: %s", err)
	}
	gotPerms := resp.Payload

	if len(gotPerms) != len(expectedPerms) {
		return fmt.Errorf("got %d perms, expected %d", len(gotPerms), len(expectedPerms))
	}

	for _, expectedPerm := range expectedPerms {
		found := false
		for _, gotPerm := range gotPerms {
			if gotPerm.PermissionName == expectedPerm.PermissionName &&
				gotPerm.Role == expectedPerm.Role &&
				gotPerm.UserID == expectedPerm.UserID &&
				gotPerm.TeamID == expectedPerm.TeamID {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("didn't find permission matching %+v", expectedPerm)
		}
	}

	return nil
}

const testAccFolderPermissionConfig_Common = `
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
`

const testAccFolderPermissionConfig_Basic = testAccFolderPermissionConfig_Common + `
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

const testAccFolderPermissionConfig_NoPermissions = testAccFolderPermissionConfig_Common + `
resource "grafana_folder_permission" "testPermission" {
  folder_uid = grafana_folder.testFolder.uid
}
`
