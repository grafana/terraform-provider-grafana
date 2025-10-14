package grafana_test

import (
	"fmt"
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDashboardPermissionItem_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.0.0")

	var (
		dashboard models.DashboardFullWithMeta
		team      models.TeamDTO
		user      models.UserProfileDTO
		sa        models.ServiceAccountDTO
	)
	name := acctest.RandString(10)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDashboardPermissionItem(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					dashboardCheckExists.exists("grafana_dashboard.foo", &dashboard),
					teamCheckExists.exists("grafana_team.team", &team),
					userCheckExists.exists("grafana_user.user", &user),
					serviceAccountCheckExists.exists("grafana_service_account.sa", &sa),
					checkDashboardPermissionsSet(&dashboard, &team, &user, &sa, true),
				),
			},
			{
				ResourceName:      "grafana_dashboard_permission_item.team",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      "grafana_dashboard_permission_item.user",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      "grafana_dashboard_permission_item.role_viewer",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      "grafana_dashboard_permission_item.sa",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Remove permissions
			{
				Config: testutils.WithoutResource(t,
					testAccDashboardPermissionItem(name),
					"grafana_dashboard_permission_item.team",
					"grafana_dashboard_permission_item.user",
					"grafana_dashboard_permission_item.role_viewer",
					"grafana_dashboard_permission_item.role_editor",
					"grafana_dashboard_permission_item.sa",
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					dashboardCheckExists.exists("grafana_dashboard.foo", &dashboard),
					checkDashboardPermissionsEmpty(&dashboard, true),
				),
			},
		},
	})
}

func testAccDashboardPermissionItem(name string) string {
	return fmt.Sprintf(`
resource "grafana_team" "team" {
	name = "%[1]s"
}

resource "grafana_dashboard" "foo" {
	config_json = jsonencode({
		uid = "%[1]s",
		title: "%[1]s",
	})
}

resource "grafana_user" "user" {
	name     = "%[1]s"
	email    = "%[1]s@example.com"
	login    = "%[1]s"
	password = "hunter2"
}

resource "grafana_service_account" "sa" {
	name = "%[1]s"
	role = "Viewer"
}

resource "grafana_dashboard_permission_item" "team" {
	dashboard_uid = grafana_dashboard.foo.uid
	team           = grafana_team.team.id
	permission     = "View"
}

resource "grafana_dashboard_permission_item" "user" {
	dashboard_uid = grafana_dashboard.foo.uid
	user           = grafana_user.user.id
	permission     = "Admin"
}

resource "grafana_dashboard_permission_item" "role_viewer" {
	dashboard_uid = grafana_dashboard.foo.uid
	role  = "Viewer"
	permission     = "View"
}

resource "grafana_dashboard_permission_item" "role_editor" {
	dashboard_uid = grafana_dashboard.foo.uid
	role  = "Editor"
	permission     = "Edit"
}

resource "grafana_dashboard_permission_item" "sa" {
	dashboard_uid = grafana_dashboard.foo.uid
	user = grafana_service_account.sa.id
	permission     = "Admin"
}`, name)
}
