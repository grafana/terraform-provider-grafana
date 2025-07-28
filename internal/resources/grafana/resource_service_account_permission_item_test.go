package grafana_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
)

func TestAccServiceAccountPermissionItem_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.2.4")

	var sa models.ServiceAccountDTO
	name := acctest.RandString(10)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             serviceAccountPermissionsCheckExists.destroyed(&sa, nil),
		Steps: []resource.TestStep{
			{
				Config: testServiceAccountPermissionsItemConfig(name),
				Check: resource.ComposeTestCheckFunc(
					serviceAccountPermissionsCheckExists.exists("grafana_service_account.test", &sa),
				),
			},
			{
				ImportState:             true,
				ResourceName:            "grafana_service_account_permission_item.admin_user",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"service_account_id"},
			},
			{
				ImportState:             true,
				ResourceName:            "grafana_service_account_permission_item.edit_user",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"service_account_id"},
			},
			{
				ImportState:             true,
				ResourceName:            "grafana_service_account_permission_item.admin_team",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"service_account_id"},
			},
			// Test no diff
			{
				Config:   testServiceAccountPermissionsItemConfig(name),
				PlanOnly: true,
			},
		},
	})
}

func TestAccServiceAccountPermissionItem_inOrg(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.2.4")

	var sa models.ServiceAccountDTO
	var org models.OrgDetailsDTO
	name := acctest.RandString(10)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             orgCheckExists.destroyed(&org, nil),
		Steps: []resource.TestStep{
			{
				Config: testServiceAccountPermissionsItemConfig_inOrg(name),
				Check: resource.ComposeTestCheckFunc(
					serviceAccountPermissionsCheckExists.exists("grafana_service_account.test", &sa),
				),
			},
			// Test destroy
			{
				Config: testutils.WithoutResource(t, testServiceAccountPermissionsItemConfig_inOrg(name),
					"grafana_service_account_permission_item.test",
					"grafana_service_account_permission_item.admin_user",
				),
				Check: resource.ComposeTestCheckFunc(
					orgCheckExists.exists("grafana_organization.test", &org),
					serviceAccountPermissionsCheckExists.destroyed(&sa, &org),
				),
			},
		},
	})
}

func testServiceAccountPermissionsItemConfig(name string) string {
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

resource "grafana_service_account_permission_item" "admin_user" {
	service_account_id = grafana_service_account.test.id
	user               = 1
	permission         = "Admin"
}

resource "grafana_service_account_permission_item" "edit_user" {
	service_account_id = grafana_service_account.test.id
	user               = grafana_user.test_user.id
	permission         = "Edit"
}

resource "grafana_service_account_permission_item" "admin_team" {
	service_account_id = grafana_service_account.test.id
	team               = grafana_team.test_team.id
	permission         = "Admin"
}
`, name)
}

func testServiceAccountPermissionsItemConfig_inOrg(name string) string {
	return fmt.Sprintf(`
	resource "grafana_organization" "test" {
		name = "%[1]s"
	}

	resource "grafana_team" "test" {
		org_id  = grafana_organization.test.id
		name    = "test"
		members = []
	}
	
	resource "grafana_service_account" "test" {
		org_id = grafana_organization.test.id
		name   = "test"
		role   = "Viewer"
	}

	resource "grafana_service_account_permission_item" "admin_user" {
		org_id             = grafana_organization.test.id
		service_account_id = grafana_service_account.test.id
		user               = 1
		permission         = "Admin"
	}

	resource "grafana_service_account_permission_item" "test" {
		org_id             = grafana_organization.test.id
		service_account_id = grafana_service_account.test.id
		team               = grafana_team.test.id
		permission         = "Edit"
	}
`, name)
}
