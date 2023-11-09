package grafana_test

import (
	"fmt"
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/grafana/terraform-provider-grafana/internal/resources/grafana"
	"github.com/grafana/terraform-provider-grafana/internal/testutils"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccLibraryPanel_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=8.0.0")

	var panel models.LibraryElementResponse

	// TODO: Make parallelizable
	resource.Test(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      libraryPanelCheckExists.destroyed(&panel, nil),
		Steps: []resource.TestStep{
			{
				// Test resource creation.
				Config: testutils.TestAccExample(t, "resources/grafana_library_panel/_acc_basic.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("grafana_library_panel.test", "id", defaultOrgIDRegexp),
					resource.TestCheckResourceAttr("grafana_library_panel.test", "org_id", "1"),
					libraryPanelCheckExists.exists("grafana_library_panel.test", &panel),
					resource.TestCheckResourceAttr("grafana_library_panel.test", "name", "basic"),
					resource.TestCheckResourceAttr("grafana_library_panel.test", "version", "1"),
					resource.TestCheckResourceAttr("grafana_library_panel.test", "model_json", `{"description":"","title":"basic","type":"","version":34}`),
				),
			},
			{
				// Updates title.
				Config: testutils.TestAccExample(t, "resources/grafana_library_panel/_acc_basic_update.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("grafana_library_panel.test", "id", defaultOrgIDRegexp),
					libraryPanelCheckExists.exists("grafana_library_panel.test", &panel),
					resource.TestCheckResourceAttr("grafana_library_panel.test", "name", "updated name"),
					resource.TestCheckResourceAttr("grafana_library_panel.test", "version", "2"),
					resource.TestCheckResourceAttr("grafana_library_panel.test", "model_json", `{"description":"","id":12,"title":"updated name","type":"","version":35}`),
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
	testutils.CheckOSSTestsEnabled(t, ">=8.0.0")

	var panel models.LibraryElementResponse

	// TODO: Make parallelizable
	resource.Test(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      libraryPanelCheckExists.destroyed(&panel, nil),
		Steps: []resource.TestStep{
			{
				// Test resource creation.
				Config: testutils.TestAccExample(t, "resources/grafana_library_panel/_acc_computed.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("grafana_library_panel.test", "id", defaultOrgIDRegexp),
					libraryPanelCheckExists.exists("grafana_library_panel.test", &panel),
					libraryPanelCheckExists.exists("grafana_library_panel.test-computed", &panel),
				),
			},
		},
	})
}

func TestAccLibraryPanel_folder(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=8.0.0")

	var panel models.LibraryElementResponse
	var folder models.Folder

	// TODO: Make parallelizable
	resource.Test(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      testAccLibraryPanelFolderCheckDestroy(&panel, &folder),
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "resources/grafana_library_panel/_acc_folder.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("grafana_library_panel.test_folder", "id", defaultOrgIDRegexp),
					libraryPanelCheckExists.exists("grafana_library_panel.test_folder", &panel),
					folderCheckExists.exists("grafana_folder.test_folder", &folder),
					testAccLibraryPanelCheckExistsInFolder(&panel, &folder),
					resource.TestCheckResourceAttr("grafana_library_panel.test_folder", "name", "test-folder"),
					resource.TestMatchResourceAttr(
						"grafana_library_panel.test_folder", "folder_id", defaultOrgIDRegexp,
					),
				),
			},
		},
	})
}

func TestAccLibraryPanel_dashboard(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=8.0.0")

	var panel models.LibraryElementResponse
	var dashboard gapi.Dashboard

	// TODO: Make parallelizable
	resource.Test(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      libraryPanelCheckExists.destroyed(&panel, nil),
		Steps: []resource.TestStep{
			{
				// Test library panel is connected to dashboard
				Config: testutils.TestAccExample(t, "data-sources/grafana_library_panel/data-source.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("grafana_library_panel.dashboard", "id", defaultOrgIDRegexp),
					libraryPanelCheckExists.exists("grafana_library_panel.dashboard", &panel),
					testAccDashboardCheckExists("grafana_dashboard.with_library_panel", &dashboard),
					testAccDashboardCheckExists("data.grafana_dashboard.from_library_panel_connection", &dashboard),
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

func testAccLibraryPanelCheckExistsInFolder(panel *models.LibraryElementResponse, folder *models.Folder) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if panel.Result.FolderID != folder.ID && folder.ID != 0 {
			return fmt.Errorf("panel.Folder(%d) does not match folder.ID(%d)", panel.Result.FolderID, folder.ID)
		}
		return nil
	}
}

func testAccLibraryPanelFolderCheckDestroy(panel *models.LibraryElementResponse, folder *models.Folder) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testutils.Provider.Meta().(*common.Client).GrafanaAPI.WithOrgID(panel.Result.OrgID)
		_, err := client.LibraryPanelByUID(panel.Result.UID)
		if err == nil {
			return fmt.Errorf("panel still exists")
		}

		OAPIclient := testutils.Provider.Meta().(*common.Client).GrafanaOAPI.Folders
		folder, err = grafana.GetFolderByIDorUID(OAPIclient, folder.UID)
		if err == nil {
			return fmt.Errorf("the following folder still exists: %s", folder.Title)
		}
		return nil
	}
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
