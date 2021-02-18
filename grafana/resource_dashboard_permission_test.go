package grafana

import (
	"fmt"
	"strconv"
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDashboardPermission_basic(t *testing.T) {
	dashboardID := int64(-1)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccDashboardPermissionCheckDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testAccDashboardPermissionConfig_Basic,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccDashboardPermissionsCheckExists("grafana_dashboard_permission.testPermission", &dashboardID),
					resource.TestCheckResourceAttr("grafana_dashboard_permission.testPermission", "permissions.#", "4"),
				),
			},
			{
				Config: testAccDashboardPermissionConfig_Remove,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccDashboardPermissionsCheckEmpty(&dashboardID),
				),
			},
		},
	})
}

func testAccDashboardPermissionsCheckExists(rn string, dashboardID *int64) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("Resource not found: %s\n %#v", rn, s.RootModule().Resources)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("Resource id not set")
		}

		client := testAccProvider.Meta().(*gapi.Client)

		gotDashboardID, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
		if err != nil {
			return fmt.Errorf("dashboard id is malformed")
		}

		_, err = client.DashboardPermissions(gotDashboardID)
		if err != nil {
			return fmt.Errorf("Error getting dashboard permissions: %s", err)
		}

		*dashboardID = gotDashboardID

		return nil
	}
}

func testAccDashboardPermissionsCheckEmpty(dashboardID *int64) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*gapi.Client)
		permissions, err := client.DashboardPermissions(*dashboardID)
		if err != nil {
			return fmt.Errorf("Error getting dashboard permissions %d: %s", *dashboardID, err)
		}
		if len(permissions) > 0 {
			return fmt.Errorf("Permissions were not empty when expected")
		}

		return nil
	}
}

func testAccDashboardPermissionCheckDestroy() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		//you can't really destroy dashboard permissions so nothing to check for
		return nil
	}
}

func testAccDashboardPermissionsRemoval(permissions *gapi.DashboardPermission) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		//since the permissions aren't deleted, let's just check if we have empty permissions
		client := testAccProvider.Meta().(*gapi.Client)
		newPermissions, err := client.DashboardPermissions(permissions.DashboardID)
		if err != nil {
			return fmt.Errorf(err.Error())
		}
		if len(newPermissions) > 0 {
			return fmt.Errorf("Permissions still exist for dashboard")
		}
		return nil
	}
}

const testAccDashboardPermissionConfig_Basic = `
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

resource "grafana_dashboard_permission" "testPermission" {
  dashboard_id = grafana_dashboard.testDashboard.dashboard_id
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
const testAccDashboardPermissionConfig_Remove = `
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
  name = "terraform-test-team-dashboard-permissions"
}

resource "grafana_user" "testAdminUser" {
  email    = "terraform-test-dashboard-permissions@localhost"
  name     = "Terraform Test Dashboard Permissions"
  login    = "ttdp"
  password = "zyx987"
}
`
