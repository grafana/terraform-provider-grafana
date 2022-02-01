package grafana

import (
	"fmt"
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccDatasourceLibraryPanelFromName(t *testing.T) {
	CheckOSSTestsEnabled(t)

	var panel gapi.LibraryPanel
	checks := []resource.TestCheckFunc{
		testAccLibraryPanelCheckExists("grafana_library_panel.test", &panel),
		resource.TestCheckResourceAttr(
			"data.grafana_library_panel.from_name", "name", "test name",
		),
		resource.TestMatchResourceAttr(
			"data.grafana_library_panel.from_name", "id", idRegexp,
		),
		resource.TestMatchResourceAttr(
			"data.grafana_library_panel.from_name", "uid", uidRegexp,
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

func TestAccDatasourceLibraryPanelFromUID(t *testing.T) {
	CheckOSSTestsEnabled(t)

	var panel gapi.LibraryPanel
	checks := []resource.TestCheckFunc{
		testAccLibraryPanelCheckExists("grafana_library_panel.test", &panel),
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

func testAccLibraryPanelCheckExists(rn string, panel *gapi.LibraryPanel) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}
		client := testAccProvider.Meta().(*client).gapi
		gotLibraryPanel, err := client.LibraryPanelByUID(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("error getting panel: %s", err)
		}
		*panel = *gotLibraryPanel
		return nil
	}
}

func testAccLibraryPanelCheckDestroy(panel *gapi.LibraryPanel) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*client).gapi
		_, err := client.LibraryPanelByUID(panel.UID)
		if err == nil {
			return fmt.Errorf("panel still exists")
		}
		return nil
	}
}
