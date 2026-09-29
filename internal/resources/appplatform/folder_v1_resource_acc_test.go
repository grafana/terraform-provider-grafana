package appplatform_test

import (
	"fmt"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	terraformresource "github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const folderV1ResourceName = "grafana_apps_folder_folder_v1.test"

func TestAccFolderV1_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=13.0.0")

	randSuffix := acctest.RandString(6)

	terraformresource.ParallelTest(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []terraformresource.TestStep{
			{
				Config: testAccFolderV1Basic(randSuffix),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttrSet(folderV1ResourceName, "id"),
					terraformresource.TestCheckResourceAttr(folderV1ResourceName, "spec.title", "Test Folder"),
					terraformresource.TestCheckResourceAttr(folderV1ResourceName, "spec.description", "Managed by Terraform"),
					// A folder with no team owner must report none, so that a configuration
					// declaring no owner_references blocks matches.
					terraformresource.TestCheckResourceAttr(folderV1ResourceName, "metadata.owner_references.#", "0"),
				),
			},
			{
				// Re-applying an unchanged configuration must be a no-op. This is what proves
				// the shared metadata block did not start churning.
				Config:   testAccFolderV1Basic(randSuffix),
				PlanOnly: true,
			},
			{
				ResourceName:      folderV1ResourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"options.%",
					"options.overwrite",
					"options.allow_ui_updates",
				},
				ImportStateIdFunc: importStateIDFunc(folderV1ResourceName),
			},
		},
	})
}

func TestAccFolderV1_nested(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=13.0.0")

	randSuffix := acctest.RandString(6)

	terraformresource.ParallelTest(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []terraformresource.TestStep{
			{
				Config: testAccFolderV1Nested(randSuffix),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(
						"grafana_apps_folder_folder_v1.child", "metadata.folder_uid",
						fmt.Sprintf("test-folder-parent-%s", randSuffix),
					),
				),
			},
			{
				Config:   testAccFolderV1Nested(randSuffix),
				PlanOnly: true,
			},
		},
	})
}

// TestAccFolderV1_teamOwner covers the team folders feature: attaching, keeping and removing an
// owner reference pointing at a team.
func TestAccFolderV1_teamOwner(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=13.0.0")

	randSuffix := acctest.RandString(6)

	terraformresource.ParallelTest(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []terraformresource.TestStep{
			{
				Config: testAccFolderV1TeamOwner(randSuffix),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(folderV1ResourceName, "metadata.owner_references.#", "1"),
					terraformresource.TestCheckResourceAttr(folderV1ResourceName, "metadata.owner_references.0.api_version", "iam.grafana.app/v0alpha1"),
					terraformresource.TestCheckResourceAttr(folderV1ResourceName, "metadata.owner_references.0.kind", "Team"),
					terraformresource.TestCheckResourceAttrPair(
						folderV1ResourceName, "metadata.owner_references.0.name",
						"grafana_team.test", "team_uid",
					),
					// The Kubernetes uid is derived from the name rather than exposed, so there
					// is no uid attribute to assert on.
					terraformresource.TestCheckNoResourceAttr(
						folderV1ResourceName, "metadata.owner_references.0.uid",
					),
				),
			},
			{
				// Ownership must be stable across applies, not re-written every time.
				Config:   testAccFolderV1TeamOwner(randSuffix),
				PlanOnly: true,
			},
			{
				ResourceName:      folderV1ResourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"options.%",
					"options.overwrite",
					"options.allow_ui_updates",
				},
				ImportStateIdFunc: importStateIDFunc(folderV1ResourceName),
			},
			{
				// Removing the block must clear the owner reference server-side. Update builds a
				// fresh object from the model, so this exercises that the empty list reaches the
				// API rather than leaving the previous owner in place.
				Config: testAccFolderV1TeamOwnerRemoved(randSuffix),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(folderV1ResourceName, "metadata.owner_references.#", "0"),
				),
			},
			{
				Config:   testAccFolderV1TeamOwnerRemoved(randSuffix),
				PlanOnly: true,
			},
		},
	})
}

func testAccFolderV1Basic(randSuffix string) string {
	return fmt.Sprintf(`
resource "grafana_apps_folder_folder_v1" "test" {
  metadata {
    uid = "test-folder-%[1]s"
  }

  spec {
    title       = "Test Folder"
    description = "Managed by Terraform"
  }
}
`, randSuffix)
}

func testAccFolderV1Nested(randSuffix string) string {
	return fmt.Sprintf(`
resource "grafana_apps_folder_folder_v1" "parent" {
  metadata {
    uid = "test-folder-parent-%[1]s"
  }

  spec {
    title = "Test Parent Folder"
  }
}

resource "grafana_apps_folder_folder_v1" "child" {
  metadata {
    uid        = "test-folder-child-%[1]s"
    folder_uid = grafana_apps_folder_folder_v1.parent.metadata.uid
  }

  spec {
    title = "Test Child Folder"
  }
}
`, randSuffix)
}

func testAccFolderV1TeamOwner(randSuffix string) string {
	return fmt.Sprintf(`
resource "grafana_team" "test" {
  name = "test-team-%[1]s"
}

resource "grafana_apps_folder_folder_v1" "test" {
  metadata {
    uid = "test-team-folder-%[1]s"

    owner_references {
      api_version = "iam.grafana.app/v0alpha1"
      kind        = "Team"
      name        = grafana_team.test.team_uid
    }
  }

  spec {
    title = "Test Team Folder"
  }
}
`, randSuffix)
}

func testAccFolderV1TeamOwnerRemoved(randSuffix string) string {
	return fmt.Sprintf(`
resource "grafana_team" "test" {
  name = "test-team-%[1]s"
}

resource "grafana_apps_folder_folder_v1" "test" {
  metadata {
    uid = "test-team-folder-%[1]s"
  }

  spec {
    title = "Test Team Folder"
  }
}
`, randSuffix)
}
