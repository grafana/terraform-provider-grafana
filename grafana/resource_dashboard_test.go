package grafana

import (
	"fmt"
	"regexp"
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDashboard_basic(t *testing.T) {
	var uid string

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccDashboardCheckDestroy(&uid),
		Steps: []resource.TestStep{
			// first step creates the resource
			{
				Config: testAccDashboardConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccDashboardCheckExists("grafana_dashboard.test", &uid),
					resource.TestMatchResourceAttr(
						"grafana_dashboard.test", "config_json", regexp.MustCompile(".*Terraform Acceptance Test.*"),
					),
				),
			},
			// second step updates it with a new title
			{
				Config: testAccDashboardConfig_update,
				Check: resource.ComposeTestCheckFunc(
					testAccDashboardCheckExists("grafana_dashboard.test", &uid),
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
	var uid string
	var folder gapi.Folder

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccDashboardFolderCheckDestroy(uid, &folder),
		Steps: []resource.TestStep{
			{
				Config: testAccDashboardConfig_folder,
				Check: resource.ComposeTestCheckFunc(
					testAccDashboardCheckExists("grafana_dashboard.test_folder", &uid),
					testAccFolderCheckExists("grafana_folder.test_folder", &folder),
					testAccDashboardCheckExistsInFolder(&uid, &folder),
					resource.TestMatchResourceAttr(
						"grafana_dashboard.test_folder", "config_json", regexp.MustCompile(".*Terraform Folder Test Dashboard.*"),
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
	var uid string

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccDashboardCheckDestroy(&uid),
		Steps: []resource.TestStep{
			{
				Config: testAccDashboardConfig_disappear,
				Check: resource.ComposeTestCheckFunc(
					testAccDashboardCheckExists("grafana_dashboard.test", &uid),
					testAccDashboardDisappear(&uid),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccDashboardCheckExists(rn string, uid *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		client := testAccProvider.Meta().(*gapi.Client)
		_, err := client.DashboardByUID(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("error getting dashboard: %s", err)
		}

		*uid = rs.Primary.ID

		return nil
	}
}

func testAccDashboardCheckExistsInFolder(uid *string, folder *gapi.Folder) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*gapi.Client)
		dashboard, err := client.DashboardByUID(*uid)
		if err != nil {
			return fmt.Errorf("error getting dashboard: %s", err)
		}

		if dashboard.Folder != folder.ID && folder.ID != 0 {
			return fmt.Errorf("dashboard.Folder(%d) does not match folder.ID(%d)", dashboard.Folder, folder.ID)
		}
		return nil
	}
}

func testAccDashboardDisappear(uid *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// At this point testAccDashboardCheckExists should have been called and
		// dashboard should have been populated
		client := testAccProvider.Meta().(*gapi.Client)
		client.DeleteDashboardByUID(*uid)
		return nil
	}
}

func testAccDashboardCheckDestroy(uid *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*gapi.Client)
		_, err := client.DashboardByUID(*uid)
		if err == nil {
			return fmt.Errorf("dashboard still exists")
		}
		return nil
	}
}

func testAccDashboardFolderCheckDestroy(uid string, folder *gapi.Folder) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*gapi.Client)
		_, err := client.DashboardByUID(uid)
		if err == nil {
			return fmt.Errorf("dashboard still exists")
		}
		_, err = client.Folder(folder.ID)
		if err == nil {
			return fmt.Errorf("folder still exists")
		}
		return nil
	}
}

// The "id" and "version" properties in the config below are there to test
// that we correctly normalize them away. They are not actually used by this
// resource, since it uses UIDs for identification.
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
