package grafana

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceDashboardsAllAndByFolderID(t *testing.T) {
	CheckOSSTestsEnabled(t)

	checks := []resource.TestCheckFunc{
		resource.TestCheckResourceAttr("data.grafana_dashboards.all", "id", "dashboards"),
		resource.TestCheckResourceAttr("data.grafana_dashboards.tags", "id", "dashboards-tags"),
		resource.TestCheckResourceAttr("data.grafana_dashboards.folder_ids", "id", "dashboards-folder_ids"),
		resource.TestCheckResourceAttr("data.grafana_dashboards.folder_ids_tags", "id", "dashboards-folder_ids-tags"),
		resource.TestCheckResourceAttr("data.grafana_dashboards.tags", "dashboards.#", "1"),
		resource.TestCheckResourceAttr("data.grafana_dashboards.folder_ids", "dashboards.#", "1"),
		resource.TestCheckResourceAttr("data.grafana_dashboards.folder_ids_tags", "dashboards.#", "1"),
		resource.TestCheckResourceAttrSet("data.grafana_dashboard.from_data_source", "config_json"),
	}

	resource.UnitTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccExample(t, "data-sources/grafana_dashboards/data-source.tf"),
				Check:  resource.ComposeTestCheckFunc(checks...),
			},
		},
	})
}
