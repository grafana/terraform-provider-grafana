package grafana

import (
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDatasourceLibraryPanelFromName(t *testing.T) {
	CheckOSSTestsEnabled(t)
	CheckOSSTestsSemver(t, ">=8.0.0")

	var panel gapi.LibraryPanel
	var dashboard gapi.Dashboard
	checks := []resource.TestCheckFunc{
		testAccLibraryPanelCheckExists("grafana_library_panel.test", &panel),
		testAccDashboardCheckExists("grafana_dashboard.test", &dashboard),
		// resource.TestCheckResourceAttr(
		// 	"data.grafana_library_panel.from_name", "name", "test name",
		// ),
		// resource.TestMatchResourceAttr(
		// 	"data.grafana_library_panel.from_name", "id", idRegexp,
		// ),
		// resource.TestMatchResourceAttr(
		// 	"data.grafana_library_panel.from_name", "uid", uidRegexp,
		// ),
		resource.TestCheckResourceAttr(
			"data.grafana_library_panel.from_uid", "name", "test name",
		),
		resource.TestMatchResourceAttr(
			"data.grafana_library_panel.from_uid", "id", idRegexp,
		),
		resource.TestMatchResourceAttr(
			"data.grafana_library_panel.from_uid", "uid", uidRegexp,
		),
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccLibraryPanelCheckDestroy(&panel),
		Steps: []resource.TestStep{
			{
				Config: testAccExample(t, "data-sources/grafana_library_panel/data-source.tf"),
				Check:  resource.ComposeTestCheckFunc(checks...),
			},
		},
	})
}
