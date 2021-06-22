package grafana

import (
	"fmt"
	"regexp"
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccDashboard_basic(t *testing.T) {
	var dashboard gapi.Dashboard

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccDashboardCheckDestroy(&dashboard),
		Steps: []resource.TestStep{
			{
				// Test resource creation.
				Config: testAccExample(t, "resources/grafana_dashboard/_acc_basic.tf"),
				Check: resource.ComposeTestCheckFunc(
					testAccDashboardCheckExists("grafana_dashboard.test", &dashboard),
					resource.TestCheckResourceAttr("grafana_dashboard.test", "id", "basic"),
					resource.TestCheckResourceAttr("grafana_dashboard.test", "uid", "basic"),
					resource.TestCheckResourceAttr(
						"grafana_dashboard.test", "config_json", `{"title":"Terraform Acceptance Test","uid":"basic"}`,
					),
				),
			},
			{
				// Updates title.
				Config: testAccExample(t, "resources/grafana_dashboard/_acc_basic_update.tf"),
				Check: resource.ComposeTestCheckFunc(
					testAccDashboardCheckExists("grafana_dashboard.test", &dashboard),
					resource.TestCheckResourceAttr("grafana_dashboard.test", "id", "basic"),
					resource.TestCheckResourceAttr("grafana_dashboard.test", "uid", "basic"),
					resource.TestCheckResourceAttr(
						"grafana_dashboard.test", "config_json", `{"title":"Updated Title","uid":"basic"}`,
					),
				),
			},
			{
				// Updates uid.
				// uid is removed from `config_json` before writing it to state so it's
				// important to ensure changing it triggers an update of `config_json`.
				Config: testAccExample(t, "resources/grafana_dashboard/_acc_basic_update_uid.tf"),
				Check: resource.ComposeTestCheckFunc(
					testAccDashboardCheckExists("grafana_dashboard.test", &dashboard),
					resource.TestCheckResourceAttr("grafana_dashboard.test", "id", "basic-update"),
					resource.TestCheckResourceAttr("grafana_dashboard.test", "uid", "basic-update"),
					resource.TestCheckResourceAttr(
						"grafana_dashboard.test", "config_json", `{"title":"Updated Title","uid":"basic-update"}`,
					),
				),
			},
			{
				// Importing matches the state of the previous step.
				ResourceName:      "grafana_dashboard.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccDashboard_uid_unset(t *testing.T) {
	var dashboard gapi.Dashboard

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccDashboardCheckDestroy(&dashboard),
		Steps: []resource.TestStep{
			{
				// Create dashboard with no uid set.
				Config: testAccExample(t, "resources/grafana_dashboard/_acc_uid_unset.tf"),
				Check: resource.ComposeTestCheckFunc(
					testAccDashboardCheckExists("grafana_dashboard.test", &dashboard),
					resource.TestCheckResourceAttr(
						"grafana_dashboard.test", "config_json", `{"title":"UID Unset"}`,
					),
				),
			},
			{
				// Update it to add a uid. We want to ensure that this causes a diff
				// and subsequent update.
				Config: testAccExample(t, "resources/grafana_dashboard/_acc_uid_unset_set.tf"),
				Check: resource.ComposeTestCheckFunc(
					testAccDashboardCheckExists("grafana_dashboard.test", &dashboard),
					resource.TestCheckResourceAttr(
						"grafana_dashboard.test", "config_json", `{"title":"UID Unset","uid":"uid-previously-unset"}`,
					),
				),
			},
			{
				// Remove the uid once again to ensure this is also supported.
				Config: testAccExample(t, "resources/grafana_dashboard/_acc_uid_unset.tf"),
				Check: resource.ComposeTestCheckFunc(
					testAccDashboardCheckExists("grafana_dashboard.test", &dashboard),
					resource.TestCheckResourceAttr(
						"grafana_dashboard.test", "config_json", `{"title":"UID Unset"}`,
					),
				),
			},
		},
	})
}

func TestAccDashboard_folder(t *testing.T) {
	var dashboard gapi.Dashboard
	var folder gapi.Folder

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccDashboardFolderCheckDestroy(&dashboard, &folder),
		Steps: []resource.TestStep{
			{
				Config: testAccExample(t, "resources/grafana_dashboard/_acc_folder.tf"),
				Check: resource.ComposeTestCheckFunc(
					testAccDashboardCheckExists("grafana_dashboard.test_folder", &dashboard),
					testAccDashboardCheckExistsInFolder(&dashboard, &folder),
					resource.TestCheckResourceAttr("grafana_dashboard.test_folder", "id", "folder"),
					resource.TestCheckResourceAttr("grafana_dashboard.test_folder", "uid", "folder"),
					resource.TestMatchResourceAttr(
						"grafana_dashboard.test_folder", "folder", regexp.MustCompile(`\d+`),
					),
				),
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
		client := testAccProvider.Meta().(*client).gapi
		gotDashboard, err := client.DashboardByUID(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("error getting dashboard: %s", err)
		}
		*dashboard = *gotDashboard
		return nil
	}
}

func testAccDashboardCheckExistsInFolder(dashboard *gapi.Dashboard, folder *gapi.Folder) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if dashboard.Folder != folder.ID && folder.ID != 0 {
			return fmt.Errorf("dashboard.Folder(%d) does not match folder.ID(%d)", dashboard.Folder, folder.ID)
		}
		return nil
	}
}

func testAccDashboardCheckDestroy(dashboard *gapi.Dashboard) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*client).gapi
		_, err := client.DashboardByUID(dashboard.Model["uid"].(string))
		if err == nil {
			return fmt.Errorf("dashboard still exists")
		}
		return nil
	}
}

func testAccDashboardFolderCheckDestroy(dashboard *gapi.Dashboard, folder *gapi.Folder) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*client).gapi
		_, err := client.DashboardByUID(dashboard.Model["uid"].(string))
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
