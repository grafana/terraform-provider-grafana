package grafana_test

import (
	"fmt"
	"testing"

	"github.com/grafana/grafana-openapi-client-go/client/folders"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccFolderPermission_basic(t *testing.T) {
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
				Config: testAccFolderPermissionConfig_Basic(randomName),
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
				Config: testAccFolderPermissionConfig_Basic(randomName),
				Check: resource.ComposeAggregateTestCheckFunc(
					folderCheckExists.exists("grafana_folder.testFolder", &folder),
					resource.TestCheckResourceAttr("grafana_folder_permission.testPermission", "permissions.#", "5"),
				),
			},
			// Test remove permissions by not setting any permissions
			{
				Config: testAccFolderPermissionConfig_NoPermissions(randomName),
				Check: resource.ComposeAggregateTestCheckFunc(
					folderCheckExists.exists("grafana_folder.testFolder", &folder),
					resource.TestCheckResourceAttr("grafana_folder_permission.testPermission", "permissions.#", "0"),
					checkFolderPermissionsEmpty(&folder),
				),
			},
			// Reapply permissions
			{
				Config: testAccFolderPermissionConfig_Basic(randomName),
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
				Config: testutils.WithoutResource(t, testAccFolderPermissionConfig_Basic(randomName), "grafana_folder_permission.testPermission"),
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
	client := grafanaTestClient()
	resp, err := client.FolderPermissions.GetFolderPermissionList(folder.UID)
	if err != nil {
		return fmt.Errorf("error getting folder permissions: %s", err)
	}
	var gotPerms []models.DashboardACLInfoDTO
	for _, perm := range resp.Payload {
		if perm.UserID == 1 { // Ignore the admin user (that created the folder)
			continue
		}
		gotPerms = append(gotPerms, *perm)
	}

	if len(gotPerms) != len(expectedPerms) {
		return fmt.Errorf("got %d perms, expected %d. Got %+v", len(gotPerms), len(expectedPerms), gotPerms)
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

func testAccFolderPermissionConfig_Common(name string) string {
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
`, name)
}

func testAccFolderPermissionConfig_Basic(name string) string {
	return testAccFolderPermissionConfig_Common(name) + `
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
}

func testAccFolderPermissionConfig_NoPermissions(name string) string {
	return testAccFolderPermissionConfig_Common(name) + `
	resource "grafana_folder_permission" "testPermission" {
	  folder_uid = grafana_folder.testFolder.uid
	}
	`
}
