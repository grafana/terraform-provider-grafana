package grafana

import (
	"fmt"
	"regexp"
	"testing"

	gapi "github.com/nytm/go-grafana-api"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

func TestAccDashboard_basic(t *testing.T) {
	var dashboard gapi.Dashboard

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccDashboardCheckDestroy(&dashboard),
		Steps: []resource.TestStep{
			// first step creates the resource
			{
				Config: testAccDashboardConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccDashboardCheckExists("grafana_dashboard.test", &dashboard),
					resource.TestMatchResourceAttr(
						"grafana_dashboard.test", "id", regexp.MustCompile(`terraform-acceptance-test.*`),
					),
					resource.TestMatchResourceAttr(
						"grafana_dashboard.test", "config_json", regexp.MustCompile(".*Terraform Acceptance Test.*"),
					),
				),
			},
			// second step updates it with a new title
			{
				Config: testAccDashboardConfig_update,
				Check: resource.ComposeTestCheckFunc(
					testAccDashboardCheckExists("grafana_dashboard.test", &dashboard),
					resource.TestMatchResourceAttr(
						"grafana_dashboard.test", "config_json", regexp.MustCompile(".*Updated Title.*"),
					),
				),
			},
			// final step checks importing the current state we reached in the step above
			{
				ResourceName:      "grafana_dashboard.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccDashboard_folder(t *testing.T) {
	var dashboard gapi.Dashboard
	var folder gapi.Folder

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccDashboardFolderCheckDestroy(&dashboard, &folder),
		Steps: []resource.TestStep{
			{
				Config: testAccDashboardConfig_folder,
				Check: resource.ComposeTestCheckFunc(
					testAccDashboardCheckExists("grafana_dashboard.test_folder", &dashboard),
					testAccFolderCheckExists("grafana_folder.test_folder", &folder),
					testAccDashboardCheckExistsInFolder(&dashboard, &folder),
					resource.TestMatchResourceAttr(
						"grafana_dashboard.test_folder", "id", regexp.MustCompile(`terraform-folder-test-dashboard`),
					),
					resource.TestMatchResourceAttr(
						"grafana_dashboard.test_folder", "folder", regexp.MustCompile(`\d+`),
					),
				),
			},
		},
	})
}

func TestAccDashboard_disappear(t *testing.T) {
	var dashboard gapi.Dashboard

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccDashboardCheckDestroy(&dashboard),
		Steps: []resource.TestStep{
			{
				Config: testAccDashboardConfig_disappear,
				Check: resource.ComposeTestCheckFunc(
					testAccDashboardCheckExists("grafana_dashboard.test", &dashboard),
					testAccDashboardDisappear(&dashboard),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccDashboardCheckExists(rn string, dashboard *gapi.Dashboard) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		client := testAccProvider.Meta().(*gapi.Client)
		gotDashboard, err := client.Dashboard(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("error getting dashboard: %s", err)
		}

		*dashboard = *gotDashboard

		return nil
	}
}

func testAccDashboardCheckExistsInFolder(dashboard *gapi.Dashboard, folder *gapi.Folder) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if dashboard.Folder != folder.Id && folder.Id != 0 {
			return fmt.Errorf("dashboard.Folder(%d) does not match folder.Id(%d)", dashboard.Folder, folder.Id)
		}
		return nil
	}
}

func testAccDashboardDisappear(dashboard *gapi.Dashboard) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// At this point testAccDashboardCheckExists should have been called and
		// dashboard should have been populated
		client := testAccProvider.Meta().(*gapi.Client)
		client.DeleteDashboard((*dashboard).Meta.Slug)
		return nil
	}
}

func testAccDashboardCheckDestroy(dashboard *gapi.Dashboard) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*gapi.Client)
		_, err := client.Dashboard(dashboard.Meta.Slug)
		if err == nil {
			return fmt.Errorf("dashboard still exists")
		}
		return nil
	}
}

func testAccDashboardFolderCheckDestroy(dashboard *gapi.Dashboard, folder *gapi.Folder) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*gapi.Client)
		_, err := client.Dashboard(dashboard.Meta.Slug)
		if err == nil {
			return fmt.Errorf("dashboard still exists")
		}
		_, err = client.Folder(folder.Id)
		if err == nil {
			return fmt.Errorf("folder still exists")
		}
		return nil
	}
}

// The "id" and "version" properties in the config below are there to test
// that we correctly normalize them away. They are not actually used by this
// resource, since it uses slugs for identification.
const testAccDashboardConfig_basic = `
resource "grafana_dashboard" "test" {
    config_json = <<EOT
{
    "title": "Terraform Acceptance Test",
    "id": 12,
    "version": "43"
}
EOT
}
`

// this is used as an update on the basic resource above
// NOTE: it leaves out id and version, as this is what users will do when updating
const testAccDashboardConfig_update = `
resource "grafana_dashboard" "test" {
	config_json = <<EOT
{
	"title": "Updated Title"
}
EOT
}
`

const testAccDashboardConfig_folder = `

resource "grafana_folder" "test_folder" {
    title = "Terraform Folder Test Folder"
}

resource "grafana_dashboard" "test_folder" {
    folder = "${grafana_folder.test_folder.id}"
    config_json = <<EOT
{
    "title": "Terraform Folder Test Dashboard",
    "id": 12,
    "version": "43"
}
EOT
}
`

const testAccDashboardConfig_disappear = `
resource "grafana_dashboard" "test" {
    config_json = <<EOT
{
    "title": "Terraform Disappear Test"
}
EOT
}
`
