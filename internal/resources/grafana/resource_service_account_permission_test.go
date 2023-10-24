package grafana_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/grafana/terraform-provider-grafana/internal/resources/grafana"
	"github.com/grafana/terraform-provider-grafana/internal/testutils"
)

func TestAccServiceAccountPermission(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)
	testutils.CheckOSSTestsSemver(t, ">=9.2.4")

	name := acctest.RandString(10)

	var saPermission gapi.ServiceAccountPermission
	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      testAccServiceAccountPermissionsCheckDestroy(saPermission.ID),
		Steps: []resource.TestStep{
			{
				Config: testServiceAccountPermissionsConfig(name),
				Check: resource.ComposeTestCheckFunc(
					testServiceAccountPermissionsCheckExists("grafana_service_account_permission.test_permissions", &saPermission),
					resource.TestMatchResourceAttr("grafana_service_account_permission.test_permissions", "service_account_id", defaultOrgIDRegexp),
					resource.TestCheckResourceAttr("grafana_service_account_permission.test_permissions", "permissions.#", "3"),
				),
			},
		},
	})
}

func TestAccServiceAccountPermission_inOrg(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)
	testutils.CheckOSSTestsSemver(t, ">=9.2.4")

	name := acctest.RandString(10)

	var saPermission gapi.ServiceAccountPermission
	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      testAccServiceAccountPermissionsCheckDestroy(saPermission.ID),
		Steps: []resource.TestStep{
			{
				Config: testServiceAccountPermissionsConfig_inOrg(name),
				Check: resource.ComposeTestCheckFunc(
					testServiceAccountPermissionsCheckExists("grafana_service_account_permission.test", &saPermission),
					resource.TestMatchResourceAttr("grafana_service_account_permission.test", "service_account_id", nonDefaultOrgIDRegexp),
					resource.TestCheckResourceAttr("grafana_service_account_permission.test", "permissions.#", "1"),
					resource.TestMatchResourceAttr("grafana_service_account_permission.test", "permissions.0.team_id", nonDefaultOrgIDRegexp),
					resource.TestCheckResourceAttr("grafana_service_account_permission.test", "permissions.0.permission", "Edit"),
				),
			},
		},
	})
}

func testServiceAccountPermissionsCheckExists(rn string, saPerm *gapi.ServiceAccountPermission) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		orgID, saIDStr := grafana.SplitOrgResourceID(rs.Primary.ID)

		saID, err := strconv.ParseInt(saIDStr, 10, 64)
		if err != nil {
			return fmt.Errorf("id is malformed: %w", err)
		}
		client := testutils.Provider.Meta().(*common.Client).GrafanaAPI.WithOrgID(orgID)

		perms, err := client.GetServiceAccountPermissions(saID)
		if err != nil {
			return fmt.Errorf("error getting service account permissions: %s", err)
		}
		if len(perms) == 0 {
			return fmt.Errorf("service account assignments do not exist")
		}

		saPerm = perms[0]
		return nil
	}
}

func testAccServiceAccountPermissionsCheckDestroy(id int64) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testutils.Provider.Meta().(*common.Client).GrafanaAPI
		saPerms, err := client.GetServiceAccountPermissions(id)
		if err != nil {
			return err
		}

		for _, perm := range saPerms {
			if perm.IsManaged {
				return fmt.Errorf("service account permissions still exist")
			}
		}

		return nil
	}
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
		name    = "test"
		members = []
	}
	
	resource "grafana_service_account" "test" {
		org_id = grafana_organization.test.id
		name   = "test"
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
