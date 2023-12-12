package grafana_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/grafana/terraform-provider-grafana/internal/resources/grafana"
	"github.com/grafana/terraform-provider-grafana/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccDatasourcePermission_basic(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t)

	datasourceID := int64(-1)
	// TODO: Admin role can only be set from Grafana 10.3.0 onwards. Test this!
	config := testutils.TestAccExample(t, "resources/grafana_data_source_permission/resource.tf")

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccDatasourcePermissionsCheckExists("grafana_data_source_permission.fooPermissions", &datasourceID),
					resource.TestCheckResourceAttr("grafana_data_source_permission.fooPermissions", "permissions.#", "4"),
				),
			},
			{
				Config: testutils.TestAccExample(t, "resources/grafana_data_source_permission/_acc_resource_remove.tf"),
				Check:  testAccDatasourcePermissionCheckDestroy(&datasourceID),
			},
		},
	})
}

func TestAccDatasourcePermission_AdminRole(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=10.3.0")

	datasourceID := int64(-1)

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "resources/grafana_data_source_permission/resource.tf"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccDatasourcePermissionsCheckExists("grafana_data_source_permission.fooPermissions", &datasourceID),
					resource.TestCheckResourceAttr("grafana_data_source_permission.fooPermissions", "permissions.#", "4"),
				),
			},
			{
				Config: testutils.TestAccExample(t, "resources/grafana_data_source_permission/_acc_resource_remove.tf"),
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
			return fmt.Errorf("resource id not set")
		}

		orgID, datasourceIDStr := grafana.SplitOrgResourceID(rs.Primary.ID)
		client := testutils.Provider.Meta().(*common.Client).DeprecatedGrafanaAPI.WithOrgID(orgID)

		gotDatasourceID, err := strconv.ParseInt(datasourceIDStr, 10, 64)
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

func testAccDatasourcePermissionCheckDestroy(datasourceID *int64) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testutils.Provider.Meta().(*common.Client).DeprecatedGrafanaAPI
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
