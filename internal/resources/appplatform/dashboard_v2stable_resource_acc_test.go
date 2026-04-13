package appplatform_test

import (
	"fmt"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	terraformresource "github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const dashboardV2StableResourceName = "grafana_apps_dashboard_dashboard_v2.test"

func TestAccDashboardV2Stable_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=13.0.0")

	randSuffix := acctest.RandString(6)

	terraformresource.ParallelTest(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []terraformresource.TestStep{
			{
				Config: testAccDashboardV2StableBasic(randSuffix),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttrSet(dashboardV2StableResourceName, "id"),
					terraformresource.TestCheckResourceAttr(dashboardV2StableResourceName, "spec.title", "Test Dashboard V2"),
				),
			},
			{
				ResourceName:      dashboardV2StableResourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"options.%",
					"options.overwrite",
					"options.allow_ui_updates",
					"spec.json",
				},
				ImportStateIdFunc: importStateIDFunc(dashboardV2StableResourceName),
			},
		},
	})
}

func testAccDashboardV2StableBasic(randSuffix string) string {
	return fmt.Sprintf(`
resource "grafana_apps_dashboard_dashboard_v2" "test" {
  metadata {
    uid = "test-v2-dashboard-%s"
  }

  spec {
    title = "Test Dashboard V2"
    json = jsonencode({
      title       = "Test Dashboard V2"
      cursorSync  = "Off"
      elements    = {}
      layout      = { kind = "GridLayout", spec = { items = [] } }
      links       = []
      preload     = false
      annotations = []
      variables   = []
      timeSettings = {
        timezone = "browser"
        from     = "now-6h"
        to       = "now"
      }
    })
  }

  options {
    overwrite = true
  }
}
`, randSuffix)
}
