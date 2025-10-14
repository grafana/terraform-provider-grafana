package grafana_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
)

func TestAccServiceAccountPermission_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.2.4")

	var sa models.ServiceAccountDTO
	name := acctest.RandString(10)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             serviceAccountPermissionsCheckExists.destroyed(&sa, nil),
		Steps: []resource.TestStep{
			{
				Config: testServiceAccountPermissionsConfig(name),
				Check: resource.ComposeTestCheckFunc(
					serviceAccountPermissionsCheckExists.exists("grafana_service_account_permission.test_permissions", &sa),
					resource.TestCheckResourceAttr("grafana_service_account_permission.test_permissions", "permissions.#", "3"),
				),
			},
			{
				ImportState:       true,
				ResourceName:      "grafana_service_account_permission.test_permissions",
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccServiceAccountPermission_inOrg(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.2.4")

	var sa models.ServiceAccountDTO
	var org models.OrgDetailsDTO
	name := acctest.RandString(10)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             orgCheckExists.destroyed(&org, nil),
		Steps: []resource.TestStep{
			{
				Config: testServiceAccountPermissionsConfig_inOrg(name),
				Check: resource.ComposeTestCheckFunc(
					checkResourceIsInOrg("grafana_service_account_permission.test", "grafana_organization.test"),
					serviceAccountPermissionsCheckExists.exists("grafana_service_account_permission.test", &sa),
					resource.TestCheckResourceAttr("grafana_service_account_permission.test", "permissions.#", "1"),
					resource.TestMatchResourceAttr("grafana_service_account_permission.test", "permissions.0.team_id", nonDefaultOrgIDRegexp),
					resource.TestCheckResourceAttr("grafana_service_account_permission.test", "permissions.0.permission", "Edit"),
				),
			},
			{
				ImportState:       true,
				ResourceName:      "grafana_service_account_permission.test",
				ImportStateVerify: true,
			},
			// Test destroy
			{
				Config: testutils.WithoutResource(t, testServiceAccountPermissionsConfig_inOrg(name), "grafana_service_account_permission.test"),
				Check: resource.ComposeTestCheckFunc(
					orgCheckExists.exists("grafana_organization.test", &org),
					serviceAccountPermissionsCheckExists.destroyed(&sa, &org),
				),
			},
		},
	})
}

func testServiceAccountPermissionsConfig(name string) string {
	return fmt.Sprintf(`
resource "grafana_service_account" "test" {
	name        = "%[1]s"
	role        = "Editor"
	is_disabled = false
}

resource "grafana_team" "test_team" {
	name = "%[1]s"
}

resource "grafana_user" "test_user" {
	email = "%[1]s@test.com"
	login    = "%[1]s@test.com"
	password = "password"
}

resource "grafana_service_account_permission" "test_permissions" {
	service_account_id = grafana_service_account.test.id
	permissions {
		user_id = 1
		permission = "Admin"
	}
	permissions {
		user_id = grafana_user.test_user.id
		permission = "Edit"
	}
	permissions {
		team_id = grafana_team.test_team.id
		permission = "Admin"
	}
}
`, name)
}

func testServiceAccountPermissionsConfig_inOrg(name string) string {
	return fmt.Sprintf(`
	resource "grafana_organization" "test" {
		name = "%[1]s"
	}

	resource "grafana_team" "test" {
		org_id  = grafana_organization.test.id
		name    = "%[1]s"
		members = []
	}
	
	resource "grafana_service_account" "test" {
		org_id = grafana_organization.test.id
		name   = "%[1]s"
		role   = "Viewer"
	}
	
	resource "grafana_service_account_permission" "test" {
		org_id             = grafana_organization.test.id
		service_account_id = grafana_service_account.test.id
	
		permissions {
			team_id    = grafana_team.test.id
			permission = "Edit"
		}
	}
`, name)
}
