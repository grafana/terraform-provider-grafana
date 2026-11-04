package appplatform_test

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"testing"

	"github.com/grafana/authlib/claims"
	sdkresource "github.com/grafana/grafana-app-sdk/resource"
	v2 "github.com/grafana/grafana/apps/dashboard/pkg/apis/dashboard/v2"
	"github.com/grafana/grafana/pkg/apimachinery/utils"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	terraformresource "github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const dashboardOverwriteSpecJSON = `{
	"title": %q,
	"cursorSync": "Off",
	"elements": {},
	"layout": {"kind": "GridLayout", "spec": {"items": []}},
	"links": [],
	"preload": false,
	"annotations": [],
	"variables": [],
	"timeSettings": {"timezone": "browser", "from": "now-6h", "to": "now"}
}`

// A dashboard created outside Terraform (e.g. in the Grafana UI) has no manager stamp
// at all. Applying a grafana_apps_dashboard_dashboard_v2 config that targets the same
// uid with options.overwrite = true should adopt it instead of failing with an
// "already exists" error.
func TestAccDashboardV2StableOverwrite_adoptsUnmanaged(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=13.0.0")

	randSuffix := acctest.RandString(6)
	uid := "test-v2-overwrite-unmanaged-" + randSuffix

	createDashboardV2Fixture(t, uid, "Pre-existing unmanaged dashboard", "")

	terraformresource.ParallelTest(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []terraformresource.TestStep{
			{
				Config: testAccDashboardV2StableOverwrite(uid, "Adopted dashboard"),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttrSet(dashboardV2StableResourceName, "id"),
					terraformresource.TestCheckResourceAttr(dashboardV2StableResourceName, "metadata.uid", uid),
					terraformresource.TestCheckResourceAttr(dashboardV2StableResourceName, "spec.title", "Adopted dashboard"),
				),
			},
		},
	})
}

// A dashboard already managed by a different manager identity (another Terraform
// workspace, a provisioning/GitSync pipeline, etc.) must not be silently taken over
// just because options.overwrite = true.
func TestAccDashboardV2StableOverwrite_refusesForeignManager(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=13.0.0")

	randSuffix := acctest.RandString(6)
	uid := "test-v2-overwrite-foreign-" + randSuffix

	createDashboardV2Fixture(t, uid, "Pre-existing foreign-managed dashboard", "other-workspace")
	t.Cleanup(func() { deleteDashboardV2Fixture(t, uid) })

	terraformresource.ParallelTest(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []terraformresource.TestStep{
			{
				Config:      testAccDashboardV2StableOverwrite(uid, "Attempted takeover"),
				ExpectError: regexp.MustCompile(`(?s)refusing to overwrite .*"other-workspace"`),
			},
		},
	})
}

func testAccDashboardV2StableOverwrite(uid, title string) string {
	return fmt.Sprintf(`
resource "grafana_apps_dashboard_dashboard_v2" "test" {
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

func dashboardV2NamespacedClient(t *testing.T) *sdkresource.NamespacedClient[*v2.Dashboard, *v2.DashboardList] {
	t.Helper()
	client := testutils.Provider.Meta().(*common.Client)

	rcli, err := client.GrafanaAppPlatformAPI.ClientFor(v2.DashboardKind())
	if err != nil {
		t.Fatalf("failed to create dashboard v2 client: %s", err)
	}

	// Mirrors appplatform.namespaceForClient: a configured stack ID (cloud-style
	// namespacing) takes precedence over org ID (on-prem namespacing).
	ns := claims.OrgNamespaceFormatter(client.GrafanaOrgID)
	if client.GrafanaStackID > 0 {
		ns = claims.CloudNamespaceFormatter(client.GrafanaStackID)
	}
	return sdkresource.NewNamespaced(sdkresource.NewTypedClient[*v2.Dashboard, *v2.DashboardList](rcli, v2.DashboardKind()), ns)
}

// createDashboardV2Fixture creates a dashboard directly through the app platform API,
// bypassing Terraform entirely -- standing in for a dashboard created in the Grafana UI
// (managerIdentity == "") or one already owned by another manager (managerIdentity set).
func createDashboardV2Fixture(t *testing.T, uid, title, managerIdentity string) {
	t.Helper()

	obj := v2.NewDashboard()
	obj.SetName(uid)

	var spec v2.DashboardSpec
	if err := json.Unmarshal([]byte(fmt.Sprintf(dashboardOverwriteSpecJSON, title)), &spec); err != nil {
		t.Fatalf("failed to unmarshal dashboard spec fixture: %s", err)
	}
	if err := obj.SetSpec(spec); err != nil {
		t.Fatalf("failed to set dashboard spec fixture: %s", err)
	}

	if managerIdentity != "" {
		meta, err := utils.MetaAccessor(obj)
		if err != nil {
			t.Fatalf("failed to get meta accessor: %s", err)
		}
		meta.SetManagerProperties(utils.ManagerProperties{
			Kind:     utils.ManagerKindTerraform,
			Identity: managerIdentity,
		})
	}

	if _, err := dashboardV2NamespacedClient(t).Create(context.Background(), obj, sdkresource.CreateOptions{}); err != nil {
		t.Fatalf("failed to create dashboard v2 fixture: %s", err)
	}
}

func deleteDashboardV2Fixture(t *testing.T, uid string) {
	t.Helper()
	if err := dashboardV2NamespacedClient(t).Delete(context.Background(), uid, sdkresource.DeleteOptions{}); err != nil {
		t.Logf("failed to clean up dashboard v2 fixture %s: %s", uid, err)
	}
}
