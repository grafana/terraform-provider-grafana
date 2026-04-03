package generic_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	terraformresource "github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccGenericResource_folderCloudNamespaceSelection(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	// Force the provider to rely on /bootdata for namespace selection.
	t.Setenv("GRAFANA_ORG_ID", "")
	t.Setenv("GRAFANA_STACK_ID", "")

	suffix := strings.ToLower(acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))
	config := testAccGenericCloudFolderConfig("generic-cloud-folder", "Generic Cloud Folder", "", suffix)

	terraformresource.Test(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckGenericFolderDestroy,
		Steps: []terraformresource.TestStep{
			{
				Config: config,
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttrSet(genericResourceName, "id"),
					terraformresource.TestCheckResourceAttr(genericResourceName, "metadata.uid", "generic-cloud-folder-"+suffix),
					terraformresource.TestCheckResourceAttr(genericResourceName, "manifest.apiVersion", "folder.grafana.app/v1"),
					terraformresource.TestCheckResourceAttr(genericResourceName, "manifest.kind", "Folder"),
				),
			},
			{
				ResourceName:      genericResourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: genericResourceImportIDFunc(genericResourceName),
			},
		},
	})
}

func testAccGenericCloudFolderConfig(uidPrefix, titlePrefix, stackID, suffix string) string {
	stackConfig := ""
	if strings.TrimSpace(stackID) != "" {
		stackConfig = fmt.Sprintf("  stack_id = %s\n", stackID)
	}

	return fmt.Sprintf(`
provider "grafana" {
  # URL and auth come from the acceptance environment.
  # org_id is intentionally omitted here.
%s}

resource "grafana_apps_generic_resource" "test" {
  manifest = {
    apiVersion = "folder.grafana.app/v1"
    kind       = "Folder"
    metadata = {
      name = "%s-%s"
    }
    spec = {
      title = "%s %s"
    }
  }
}
`, stackConfig, uidPrefix, suffix, titlePrefix, suffix)
}
