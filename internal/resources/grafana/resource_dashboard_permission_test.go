package grafana_test

import (
	"fmt"
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccDashboardPermission_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.0.0")

	randomName := acctest.RandString(6)
	var (
		dashboard models.DashboardFullWithMeta
		team      models.TeamDTO
		user      models.UserProfileDTO
		sa        models.ServiceAccountDTO
	)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDashboardPermissionConfig(randomName, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					dashboardCheckExists.exists("grafana_dashboard.testDashboard", &dashboard),
					teamCheckExists.exists("grafana_team.testTeam", &team),
					userCheckExists.exists("grafana_user.testAdminUser", &user),
					serviceAccountCheckExists.exists("grafana_service_account.test", &sa),

					resource.TestCheckResourceAttr("grafana_dashboard_permission.testPermission", "permissions.#", "5"),
					checkDashboardPermissionsSet(&dashboard, &team, &user, &sa, false),
				),
			},
			{
				ImportState:       true,
				ResourceName:      "grafana_dashboard_permission.testPermission",
				ImportStateVerify: true,
			},
			// Test remove permissions by not setting any permissions
			{
				Config: testAccDashboardPermissionConfig(randomName, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					dashboardCheckExists.exists("grafana_dashboard.testDashboard", &dashboard),
					resource.TestCheckResourceAttr("grafana_dashboard_permission.testPermission", "permissions.#", "0"),
					checkDashboardPermissionsEmpty(&dashboard, false),
				),
			},
			// Reapply permissions
			{
				Config: testAccDashboardPermissionConfig(randomName, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					dashboardCheckExists.exists("grafana_dashboard.testDashboard", &dashboard),
					teamCheckExists.exists("grafana_team.testTeam", &team),
					userCheckExists.exists("grafana_user.testAdminUser", &user),
					serviceAccountCheckExists.exists("grafana_service_account.test", &sa),

					resource.TestCheckResourceAttr("grafana_dashboard_permission.testPermission", "permissions.#", "5"),
					checkDashboardPermissionsSet(&dashboard, &team, &user, &sa, false),
				),
			},
			// Test remove permissions by removing the resource
			{
				Config: testutils.WithoutResource(t, testAccDashboardPermissionConfig(randomName, true), "grafana_dashboard_permission.testPermission"),
				Check: resource.ComposeAggregateTestCheckFunc(
					dashboardCheckExists.exists("grafana_dashboard.testDashboard", &dashboard),
					checkDashboardPermissionsEmpty(&dashboard, false),
				),
			},
		},
	})
}

func checkDashboardPermissionsSet(dashboard *models.DashboardFullWithMeta, team *models.TeamDTO, user *models.UserProfileDTO, sa *models.ServiceAccountDTO, expectAdminPerm bool) resource.TestCheckFunc {
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

		return checkDashboardPermissions(dashboard, expectedPerms, expectAdminPerm)
	}
}

func checkDashboardPermissionsEmpty(dashboard *models.DashboardFullWithMeta, expectAdminPerm bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		return checkDashboardPermissions(dashboard, []*models.DashboardACLInfoDTO{}, expectAdminPerm)
	}
}

func checkDashboardPermissions(dashboard *models.DashboardFullWithMeta, expectedPerms []*models.DashboardACLInfoDTO, expectAdminPerm bool) error {
	if expectAdminPerm {
		expectedPerms = append(expectedPerms, &models.DashboardACLInfoDTO{
			UserID:         1,
			PermissionName: "Admin",
		})
	}

	client := grafanaTestClient()
	uid := dashboard.Dashboard.(map[string]any)["uid"].(string)
	resp, err := client.DashboardPermissions.GetDashboardPermissionsListByUID(uid)
	if err != nil {
		return fmt.Errorf("error getting dashboard permissions: %s", err)
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

func testAccDashboardPermissionConfig(name string, hasPermissions bool) string {
	perms := ""
	if hasPermissions {
		perms = `permissions {
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
		  }`
	}

	return fmt.Sprintf(`
resource "grafana_dashboard" "testDashboard" {
    config_json = <<EOT
{
    "title": "%[1]s",
    "id": 14,
    "version": "43",
    "uid": "%[1]s"
}
EOT
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
	role 	    = "Editor"
}

resource "grafana_dashboard_permission" "testPermission" {
  dashboard_uid = grafana_dashboard.testDashboard.uid
  %[2]s
}
`, name, perms)
}
