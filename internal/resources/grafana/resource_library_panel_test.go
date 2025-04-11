package grafana_test

import (
	"fmt"
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccLibraryPanel_basic(t *testing.T) {
	testCases := []struct {
		versionConstraint string
		folder            string
	}{
		{
			versionConstraint: ">=8.0.0,<=11.0.0",
			folder:            "",
		},
		{
			versionConstraint: ">11.0.0",
			folder:            "general",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.versionConstraint, func(t *testing.T) {
			testutils.CheckOSSTestsEnabled(t, tc.versionConstraint)

			name := acctest.RandString(10)
			var panel models.LibraryElementResponse

			resource.ParallelTest(t, resource.TestCase{
				ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
				CheckDestroy:             libraryPanelCheckExists.destroyed(&panel, nil),
				Steps: []resource.TestStep{
					{
						// Test resource creation.
						Config: testAccLibraryPanelBasic(name, tc.folder),
						Check: resource.ComposeTestCheckFunc(
							libraryPanelCheckExists.exists("grafana_library_panel.test", &panel),
							resource.TestCheckResourceAttrSet("grafana_library_panel.test", "uid"),
							resource.TestMatchResourceAttr("grafana_library_panel.test", "id", defaultOrgIDRegexp),
							resource.TestCheckResourceAttr("grafana_library_panel.test", "org_id", "1"),
							resource.TestCheckResourceAttr("grafana_library_panel.test", "name", name),
							resource.TestCheckResourceAttr("grafana_library_panel.test", "version", "1"),
							resource.TestCheckResourceAttr("grafana_library_panel.test", "model_json", fmt.Sprintf(`{"description":"","title":"%s","type":""}`, name)),
							testutils.CheckLister("grafana_library_panel.test"),
						),
					},
					{
						// Updates title.
						Config: testAccLibraryPanelBasic("updated "+name, tc.folder),
						Check: resource.ComposeTestCheckFunc(
							libraryPanelCheckExists.exists("grafana_library_panel.test", &panel),
							resource.TestCheckResourceAttrSet("grafana_library_panel.test", "uid"),
							resource.TestMatchResourceAttr("grafana_library_panel.test", "id", defaultOrgIDRegexp),
							resource.TestCheckResourceAttr("grafana_library_panel.test", "name", "updated "+name),
							resource.TestCheckResourceAttr("grafana_library_panel.test", "version", "2"),
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
		})
	}
}

func TestAccLibraryPanel_folder(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.0.0")

	name := acctest.RandString(10)
	var panel models.LibraryElementResponse
	var folder models.Folder

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
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
					resource.TestCheckResourceAttrSet("grafana_library_panel.test_folder", "folder_uid"),
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
	testCases := []struct {
		versionConstraint string
		replacements      map[string]string
	}{
		{
			versionConstraint: ">=8.0.0,<=11.0.0",
			replacements: map[string]string{
				`"general"`: "null",
			},
		},
		{
			versionConstraint: ">11.0.0",
			replacements:      map[string]string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.versionConstraint, func(t *testing.T) {
			testutils.CheckOSSTestsEnabled(t, tc.versionConstraint)

			var panel models.LibraryElementResponse
			var dashboard models.DashboardFullWithMeta

			resource.ParallelTest(t, resource.TestCase{
				ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
				CheckDestroy:             libraryPanelCheckExists.destroyed(&panel, nil),
				Steps: []resource.TestStep{
					{
						// Test library panel is connected to dashboard
						Config: testutils.TestAccExampleWithReplace(t,
							"data-sources/grafana_library_panel/data-source.tf",
							tc.replacements,
						),
						Check: resource.ComposeTestCheckFunc(
							resource.TestMatchResourceAttr("grafana_library_panel.dashboard", "id", defaultOrgIDRegexp),
							libraryPanelCheckExists.exists("grafana_library_panel.dashboard", &panel),
							dashboardCheckExists.exists("grafana_dashboard.with_library_panel", &dashboard),
						),
					},
				},
			})
		})
	}
}

func TestAccLibraryPanel_inOrg(t *testing.T) {
	testCases := []struct {
		versionConstraint string
		folder            string
	}{
		{
			versionConstraint: ">=8.0.0,<=11.0.0",
			folder:            "",
		},
		{
			versionConstraint: ">11.0.0",
			folder:            "general",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.versionConstraint, func(t *testing.T) {
			testutils.CheckOSSTestsEnabled(t, tc.versionConstraint)

			var panel models.LibraryElementResponse
			orgName := acctest.RandString(10)

			resource.ParallelTest(t, resource.TestCase{
				ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
				CheckDestroy:             libraryPanelCheckExists.destroyed(&panel, nil),
				Steps: []resource.TestStep{
					{
						Config: testAccLibraryPanelInOrganization(orgName, tc.folder),
						Check: resource.ComposeTestCheckFunc(
							resource.TestMatchResourceAttr("grafana_library_panel.test", "id", nonDefaultOrgIDRegexp),
							libraryPanelCheckExists.exists("grafana_library_panel.test", &panel),
							checkResourceIsInOrg("grafana_library_panel.test", "grafana_organization.test"),
						),
					},
				},
			})
		})
	}
}

func testAccLibraryPanelBasic(name, folder string) string {
	var fuid string
	if folder == "" {
		fuid = "null"
	} else {
		fuid = fmt.Sprintf(`"%s"`, folder)
	}

	return fmt.Sprintf(`
resource "grafana_library_panel" "test" {
	name       = "%[1]s"
	folder_uid = %[2]s
	model_json = jsonencode({
		title   = "%[1]s",
	})
}
`, name, fuid)
}

func testAccLibraryPanelInFolder(name string) string {
	return fmt.Sprintf(`
resource "grafana_folder" "test_folder" {
	title = "%[1]s"
}

resource "grafana_library_panel" "test_folder" {
	name      = "%[1]s"
	folder_uid = grafana_folder.test_folder.uid
	model_json = jsonencode({
		title   = "%[1]s",
		id      = 12,
		version = 43,
	})
}`, name)
}

func testAccLibraryPanelInOrganization(orgName, folder string) string {
	var fuid string
	if folder == "" {
		fuid = "null"
	} else {
		fuid = fmt.Sprintf(`"%s"`, folder)
	}

	return fmt.Sprintf(`
resource "grafana_organization" "test" {
	name = "%[1]s"
}

resource "grafana_library_panel" "test" {
	org_id     = grafana_organization.test.id
	name       = "%[1]s"
    folder_uid = %[2]s
	model_json = jsonencode({
	  title   = "%[1]s",
	})
  }`, orgName, fuid)
}
