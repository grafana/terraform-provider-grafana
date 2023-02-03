package provider

import (
	"fmt"
	"os"
	"strings"
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/provider/common"
	"github.com/grafana/terraform-provider-grafana/provider/testutils"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccDashboard_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	var dashboard gapi.Dashboard

	for _, useSHA256 := range []bool{false, true} {
		t.Run(fmt.Sprintf("useSHA256=%t", useSHA256), func(t *testing.T) {
			os.Setenv("GRAFANA_STORE_DASHBOARD_SHA256", fmt.Sprintf("%t", useSHA256))
			defer os.Unsetenv("GRAFANA_STORE_DASHBOARD_SHA256")

			expectedInitialConfig := `{"title":"Terraform Acceptance Test","uid":"basic"}`
			expectedUpdatedTitleConfig := `{"title":"Updated Title","uid":"basic"}`
			expectedUpdatedUIDConfig := `{"title":"Updated Title","uid":"basic-update"}`
			if useSHA256 {
				expectedInitialConfig = "fadbc115a19bfd7962d8f8d749d22c20d0a44043d390048bf94b698776d9f7f1"
				expectedUpdatedTitleConfig = "4669abda43a4a6d6ae9ecaa19f8508faf4095682b679da0b5ce4176aa9171ab2"
				expectedUpdatedUIDConfig = "2934e80938a672bd09d8e56385159a1bf8176e2a2ef549437f200d82ff398bfb"
			}

			// TODO: Make parallelizable
			resource.Test(t, resource.TestCase{
				ProviderFactories: testAccProviderFactories,
				CheckDestroy:      testAccDashboardCheckDestroy(&dashboard, 0),
				Steps: []resource.TestStep{
					{
						// Test resource creation.
						Config: testutils.TestAccExample(t, "resources/grafana_dashboard/_acc_basic.tf"),
						Check: resource.ComposeTestCheckFunc(
							testAccDashboardCheckExists("grafana_dashboard.test", &dashboard),
							resource.TestCheckResourceAttr("grafana_dashboard.test", "id", "0:basic"), // <org id>:<uid>
							resource.TestCheckResourceAttr("grafana_dashboard.test", "uid", "basic"),
							resource.TestCheckResourceAttr("grafana_dashboard.test", "url", strings.TrimRight(os.Getenv("GRAFANA_URL"), "/")+"/d/basic/terraform-acceptance-test"),
							resource.TestCheckResourceAttr(
								"grafana_dashboard.test", "config_json", expectedInitialConfig,
							),
						),
					},
					{
						// Updates title.
						Config: testutils.TestAccExample(t, "resources/grafana_dashboard/_acc_basic_update.tf"),
						Check: resource.ComposeTestCheckFunc(
							testAccDashboardCheckExists("grafana_dashboard.test", &dashboard),
							resource.TestCheckResourceAttr("grafana_dashboard.test", "id", "0:basic"), // <org id>:<uid>
							resource.TestCheckResourceAttr("grafana_dashboard.test", "uid", "basic"),
							resource.TestCheckResourceAttr(
								"grafana_dashboard.test", "config_json", expectedUpdatedTitleConfig,
							),
						),
					},
					{
						// Updates uid.
						// uid is removed from `config_json` before writing it to state so it's
						// important to ensure changing it triggers an update of `config_json`.
						Config: testutils.TestAccExample(t, "resources/grafana_dashboard/_acc_basic_update_uid.tf"),
						Check: resource.ComposeTestCheckFunc(
							testAccDashboardCheckExists("grafana_dashboard.test", &dashboard),
							resource.TestCheckResourceAttr("grafana_dashboard.test", "id", "0:basic-update"), // <org id>:<uid>
							resource.TestCheckResourceAttr("grafana_dashboard.test", "uid", "basic-update"),
							resource.TestCheckResourceAttr("grafana_dashboard.test", "url", strings.TrimRight(os.Getenv("GRAFANA_URL"), "/")+"/d/basic-update/updated-title"),
							resource.TestCheckResourceAttr(
								"grafana_dashboard.test", "config_json", expectedUpdatedUIDConfig,
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
		})
	}
}

func TestAccDashboard_uid_unset(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	var dashboard gapi.Dashboard

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccDashboardCheckDestroy(&dashboard, 0),
		Steps: []resource.TestStep{
			{
				// Create dashboard with no uid set.
				Config: testutils.TestAccExample(t, "resources/grafana_dashboard/_acc_uid_unset.tf"),
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
				Config: testutils.TestAccExample(t, "resources/grafana_dashboard/_acc_uid_unset_set.tf"),
				Check: resource.ComposeTestCheckFunc(
					testAccDashboardCheckExists("grafana_dashboard.test", &dashboard),
					resource.TestCheckResourceAttr(
						"grafana_dashboard.test", "config_json", `{"title":"UID Unset","uid":"uid-previously-unset"}`,
					),
				),
			},
			{
				// Remove the uid once again to ensure this is also supported.
				Config: testutils.TestAccExample(t, "resources/grafana_dashboard/_acc_uid_unset.tf"),
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
	testutils.CheckOSSTestsEnabled(t)

	var dashboard gapi.Dashboard

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccDashboardCheckDestroy(&dashboard, 0),
		Steps: []resource.TestStep{
			{
				// Test resource creation.
				Config: testutils.TestAccExample(t, "resources/grafana_dashboard/_acc_computed.tf"),
				Check: resource.ComposeTestCheckFunc(
					testAccDashboardCheckExists("grafana_dashboard.test", &dashboard),
					testAccDashboardCheckExists("grafana_dashboard.test-computed", &dashboard),
				),
			},
		},
	})
}

func TestAccDashboard_folder(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	var dashboard gapi.Dashboard
	var folder gapi.Folder

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccDashboardFolderCheckDestroy(&dashboard, &folder),
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "resources/grafana_dashboard/_acc_folder.tf"),
				Check: resource.ComposeTestCheckFunc(
					testAccDashboardCheckExists("grafana_dashboard.test_folder", &dashboard),
					testAccFolderCheckExists("grafana_folder.test_folder", &folder),
					testAccDashboardCheckExistsInFolder(&dashboard, &folder),
					resource.TestCheckResourceAttr("grafana_dashboard.test_folder", "id", "0:folder"), // <org id>:<uid>
					resource.TestCheckResourceAttr("grafana_dashboard.test_folder", "uid", "folder"),
					resource.TestMatchResourceAttr(
						"grafana_dashboard.test_folder", "folder", idRegexp,
					),
				),
			},
		},
	})
}

func TestAccDashboard_inOrg(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	var dashboard gapi.Dashboard
	var org gapi.Org

	orgName := acctest.RandString(10)

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccDashboardCheckDestroy(&dashboard, org.ID),
		Steps: []resource.TestStep{
			{
				Config: testAccDashboardInOrganization(orgName),
				Check: resource.ComposeTestCheckFunc(
					testAccOrganizationCheckExists("grafana_organization.test", &org),
					testAccDashboardCheckExists("grafana_dashboard.test", &dashboard),
					resource.TestCheckResourceAttr("grafana_dashboard.test", "uid", "dashboard-"+orgName),
					// Check that the dashboard is not in the default org, from its ID
					func(s *terraform.State) error {
						expectedOrgID := org.ID
						if org.ID <= 1 {
							return fmt.Errorf("expected org ID higher than 1, got %d", org.ID)
						}
						expectedDashboardID := fmt.Sprintf("%d:dashboard-%s", expectedOrgID, orgName) // <org id>:<uid>
						checkFunc := resource.TestCheckResourceAttr("grafana_dashboard.test", "id", expectedDashboardID)
						return checkFunc(s)
					},
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
		orgID, dashboardUID := splitOSSOrgID(rs.Primary.ID)

		client := testAccProvider.Meta().(*common.Client).GrafanaAPI
		// If the org ID is set, check that the dashboard doesn't exist in the default org
		if orgID > 0 {
			dashboard, err := client.DashboardByUID(dashboardUID)
			if err == nil || dashboard != nil {
				return fmt.Errorf("dashboard %s exists in the default org", dashboardUID)
			}
			client = client.WithOrgID(orgID)
		}

		// Check that the dashboard is in the expected org
		gotDashboard, err := client.DashboardByUID(dashboardUID)
		if err != nil {
			return fmt.Errorf("error getting dashboard: %s", err)
		}
		*dashboard = *gotDashboard
		return nil
	}
}

func testAccDashboardCheckExistsInFolder(dashboard *gapi.Dashboard, folder *gapi.Folder) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if dashboard.FolderID != folder.ID && folder.ID != 0 {
			return fmt.Errorf("dashboard.Folder(%d) does not match folder.ID(%d)", dashboard.FolderID, folder.ID)
		}
		return nil
	}
}

func testAccDashboardCheckDestroy(dashboard *gapi.Dashboard, orgID int64) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Check that the dashboard was deleted from the default organization
		client := testAccProvider.Meta().(*common.Client).GrafanaAPI
		dashboard, err := client.DashboardByUID(dashboard.Model["uid"].(string))
		if dashboard != nil || err == nil {
			return fmt.Errorf("dashboard still exists")
		}

		// If the dashboard was created in an organization, check that it was deleted from that organization
		if orgID != 0 {
			client = client.WithOrgID(orgID)
			dashboard, err = client.DashboardByUID(dashboard.Model["uid"].(string))
			if dashboard != nil || err == nil {
				return fmt.Errorf("dashboard still exists")
			}
		}

		return nil
	}
}

func testAccDashboardFolderCheckDestroy(dashboard *gapi.Dashboard, folder *gapi.Folder) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*common.Client).GrafanaAPI
		_, err := client.DashboardByUID(dashboard.Model["uid"].(string))
		if err == nil {
			return fmt.Errorf("dashboard still exists")
		}
		folder, err = getFolderByID(client, folder.ID)
		if err == nil {
			return fmt.Errorf("the following folder still exists: %s", folder.Title)
		}
		return nil
	}
}

func Test_normalizeDashboardConfigJSON(t *testing.T) {
	testutils.IsUnitTest(t)

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

func testAccDashboardInOrganization(orgName string) string {
	return fmt.Sprintf(`
resource "grafana_organization" "test" {
	name = "%[1]s"
}

resource "grafana_dashboard" "test" {
	org_id      = grafana_organization.test.id
	config_json = jsonencode({
	  title = "dashboard-%[1]s"
	  uid   = "dashboard-%[1]s"
	})
}`, orgName)
}
