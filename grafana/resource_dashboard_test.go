package grafana

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"

	gapi "github.com/emerald-squad/go-grafana-api"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDashboard_basic(t *testing.T) {
	var dashboard gapi.Dashboard
	var testOrgID int64 = 1

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccDashboardCheckDestroy(&dashboard, testOrgID),
		Steps: []resource.TestStep{
			{
				Config: testAccDashboardConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccDashboardCheckExists("grafana_dashboard.test", &dashboard),
					resource.TestMatchResourceAttr(
						"grafana_dashboard.test", "id", regexp.MustCompile(`terraform-acceptance-test.*`),
					),
				),
			},
		},
	})
}

func TestAccDashboard_folder(t *testing.T) {
	var dashboard gapi.Dashboard
	var folder gapi.Folder
	var testOrgID int64 = 1

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccDashboardFolderCheckDestroy(&dashboard, &folder, testOrgID),
		Steps: []resource.TestStep{
			{
				Config: testAccDashboardConfig_folder,
				Check: resource.ComposeTestCheckFunc(
					testAccDashboardCheckExists("grafana_dashboard.test_folder", &dashboard),
					testAccFolderCheckExists("grafana_folder.test_folder", &folder),
					testAccDashboardCheckExistsInFolder(&dashboard, &folder),
					resource.TestMatchResourceAttr(
						"grafana_dashboard.test_folder", "id", regexp.MustCompile(`terraform-acceptance-test.*`),
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
	var testOrgID int64 = 1

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccDashboardCheckDestroy(&dashboard, testOrgID),
		Steps: []resource.TestStep{
			{
				Config: testAccDashboardConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccDashboardCheckExists("grafana_dashboard.test", &dashboard),
					testAccDashboardDisappear(&dashboard, testOrgID),
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

		orgID, err := strconv.ParseInt(rs.Primary.Attributes["org_id"], 10, 64)
		if err != nil {
			return fmt.Errorf("could not find org_id")
		}

		client := testAccProvider.Meta().(*gapi.Client)

		gotDashboard, err := client.Dashboard(rs.Primary.ID, orgID)
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

func testAccDashboardDisappear(dashboard *gapi.Dashboard, orgID int64) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// At this point testAccDashboardCheckExists should have been called and
		// dashboard should have been populated
		client := testAccProvider.Meta().(*gapi.Client)
		client.DeleteDashboard((*dashboard).Meta.Slug, orgID)
		return nil
	}
}

func testAccDashboardCheckDestroy(dashboard *gapi.Dashboard, orgID int64) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*gapi.Client)
		_, err := client.Dashboard(dashboard.Meta.Slug, orgID)
		if err == nil {
			return fmt.Errorf("dashboard still exists")
		}
		return nil
	}
}

func testAccDashboardFolderCheckDestroy(dashboard *gapi.Dashboard, folder *gapi.Folder, orgID int64) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*gapi.Client)
		_, err := client.Dashboard(dashboard.Meta.Slug, orgID)
		if err == nil {
			return fmt.Errorf("dashboard still exists")
		}
		_, err = client.Folder(folder.Id, orgID)
		if err == nil {
			return fmt.Errorf("folder still exists")
		}
		return nil
	}
}

// The "id" and "version" properties in the config below are there to test
// that we correctly normalize them away. They are not actually used by this
// resource, since it uses slugs for identification and never modifies an
// existing dashboard.
const testAccDashboardConfig_basic = `
resource "grafana_dashboard" "test" {
	org_id = 1
    config_json = <<EOT
{
    "title": "Terraform Acceptance Test",
    "id": 12,
    "version": "43"
}
EOT
}
`

const testAccDashboardConfig_folder = `

resource "grafana_folder" "test_folder" {
	org_id = 1
    title = "Terraform Dashboard Folder Acceptance Test"
}

resource "grafana_dashboard" "test_folder" {
	org_id = 1
    folder = "${grafana_folder.test_folder.id}"
    config_json = <<EOT
{
    "title": "Terraform Acceptance Test",
    "id": 12,
    "version": "43"
}
EOT
}
`
