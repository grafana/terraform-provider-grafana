package provider

import (
	"fmt"
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/provider/common"
	"github.com/grafana/terraform-provider-grafana/provider/testutils"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccLibraryPanel_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)
	testutils.CheckOSSTestsSemver(t, ">=8.0.0")

	var panel gapi.LibraryPanel

	// TODO: Make parallelizable
	resource.Test(t, resource.TestCase{
		ProviderFactories: testutils.GetProviderFactories(),
		CheckDestroy:      testAccLibraryPanelCheckDestroy(&panel),
		Steps: []resource.TestStep{
			{
				// Test resource creation.
				Config: testutils.TestAccExample(t, "resources/grafana_library_panel/_acc_basic.tf"),
				Check: resource.ComposeTestCheckFunc(
					testAccLibraryPanelCheckExists("grafana_library_panel.test", &panel),
					resource.TestCheckResourceAttr("grafana_library_panel.test", "name", "basic"),
					resource.TestCheckResourceAttr("grafana_library_panel.test", "version", "1"),
				),
			},
			{
				// Updates title.
				Config: testutils.TestAccExample(t, "resources/grafana_library_panel/_acc_basic_update.tf"),
				Check: resource.ComposeTestCheckFunc(
					testAccLibraryPanelCheckExists("grafana_library_panel.test", &panel),
					resource.TestCheckResourceAttr("grafana_library_panel.test", "name", "updated name"),
					resource.TestCheckResourceAttr("grafana_library_panel.test", "version", "2"),
				),
			},
			{
				// Importing matches the state of the previous step.
				ResourceName:      "grafana_library_panel.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccLibraryPanel_computed_config(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)
	testutils.CheckOSSTestsSemver(t, ">=8.0.0")

	var panel gapi.LibraryPanel

	// TODO: Make parallelizable
	resource.Test(t, resource.TestCase{
		ProviderFactories: testutils.GetProviderFactories(),
		CheckDestroy:      testAccLibraryPanelCheckDestroy(&panel),
		Steps: []resource.TestStep{
			{
				// Test resource creation.
				Config: testutils.TestAccExample(t, "resources/grafana_library_panel/_acc_computed.tf"),
				Check: resource.ComposeTestCheckFunc(
					testAccLibraryPanelCheckExists("grafana_library_panel.test", &panel),
					testAccLibraryPanelCheckExists("grafana_library_panel.test-computed", &panel),
				),
			},
		},
	})
}

func TestAccLibraryPanel_folder(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)
	testutils.CheckOSSTestsSemver(t, ">=8.0.0")

	var panel gapi.LibraryPanel
	var folder gapi.Folder

	// TODO: Make parallelizable
	resource.Test(t, resource.TestCase{
		ProviderFactories: testutils.GetProviderFactories(),
		CheckDestroy:      testAccLibraryPanelFolderCheckDestroy(&panel, &folder),
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "resources/grafana_library_panel/_acc_folder.tf"),
				Check: resource.ComposeTestCheckFunc(
					testAccLibraryPanelCheckExists("grafana_library_panel.test_folder", &panel),
					testAccFolderCheckExists("grafana_folder.test_folder", &folder),
					testAccLibraryPanelCheckExistsInFolder(&panel, &folder),
					resource.TestCheckResourceAttr("grafana_library_panel.test_folder", "name", "test-folder"),
					resource.TestMatchResourceAttr(
						"grafana_library_panel.test_folder", "folder_id", idRegexp,
					),
				),
			},
		},
	})
}

func TestAccLibraryPanel_dashboard(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)
	testutils.CheckOSSTestsSemver(t, ">=8.0.0")

	var panel gapi.LibraryPanel
	var dashboard gapi.Dashboard

	// TODO: Make parallelizable
	resource.Test(t, resource.TestCase{
		ProviderFactories: testutils.GetProviderFactories(),
		CheckDestroy:      testAccLibraryPanelCheckDestroy(&panel),
		Steps: []resource.TestStep{
			{
				// Test library panel is connected to dashboard
				Config: testutils.TestAccExample(t, "data-sources/grafana_library_panel/data-source.tf"),
				Check: resource.ComposeTestCheckFunc(
					testAccLibraryPanelCheckExists("grafana_library_panel.dashboard", &panel),
					testAccDashboardCheckExists("grafana_dashboard.with_library_panel", &dashboard),
					testAccDashboardCheckExists("data.grafana_dashboard.from_library_panel_connection", &dashboard),
				),
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
		client := testutils.GetProvider().Meta().(*common.Client).GrafanaAPI
		gotLibraryPanel, err := client.LibraryPanelByUID(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("error getting panel: %s", err)
		}
		*panel = *gotLibraryPanel
		return nil
	}
}

func testAccLibraryPanelCheckExistsInFolder(panel *gapi.LibraryPanel, folder *gapi.Folder) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if panel.Folder != folder.ID && folder.ID != 0 {
			return fmt.Errorf("panel.Folder(%d) does not match folder.ID(%d)", panel.Folder, folder.ID)
		}
		return nil
	}
}

func testAccLibraryPanelCheckDestroy(panel *gapi.LibraryPanel) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testutils.GetProvider().Meta().(*common.Client).GrafanaAPI
		_, err := client.LibraryPanelByUID(panel.UID)
		if err == nil {
			return fmt.Errorf("panel still exists")
		}
		return nil
	}
}

func testAccLibraryPanelFolderCheckDestroy(panel *gapi.LibraryPanel, folder *gapi.Folder) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testutils.GetProvider().Meta().(*common.Client).GrafanaAPI
		_, err := client.LibraryPanelByUID(panel.UID)
		if err == nil {
			return fmt.Errorf("panel still exists")
		}
		folder, err = getFolderByID(client, folder.ID)
		if err == nil {
			return fmt.Errorf("the following folder still exists: %s", folder.Title)
		}
		return nil
	}
}
