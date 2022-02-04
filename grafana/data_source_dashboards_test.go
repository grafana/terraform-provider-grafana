package grafana

import (
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceDashboardsAllAndByFolderID(t *testing.T) {
	CheckCloudTestsEnabled(t)

	var dashboard1 gapi.Dashboard
	var dashboard2 gapi.Dashboard
	var folder gapi.Folder

	checks := []resource.TestCheckFunc{
		testAccDashboardCheckExists("grafana_dashboard.test1", &dashboard1),
		testAccDashboardCheckExists("grafana_dashboard.test2", &dashboard2),
		testAccFolderCheckExists("grafana_folder.test", &folder),
		// make sure only one dashboard in one folder when specifying folder
		resource.TestCheckResourceAttr("data.grafana_dashboards.with_folder_id", "dashboards.%", "1"),
		resource.TestCheckResourceAttr("data.grafana_dashboards.with_folder_id", "folder_ids.#", "1"),
		resource.TestCheckResourceAttr("data.grafana_dashboards.with_folder_id", "uids.#", "1"),
		resource.TestCheckResourceAttr("data.grafana_dashboards.with_folder_id", "ids.#", "1"),
		// make sure exactly two dashboards in two folders
		resource.TestCheckResourceAttr("data.grafana_dashboards.all", "dashboards.%", "2"),
		resource.TestCheckResourceAttr("data.grafana_dashboards.all", "folder_ids.#", "2"),
		resource.TestCheckResourceAttr("data.grafana_dashboards.all", "uids.#", "2"),
		resource.TestCheckResourceAttr("data.grafana_dashboards.all", "ids.#", "2"),
	}

	resource.Test(t, resource.TestCase{
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
