package grafana

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"

	gapi "github.com/emerald-squad/go-grafana-api"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccFolder_basic(t *testing.T) {
	var folder gapi.Folder
	var testOrgID int64 = 1

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccFolderCheckDestroy(&folder, testOrgID),
		Steps: []resource.TestStep{
			{
				Config: testAccFolderConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccFolderCheckExists("grafana_folder.test_folder", &folder),
					resource.TestMatchResourceAttr(
						"grafana_folder.test_folder", "id", regexp.MustCompile(`\d+`),
					),
					resource.TestMatchResourceAttr(
						"grafana_folder.test_folder", "uid", regexp.MustCompile(`\w+`),
					),
				),
			},
		},
	})
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

		client := testAccProvider.Meta().(*gapi.Client)
		id, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
		if err != nil {
			return err
		}

		if id == 0 {
			return fmt.Errorf("got a folder id of 0")
		}

		orgID, err := strconv.ParseInt(rs.Primary.Attributes["org_id"], 10, 64)
		if err != nil {
			return fmt.Errorf("could not find org_id")
		}

		gotFolder, err := client.Folder(id, orgID)
		if err != nil {
			return fmt.Errorf("error getting folder: %s", err)
		}

		*folder = *gotFolder

		return nil
	}
}

func testAccFolderDisappear(folder *gapi.Folder, orgID int64) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// At this point testAccFolderCheckExists should have been called and
		// folder should have been populated
		client := testAccProvider.Meta().(*gapi.Client)
		return client.DeleteFolder((*folder).Uid, orgID)
	}
}

func testAccFolderCheckDestroy(folder *gapi.Folder, orgID int64) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*gapi.Client)
		_, err := client.Folder(folder.Id, orgID)
		if err == nil {
			return fmt.Errorf("folder still exists")
		}
		return nil
	}
}

const testAccFolderConfig_basic = `
resource "grafana_folder" "test_folder" {
	org_id = 1
    title = "Terraform Acceptance Test Folder"
}
`
