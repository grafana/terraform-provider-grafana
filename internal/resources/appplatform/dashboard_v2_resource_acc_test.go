package appplatform_test

import (
	"fmt"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	terraformresource "github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
)

const dashboardV2ResourceName = "grafana_apps_dashboard_dashboard_v2beta1.test"

func TestAccDashboardV2_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=12.1.0")

	randSuffix := acctest.RandString(6)

	terraformresource.ParallelTest(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []terraformresource.TestStep{
			{
				Config: testAccDashboardV2Basic(randSuffix),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttrSet(dashboardV2ResourceName, "id"),
					terraformresource.TestCheckResourceAttr(dashboardV2ResourceName, "spec.title", "Test Dashboard V2"),
				),
			},
			{
				ResourceName:      dashboardV2ResourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"options.%",
					"options.overwrite",
					"spec.json",
				},
				ImportStateIdFunc: importStateIDFunc(dashboardV2ResourceName),
			},
		},
	})
}

func testAccDashboardV2Basic(randSuffix string) string {
	return fmt.Sprintf(`
resource "grafana_apps_dashboard_dashboard_v2beta1" "test" {
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
