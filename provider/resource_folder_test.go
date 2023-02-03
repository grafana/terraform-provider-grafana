package provider

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/provider/common"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccFolder_basic(t *testing.T) {
	CheckOSSTestsEnabled(t)

	var folder gapi.Folder
	var folderWithUID gapi.Folder

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccFolderCheckDestroy(&folder),
			testAccFolderCheckDestroy(&folderWithUID),
		),
		Steps: []resource.TestStep{
			{
				Config: testAccExample(t, "resources/grafana_folder/resource.tf"),
				Check: resource.ComposeTestCheckFunc(
					testAccFolderCheckExists("grafana_folder.test_folder", &folder),
					resource.TestMatchResourceAttr("grafana_folder.test_folder", "id", idRegexp),
					resource.TestMatchResourceAttr("grafana_folder.test_folder", "uid", uidRegexp),
					resource.TestCheckResourceAttr("grafana_folder.test_folder", "title", "Terraform Test Folder"),

					testAccFolderCheckExists("grafana_folder.test_folder_with_uid", &folderWithUID),
					resource.TestMatchResourceAttr("grafana_folder.test_folder_with_uid", "id", idRegexp),
					resource.TestCheckResourceAttr("grafana_folder.test_folder_with_uid", "uid", "test-folder-uid"),
					resource.TestCheckResourceAttr("grafana_folder.test_folder_with_uid", "title", "Terraform Test Folder With UID"),
					resource.TestCheckResourceAttr("grafana_folder.test_folder_with_uid", "url", strings.TrimRight(os.Getenv("GRAFANA_URL"), "/")+"/dashboards/f/test-folder-uid/terraform-test-folder-with-uid"),
				),
			},
			{
				ResourceName:      "grafana_folder.test_folder",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      "grafana_folder.test_folder_with_uid",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Change the title of one folder, change the UID of the other. They shouldn't change IDs (the folder doesn't have to be recreated)
			{
				Config: testAccExampleWithReplace(t, "resources/grafana_folder/resource.tf", map[string]string{
					"Terraform Test Folder": "Terraform Test Folder Updated",
					"test-folder-uid":       "test-folder-uid-other",
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccFolderIDDidntChange("grafana_folder.test_folder", &folder),
					resource.TestMatchResourceAttr("grafana_folder.test_folder", "id", idRegexp),
					resource.TestMatchResourceAttr("grafana_folder.test_folder", "uid", uidRegexp),
					resource.TestCheckResourceAttr("grafana_folder.test_folder", "title", "Terraform Test Folder Updated"),

					testAccFolderIDDidntChange("grafana_folder.test_folder_with_uid", &folderWithUID),
					resource.TestMatchResourceAttr("grafana_folder.test_folder_with_uid", "id", idRegexp),
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

// This is a bug in Grafana, not the provider. It was fixed in 9.2.7+ and 9.3.0+, this test will check for regressions
func TestAccFolder_createFromDifferentRoles(t *testing.T) {
	CheckOSSTestsEnabled(t)
	CheckOSSTestsSemver(t, ">=9.2.7")

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
			var folder gapi.Folder
			var name = acctest.RandomWithPrefix(tc.role + "-key")

			// Create an API key with the correct role and inject it in envvars. This auth will be used when the test runs
			client := testAccProvider.Meta().(*common.Client).GrafanaAPI
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
				ProviderFactories: testAccProviderFactories,
				CheckDestroy: resource.ComposeTestCheckFunc(
					testAccFolderCheckDestroy(&folder),
				),
				Steps: []resource.TestStep{
					{
						Config:      config,
						ExpectError: tc.expectError,
						Check: resource.ComposeTestCheckFunc(
							testAccFolderCheckExists("grafana_folder.bar", &folder),
							resource.TestMatchResourceAttr("grafana_folder.bar", "id", idRegexp),
							resource.TestMatchResourceAttr("grafana_folder.bar", "uid", uidRegexp),
							resource.TestCheckResourceAttr("grafana_folder.bar", "title", name),
						),
					},
				},
			})
		})
	}
}

func testAccFolderIDDidntChange(rn string, folder *gapi.Folder) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		oldID := strconv.FormatInt(folder.ID, 10)
		newFolder, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("folder not found: %s", rn)
		}
		if newFolder.Primary.ID != oldID {
			return fmt.Errorf("folder id has changed: %s -> %s", oldID, newFolder.Primary.ID)
		}
		return nil
	}
}

func testAccFolderCheckExists(rn string, folder *gapi.Folder) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s\n %#v", rn, s.RootModule().Resources)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		client := testAccProvider.Meta().(*common.Client).GrafanaAPI
		id, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
		if err != nil {
			return err
		}

		if id == 0 {
			return fmt.Errorf("got a folder id of 0")
		}
		gotFolder, err := getFolderByID(client, id)
		if err != nil {
			return fmt.Errorf("error getting folder: %s", err)
		}

		*folder = *gotFolder

		return nil
	}
}

func testAccFolderCheckDestroy(folder *gapi.Folder) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*common.Client).GrafanaAPI
		_, err := getFolderByID(client, folder.ID)
		if err == nil {
			return fmt.Errorf("folder still exists")
		}
		return nil
	}
}
