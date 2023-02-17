package grafana_test

import (
	"errors"
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

func testServiceAccountPermissionsCheckExists(rn string, saPerm *gapi.ServiceAccountPermission) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		client := testutils.Provider.Meta().(*common.Client).GrafanaAPI
		orgID, saIDStr := grafana.SplitOrgResourceID(rs.Primary.ID)

		saID, err := strconv.ParseInt(saIDStr, 10, 64)
		if err != nil {
			return fmt.Errorf("id is malformed: %w", err)
		}

		// If orgID is not the default org, check that the SA doesn't exist in the default org
		if orgID > 1 {
			perms, err := client.GetServiceAccountPermissions(saID)
			if err == nil || len(perms) > 0 {
				return errors.New("got SA permissions from the default org, while the SA shouldn't exist")
			}
			client = client.WithOrgID(orgID)
		}

		perms, err := client.GetServiceAccountPermissions(saID)
		if err != nil {
			return fmt.Errorf("error getting role assignments: %s", err)
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
