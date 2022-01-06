package grafana

import (
	"fmt"
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccLibraryPanel_basic(t *testing.T) {
	CheckOSSTestsEnabled(t)

	var panel gapi.LibraryPanel

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccLibraryPanelCheckDestroy(&panel),
		Steps: []resource.TestStep{
			{
				// Test resource creation.
				Config: testAccExample(t, "resources/grafana_library_panel/_acc_basic.tf"),
				Check: resource.ComposeTestCheckFunc(
					testAccLibraryPanelCheckExists("grafana_library_panel.test", &panel),
					resource.TestCheckResourceAttr("grafana_library_panel.test", "id", "basic"),
					resource.TestCheckResourceAttr("grafana_library_panel.test", "uid", "basic"),
					resource.TestCheckResourceAttr(
						"grafana_library_panel.test", "config_json", `{"title":"Terraform Acceptance Test","uid":"basic"}`,
					),
				),
			},
			{
				// Updates title.
				Config: testAccExample(t, "resources/grafana_library_panel/_acc_basic_update.tf"),
				Check: resource.ComposeTestCheckFunc(
					testAccLibraryPanelCheckExists("grafana_library_panel.test", &panel),
					resource.TestCheckResourceAttr("grafana_library_panel.test", "id", "basic"),
					resource.TestCheckResourceAttr("grafana_library_panel.test", "uid", "basic"),
					resource.TestCheckResourceAttr(
						"grafana_library_panel.test", "config_json", `{"title":"Updated Name","uid":"basic"}`,
					),
				),
			},
			{
				// Updates uid.
				// uid is removed from `config_json` before writing it to state so it's
				// important to ensure changing it triggers an update of `config_json`.
				Config: testAccExample(t, "resources/grafana_library_panel/_acc_basic_update_uid.tf"),
				Check: resource.ComposeTestCheckFunc(
					testAccLibraryPanelCheckExists("grafana_library_panel.test", &panel),
					resource.TestCheckResourceAttr("grafana_library_panel.test", "id", "basic-update"),
					resource.TestCheckResourceAttr("grafana_library_panel.test", "uid", "basic-update"),
					resource.TestCheckResourceAttr(
						"grafana_library_panel.test", "config_json", `{"title":"Updated Name","uid":"basic-update"}`,
					),
				),
			},
			{
				// Importing matches the state of the previous step.
				ResourceName:            "grafana_library_panel.test",
				ImportState:             true,
				ImportStateVerify:       true,
			},
		},
	})
}

func TestAccLibraryPanel_uid_unset(t *testing.T) {
	CheckOSSTestsEnabled(t)

	var panel gapi.LibraryPanel

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccLibraryPanelCheckDestroy(&panel),
		Steps: []resource.TestStep{
			{
				// Create panel with no uid set.
				Config: testAccExample(t, "resources/grafana_library_panel/_acc_uid_unset.tf"),
				Check: resource.ComposeTestCheckFunc(
					testAccLibraryPanelCheckExists("grafana_library_panel.test", &panel),
					resource.TestCheckResourceAttr(
						"grafana_library_panel.test", "config_json", `{"title":"UID Unset"}`,
					),
				),
			},
			{
				// Update it to add a uid. We want to ensure that this causes a diff
				// and subsequent update.
				Config: testAccExample(t, "resources/grafana_library_panel/_acc_uid_unset_set.tf"),
				Check: resource.ComposeTestCheckFunc(
					testAccLibraryPanelCheckExists("grafana_library_panel.test", &panel),
					resource.TestCheckResourceAttr(
						"grafana_library_panel.test", "config_json", `{"title":"UID Unset","uid":"uid-previously-unset"}`,
					),
				),
			},
			{
				// Remove the uid once again to ensure this is also supported.
				Config: testAccExample(t, "resources/grafana_library_panel/_acc_uid_unset.tf"),
				Check: resource.ComposeTestCheckFunc(
					testAccLibraryPanelCheckExists("grafana_library_panel.test", &panel),
					resource.TestCheckResourceAttr(
						"grafana_library_panel.test", "config_json", `{"title":"UID Unset"}`,
					),
				),
			},
		},
	})
}

func TestAccLibraryPanel_computed_config(t *testing.T) {
	CheckOSSTestsEnabled(t)

	var panel gapi.LibraryPanel

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccLibraryPanelCheckDestroy(&panel),
		Steps: []resource.TestStep{
			{
				// Test resource creation.
				Config: testAccExample(t, "resources/grafana_library_panel/_acc_computed.tf"),
				Check: resource.ComposeTestCheckFunc(
					testAccLibraryPanelCheckExists("grafana_library_panel.test", &panel),
					testAccLibraryPanelCheckExists("grafana_library_panel.test-computed", &panel),
				),
			},
		},
	})
}

func TestAccLibraryPanel_folder(t *testing.T) {
	CheckOSSTestsEnabled(t)

	var panel gapi.LibraryPanel
	var folder gapi.Folder

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccLibraryPanelFolderCheckDestroy(&panel, &folder),
		Steps: []resource.TestStep{
			{
				Config: testAccExample(t, "resources/grafana_library_panel/_acc_folder.tf"),
				Check: resource.ComposeTestCheckFunc(
					testAccLibraryPanelCheckExists("grafana_library_panel.test_folder", &panel),
					testAccFolderCheckExists("grafana_folder.test_folder", &folder),
					testAccLibraryPanelCheckExistsInFolder(&panel, &folder),
					resource.TestCheckResourceAttr("grafana_library_panel.test_folder", "id", "folder"),
					resource.TestCheckResourceAttr("grafana_library_panel.test_folder", "uid", "folder"),
					resource.TestMatchResourceAttr(
						"grafana_library_panel.test_folder", "folder", idRegexp,
					),
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
		client := testAccProvider.Meta().(*client).gapi
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
		client := testAccProvider.Meta().(*client).gapi
		_, err := client.LibraryPanelByUID(panel.Model["uid"].(string))
		if err == nil {
			return fmt.Errorf("panel still exists")
		}
		return nil
	}
}

func testAccLibraryPanelFolderCheckDestroy(panel *gapi.LibraryPanel, folder *gapi.Folder) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*client).gapi
		_, err := client.LibraryPanelByUID(panel.Model["uid"].(string))
		if err == nil {
			return fmt.Errorf("panel still exists")
		}
		folder, err = client.Folder(folder.ID)
		if err == nil {
			return fmt.Errorf("the following folder still exists: %s", folder.Title)
		}
		return nil
	}
}

func Test_normalizeLibraryPanelConfigJSON(t *testing.T) {
	type args struct {
		config interface{}
	}

	d := "New LibraryPanel"
	expected := fmt.Sprintf("{\"title\":\"%s\"}", d)

	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "String panel is valid",
			args: args{config: fmt.Sprintf("{\"title\":\"%s\"}", d)},
			want: expected,
		},
		{
			name: "Map panel is valid",
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeLibraryPanelConfigJSON(tt.args.config); got != tt.want {
				t.Errorf("normalizeLibraryPanelConfigJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}
