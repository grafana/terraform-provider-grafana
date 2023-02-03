package provider

import (
	"net/url"
	"testing"

	"github.com/grafana/terraform-provider-grafana/provider/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceDashboardsAllAndByFolderID(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

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

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "data-sources/grafana_dashboards/data-source.tf"),
				Check:  resource.ComposeTestCheckFunc(checks...),
			},
		},
	})
}
