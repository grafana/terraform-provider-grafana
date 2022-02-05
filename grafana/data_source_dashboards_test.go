package grafana

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceDashboardsAllAndByFolderID(t *testing.T) {
	CheckCloudTestsEnabled(t)

	// var dashboard1 gapi.Dashboard
	// var dashboard2 gapi.Dashboard
	// var folder gapi.Folder

	checks := []resource.TestCheckFunc{
		// testAccDashboardCheckExists("grafana_dashboard.test1", &dashboard1),
		// testAccDashboardCheckExists("grafana_dashboard.test2", &dashboard2),
		// testAccFolderCheckExists("grafana_folder.test", &folder),
		// make sure only one dashboard in one folder when specifying folder
		// resource.TestCheckResourceAttr("data.grafana_dashboards.with_folder_id", "dashboards.%", "1"),
		// make sure exactly two dashboards in two folders when omitting folder_ids
		resource.TestCheckResourceAttr("data.grafana_dashboards.all", "dashboards.%", "2"),
		// make sure only one dashboard in one folder when specifying tags
		// resource.TestCheckResourceAttr("data.grafana_dashboards.with_tags", "dashboards.%", "1"),
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
