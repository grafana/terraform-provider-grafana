package grafana_test

import (
	"fmt"
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDatasourcePermissionItem_basic(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=9.0.0")

	var ds models.DataSource
	name := acctest.RandString(10)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDatasourcePermissionItem(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					datasourcePermissionsCheckExists.exists("grafana_data_source.foo", &ds),
				),
			},
			{
				ResourceName:      "grafana_data_source_permission_item.team",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      "grafana_data_source_permission_item.user",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      "grafana_data_source_permission_item.role",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      "grafana_data_source_permission_item.sa",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccDatasourcePermissionItem(name string) string {
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

resource "grafana_data_source_permission_item" "team" {
	datasource_uid = grafana_data_source.foo.uid
	team           = grafana_team.team.id
	permission     = "Edit"
}

resource "grafana_data_source_permission_item" "user" {
	datasource_uid = grafana_data_source.foo.uid
	user           = grafana_user.user.id
	permission     = "Edit"
}

resource "grafana_data_source_permission_item" "role" {
	datasource_uid = grafana_data_source.foo.uid
	role  = "Viewer"
	permission     = "Query"
}

resource "grafana_data_source_permission_item" "sa" {
	datasource_uid = grafana_data_source.foo.uid
	user = grafana_service_account.sa.id
	permission     = "Query"
}`, name)
}
