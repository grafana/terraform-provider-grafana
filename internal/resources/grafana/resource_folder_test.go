package grafana_test

import (
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/grafana/grafana-openapi-client-go/client/service_accounts"
	"github.com/grafana/grafana-openapi-client-go/models"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/resources/grafana"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
)

func TestAccFolder_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	var folder models.Folder
	var folderWithUID models.Folder

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			folderCheckExists.destroyed(&folder, nil),
			folderCheckExists.destroyed(&folderWithUID, nil),
		),
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "resources/grafana_folder/resource.tf"),
				Check: resource.ComposeTestCheckFunc(
					folderCheckExists.exists("grafana_folder.test_folder", &folder),
					resource.TestMatchResourceAttr("grafana_folder.test_folder", "id", defaultOrgIDRegexp),
					resource.TestCheckResourceAttr("grafana_folder.test_folder", "org_id", "1"),
					resource.TestMatchResourceAttr("grafana_folder.test_folder", "uid", common.UIDRegexp),
					resource.TestCheckResourceAttr("grafana_folder.test_folder", "title", "Terraform Test Folder"),

					folderCheckExists.exists("grafana_folder.test_folder_with_uid", &folderWithUID),
					resource.TestMatchResourceAttr("grafana_folder.test_folder_with_uid", "id", defaultOrgIDRegexp),
					resource.TestCheckResourceAttr("grafana_folder.test_folder_with_uid", "uid", "test-folder-uid"),
					resource.TestCheckResourceAttr("grafana_folder.test_folder_with_uid", "title", "Terraform Test Folder With UID"),
					resource.TestCheckResourceAttr("grafana_folder.test_folder_with_uid", "url", strings.TrimRight(os.Getenv("GRAFANA_URL"), "/")+"/dashboards/f/test-folder-uid/terraform-test-folder-with-uid"),
					testutils.CheckLister("grafana_folder.test_folder_with_uid"),
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
			// Change the title of a folder. This shouldn't recreate the folder
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_folder/resource.tf", map[string]string{
					"Terraform Test Folder": "Terraform Test Folder Updated",
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccFolderWasntRecreated("grafana_folder.test_folder", &folder),
					resource.TestMatchResourceAttr("grafana_folder.test_folder", "id", defaultOrgIDRegexp),
					resource.TestMatchResourceAttr("grafana_folder.test_folder", "uid", common.UIDRegexp),
					resource.TestCheckResourceAttr("grafana_folder.test_folder", "title", "Terraform Test Folder Updated"),
					resource.TestCheckResourceAttr("grafana_folder.test_folder", "url", strings.TrimRight(os.Getenv("GRAFANA_URL"), "/")+"/dashboards/f/test-folder-uid/terraform-test-folder-updated"),
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

func TestAccFolder_nested(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=10.3.0")

	var parentFolder models.Folder
	var childFolder1 models.Folder
	var childFolder2 models.Folder
	name := acctest.RandStringFromCharSet(10, acctest.CharSetAlpha)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			folderCheckExists.destroyed(&parentFolder, nil),
			folderCheckExists.destroyed(&childFolder1, nil),
			folderCheckExists.destroyed(&childFolder2, nil),
		),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource grafana_folder parent {
	title = "Nested Test: Parent %[1]s"
}

resource grafana_folder child1 {
	title = "Nested Test: Child 1 %[1]s"
	uid = "%[1]s-child1"
	parent_folder_uid = grafana_folder.parent.uid
}

resource grafana_folder child2 {
	title = "Nested Test: Child 2 %[1]s"
	parent_folder_uid = grafana_folder.child1.uid
}
`, name),
				Check: resource.ComposeTestCheckFunc(
					folderCheckExists.exists("grafana_folder.parent", &parentFolder),
					resource.TestMatchResourceAttr("grafana_folder.parent", "id", defaultOrgIDRegexp),
					resource.TestCheckResourceAttr("grafana_folder.parent", "title", "Nested Test: Parent "+name),
					resource.TestCheckResourceAttr("grafana_folder.parent", "parent_folder_uid", ""),

					folderCheckExists.exists("grafana_folder.child1", &childFolder1),
					resource.TestMatchResourceAttr("grafana_folder.child1", "id", defaultOrgIDRegexp),
					resource.TestCheckResourceAttr("grafana_folder.child1", "title", "Nested Test: Child 1 "+name),
					resource.TestCheckResourceAttrSet("grafana_folder.child1", "parent_folder_uid"),

					folderCheckExists.exists("grafana_folder.child2", &childFolder2),
					resource.TestMatchResourceAttr("grafana_folder.child2", "id", defaultOrgIDRegexp),
					resource.TestCheckResourceAttr("grafana_folder.child2", "title", "Nested Test: Child 2 "+name),
					resource.TestCheckResourceAttr("grafana_folder.child2", "parent_folder_uid", name+"-child1"),
				),
			},
			{
				ResourceName:            "grafana_folder.parent",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"prevent_destroy_if_not_empty"},
			},
			{
				ResourceName:            "grafana_folder.child1",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"prevent_destroy_if_not_empty"},
			},
			{
				ResourceName:            "grafana_folder.child2",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"prevent_destroy_if_not_empty"},
			},
		},
	})
}

func TestAccFolder_ChangeParent(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=10.3.0")

	var parentFolder models.Folder
	var childFolder1 models.Folder
	name := acctest.RandStringFromCharSet(10, acctest.CharSetAlpha)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			folderCheckExists.destroyed(&parentFolder, nil),
			folderCheckExists.destroyed(&childFolder1, nil),
		),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource grafana_folder parent {
	title = "Nested Test: Parent %[1]s"
}

resource grafana_folder child1 {
	title = "Nested Test: Child 1 %[1]s"
	uid = "%[1]s-child1"
}
`, name),
				Check: resource.ComposeTestCheckFunc(
					folderCheckExists.exists("grafana_folder.parent", &parentFolder),
					resource.TestMatchResourceAttr("grafana_folder.parent", "id", defaultOrgIDRegexp),
					resource.TestCheckResourceAttr("grafana_folder.parent", "title", "Nested Test: Parent "+name),
					resource.TestCheckResourceAttr("grafana_folder.parent", "parent_folder_uid", ""),

					folderCheckExists.exists("grafana_folder.child1", &childFolder1),
					resource.TestMatchResourceAttr("grafana_folder.child1", "id", defaultOrgIDRegexp),
					resource.TestCheckResourceAttr("grafana_folder.child1", "title", "Nested Test: Child 1 "+name),
					resource.TestCheckResourceAttr("grafana_folder.child1", "parent_folder_uid", ""),
				),
			},
			{
				Config: fmt.Sprintf(`
resource grafana_folder parent {
	title = "Nested Test: Parent %[1]s"
}

resource grafana_folder child1 {
	title = "Nested Test: Child 1 %[1]s"
	uid = "%[1]s-child1"
	parent_folder_uid = grafana_folder.parent.uid
}
`, name),
				Check: resource.ComposeTestCheckFunc(
					folderCheckExists.exists("grafana_folder.parent", &parentFolder),
					resource.TestMatchResourceAttr("grafana_folder.parent", "id", defaultOrgIDRegexp),
					resource.TestCheckResourceAttr("grafana_folder.parent", "title", "Nested Test: Parent "+name),
					resource.TestCheckResourceAttr("grafana_folder.parent", "parent_folder_uid", ""),

					folderCheckExists.exists("grafana_folder.child1", &childFolder1),
					resource.TestMatchResourceAttr("grafana_folder.child1", "id", defaultOrgIDRegexp),
					resource.TestCheckResourceAttr("grafana_folder.child1", "title", "Nested Test: Child 1 "+name),
					resource.TestCheckResourceAttrSet("grafana_folder.child1", "parent_folder_uid"),
				),
			},
			{
				Config: fmt.Sprintf(`
resource grafana_folder parent {
	title = "Nested Test: Parent %[1]s"
}

resource grafana_folder child1 {
	title = "Nested Test: Child 1 %[1]s"
	uid = "%[1]s-child1"
}
`, name),
				Check: resource.ComposeTestCheckFunc(
					folderCheckExists.exists("grafana_folder.parent", &parentFolder),
					resource.TestMatchResourceAttr("grafana_folder.parent", "id", defaultOrgIDRegexp),
					resource.TestCheckResourceAttr("grafana_folder.parent", "title", "Nested Test: Parent "+name),
					resource.TestCheckResourceAttr("grafana_folder.parent", "parent_folder_uid", ""),

					folderCheckExists.exists("grafana_folder.child1", &childFolder1),
					resource.TestMatchResourceAttr("grafana_folder.child1", "id", defaultOrgIDRegexp),
					resource.TestCheckResourceAttr("grafana_folder.child1", "title", "Nested Test: Child 1 "+name),
					resource.TestCheckResourceAttr("grafana_folder.child1", "parent_folder_uid", ""),
				),
			},
			{
				ResourceName:            "grafana_folder.parent",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"prevent_destroy_if_not_empty"},
			},
			{
				ResourceName:            "grafana_folder.child1",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"prevent_destroy_if_not_empty"},
			},
		},
	})
}

func TestAccFolder_PreventDeletion(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=10.2.0") // Searching by folder UID was added in 10.2.0

	name := acctest.RandStringFromCharSet(10, acctest.CharSetAlpha)
	var folder models.Folder

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFolderExample_PreventDeletion(name, true), // Create protected folder
			},
			{
				Config:  testAccFolderExample_PreventDeletion(name, true), // Create protected folder
				Destroy: true,
			},
			// test with dashboard added to folder from outside Terraform:
			{
				Config: testAccFolderExample_PreventDeletion(name, true), // Create protected folder again
				Check: resource.ComposeTestCheckFunc(
					folderCheckExists.exists("grafana_folder.test_folder", &folder),
					// Create a dashboard in the protected folder
					func(s *terraform.State) error {
						client := grafanaTestClient()
						_, err := client.Dashboards.PostDashboard(&models.SaveDashboardCommand{
							FolderUID: folder.UID,
							Dashboard: map[string]any{
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

func TestAccFolder_PreventDeletionNested(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=10.2.0") // Searching by folder UID was added in 10.2.0

	name := acctest.RandStringFromCharSet(10, acctest.CharSetAlpha)
	var folder models.Folder

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			// test with a nested folder inside the folder:
			{
				Config: testAccFolderExample_PreventDeletion(name, true), // Create protected folder again
				Check: resource.ComposeTestCheckFunc(
					folderCheckExists.exists("grafana_folder.test_folder", &folder),
					// Create a dashboard in the protected folder
					func(s *terraform.State) error {
						client := grafanaTestClient()
						_, err := client.Folders.CreateFolder(&models.CreateFolderCommand{
							Title:     "Inner folder",
							ParentUID: name,
							UID:       acctest.RandStringFromCharSet(10, acctest.CharSetAlpha),
						})
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

func TestAccFolder_RapidCreation(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	folderCount := 100

	var checks []resource.TestCheckFunc
	for i := range folderCount {
		name := fmt.Sprintf("grafana_folder.rapid.%d", i)
		checks = append(checks, resource.TestCheckResourceAttr(name, "title", fmt.Sprintf("Rapid Test Folder %d", i)))
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "grafana_folder" "rapid" {
						count = %[1]d
						uid   = "rapid_test_${count.index}"
						title = "Rapid Test Folder ${count.index}"
					}
				`, folderCount),
				Check: resource.ComposeTestCheckFunc(checks...),
			},
		},
	})
}

// This is a bug in Grafana, not the provider. It was fixed in 9.2.7+ and 9.3.0+, this test will check for regressions
func TestAccFolder_createFromDifferentRoles(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.2.7, <11.6.0 || >=12.0.0") // TODO: folder permission regression in Grafana 11.6.x

	for _, tc := range []struct {
		role        string
		expectError *regexp.Regexp
	}{
		{
			role:        "Viewer",
			expectError: regexp.MustCompile(fmt.Sprint(http.StatusForbidden)),
		},
		{
			role:        "Editor",
			expectError: nil,
		},
	} {
		t.Run(tc.role, func(t *testing.T) {
			var folder models.Folder
			var saName = acctest.RandomWithPrefix(tc.role + "-sa")
			var saTokenName = acctest.RandomWithPrefix(tc.role + "-token")

			// Create a service account token with the correct role and inject it in envvars. This auth will be used when the test runs
			client := grafanaTestClient()

			sa, err := client.ServiceAccounts.CreateServiceAccount(
				service_accounts.NewCreateServiceAccountParams().WithBody(&models.CreateServiceAccountForm{
					Name: saName,
					Role: tc.role,
				}),
			)
			if err != nil {
				t.Fatal(err)
			}
			defer client.ServiceAccounts.DeleteServiceAccount(sa.Payload.ID)

			saToken, err := client.ServiceAccounts.CreateToken(
				service_accounts.NewCreateTokenParams().WithBody(&models.AddServiceAccountTokenCommand{
					Name: saTokenName,
				}).WithServiceAccountID(sa.Payload.ID),
			)
			if err != nil {
				t.Fatal(err)
			}

			oldValue := os.Getenv("GRAFANA_AUTH")
			defer os.Setenv("GRAFANA_AUTH", oldValue)
			os.Setenv("GRAFANA_AUTH", saToken.Payload.Key)

			config := fmt.Sprintf(`
		resource "grafana_folder" "bar" {
			title    = "%[1]s"
		}`, saName)

			// Do not make parallel, fiddling with auth will break other tests that run in parallel
			resource.Test(t, resource.TestCase{
				ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
				CheckDestroy:             folderCheckExists.destroyed(&folder, nil),
				Steps: []resource.TestStep{
					{
						Config:      config,
						ExpectError: tc.expectError,
						Check: resource.ComposeTestCheckFunc(
							folderCheckExists.exists("grafana_folder.bar", &folder),
							resource.TestMatchResourceAttr("grafana_folder.bar", "id", defaultOrgIDRegexp),
							resource.TestMatchResourceAttr("grafana_folder.bar", "uid", common.UIDRegexp),
							resource.TestCheckResourceAttr("grafana_folder.bar", "title", saName),
						),
					},
				},
			})
		})
	}
}

func testAccFolderWasntRecreated(rn string, oldFolder *models.Folder) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		newFolderResource, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("folder not found: %s", rn)
		}
		orgID, folderUID := grafana.SplitOrgResourceID(newFolderResource.Primary.ID)
		client := testutils.Provider.Meta().(*common.Client).GrafanaAPI.WithOrgID(orgID)
		newFolder, err := grafana.GetFolderByIDorUID(client.Folders, folderUID)
		if err != nil {
			return fmt.Errorf("error getting folder: %s", err)
		}
		if newFolder.Created != oldFolder.Created {
			return fmt.Errorf("folder creation date has changed: %s -> %s", oldFolder.Created, newFolder.Created)
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
