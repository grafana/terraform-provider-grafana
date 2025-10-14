package grafana_test

import (
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceDashboardsAllAndByFolderUID(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=10.0.0")

	// Do not use parallel tests here because it tests a listing datasource on the default org
	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "data-sources/grafana_dashboards/data-source.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.grafana_dashboards.tags", "dashboards.#", "1"),
					resource.TestCheckResourceAttr("data.grafana_dashboards.tags", "dashboards.0.uid", "data-source-dashboards-1"),
					resource.TestCheckResourceAttr("data.grafana_dashboards.tags", "dashboards.0.title", "data_source_dashboards 1"),
					resource.TestCheckResourceAttr("data.grafana_dashboards.tags", "dashboards.0.folder_title", "test folder data_source_dashboards"),

					resource.TestCheckResourceAttr("data.grafana_dashboards.folder_uids", "dashboards.#", "1"),
					resource.TestCheckResourceAttr("data.grafana_dashboards.folder_uids", "dashboards.0.uid", "data-source-dashboards-1"),
					resource.TestCheckResourceAttr("data.grafana_dashboards.folder_uids", "dashboards.0.title", "data_source_dashboards 1"),
					resource.TestCheckResourceAttr("data.grafana_dashboards.folder_uids", "dashboards.0.folder_title", "test folder data_source_dashboards"),

					resource.TestCheckResourceAttr("data.grafana_dashboards.folder_uids_tags", "dashboards.#", "1"),
					resource.TestCheckResourceAttr("data.grafana_dashboards.folder_uids_tags", "dashboards.0.uid", "data-source-dashboards-1"),
					resource.TestCheckResourceAttr("data.grafana_dashboards.folder_uids_tags", "dashboards.0.title", "data_source_dashboards 1"),
					resource.TestCheckResourceAttr("data.grafana_dashboards.folder_uids_tags", "dashboards.0.folder_title", "test folder data_source_dashboards"),

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
