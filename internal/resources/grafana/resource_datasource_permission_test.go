package grafana_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/grafana/terraform-provider-grafana/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccDatasourcePermission_basic(t *testing.T) {
	t.Skip("This test is failing in Grafana Cloud 9.3+")
	testutils.CheckCloudInstanceTestsEnabled(t)

	datasourceID := int64(-1)

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "resources/grafana_data_source_permission/resource.tf"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccDatasourcePermissionsCheckExists("grafana_data_source_permission.fooPermissions", &datasourceID),
					resource.TestCheckResourceAttr("grafana_data_source_permission.fooPermissions", "permissions.#", "3"),
				),
			},
			{
				Config: testutils.TestAccExample(t, "resources/grafana_data_source_permission/_acc_resource_remove.tf"),
				Check:  testAccDatasourcePermissionCheckDestroy(&datasourceID),
			},
		},
	})
}

//nolint:unused
func testAccDatasourcePermissionsCheckExists(rn string, datasourceID *int64) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("Resource not found: %s\n %#v", rn, s.RootModule().Resources)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		client := testutils.Provider.Meta().(*common.Client).GrafanaAPI

		gotDatasourceID, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
		if err != nil {
			return fmt.Errorf("datasource id is malformed")
		}

		_, err = client.DatasourcePermissions(gotDatasourceID)
		if err != nil {
			return fmt.Errorf("error getting datasource permissions: %s", err)
		}

		*datasourceID = gotDatasourceID

		return nil
	}
}

//nolint:unused
func testAccDatasourcePermissionCheckDestroy(datasourceID *int64) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testutils.Provider.Meta().(*common.Client).GrafanaAPI
		response, err := client.DatasourcePermissions(*datasourceID)
		if err != nil {
			return fmt.Errorf("error getting datasource permissions %d: %s", *datasourceID, err)
		}
		if len(response.Permissions) > 0 {
			return fmt.Errorf("permissions were not empty when expected")
		}

		return nil
	}
}
