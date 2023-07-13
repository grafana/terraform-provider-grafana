package grafana_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/grafana/terraform-provider-grafana/internal/resources/grafana"
	"github.com/grafana/terraform-provider-grafana/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccDashboardPermission_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)
	testutils.CheckOSSTestsSemver(t, ">=9.0.0") // Dashboard UIDs are only available as references in Grafana 9+

	dashboardUID := ""

	// TODO: Make parallelizable
	resource.Test(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDashboardPermissionConfig_Basic,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccDashboardPermissionsCheckExistsUID("grafana_dashboard_permission.testPermission", &dashboardUID),
					resource.TestCheckResourceAttr("grafana_dashboard_permission.testPermission", "permissions.#", "4"),
				),
			},
			{
				ImportState:       true,
				ResourceName:      "grafana_dashboard_permission.testPermission",
				ImportStateVerify: true,
			},
			{
				Config: testAccDashboardPermissionConfig_Remove,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccDashboardPermissionsCheckEmptyUID(&dashboardUID),
				),
			},
		},
	})
}

// Testing the deprecated case of using a dashboard ID instead of a dashboard UID
// TODO: Remove in next major version
func TestAccDashboardPermission_fromDashboardID(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	dashboardID := int64(-1)

	// TODO: Make parallelizable
	resource.Test(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDashboardPermissionConfig_FromID,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccDashboardPermissionsCheckExists("grafana_dashboard_permission.testPermission", &dashboardID),
					resource.TestCheckResourceAttr("grafana_dashboard_permission.testPermission", "permissions.#", "4"),
				),
			},
		},
	})
}

func testAccDashboardPermissionsCheckExistsUID(rn string, dashboardUID *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("Resource not found: %s\n %#v", rn, s.RootModule().Resources)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("Resource id not set")
		}

		orgID, gotDashboardUID := grafana.SplitOrgResourceID(rs.Primary.ID)
		client := testutils.Provider.Meta().(*common.Client).GrafanaAPI.WithOrgID(orgID)

		_, err := client.DashboardPermissionsByUID(gotDashboardUID)
		if err != nil {
			return fmt.Errorf("Error getting dashboard permissions: %s", err)
		}

		*dashboardUID = gotDashboardUID

		return nil
	}
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

		orgID, dashboardIDStr := grafana.SplitOrgResourceID(rs.Primary.ID)
		client := testutils.Provider.Meta().(*common.Client).GrafanaAPI.WithOrgID(orgID)

		gotDashboardID, err := strconv.ParseInt(dashboardIDStr, 10, 64)
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

func testAccDashboardPermissionsCheckEmptyUID(dashboardUID *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testutils.Provider.Meta().(*common.Client).GrafanaAPI
		permissions, err := client.DashboardPermissionsByUID(*dashboardUID)
		if err != nil {
			return fmt.Errorf("Error getting dashboard permissions %s: %s", *dashboardUID, err)
		}
		if len(permissions) > 0 {
			return fmt.Errorf("Permissions were not empty when expected")
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
  dashboard_uid = grafana_dashboard.testDashboard.uid
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

const testAccDashboardPermissionConfig_FromID = `
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
