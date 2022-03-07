package grafana

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccDatasourcePermission_basic(t *testing.T) {
	CheckCloudInstanceTestsEnabled(t)

	datasourceID := int64(-1)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccExample(t, "resources/grafana_data_source_permission/resource.tf"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccDatasourcePermissionsCheckExists("grafana_data_source_permission.fooPermissions", &datasourceID),
					resource.TestCheckResourceAttr("grafana_data_source_permission.fooPermissions", "permissions.#", "2"),
				),
			},
			{
				Config: testAccExample(t, "resources/grafana_data_source_permission/_acc_resource_remove.tf"),
				Check:  testAccDatasourcePermissionCheckDestroy(&datasourceID),
			},
		},
	})
}

func testAccDatasourcePermissionsCheckExists(rn string, datasourceID *int64) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("Resource not found: %s\n %#v", rn, s.RootModule().Resources)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("Resource id not set")
		}

		client := testAccProvider.Meta().(*client).gapi

		gotDatasourceID, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
		if err != nil {
			return fmt.Errorf("datasource id is malformed")
		}

		_, err = client.DatasourcePermissions(gotDatasourceID)
		if err != nil {
			return fmt.Errorf("Error getting datasource permissions: %s", err)
		}

		*datasourceID = gotDatasourceID

		return nil
	}
}

func testAccDatasourcePermissionCheckDestroy(datasourceID *int64) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*client).gapi
		response, err := client.DatasourcePermissions(*datasourceID)
		if err != nil {
			return fmt.Errorf("Error getting datasource permissions %d: %s", *datasourceID, err)
		}
		if response.Enabled {
			return fmt.Errorf("Datasource permissions %d still enabled", *datasourceID)
		}
		if len(response.Permissions) > 0 {
			return fmt.Errorf("Permissions were not empty when expected")
		}

		return nil
	}
}
