package grafana

import (
	"net/url"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceDashboardsAllAndByFolderID(t *testing.T) {
	CheckOSSTestsEnabled(t)

	params := url.Values{
		"limit": {"5000"},
		"type":  {"dash-db"},
	}
	idAll := hashDashboardSearchParameters(params)

	params["tag"] = []string{"data_source_dashboards"}
	idTags := hashDashboardSearchParameters(params)

	checks := []resource.TestCheckFunc{
		resource.TestCheckResourceAttr("data.grafana_dashboards.all", "id", idAll),
		resource.TestCheckResourceAttr("data.grafana_dashboards.tags", "id", idTags),
		resource.TestCheckResourceAttr("data.grafana_dashboards.tags", "dashboards.#", "1"),
		resource.TestCheckResourceAttr("data.grafana_dashboards.folder_ids", "dashboards.#", "1"),
		resource.TestCheckResourceAttr("data.grafana_dashboards.folder_ids_tags", "dashboards.#", "1"),
		resource.TestCheckResourceAttr("data.grafana_dashboards.limit_one", "dashboards.#", "1"),
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
