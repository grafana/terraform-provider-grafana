package grafana_test

import (
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDatasourcePermission_basic(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t)

	var ds models.DataSource

	// TODO: Admin role can only be set from Grafana 10.3.0 onwards. Test this!
	config := testutils.TestAccExample(t, "resources/grafana_data_source_permission/resource.tf")

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					datasourcePermissionsCheckExists.exists("grafana_data_source_permission.fooPermissions", &ds),
					resource.TestCheckResourceAttr("grafana_data_source_permission.fooPermissions", "permissions.#", "4"),
				),
			},
			{
				Config: testutils.WithoutResource(t, config, "grafana_data_source_permission.fooPermissions"),
				Check:  datasourcePermissionsCheckExists.destroyed(&ds, nil),
			},
		},
	})
}

func TestAccDatasourcePermission_AdminRole(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=10.3.0")

	var ds models.DataSource

	config := testutils.TestAccExample(t, "resources/grafana_data_source_permission/resource.tf")

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					datasourcePermissionsCheckExists.exists("grafana_data_source_permission.fooPermissions", &ds),
					resource.TestCheckResourceAttr("grafana_data_source_permission.fooPermissions", "permissions.#", "4"),
				),
			},
			{
				Config: testutils.WithoutResource(t, config, "grafana_data_source_permission.fooPermissions"),
				Check:  datasourcePermissionsCheckExists.destroyed(&ds, nil),
			},
		},
	})
}
