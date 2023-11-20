package grafana_test

import (
	"testing"

	"github.com/grafana/terraform-provider-grafana/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceDashboardsAllAndByFolderID(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	// Do not use parallel tests here because it tests a listing datasource on the default org
	resource.Test(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "data-sources/grafana_dashboards/data-source.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.grafana_dashboards.tags", "dashboards.#", "1"),
					resource.TestCheckResourceAttr("data.grafana_dashboards.tags", "dashboards.0.uid", "data-source-dashboards-1"),
					resource.TestCheckResourceAttr("data.grafana_dashboards.tags", "dashboards.0.title", "data_source_dashboards 1"),
					resource.TestCheckResourceAttr("data.grafana_dashboards.tags", "dashboards.0.folder_title", "test folder data_source_dashboards"),

					resource.TestCheckResourceAttr("data.grafana_dashboards.folder_ids", "dashboards.#", "1"),
					resource.TestCheckResourceAttr("data.grafana_dashboards.folder_ids", "dashboards.0.uid", "data-source-dashboards-1"),
					resource.TestCheckResourceAttr("data.grafana_dashboards.folder_ids", "dashboards.0.title", "data_source_dashboards 1"),
					resource.TestCheckResourceAttr("data.grafana_dashboards.folder_ids", "dashboards.0.folder_title", "test folder data_source_dashboards"),

					resource.TestCheckResourceAttr("data.grafana_dashboards.folder_ids_tags", "dashboards.#", "1"),
					resource.TestCheckResourceAttr("data.grafana_dashboards.folder_ids_tags", "dashboards.0.uid", "data-source-dashboards-1"),
					resource.TestCheckResourceAttr("data.grafana_dashboards.folder_ids_tags", "dashboards.0.title", "data_source_dashboards 1"),
					resource.TestCheckResourceAttr("data.grafana_dashboards.folder_ids_tags", "dashboards.0.folder_title", "test folder data_source_dashboards"),

					resource.TestCheckResourceAttr("data.grafana_dashboards.limit_one", "dashboards.#", "1"),
					resource.TestCheckResourceAttr("data.grafana_dashboards.limit_one", "dashboards.0.uid", "data-source-dashboards-1"),
					resource.TestCheckResourceAttr("data.grafana_dashboards.limit_one", "dashboards.0.title", "data_source_dashboards 1"),
					resource.TestCheckResourceAttr("data.grafana_dashboards.limit_one", "dashboards.0.folder_title", "test folder data_source_dashboards"),

					resource.TestCheckResourceAttr("data.grafana_dashboards.all", "dashboards.#", "2"),
					resource.TestCheckResourceAttr("data.grafana_dashboards.all", "dashboards.0.uid", "data-source-dashboards-1"),
					resource.TestCheckResourceAttr("data.grafana_dashboards.all", "dashboards.0.title", "data_source_dashboards 1"),
					resource.TestCheckResourceAttr("data.grafana_dashboards.all", "dashboards.0.folder_title", "test folder data_source_dashboards"),
					resource.TestCheckResourceAttr("data.grafana_dashboards.all", "dashboards.1.uid", "data-source-dashboards-2"),
					resource.TestCheckResourceAttr("data.grafana_dashboards.all", "dashboards.1.title", "data_source_dashboards 2"),
					resource.TestCheckResourceAttr("data.grafana_dashboards.all", "dashboards.1.folder_title", ""),

					resource.TestCheckResourceAttr("data.grafana_dashboards.wrong_org", "dashboards.#", "0"),
				),
			},
		},
	})
}
