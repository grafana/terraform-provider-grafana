package provider

import (
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/provider/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDatasourceLibraryPanel(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)
	testutils.CheckOSSTestsSemver(t, ">=8.0.0")

	var panel gapi.LibraryPanel
	// var dashboard gapi.Dashboard
	checks := []resource.TestCheckFunc{
		testAccLibraryPanelCheckExists("grafana_library_panel.test", &panel),
		resource.TestCheckResourceAttr(
			"data.grafana_library_panel.from_name", "name", "test name",
		),
		resource.TestMatchResourceAttr(
			"data.grafana_library_panel.from_name", "uid", uidRegexp,
		),
		resource.TestCheckResourceAttr(
			"data.grafana_library_panel.from_uid", "name", "test name",
		),
		resource.TestMatchResourceAttr(
			"data.grafana_library_panel.from_uid", "uid", uidRegexp,
		),
	}

	// TODO: Make parallelizable
	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccLibraryPanelCheckDestroy(&panel),
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "data-sources/grafana_library_panel/data-source.tf"),
				Check:  resource.ComposeTestCheckFunc(checks...),
			},
		},
	})
}
