package grafana

import (
	"fmt"
	"testing"

	gapi "github.com/albeego/grafana-api-golang-client"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccDashboard_basic(t *testing.T) {
	CheckOSSTestsEnabled(t)

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
				ResourceName:            "grafana_dashboard.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"message"},
			},
		},
	})
}

func TestAccDashboard_uid_unset(t *testing.T) {
	CheckOSSTestsEnabled(t)

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

func TestAccDashboard_computed_config(t *testing.T) {
	CheckOSSTestsEnabled(t)

	var dashboard gapi.Dashboard

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccDashboardCheckDestroy(&dashboard),
		Steps: []resource.TestStep{
			{
				// Test resource creation.
				Config: testAccExample(t, "resources/grafana_dashboard/_acc_computed.tf"),
				Check: resource.ComposeTestCheckFunc(
					testAccDashboardCheckExists("grafana_dashboard.test", &dashboard),
					testAccDashboardCheckExists("grafana_dashboard.test-computed", &dashboard),
				),
			},
		},
	})
}

func TestAccDashboard_folder(t *testing.T) {
	CheckOSSTestsEnabled(t)

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
					testAccFolderCheckExists("grafana_folder.test_folder", &folder),
					testAccDashboardCheckExistsInFolder(&dashboard, &folder),
					resource.TestCheckResourceAttr("grafana_dashboard.test_folder", "id", "folder"),
					resource.TestCheckResourceAttr("grafana_dashboard.test_folder", "uid", "folder"),
					resource.TestMatchResourceAttr(
						"grafana_dashboard.test_folder", "folder", idRegexp,
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
		folder, err = client.Folder(folder.ID)
		if err == nil {
			return fmt.Errorf("the following folder still exists: %s", folder.Title)
		}
		return nil
	}
}

func Test_normalizeDashboardConfigJSON(t *testing.T) {
	type args struct {
		config interface{}
	}

	d := "New Dashboard"
	expected := fmt.Sprintf("{\"title\":\"%s\"}", d)
	givenPanels, err := unmarshalDashboardConfigJSON(fmt.Sprintf("{\"panels\":[{\"libraryPanel\":{\"name\":\"%s\",\"uid\":\"%s\",\"description\":\"%s\"}}]}", "test", "test", "test"))
	if err != nil {
		t.Error(err)
	}
	expectedPanels := fmt.Sprintf("{\"panels\":[{\"libraryPanel\":{\"name\":\"%s\",\"uid\":\"%s\"}}]}", "test", "test")

	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "String dashboard is valid",
			args: args{config: fmt.Sprintf("{\"title\":\"%s\"}", d)},
			want: expected,
		},
		{
			name: "Map dashboard is valid",
			args: args{config: map[string]interface{}{"title": d}},
			want: expected,
		},
		{
			name: "Version is removed",
			args: args{config: map[string]interface{}{"title": d, "version": 10}},
			want: expected,
		},
		{
			name: "Id is removed",
			args: args{config: map[string]interface{}{"title": d, "id": 10}},
			want: expected,
		},
		{
			name: "Bad json is ignored",
			args: args{config: "74D93920-ED26–11E3-AC10–0800200C9A66"},
			want: "74D93920-ED26–11E3-AC10–0800200C9A66",
		},
		{
			name: "panels[].libraryPanel.!<name|uid> is removed",
			args: args{config: givenPanels},
			want: expectedPanels,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeDashboardConfigJSON(tt.args.config); got != tt.want {
				t.Errorf("normalizeDashboardConfigJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}
