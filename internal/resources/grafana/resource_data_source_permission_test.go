package grafana_test

import (
	"fmt"
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDatasourcePermission_basic(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=9.0.0")

	var ds models.DataSource
	name := acctest.RandString(10)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDatasourcePermission(name, "Edit"),
				Check: resource.ComposeAggregateTestCheckFunc(
					datasourcePermissionsCheckExists.exists("grafana_data_source_permission.fooPermissions", &ds),
					resource.TestCheckResourceAttr("grafana_data_source_permission.fooPermissions", "permissions.#", "4"),
					resource.TestCheckResourceAttr("grafana_data_source_permission.fooPermissions", "permissions.0.permission", "Edit"),
				),
			},
			{
				Config: testutils.WithoutResource(t, testAccDatasourcePermission(name, "Edit"), "grafana_data_source_permission.fooPermissions"),
				Check:  datasourcePermissionsCheckExists.destroyed(&ds, nil),
			},
		},
	})
}

func TestAccDatasourcePermission_AdminRole(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=10.3.0")

	var ds models.DataSource
	name := acctest.RandString(10)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDatasourcePermission(name, "Admin"),
				Check: resource.ComposeAggregateTestCheckFunc(
					datasourcePermissionsCheckExists.exists("grafana_data_source_permission.fooPermissions", &ds),
					resource.TestCheckResourceAttr("grafana_data_source_permission.fooPermissions", "permissions.#", "4"),
					resource.TestCheckResourceAttr("grafana_data_source_permission.fooPermissions", "permissions.0.permission", "Admin"),
				),
			},
			{
				Config: testutils.WithoutResource(t, testAccDatasourcePermission(name, "Admin"), "grafana_data_source_permission.fooPermissions"),
				Check:  datasourcePermissionsCheckExists.destroyed(&ds, nil),
			},
		},
	})
}

func testAccDatasourcePermission(name string, teamPermission string) string {
	return fmt.Sprintf(`
resource "grafana_team" "team" {
	name = "%[1]s"
}

resource "grafana_data_source" "foo" {
	name = "%[1]s"
	type = "cloudwatch"

	json_data_encoded = jsonencode({
		defaultRegion = "us-east-1"
		authType      = "keys"
	})

	secure_json_data_encoded = jsonencode({
		accessKey = "123"
		secretKey = "456"
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

resource "grafana_data_source_permission" "fooPermissions" {
	datasource_uid = grafana_data_source.foo.uid
	permissions {
		team_id    = grafana_team.team.id
		permission = "%[2]s"
	}
	permissions {
		user_id    = grafana_user.user.id
		permission = "Edit"
	}
	permissions {
		built_in_role = "Viewer"
		permission    = "Query"
	}
	permissions {
		user_id    = grafana_service_account.sa.id
		permission = "Query"
	}
}`, name, teamPermission)
}
