package grafana_test

import (
	"fmt"
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v2/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccDashboardPermission_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.0.0")

	var (
		dashboard models.DashboardFullWithMeta
		team      models.TeamDTO
		user      models.UserProfileDTO
		sa        models.ServiceAccountDTO
	)

	// TODO: Make parallelizable
	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDashboardPermissionConfig(true, true),
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
				Config: testAccDashboardPermissionConfig(true, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					dashboardCheckExists.exists("grafana_dashboard.testDashboard", &dashboard),
					resource.TestCheckResourceAttr("grafana_dashboard_permission.testPermission", "permissions.#", "0"),
					checkDashboardPermissionsEmpty(&dashboard, false),
				),
			},
			// Reapply permissions
			{
				Config: testAccDashboardPermissionConfig(true, true),
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
				Config: testutils.WithoutResource(t, testAccDashboardPermissionConfig(true, true), "grafana_dashboard_permission.testPermission"),
				Check: resource.ComposeAggregateTestCheckFunc(
					dashboardCheckExists.exists("grafana_dashboard.testDashboard", &dashboard),
					checkDashboardPermissionsEmpty(&dashboard, false),
				),
			},
		},
	})
}

// Testing the deprecated case of using a dashboard ID instead of a dashboard UID
// TODO: Remove in next major version
func TestAccDashboardPermission_fromDashboardID(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.0.0")

	var (
		dashboard models.DashboardFullWithMeta
		team      models.TeamDTO
		user      models.UserProfileDTO
		sa        models.ServiceAccountDTO
	)

	// TODO: Make parallelizable
	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDashboardPermissionConfig(false, true),
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
	uid := dashboard.Dashboard.(map[string]interface{})["uid"].(string)
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

func testAccDashboardPermissionConfig(refDashboardByUID bool, hasPermissions bool) string {
	ref := "dashboard_id = grafana_dashboard.testDashboard.dashboard_id"
	if refDashboardByUID {
		ref = "dashboard_uid = grafana_dashboard.testDashboard.uid"
	}

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
    "title": "Terraform Dashboard Permission Test Dashboard",
    "id": 14,
    "version": "43",
    "uid": "someuid"
}
EOT
}

resource "grafana_team" "testTeam" {
  name = "terraform-test-team-permissions"
}

resource "grafana_user" "testAdminUser" {
  email    = "terraform-test-dashboard-permissions@localhost"
  name     = "Terraform Test Dashboard Permissions"
  login    = "ttdp"
  password = "zyx987"
}

resource "grafana_service_account" "test" {
	name        = "terraform-test-service-account-dashboard-perms"
	role 	    = "Editor"
}

resource "grafana_dashboard_permission" "testPermission" {
  %s
  %s
}
`, ref, perms)
}
