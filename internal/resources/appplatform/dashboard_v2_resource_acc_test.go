package appplatform_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/grafana/authlib/claims"
	sdkresource "github.com/grafana/grafana-app-sdk/resource"
	"github.com/grafana/grafana/apps/dashboard/pkg/apis/dashboard/v2beta1"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	terraformresource "github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const dashboardV2ResourceName = "grafana_apps_dashboard_dashboard_v2beta1.test"

func TestAccDashboardV2_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=12.2.0")

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
					"options.allow_ui_updates",
					"spec.json",
				},
				ImportStateIdFunc: importStateIDFunc(dashboardV2ResourceName),
			},
		},
	})
}

func TestAccDashboardV2_update(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=12.2.0")

	randSuffix := acctest.RandString(6)
	uid := fmt.Sprintf("test-v2-dashboard-%s", randSuffix)

	terraformresource.ParallelTest(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []terraformresource.TestStep{
			// Create the dashboard.
			{
				Config: testAccDashboardV2WithTitle(uid, "Initial Title"),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttrSet(dashboardV2ResourceName, "id"),
					terraformresource.TestCheckResourceAttr(dashboardV2ResourceName, "spec.title", "Initial Title"),
				),
			},
			// Update the dashboard title.
			{
				Config: testAccDashboardV2WithTitle(uid, "Updated Title"),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttrSet(dashboardV2ResourceName, "id"),
					terraformresource.TestCheckResourceAttr(dashboardV2ResourceName, "spec.title", "Updated Title"),
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

func TestAccDashboardV2_overwrite(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=12.2.0")

	randSuffix := acctest.RandString(6)
	uid := fmt.Sprintf("test-v2-dashboard-%s", randSuffix)

	terraformresource.ParallelTest(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []terraformresource.TestStep{
			// Pre-create the dashboard directly via the API to simulate an externally-existing
			// resource (e.g. created manually). Then apply with overwrite=true to verify the
			// Create→Update fallback introduced to handle the 409 AlreadyExists case.
			{
				PreConfig: func() {
					client := testutils.Provider.Meta().(*common.Client)

					rcli, err := client.GrafanaAppPlatformAPI.ClientFor(v2beta1.DashboardKind())
					if err != nil {
						t.Fatalf("failed to create app platform client: %v", err)
					}

					ns := claims.OrgNamespaceFormatter(client.GrafanaOrgID)
					namespacedClient := sdkresource.NewNamespaced(
						sdkresource.NewTypedClient[*v2beta1.Dashboard, *v2beta1.DashboardList](rcli, v2beta1.DashboardKind()),
						ns,
					)

					dashboard := v2beta1.DashboardKind().Schema.ZeroValue().(*v2beta1.Dashboard)
					dashboard.SetName(uid)
					if err := dashboard.SetSpec(v2beta1.DashboardSpec{Title: "Pre-existing Title"}); err != nil {
						t.Fatalf("failed to set dashboard spec: %v", err)
					}

					if _, err := namespacedClient.Create(context.Background(), dashboard, sdkresource.CreateOptions{}); err != nil {
						t.Fatalf("failed to pre-create dashboard: %v", err)
					}
				},
				Config: testAccDashboardV2WithTitle(uid, "Overwritten Title"),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttrSet(dashboardV2ResourceName, "id"),
					terraformresource.TestCheckResourceAttr(dashboardV2ResourceName, "spec.title", "Overwritten Title"),
					terraformresource.TestCheckResourceAttr(dashboardV2ResourceName, "options.overwrite", "true"),
				),
			},
		},
	})
}

func testAccDashboardV2WithTitle(uid, title string) string {
	return fmt.Sprintf(`
resource "grafana_apps_dashboard_dashboard_v2beta1" "test" {
  metadata {
    uid = %q
  }

  spec {
    title = %q
    json = jsonencode({
      title       = %q
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
`, uid, title, title)
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
