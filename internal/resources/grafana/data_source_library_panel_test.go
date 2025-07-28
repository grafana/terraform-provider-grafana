package grafana_test

import (
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDatasourceLibraryPanel_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=8.0.0")

	var panel models.LibraryElementResponse
	// var dashboard gapi.Dashboard
	checks := []resource.TestCheckFunc{
		libraryPanelCheckExists.exists("grafana_library_panel.test", &panel),
		resource.TestCheckResourceAttr(
			"data.grafana_library_panel.from_name", "name", "test name",
		),
		resource.TestMatchResourceAttr(
			"data.grafana_library_panel.from_name", "uid", common.UIDRegexp,
		),
		resource.TestCheckResourceAttr(
			"data.grafana_library_panel.from_uid", "name", "test name",
		),
		resource.TestMatchResourceAttr(
			"data.grafana_library_panel.from_uid", "uid", common.UIDRegexp,
		),
	}

	// TODO: Make parallelizable
	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             libraryPanelCheckExists.destroyed(&panel, nil),
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "data-sources/grafana_library_panel/data-source.tf"),
				Check:  resource.ComposeTestCheckFunc(checks...),
			},
		},
	})
}
