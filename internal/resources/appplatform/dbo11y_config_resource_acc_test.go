package appplatform_test

import (
	"fmt"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	terraformresource "github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	dbO11yConfigResourceType = "grafana_apps_productactivation_dbo11yconfig_v1alpha1"
	dbO11yConfigResourceName = dbO11yConfigResourceType + ".test"
)

func TestAccDBO11yConfig_basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	terraformresource.Test(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []terraformresource.TestStep{
			{
				Config: testAccDBO11yConfigConfig(true),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(dbO11yConfigResourceName, "metadata.uid", "global"),
					terraformresource.TestCheckResourceAttr(dbO11yConfigResourceName, "spec.enabled", "true"),
					terraformresource.TestCheckResourceAttrSet(dbO11yConfigResourceName, "id"),
				),
			},
			{
				Config: testAccDBO11yConfigConfig(false),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(dbO11yConfigResourceName, "metadata.uid", "global"),
					terraformresource.TestCheckResourceAttr(dbO11yConfigResourceName, "spec.enabled", "false"),
				),
			},
			{
				ResourceName:      dbO11yConfigResourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"metadata.version",
					"options.%",
					"options.overwrite",
				},
				ImportStateIdFunc: importStateIDFunc(dbO11yConfigResourceName),
			},
		},
	})
}

func testAccDBO11yConfigConfig(enabled bool) string {
	return fmt.Sprintf(`
resource "grafana_apps_productactivation_dbo11yconfig_v1alpha1" "test" {
  metadata {
    uid = "global"
  }

  spec {
    enabled = %v
  }
}
`, enabled)
}
