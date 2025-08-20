package grafana_test

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/grafana/terraform-provider-grafana/v4/internal/resources/grafana"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
)

func TestAccDatasourceFolder_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	var folder models.Folder
	checks := []resource.TestCheckFunc{
		folderCheckExists.exists("grafana_folder.test", &folder),
		resource.TestCheckResourceAttr("data.grafana_folder.from_title", "title", "test-folder"),
		resource.TestMatchResourceAttr("data.grafana_folder.from_title", "id", defaultOrgIDRegexp),
		resource.TestCheckResourceAttr("data.grafana_folder.from_title", "uid", "test-ds-folder-uid"),
		resource.TestCheckResourceAttr("data.grafana_folder.from_title", "url", strings.TrimRight(os.Getenv("GRAFANA_URL"), "/")+"/dashboards/f/test-ds-folder-uid/test-folder"),
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             folderCheckExists.destroyed(&folder, nil),
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "data-sources/grafana_folder/data-source.tf"),
				Check:  resource.ComposeTestCheckFunc(checks...),
			},
		},
	})
}

func TestAccDatasourceFolder_nested(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=10.3.0")

	var parent models.Folder
	var child models.Folder
	randomName := acctest.RandStringFromCharSet(6, acctest.CharSetAlpha)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			folderCheckExists.destroyed(&parent, nil),
			folderCheckExists.destroyed(&child, nil),
		),
		Steps: []resource.TestStep{
			{
				Config: testNestedFolderData(randomName),
				Check: resource.ComposeTestCheckFunc(
					folderCheckExists.exists("grafana_folder.parent", &parent),
					folderCheckExists.exists("grafana_folder.child", &child),
					resource.TestCheckResourceAttr("data.grafana_folder.parent", "title", randomName),
					resource.TestCheckResourceAttr("data.grafana_folder.parent", "uid", randomName),
					resource.TestMatchResourceAttr("data.grafana_folder.parent", "id", defaultOrgIDRegexp),
					resource.TestCheckResourceAttr("data.grafana_folder.parent", "parent_folder_uid", ""),

					resource.TestCheckResourceAttr("data.grafana_folder.child", "title", randomName+"-child"),
					resource.TestCheckResourceAttr("data.grafana_folder.child", "uid", randomName+"-child"),
					resource.TestMatchResourceAttr("data.grafana_folder.child", "id", defaultOrgIDRegexp),
					resource.TestCheckResourceAttr("data.grafana_folder.child", "parent_folder_uid", randomName),
				),
			},
		},
	})
}

func testNestedFolderData(name string) string {
	return fmt.Sprintf(`
resource "grafana_folder" "parent" {
	title = "%[1]s"
	uid   = "%[1]s"
}
	
resource "grafana_folder" "child" {
	title = "%[1]s-child"
	uid   = "%[1]s-child"
	parent_folder_uid = grafana_folder.parent.uid
}
	
data "grafana_folder" "parent" {
	title = grafana_folder.parent.title
}
	
data "grafana_folder" "child" {
	title = grafana_folder.child.title
}
`, name)
}

func TestAccDatasourceFolderByTitleAndUid(t *testing.T) {
	// This test uses duplicate folder names, a feature that was introduced in Grafana 11.4: https://github.com/grafana/grafana/pull/90687
	testutils.CheckOSSTestsEnabled(t, ">=11.4.0")

	var folder1 models.Folder
	var folder2 models.Folder
	var folder3 models.Folder
	randomName := acctest.RandStringFromCharSet(6, acctest.CharSetAlpha)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			folderCheckExists.destroyed(&folder1, nil),
			folderCheckExists.destroyed(&folder2, nil),
			folderCheckExists.destroyed(&folder3, nil),
		),
		Steps: []resource.TestStep{
			{
				Config: folderResources(randomName),
				Check: resource.ComposeTestCheckFunc(
					folderCheckExists.exists("grafana_folder.folder1", &folder1),
					folderCheckExists.exists("grafana_folder.folder2", &folder2),
					folderCheckExists.exists("grafana_folder.folder3", &folder3),
				),
			},
			{
				Config: folderResources(randomName) + folderData(randomName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.grafana_folder.f1", "title", randomName),
					// f1 must be one of our folders, but we cannot guarantee which one.
					resource.TestMatchResourceAttr("data.grafana_folder.f1", "uid", regexp.MustCompile(regexp.QuoteMeta(randomName)+"[123]")),

					resource.TestCheckResourceAttr("data.grafana_folder.f2", "title", randomName),
					resource.TestCheckResourceAttr("data.grafana_folder.f2", "uid", randomName+"2"),

					resource.TestCheckResourceAttr("data.grafana_folder.f3", "title", randomName),
					resource.TestCheckResourceAttr("data.grafana_folder.f3", "uid", randomName+"3"),
				),
			},
			{
				Config: folderResources(randomName) + fmt.Sprintf(`
					data "grafana_folder" "unknown" {
						title = "unknown %[1]s"
					}
				`, randomName),
				ExpectError: regexp.MustCompile(regexp.QuoteMeta(fmt.Sprintf(grafana.FolderWithTitleNotFound, "unknown "+randomName))),
			},
			{
				Config: folderResources(randomName) + fmt.Sprintf(`
					data "grafana_folder" "unknown" {
						title = "unknown %[1]s"
					}
				`, randomName),
				ExpectError: regexp.MustCompile(regexp.QuoteMeta(fmt.Sprintf(grafana.FolderWithTitleNotFound, "unknown "+randomName))),
			},
			{
				Config: folderResources(randomName) + fmt.Sprintf(`
					data "grafana_folder" "unknown" {
						title = "unknown %[1]s"
					}
				`, randomName),
				ExpectError: regexp.MustCompile(regexp.QuoteMeta(fmt.Sprintf(grafana.FolderWithTitleNotFound, "unknown "+randomName))),
			},
			{
				Config: folderResources(randomName) + fmt.Sprintf(`
					data "grafana_folder" "unknown" {
						title = "unknown %[1]s"
					}
				`, randomName),
				ExpectError: regexp.MustCompile(regexp.QuoteMeta(fmt.Sprintf(grafana.FolderWithTitleNotFound, "unknown "+randomName))),
			},
			// Don't find the folder if neither title or uid is provided.
			{
				Config: folderResources(randomName) + `
					data "grafana_folder" "unknown" {
					}
				`,
				ExpectError: regexp.MustCompile(regexp.QuoteMeta(grafana.FolderTitleOrUIDMissing)),
			},
			// Don't find the folder if title is wrong.
			{
				Config: folderResources(randomName) + fmt.Sprintf(`
					data "grafana_folder" "unknown" {
						title = "unknown %[1]s"
					}
				`, randomName),
				ExpectError: regexp.MustCompile(regexp.QuoteMeta(fmt.Sprintf(grafana.FolderWithTitleNotFound, "unknown "+randomName))),
			},
			// Don't find the folder if uid is wrong.
			{
				Config: folderResources(randomName) + fmt.Sprintf(`
					data "grafana_folder" "unknown" {
						uid = "%[1]s9"
					}
				`, randomName),
				ExpectError: regexp.MustCompile(regexp.QuoteMeta(fmt.Sprintf(grafana.FolderWithUIDNotFound, randomName+"9"))),
			},
			// Don't find the folder if the title is wrong, even if uid matches.
			{
				Config: folderResources(randomName) + fmt.Sprintf(`
					data "grafana_folder" "unknown" {
						title = "unknown %[1]s"
						uid = "%[1]s1"
					}
				`, randomName),
				ExpectError: regexp.MustCompile(regexp.QuoteMeta(fmt.Sprintf(grafana.FolderWithTitleAndUIDNotFound, "unknown "+randomName, randomName+"1"))),
			},
			// Don't find the folder if uid is wrong, even if the title matches.
			{
				Config: folderResources(randomName) + fmt.Sprintf(`
					data "grafana_folder" "unknown" {
						title = "%[1]s"
						uid = "%[1]s9"
					}
				`, randomName),
				ExpectError: regexp.MustCompile(regexp.QuoteMeta(fmt.Sprintf(grafana.FolderWithTitleAndUIDNotFound, randomName, randomName+"9"))),
			},
		},
	})
}

// Creates three folders with the same title, but different UIDs
func folderResources(name string) string {
	return fmt.Sprintf(`
resource "grafana_folder" "folder1" {
	title = "%[1]s"
	uid   = "%[1]s1"
}

resource "grafana_folder" "folder2" {
	title = "%[1]s"
	uid   = "%[1]s2"
}

resource "grafana_folder" "folder3" {
	title = "%[1]s"
	uid   = "%[1]s3"
}
`, name)
}

// Creates data sources that find folders by title and/or uid.
func folderData(name string) string {
	return fmt.Sprintf(`
# Find folder by title only -- random folder
data "grafana_folder" "f1" {
	title = "%[1]s"
}
 
# Find folder by uid only -- matches second folder
data "grafana_folder" "f2" {
	uid = "%[1]s2"
}
	
# Find folder by title and uid -- matches third folder
data "grafana_folder" "f3" {
	title = "%[1]s"
	uid = "%[1]s3"
}
`, name)
}

func TestAccDatasourceFolderUidSetCorrectly(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	var folder models.Folder
	randomName := acctest.RandStringFromCharSet(6, acctest.CharSetAlpha)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			folderCheckExists.destroyed(&folder, nil),
		),
		Steps: []resource.TestStep{
			{
				// Find folder by title. We should get correct uid back.
				Config: fmt.Sprintf(`
					resource "grafana_folder" "folder" {
						title = "%[1]s"
					}
					data "grafana_folder" "test" {
						depends_on = ["grafana_folder.folder"]
						title = "%[1]s"		
					}
					resource "grafana_folder" "nested_folder" {
						title = "%[1]s Nested"
						parent_folder_uid = data.grafana_folder.test.uid
					}
				`, randomName),
				Check: resource.ComposeTestCheckFunc(
					folderCheckExists.exists("grafana_folder.folder", &folder),
					resource.TestCheckResourceAttr("grafana_folder.nested_folder", "title", randomName+" Nested"),
					// Next line doesn't work, because resource.TestCheckResourceAttr is called too early -- before folder.UID is set.
					//	   resource.TestCheckResourceAttr("grafana_folder.nested_folder", "parent_folder_uid", folder.UID)
					//
					// This works:
					resource.TestCheckResourceAttrWith("grafana_folder.nested_folder", "parent_folder_uid", func(value string) error {
						if folder.UID == "" {
							return fmt.Errorf("grafana_folder.folder.uid should not be empty")
						}
						if value == folder.UID {
							return nil
						}
						return fmt.Errorf("expected uid to be %q but got %q", folder.UID, value)
					}),
				),
			},
		},
	})
}
