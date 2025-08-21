package grafana_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/resources/grafana"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccDashboard_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	var dashboard models.DashboardFullWithMeta

	for _, useSHA256 := range []bool{false, true} {
		t.Run(fmt.Sprintf("useSHA256=%t", useSHA256), func(t *testing.T) {
			os.Setenv("GRAFANA_STORE_DASHBOARD_SHA256", fmt.Sprintf("%t", useSHA256))
			defer os.Unsetenv("GRAFANA_STORE_DASHBOARD_SHA256")

			expectedInitialConfig := `{"title":"Terraform Acceptance Test","uid":"basic"}`
			expectedUpdatedTitleConfig := `{"title":"Updated Title","uid":"basic"}`
			expectedUpdatedUIDConfig := `{"title":"Updated Title","uid":"basic-update"}`
			if useSHA256 {
				expectedInitialConfig = "fadbc115a19bfd7962d8f8d749d22c20d0a44043d390048bf94b698776d9f7f1"      //nolint:gosec
				expectedUpdatedTitleConfig = "4669abda43a4a6d6ae9ecaa19f8508faf4095682b679da0b5ce4176aa9171ab2" //nolint:gosec
				expectedUpdatedUIDConfig = "2934e80938a672bd09d8e56385159a1bf8176e2a2ef549437f200d82ff398bfb"   //nolint:gosec
			}

			// TODO: Make parallelizable
			resource.Test(t, resource.TestCase{
				ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
				CheckDestroy:             dashboardCheckExists.destroyed(&dashboard, nil),
				Steps: []resource.TestStep{
					{
						// Test resource creation.
						Config: testutils.TestAccExample(t, "resources/grafana_dashboard/_acc_basic.tf"),
						Check: resource.ComposeTestCheckFunc(
							dashboardCheckExists.exists("grafana_dashboard.test", &dashboard),
							resource.TestCheckResourceAttr("grafana_dashboard.test", "id", "1:basic"), // <org id>:<uid>
							resource.TestCheckResourceAttr("grafana_dashboard.test", "org_id", "1"),
							resource.TestCheckResourceAttr("grafana_dashboard.test", "uid", "basic"),
							resource.TestCheckResourceAttr("grafana_dashboard.test", "url", strings.TrimRight(os.Getenv("GRAFANA_URL"), "/")+"/d/basic/terraform-acceptance-test"),
							resource.TestCheckResourceAttr(
								"grafana_dashboard.test", "config_json", expectedInitialConfig,
							),
							testutils.CheckLister("grafana_dashboard.test"),
						),
					},
					{
						// Updates title.
						Config: testutils.TestAccExample(t, "resources/grafana_dashboard/_acc_basic_update.tf"),
						Check: resource.ComposeTestCheckFunc(
							dashboardCheckExists.exists("grafana_dashboard.test", &dashboard),
							resource.TestCheckResourceAttr("grafana_dashboard.test", "id", "1:basic"), // <org id>:<uid>
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
							dashboardCheckExists.exists("grafana_dashboard.test", &dashboard),
							resource.TestCheckResourceAttr("grafana_dashboard.test", "id", "1:basic-update"), // <org id>:<uid>
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

	var dashboard models.DashboardFullWithMeta

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             dashboardCheckExists.destroyed(&dashboard, nil),
		Steps: []resource.TestStep{
			{
				// Create dashboard with no uid set.
				Config: testutils.TestAccExample(t, "resources/grafana_dashboard/_acc_uid_unset.tf"),
				Check: resource.ComposeTestCheckFunc(
					dashboardCheckExists.exists("grafana_dashboard.test", &dashboard),
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
					dashboardCheckExists.exists("grafana_dashboard.test", &dashboard),
					resource.TestCheckResourceAttr(
						"grafana_dashboard.test", "config_json", `{"title":"UID Unset","uid":"uid-previously-unset"}`,
					),
				),
			},
			{
				// Remove the uid once again to ensure this is also supported.
				Config: testutils.TestAccExample(t, "resources/grafana_dashboard/_acc_uid_unset.tf"),
				Check: resource.ComposeTestCheckFunc(
					dashboardCheckExists.exists("grafana_dashboard.test", &dashboard),
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

	var dashboard models.DashboardFullWithMeta

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             dashboardCheckExists.destroyed(&dashboard, nil),
		Steps: []resource.TestStep{
			{
				// Test resource creation.
				Config: testutils.TestAccExample(t, "resources/grafana_dashboard/_acc_computed.tf"),
				Check: resource.ComposeTestCheckFunc(
					dashboardCheckExists.exists("grafana_dashboard.test", &dashboard),
					dashboardCheckExists.exists("grafana_dashboard.test-computed", &dashboard),
				),
			},
		},
	})
}

func TestAccDashboard_folder_uid(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=8.0.0") // UID in folders were added in v8

	uid := acctest.RandString(10)

	var dashboard models.DashboardFullWithMeta
	var folder models.Folder

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			dashboardCheckExists.destroyed(&dashboard, nil),
			folderCheckExists.destroyed(&folder, nil),
		),
		Steps: []resource.TestStep{
			{
				Config: testAccDashboardFolder(uid, "grafana_folder.test_folder1.uid"),
				Check: resource.ComposeTestCheckFunc(
					folderCheckExists.exists("grafana_folder.test_folder1", &folder),
					dashboardCheckExists.exists("grafana_dashboard.test_folder", &dashboard),
					testAccDashboardCheckExistsInFolder(&dashboard, &folder),
					resource.TestCheckResourceAttr("grafana_dashboard.test_folder", "id", "1:"+uid), // <org id>:<uid>
					resource.TestCheckResourceAttr("grafana_dashboard.test_folder", "uid", uid),
					resource.TestCheckResourceAttr("grafana_dashboard.test_folder", "folder", uid+"-1"),
				),
			},
			// Update folder
			{
				Config: testAccDashboardFolder(uid, "grafana_folder.test_folder2.uid"),
				Check: resource.ComposeTestCheckFunc(
					folderCheckExists.exists("grafana_folder.test_folder2", &folder),
					dashboardCheckExists.exists("grafana_dashboard.test_folder", &dashboard),
					testAccDashboardCheckExistsInFolder(&dashboard, &folder),
					resource.TestCheckResourceAttr("grafana_dashboard.test_folder", "id", "1:"+uid), // <org id>:<uid>
					resource.TestCheckResourceAttr("grafana_dashboard.test_folder", "uid", uid),
					resource.TestCheckResourceAttr("grafana_dashboard.test_folder", "folder", uid+"-2"),
				),
			},
			{
				ImportState:       true,
				ResourceName:      "grafana_dashboard.test_folder",
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccDashboard_inOrg(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	var dashboard models.DashboardFullWithMeta
	var folder models.Folder
	var org models.OrgDetailsDTO

	orgName := acctest.RandString(10)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			dashboardCheckExists.destroyed(&dashboard, &org),
			folderCheckExists.destroyed(&folder, &org),
		),
		Steps: []resource.TestStep{
			{
				Config: testAccDashboardInOrganization(orgName),
				Check: resource.ComposeTestCheckFunc(
					orgCheckExists.exists("grafana_organization.test", &org),
					// Check that the folder is in the correct organization
					folderCheckExists.exists("grafana_folder.test", &folder),
					resource.TestCheckResourceAttr("grafana_folder.test", "uid", "folder-"+orgName),
					resource.TestMatchResourceAttr("grafana_folder.test", "id", nonDefaultOrgIDRegexp),
					checkResourceIsInOrg("grafana_folder.test", "grafana_organization.test"),

					// Check that the dashboard is in the correct organization
					dashboardCheckExists.exists("grafana_dashboard.test", &dashboard),
					resource.TestCheckResourceAttr("grafana_dashboard.test", "uid", "dashboard-"+orgName),
					resource.TestMatchResourceAttr("grafana_dashboard.test", "id", nonDefaultOrgIDRegexp),
					checkResourceIsInOrg("grafana_dashboard.test", "grafana_organization.test"),

					testAccDashboardCheckExistsInFolder(&dashboard, &folder),
					testutils.CheckLister("grafana_dashboard.test"),
				),
			},
		},
	})
}

func testAccDashboardCheckExistsInFolder(dashboard *models.DashboardFullWithMeta, folder *models.Folder) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if dashboard.Meta.FolderUID != folder.UID && folder.UID != "" {
			return fmt.Errorf("dashboard.Folder(%s) does not match folder.ID(%s)", dashboard.Meta.FolderUID, folder.UID)
		}
		return nil
	}
}

func Test_NormalizeDashboardConfigJSON(t *testing.T) {
	testutils.IsUnitTest(t)

	type args struct {
		config interface{}
	}

	d := "New Dashboard"
	expected := fmt.Sprintf("{\"title\":\"%s\"}", d)
	givenPanels, err := grafana.UnmarshalDashboardConfigJSON(fmt.Sprintf("{\"panels\":[{\"libraryPanel\":{\"name\":\"%s\",\"uid\":\"%s\",\"description\":\"%s\"}}]}", "test", "test", "test"))
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
			if got := grafana.NormalizeDashboardConfigJSON(tt.args.config); got != tt.want {
				t.Errorf("NormalizeDashboardConfigJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func testAccDashboardFolder(uid string, folderRef string) string {
	return fmt.Sprintf(`
resource "grafana_folder" "test_folder1" {
	title = "%[1]s-1"
	uid   = "%[1]s-1"
}

resource "grafana_folder" "test_folder2" {
	title = "%[1]s-2"
	uid   = "%[1]s-2"
}

resource "grafana_dashboard" "test_folder" {
	folder = %[2]s
	config_json = jsonencode({
		"title" : "%[1]s",
		"uid" : "%[1]s"
	})
}`, uid, folderRef)
}

func testAccDashboardInOrganization(orgName string) string {
	return fmt.Sprintf(`
resource "grafana_organization" "test" {
	name = "%[1]s"
}

resource "grafana_folder" "test" {
	org_id  = grafana_organization.test.id
	title   = "folder-%[1]s"
	uid     = "folder-%[1]s"
}

resource "grafana_dashboard" "test" {
	org_id      = grafana_organization.test.id
	folder      = grafana_folder.test.id
	config_json = jsonencode({
	  title = "dashboard-%[1]s"
	  uid   = "dashboard-%[1]s"
	})
}`, orgName)
}
