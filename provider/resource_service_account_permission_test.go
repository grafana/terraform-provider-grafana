package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/provider/common"
)

func TestAccServiceAccountPermission(t *testing.T) {
	CheckOSSTestsEnabled(t)
	CheckOSSTestsSemver(t, ">=9.2.4")

	var saPermission gapi.ServiceAccountPermission
	// TODO: Make parallelizable
	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccServiceAccountPermissionsCheckDestroy(saPermission.ID),
		Steps: []resource.TestStep{
			{
				Config: testServiceAccountPermissionsConfig,
				Check: resource.ComposeTestCheckFunc(
					testServiceAccountPermissionsCheckExists("grafana_service_account_permission.test_permissions", &saPermission),
					resource.TestMatchResourceAttr(
						"grafana_service_account_permission.test_permissions", "service_account_id", idRegexp,
					),
					resource.TestCheckResourceAttr(
						"grafana_service_account_permission.test_permissions", "permissions.#", "3",
					),
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

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}
		id, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
		if err != nil {
			return fmt.Errorf("resource id is malformed")
		}

		client := testAccProvider.Meta().(*common.Client).GrafanaAPI
		perms, err := client.GetServiceAccountPermissions(id)
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
		client := testAccProvider.Meta().(*common.Client).GrafanaAPI
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

const testServiceAccountPermissionsConfig = `
resource "grafana_service_account" "test" {
	name        = "sa-terraform-test"
	role        = "Editor"
	is_disabled = false
}

resource "grafana_team" "test_team" {
	name = "tf_test_team"
}

resource "grafana_user" "test_user" {
	email = "tf_user@test.com"
	login    = "tf_user@test.com"
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
`
