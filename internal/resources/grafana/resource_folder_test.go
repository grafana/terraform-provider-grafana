package grafana_test

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	goapi "github.com/grafana/grafana-openapi-client-go/models"

	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/grafana/terraform-provider-grafana/internal/resources/grafana"
	"github.com/grafana/terraform-provider-grafana/internal/testutils"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccFolder_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	var folder goapi.Folder
	var folderWithUID goapi.Folder

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccFolderCheckDestroy(&folder, 0),
			testAccFolderCheckDestroy(&folderWithUID, 0),
		),
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "resources/grafana_folder/resource.tf"),
				Check: resource.ComposeTestCheckFunc(
					testAccFolderCheckExists("grafana_folder.test_folder", &folder),
					resource.TestMatchResourceAttr("grafana_folder.test_folder", "id", defaultOrgIDRegexp),
					resource.TestCheckResourceAttr("grafana_folder.test_folder", "org_id", "1"),
					resource.TestMatchResourceAttr("grafana_folder.test_folder", "uid", common.UIDRegexp),
					resource.TestCheckResourceAttr("grafana_folder.test_folder", "title", "Terraform Test Folder"),

					testAccFolderCheckExists("grafana_folder.test_folder_with_uid", &folderWithUID),
					resource.TestMatchResourceAttr("grafana_folder.test_folder_with_uid", "id", defaultOrgIDRegexp),
					resource.TestCheckResourceAttr("grafana_folder.test_folder_with_uid", "uid", "test-folder-uid"),
					resource.TestCheckResourceAttr("grafana_folder.test_folder_with_uid", "title", "Terraform Test Folder With UID"),
					resource.TestCheckResourceAttr("grafana_folder.test_folder_with_uid", "url", strings.TrimRight(os.Getenv("GRAFANA_URL"), "/")+"/dashboards/f/test-folder-uid/terraform-test-folder-with-uid"),
				),
			},
			{
				ResourceName:            "grafana_folder.test_folder",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"prevent_destroy_if_not_empty"},
			},
			{
				ResourceName:            "grafana_folder.test_folder_with_uid",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"prevent_destroy_if_not_empty"},
			},
			// Change the title of one folder, change the UID of the other. They shouldn't change IDs (the folder doesn't have to be recreated)
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_folder/resource.tf", map[string]string{
					"Terraform Test Folder": "Terraform Test Folder Updated",
					"test-folder-uid":       "test-folder-uid-other",
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccFolderIDDidntChange("grafana_folder.test_folder", &folder),
					resource.TestMatchResourceAttr("grafana_folder.test_folder", "id", defaultOrgIDRegexp),
					resource.TestMatchResourceAttr("grafana_folder.test_folder", "uid", common.UIDRegexp),
					resource.TestCheckResourceAttr("grafana_folder.test_folder", "title", "Terraform Test Folder Updated"),

					testAccFolderIDDidntChange("grafana_folder.test_folder_with_uid", &folderWithUID),
					resource.TestMatchResourceAttr("grafana_folder.test_folder_with_uid", "id", defaultOrgIDRegexp),
					resource.TestCheckResourceAttr("grafana_folder.test_folder_with_uid", "uid", "test-folder-uid-other"),
					resource.TestCheckResourceAttr("grafana_folder.test_folder_with_uid", "title", "Terraform Test Folder Updated With UID"),
				),
			},
			// Test import using ID
			{
				ResourceName: "grafana_folder.test_folder",
				ImportState:  true,
			},
			// Test import using UID
			{
				ResourceName: "grafana_folder.test_folder_with_uid",
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["grafana_folder.test_folder_with_uid"]
					if !ok {
						return "", fmt.Errorf("resource not found: %s", "grafana_folder.test_folder_with_uid")
					}

					if rs.Primary.ID == "" {
						return "", fmt.Errorf("resource id not set")
					}
					return rs.Primary.Attributes["uid"], nil
				},
			},
		},
	})
}

func TestAccFolder_PreventDeletion(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	name := acctest.RandStringFromCharSet(10, acctest.CharSetAlpha)
	var folder goapi.Folder

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFolderExample_PreventDeletion(name, true), // Create protected folder
			},
			{
				Config:  testAccFolderExample_PreventDeletion(name, true), // Create protected folder
				Destroy: true,
			},
			{
				Config: testAccFolderExample_PreventDeletion(name, true), // Create protected folder again
				Check: resource.ComposeTestCheckFunc(
					testAccFolderCheckExists("grafana_folder.test_folder", &folder),
					// Create a dashboard in the protected folder
					func(s *terraform.State) error {
						client := testutils.Provider.Meta().(*common.Client).GrafanaAPI
						_, err := client.NewDashboard(gapi.Dashboard{
							FolderUID: folder.UID,
							FolderID:  folder.ID,
							Model: map[string]interface{}{
								"uid":   name + "-dashboard",
								"title": name + "-dashboard",
							}})
						return err
					},
				),
			},
			{
				Config:  testAccFolderExample_PreventDeletion(name, true),
				Destroy: true, // Try to delete the protected folder
				ExpectError: regexp.MustCompile(
					fmt.Sprintf(`.+folder %s is not empty and prevent_destroy_if_not_empty is set.+`, name),
				), // Fail because it's protected
			},
			{
				Config: testAccFolderExample_PreventDeletion(name, false), // Remove protected flag
			},
			{
				Config:  testAccFolderExample_PreventDeletion(name, false),
				Destroy: true, // No error if the folder is not protected
			},
		},
	})
}

// This is a bug in Grafana, not the provider. It was fixed in 9.2.7+ and 9.3.0+, this test will check for regressions
func TestAccFolder_createFromDifferentRoles(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)
	testutils.CheckOSSTestsSemver(t, ">=9.2.7")

	for _, tc := range []struct {
		role        string
		expectError *regexp.Regexp
	}{
		{
			role:        "Viewer",
			expectError: regexp.MustCompile(".*Access denied.*"),
		},
		{
			role:        "Editor",
			expectError: nil,
		},
	} {
		t.Run(tc.role, func(t *testing.T) {
			var folder goapi.Folder
			var name = acctest.RandomWithPrefix(tc.role + "-key")

			// Create an API key with the correct role and inject it in envvars. This auth will be used when the test runs
			client := testutils.Provider.Meta().(*common.Client).GrafanaAPI
			key, err := client.CreateAPIKey(gapi.CreateAPIKeyRequest{
				Name: name,
				Role: tc.role,
			})
			if err != nil {
				t.Fatal(err)
			}
			defer client.DeleteAPIKey(key.ID)
			oldValue := os.Getenv("GRAFANA_AUTH")
			defer os.Setenv("GRAFANA_AUTH", oldValue)
			os.Setenv("GRAFANA_AUTH", key.Key)

			config := fmt.Sprintf(`
		resource "grafana_folder" "bar" {
			title    = "%[1]s"
		}`, name)

			// Do not make parallel, fiddling with auth will break other tests that run in parallel
			resource.Test(t, resource.TestCase{
				ProviderFactories: testutils.ProviderFactories,
				CheckDestroy: resource.ComposeTestCheckFunc(
					testAccFolderCheckDestroy(&folder, 0),
				),
				Steps: []resource.TestStep{
					{
						Config:      config,
						ExpectError: tc.expectError,
						Check: resource.ComposeTestCheckFunc(
							testAccFolderCheckExists("grafana_folder.bar", &folder),
							resource.TestMatchResourceAttr("grafana_folder.bar", "id", defaultOrgIDRegexp),
							resource.TestMatchResourceAttr("grafana_folder.bar", "uid", common.UIDRegexp),
							resource.TestCheckResourceAttr("grafana_folder.bar", "title", name),
						),
					},
				},
			})
		})
	}
}

func testAccFolderIDDidntChange(rn string, oldFolder *goapi.Folder) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		newFolderResource, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("folder not found: %s", rn)
		}
		_, folderUID := grafana.SplitOrgResourceID(newFolderResource.Primary.ID)
		client := testutils.Provider.Meta().(*common.Client).GrafanaOAPI.Folders
		newFolder, err := grafana.GetFolderByIDorUID(client, folderUID)
		if err != nil {
			return fmt.Errorf("error getting folder: %s", err)
		}
		if newFolder.ID != oldFolder.ID {
			return fmt.Errorf("folder id has changed: %d -> %d", oldFolder.ID, newFolder.ID)
		}
		return nil
	}
}

func testAccFolderCheckExists(rn string, folder *goapi.Folder) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s\n %#v", rn, s.RootModule().Resources)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		_, folderID := grafana.SplitOrgResourceID(rs.Primary.ID)
		client := testutils.Provider.Meta().(*common.Client).GrafanaOAPI.Folders
		gotFolder, err := grafana.GetFolderByIDorUID(client, folderID)
		if err != nil {
			return fmt.Errorf("error getting folder: %s", err)
		}

		folder.ID = gotFolder.ID

		return nil
	}
}

func testAccFolderCheckDestroy(folder *goapi.Folder, orgID int64) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testutils.Provider.Meta().(*common.Client).GrafanaOAPI.Folders
		_, err := grafana.GetFolderByIDorUID(client, folder.UID)
		if err == nil {
			return fmt.Errorf("folder still exists")
		}
		return nil
	}
}

func testAccFolderExample_PreventDeletion(name string, preventDeletion bool) string {
	preventDeletionStr := ""
	if preventDeletion {
		preventDeletionStr = "prevent_destroy_if_not_empty = true"
	}

	return fmt.Sprintf(`
		resource "grafana_folder" "test_folder" {
			uid      = "%[1]s"
			title    = "%[1]s"
			%[2]s
		}
	`, name, preventDeletionStr)
}
