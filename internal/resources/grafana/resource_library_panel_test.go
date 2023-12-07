package grafana_test

import (
	"fmt"
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/internal/testutils"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccLibraryPanel_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=8.0.0")

	name := acctest.RandString(10)
	var panel models.LibraryElementResponse

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      libraryPanelCheckExists.destroyed(&panel, nil),
		Steps: []resource.TestStep{
			{
				// Test resource creation.
				Config: testAccLibraryPanelBasic(name),
				Check: resource.ComposeTestCheckFunc(
					libraryPanelCheckExists.exists("grafana_library_panel.test", &panel),
					resource.TestCheckResourceAttrSet("grafana_library_panel.test", "uid"),
					resource.TestMatchResourceAttr("grafana_library_panel.test", "id", defaultOrgIDRegexp),
					resource.TestCheckResourceAttr("grafana_library_panel.test", "org_id", "1"),
					resource.TestCheckResourceAttr("grafana_library_panel.test", "name", name),
					resource.TestCheckResourceAttr("grafana_library_panel.test", "version", "1"),
					resource.TestCheckResourceAttr("grafana_library_panel.test", "folder_id", "1:0"),
					resource.TestCheckResourceAttr("grafana_library_panel.test", "model_json", fmt.Sprintf(`{"description":"","title":"%s","type":""}`, name)),
				),
			},
			{
				// Updates title.
				Config: testAccLibraryPanelBasic("updated " + name),
				Check: resource.ComposeTestCheckFunc(
					libraryPanelCheckExists.exists("grafana_library_panel.test", &panel),
					resource.TestCheckResourceAttrSet("grafana_library_panel.test", "uid"),
					resource.TestMatchResourceAttr("grafana_library_panel.test", "id", defaultOrgIDRegexp),
					resource.TestCheckResourceAttr("grafana_library_panel.test", "name", "updated "+name),
					resource.TestCheckResourceAttr("grafana_library_panel.test", "version", "2"),
					resource.TestCheckResourceAttr("grafana_library_panel.test", "folder_id", "1:0"),
					resource.TestCheckResourceAttr("grafana_library_panel.test", "model_json", fmt.Sprintf(`{"description":"","title":"updated %s","type":""}`, name)),
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

func TestAccLibraryPanel_folder(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=8.0.0")

	name := acctest.RandString(10)
	var panel models.LibraryElementResponse
	var folder models.Folder

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			libraryPanelCheckExists.destroyed(&panel, nil),
			folderCheckExists.destroyed(&folder, nil),
		),
		Steps: []resource.TestStep{
			{
				Config: testAccLibraryPanelInFolder(name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("grafana_library_panel.test_folder", "id", defaultOrgIDRegexp),
					libraryPanelCheckExists.exists("grafana_library_panel.test_folder", &panel),
					folderCheckExists.exists("grafana_folder.test_folder", &folder),
					resource.TestCheckResourceAttr("grafana_library_panel.test_folder", "name", name),
					resource.TestMatchResourceAttr(
						"grafana_library_panel.test_folder", "folder_id", defaultOrgIDRegexp,
					),
				),
			},
			{
				ImportState:       true,
				ResourceName:      "grafana_library_panel.test_folder",
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccLibraryPanel_dashboard(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=8.0.0")

	var panel models.LibraryElementResponse
	var dashboard models.DashboardFullWithMeta

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      libraryPanelCheckExists.destroyed(&panel, nil),
		Steps: []resource.TestStep{
			{
				// Test library panel is connected to dashboard
				Config: testutils.TestAccExample(t, "data-sources/grafana_library_panel/data-source.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("grafana_library_panel.dashboard", "id", defaultOrgIDRegexp),
					libraryPanelCheckExists.exists("grafana_library_panel.dashboard", &panel),
					dashboardCheckExists.exists("grafana_dashboard.with_library_panel", &dashboard),
					dashboardCheckExists.exists("data.grafana_dashboard.from_library_panel_connection", &dashboard),
				),
			},
		},
	})
}

func TestAccLibraryPanel_inOrg(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=8.0.0")

	var panel models.LibraryElementResponse
	orgName := acctest.RandString(10)

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      libraryPanelCheckExists.destroyed(&panel, nil),
		Steps: []resource.TestStep{
			{
				Config: testAccLibraryPanelInOrganization(orgName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("grafana_library_panel.test", "id", nonDefaultOrgIDRegexp),
					libraryPanelCheckExists.exists("grafana_library_panel.test", &panel),
					checkResourceIsInOrg("grafana_library_panel.test", "grafana_organization.test"),
				),
			},
		},
	})
}

func testAccLibraryPanelBasic(name string) string {
	return fmt.Sprintf(`
resource "grafana_library_panel" "test" {
	name      = "%[1]s"
	model_json = jsonencode({
		title   = "%[1]s",
	})
}
`, name)
}

func testAccLibraryPanelInFolder(name string) string {
	return fmt.Sprintf(`
resource "grafana_folder" "test_folder" {
	title = "%[1]s"
}

resource "grafana_library_panel" "test_folder" {
	name      = "%[1]s"
	folder_id = grafana_folder.test_folder.id
	model_json = jsonencode({
		title   = "%[1]s",
		id      = 12,
		version = 43,
	})
}`, name)
}

func testAccLibraryPanelInOrganization(orgName string) string {
	return fmt.Sprintf(`
resource "grafana_organization" "test" {
	name = "%[1]s"
}

resource "grafana_library_panel" "test" {
	org_id    = grafana_organization.test.id
	name      = "%[1]s"
	model_json = jsonencode({
	  title   = "%[1]s",
	})
  }`, orgName)
}
