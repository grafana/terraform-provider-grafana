package generic_test

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	sdkresource "github.com/grafana/grafana-app-sdk/resource"
	dashboardv2 "github.com/grafana/grafana/apps/dashboard/pkg/apis/dashboard/v2"
	"github.com/grafana/grafana/pkg/apimachinery/utils"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	terraformresource "github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	genericDashboardManagedAnnotation = "example.grafana.app/managed-annotation"
	genericDashboardManagedLabel      = "example.grafana.app/managed"
	genericDashboardUIOnlyLabel       = "grafana.app/ui-only"
)

func TestAccGenericResource_dashboardRepairsImportedMetadataDrift(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=13.0.0")

	suffix := strings.ToLower(acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))
	config := testAccGenericDashboardConfig(t, suffix)
	expectedTitle := "Generic Dashboard " + suffix
	driftedTitle := "Generic Dashboard Drifted " + suffix
	expectedFolderUID := "generic-dashboard-home-" + suffix
	driftedFolderUID := "generic-dashboard-drift-" + suffix

	terraformresource.Test(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckGenericDashboardDestroy,
		Steps: []terraformresource.TestStep{
			{
				Config: config,
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttrSet(genericResourceName, "id"),
					terraformresource.TestCheckResourceAttr(genericResourceName, "manifest.metadata.name", "generic-dashboard-"+suffix),
					terraformresource.TestCheckResourceAttr(genericResourceName, "manifest.apiVersion", "dashboard.grafana.app/v2"),
					terraformresource.TestCheckResourceAttr(genericResourceName, "manifest.kind", "Dashboard"),
					terraformresource.TestCheckResourceAttr(genericResourceName, "manifest.spec.title", expectedTitle),
				),
			},
			{
				ResourceName:      genericResourceName,
				ImportState:       true,
				ImportStateIdFunc: genericResourceImportIDFunc(genericResourceName),
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if len(states) != 1 {
						return fmt.Errorf("expected one imported state, got %d", len(states))
					}
					attrs := states[0].Attributes
					if attrs["manifest.metadata.name"] != "generic-dashboard-"+suffix {
						return fmt.Errorf("expected imported manifest.metadata.name = %q, got %q", "generic-dashboard-"+suffix, attrs["manifest.metadata.name"])
					}
					return nil
				},
			},
			{
				Config:             config,
				Check:              testAccMutateGenericDashboard(genericResourceName, driftedTitle, driftedFolderUID),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: config,
				Check: genericEventually(genericResourceName, getGenericDashboardV2, func(dashboard *dashboardv2.Dashboard) error {
					meta, err := utils.MetaAccessor(dashboard)
					if err != nil {
						return err
					}
					if dashboard.Spec.Title != expectedTitle {
						return fmt.Errorf("expected dashboard title %q after drift repair, got %q", expectedTitle, dashboard.Spec.Title)
					}
					if meta.GetFolder() != expectedFolderUID {
						return fmt.Errorf("expected dashboard folder %q after drift repair, got %q", expectedFolderUID, meta.GetFolder())
					}
					if dashboard.GetLabels()[genericDashboardManagedLabel] != "desired" {
						return fmt.Errorf("expected configured metadata label %q to be restored, got %q", genericDashboardManagedLabel, dashboard.GetLabels()[genericDashboardManagedLabel])
					}
					if dashboard.GetLabels()[genericDashboardUIOnlyLabel] != "true" {
						return fmt.Errorf("expected unconfigured metadata label %q to survive drift repair", genericDashboardUIOnlyLabel)
					}
					return nil
				}),
			},
			{
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAccGenericResource_dashboardSupportsConfiguredFinalizers(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=13.0.0")

	suffix := strings.ToLower(acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))
	configWithFinalizer := testAccGenericDashboardFinalizerConfig(t, suffix, []string{"protect"})
	configWithoutFinalizer := testAccGenericDashboardFinalizerConfig(t, suffix, []string{})

	terraformresource.Test(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckGenericDashboardDestroy,
		Steps: []terraformresource.TestStep{
			{
				Config: configWithFinalizer,
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttrSet(genericResourceName, "id"),
					genericEventually(genericResourceName, getGenericDashboardV2, func(dashboard *dashboardv2.Dashboard) error {
						if got := dashboard.GetFinalizers(); len(got) != 1 || got[0] != "protect" {
							return fmt.Errorf("expected dashboard finalizers [protect], got %v", got)
						}
						return nil
					}),
				),
			},
			{
				Config:             configWithFinalizer,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				Config: configWithoutFinalizer,
				Check: genericEventually(genericResourceName, getGenericDashboardV2, func(dashboard *dashboardv2.Dashboard) error {
					if got := dashboard.GetFinalizers(); len(got) != 0 {
						return fmt.Errorf("expected dashboard finalizers to be cleared, got %v", got)
					}
					return nil
				}),
			},
			{
				Config:             configWithoutFinalizer,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAccGenericResource_dashboardRemovesManagedMetadataKeys(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=13.0.0")

	suffix := strings.ToLower(acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))
	configWithManagedMetadata := testAccGenericDashboardManagedMetadataConfig(t, suffix, true)
	configWithoutManagedMetadata := testAccGenericDashboardManagedMetadataConfig(t, suffix, false)
	expectedFolderUID := "generic-dashboard-managed-home-" + suffix

	terraformresource.Test(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckGenericDashboardDestroy,
		Steps: []terraformresource.TestStep{
			{
				Config: configWithManagedMetadata,
				Check: genericEventually(genericResourceName, getGenericDashboardV2, func(dashboard *dashboardv2.Dashboard) error {
					annotations := dashboard.GetAnnotations()
					if annotations[genericDashboardManagedAnnotation] != "true" {
						return fmt.Errorf("expected managed annotation %q to be present, got %q", genericDashboardManagedAnnotation, annotations[genericDashboardManagedAnnotation])
					}
					if dashboard.GetLabels()[genericDashboardManagedLabel] != "desired" {
						return fmt.Errorf("expected managed label %q to be present, got %q", genericDashboardManagedLabel, dashboard.GetLabels()[genericDashboardManagedLabel])
					}
					return nil
				}),
			},
			{
				Config: configWithoutManagedMetadata,
				Check: genericEventually(genericResourceName, getGenericDashboardV2, func(dashboard *dashboardv2.Dashboard) error {
					annotations := dashboard.GetAnnotations()
					if annotations[genericDashboardManagedAnnotation] != "" {
						return fmt.Errorf("expected managed annotation %q to be removed, got %q", genericDashboardManagedAnnotation, annotations[genericDashboardManagedAnnotation])
					}
					if annotations["grafana.app/folder"] != expectedFolderUID {
						return fmt.Errorf("expected folder annotation to remain %q, got %q", expectedFolderUID, annotations["grafana.app/folder"])
					}
					if dashboard.GetLabels()[genericDashboardManagedLabel] != "" {
						return fmt.Errorf("expected managed label %q to be removed, got %q", genericDashboardManagedLabel, dashboard.GetLabels()[genericDashboardManagedLabel])
					}
					return nil
				}),
			},
			{
				Config:             configWithoutManagedMetadata,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAccGenericResource_dashboardRejectsConflictingMetadataUID(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=13.0.0")

	suffix := strings.ToLower(acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))

	// manifest has both metadata.name and metadata.uid with different values
	configConflictingManifestNameAndUID := fmt.Sprintf(`
%s

resource "grafana_apps_generic_resource" "test" {
  manifest = {
    apiVersion = "dashboard.grafana.app/v2"
    kind       = "Dashboard"
    metadata = {
      name = "manifest-name-%s"
      uid  = "manifest-uid-%s"
    }
    spec = {
      title = "Conflicting Name And UID Dashboard %s"
    }
  }
}
`, genericProviderConfig(t), suffix, suffix, suffix)

	terraformresource.Test(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []terraformresource.TestStep{
			{
				Config:      configConflictingManifestNameAndUID,
				ExpectError: regexp.MustCompile("(?i)(Conflicting|conflict)"),
			},
		},
	})
}

func TestAccGenericResource_dashboardManifestMetadataUIDAlias(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=13.0.0")

	suffix := strings.ToLower(acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))
	expectedTitle := "Generic UID Alias Dashboard " + suffix

	config := fmt.Sprintf(`
%s

resource "grafana_folder" "home" {
  title = "Generic UID Alias Folder %s"
  uid   = "generic-uid-alias-home-%s"
}

resource "grafana_apps_generic_resource" "test" {
  manifest = {
    apiVersion = "dashboard.grafana.app/v2"
    kind       = "Dashboard"
    metadata = {
      uid = "generic-uid-alias-%s"
      annotations = {
        "grafana.app/folder" = grafana_folder.home.uid
      }
    }
    spec = {
      title       = %q
      cursorSync  = "Off"
      elements    = {}
      layout      = { kind = "GridLayout", spec = { items = [] } }
      links       = []
      preload     = false
      tags        = null
      annotations = []
      variables   = []
      timeSettings = {
        timezone               = "browser"
        from                   = "now-6h"
        to                     = "now"
        autoRefresh            = ""
        autoRefreshIntervals   = null
        hideTimepicker         = false
        fiscalYearStartMonth   = 0
      }
    }
  }
}
`, genericProviderConfig(t), suffix, suffix, suffix, expectedTitle)

	terraformresource.Test(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckGenericDashboardDestroy,
		Steps: []terraformresource.TestStep{
			{
				Config: config,
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttrSet(genericResourceName, "id"),
					terraformresource.TestCheckResourceAttr(genericResourceName, "manifest.metadata.uid", "generic-uid-alias-"+suffix),
					terraformresource.TestCheckResourceAttr(genericResourceName, "manifest.spec.title", expectedTitle),
				),
			},
			{
				ResourceName:      genericResourceName,
				ImportState:       true,
				ImportStateIdFunc: genericResourceImportIDFunc(genericResourceName),
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if len(states) != 1 {
						return fmt.Errorf("expected one imported state, got %d", len(states))
					}
					attrs := states[0].Attributes
					if attrs["manifest.metadata.name"] != "generic-uid-alias-"+suffix {
						return fmt.Errorf("expected imported manifest.metadata.name = %q, got %q", "generic-uid-alias-"+suffix, attrs["manifest.metadata.name"])
					}
					if attrs["manifest.spec.title"] != expectedTitle {
						return fmt.Errorf("expected imported manifest.spec.title = %q, got %q", expectedTitle, attrs["manifest.spec.title"])
					}
					return nil
				},
			},
		},
	})
}

func TestAccGenericResource_dashboardRequiresReplaceOnIdentityChange(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=13.0.0")

	suffix := strings.ToLower(acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))
	originalUID := "generic-replace-orig-" + suffix
	changedUID := "generic-replace-new-" + suffix

	makeConfig := func(uid, title string) string {
		return fmt.Sprintf(`
%s

resource "grafana_folder" "home" {
  title = "Generic Replace Folder %s"
  uid   = "generic-replace-home-%s"
}

resource "grafana_apps_generic_resource" "test" {
  manifest = {
    apiVersion = "dashboard.grafana.app/v2"
    kind       = "Dashboard"
    metadata = {
      name = %q
      annotations = {
        "grafana.app/folder" = grafana_folder.home.uid
      }
    }
    spec = {
      title       = %q
      cursorSync  = "Off"
      elements    = {}
      layout      = { kind = "GridLayout", spec = { items = [] } }
      links       = []
      preload     = false
      tags        = null
      annotations = []
      variables   = []
      timeSettings = {
        timezone               = "browser"
        from                   = "now-6h"
        to                     = "now"
        autoRefresh            = ""
        autoRefreshIntervals   = null
        hideTimepicker         = false
        fiscalYearStartMonth   = 0
      }
    }
  }
}
`, genericProviderConfig(t), suffix, suffix, uid, title)
	}

	configOriginal := makeConfig(originalUID, "Generic Replace Original "+suffix)
	configChanged := makeConfig(changedUID, "Generic Replace Changed "+suffix)

	terraformresource.Test(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckGenericDashboardDestroy,
		Steps: []terraformresource.TestStep{
			{
				Config: configOriginal,
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttrSet(genericResourceName, "id"),
					terraformresource.TestCheckResourceAttr(genericResourceName, "manifest.metadata.name", originalUID),
				),
			},
			{
				Config: configChanged,
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(genericResourceName, "manifest.metadata.name", changedUID),
					genericEventually(genericResourceName, getGenericDashboardV2, func(dashboard *dashboardv2.Dashboard) error {
						if dashboard.GetName() != changedUID {
							return fmt.Errorf("expected dashboard name %q after replacement, got %q", changedUID, dashboard.GetName())
						}
						return nil
					}),
				),
			},
			{
				Config:             configChanged,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAccGenericResource_dashboardManagerPropertiesDefaultAllowsEdits(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=13.0.0")

	suffix := strings.ToLower(acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))

	config := renderGenericDashboardConfig(t, genericDashboardConfig{
		HomeFolderTitle: "Generic Dashboard Manager Props " + suffix,
		HomeFolderUID:   "gen-dash-mgr-home-" + suffix,
		ResourceUID:     "gen-dash-mgr-default-" + suffix,
		Title:           "Generic Dashboard Manager Default " + suffix,
	})

	terraformresource.Test(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckGenericDashboardDestroy,
		Steps: []terraformresource.TestStep{
			{
				Config: config,
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttrSet(genericResourceName, "id"),
					// Default is true when not configured.
					terraformresource.TestCheckResourceAttr(genericResourceName, "allow_ui_updates", "true"),
					genericEventually(genericResourceName, getGenericDashboardV2, func(dashboard *dashboardv2.Dashboard) error {
						meta, err := utils.MetaAccessor(dashboard)
						if err != nil {
							return err
						}
						mgr, ok := meta.GetManagerProperties()
						if !ok {
							return fmt.Errorf("expected manager properties to be set")
						}
						if !mgr.AllowsEdits {
							return fmt.Errorf("expected AllowsEdits=true by default, got false")
						}
						if mgr.Kind != utils.ManagerKindTerraform {
							return fmt.Errorf("expected manager kind %q, got %q", utils.ManagerKindTerraform, mgr.Kind)
						}
						return nil
					}),
				),
			},
			{
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAccGenericResource_dashboardManagerPropertiesDisablesEdits(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=13.0.0")

	suffix := strings.ToLower(acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))

	makeConfig := func(allowUIUpdates bool) string {
		return fmt.Sprintf(`
%s

resource "grafana_folder" "home" {
  title = "Generic Dashboard Manager Edit %s"
  uid   = "gen-dash-mgr-edit-%s"
}

resource "grafana_apps_generic_resource" "test" {
  allow_ui_updates = %v

  manifest = {
    apiVersion = "dashboard.grafana.app/v2"
    kind       = "Dashboard"
    metadata = {
      name = "gen-dash-mgr-edits-%s"
      annotations = {
        "grafana.app/folder" = grafana_folder.home.uid
      }
    }
    spec = {
      title       = "Manager Edits Dashboard %s"
      cursorSync  = "Off"
      elements    = {}
      layout      = { kind = "GridLayout", spec = { items = [] } }
      links       = []
      preload     = false
      tags        = null
      annotations = []
      variables   = []
      timeSettings = {
        timezone               = "browser"
        from                   = "now-6h"
        to                     = "now"
        autoRefresh            = ""
        autoRefreshIntervals   = null
        hideTimepicker         = false
        fiscalYearStartMonth   = 0
      }
    }
  }
}
`, genericProviderConfig(t), suffix, suffix, allowUIUpdates, suffix, suffix)
	}

	terraformresource.Test(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckGenericDashboardDestroy,
		Steps: []terraformresource.TestStep{
			// Step 1: create with allow_ui_updates = false.
			{
				Config: makeConfig(false),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(genericResourceName, "allow_ui_updates", "false"),
					genericEventually(genericResourceName, getGenericDashboardV2, func(dashboard *dashboardv2.Dashboard) error {
						meta, err := utils.MetaAccessor(dashboard)
						if err != nil {
							return err
						}
						mgr, ok := meta.GetManagerProperties()
						if !ok {
							return fmt.Errorf("expected manager properties to be set")
						}
						if mgr.AllowsEdits {
							return fmt.Errorf("expected AllowsEdits=false, got true")
						}
						return nil
					}),
				),
			},
			{
				Config:             makeConfig(false),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Step 3: update to allow_ui_updates = true.
			{
				Config: makeConfig(true),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(genericResourceName, "allow_ui_updates", "true"),
					genericEventually(genericResourceName, getGenericDashboardV2, func(dashboard *dashboardv2.Dashboard) error {
						meta, err := utils.MetaAccessor(dashboard)
						if err != nil {
							return err
						}
						mgr, ok := meta.GetManagerProperties()
						if !ok {
							return fmt.Errorf("expected manager properties to be set after update")
						}
						if !mgr.AllowsEdits {
							return fmt.Errorf("expected AllowsEdits=true after update, got false")
						}
						return nil
					}),
				),
			},
			{
				Config:             makeConfig(true),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAccGenericResource_dashboardImportPreservesManagerProperties(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=13.0.0")

	suffix := strings.ToLower(acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))

	config := fmt.Sprintf(`
%s

resource "grafana_folder" "home" {
  title = "Generic Dashboard Import Manager %s"
  uid   = "gen-dash-imp-mgr-%s"
}

resource "grafana_apps_generic_resource" "test" {
  allow_ui_updates = false

  manifest = {
    apiVersion = "dashboard.grafana.app/v2"
    kind       = "Dashboard"
    metadata = {
      name = "gen-dash-imp-mgr-%s"
      annotations = {
        "grafana.app/folder" = grafana_folder.home.uid
      }
    }
    spec = {
      title       = "Import Manager Props Dashboard %s"
      cursorSync  = "Off"
      elements    = {}
      layout      = { kind = "GridLayout", spec = { items = [] } }
      links       = []
      preload     = false
      tags        = null
      annotations = []
      variables   = []
      timeSettings = {
        timezone               = "browser"
        from                   = "now-6h"
        to                     = "now"
        autoRefresh            = ""
        autoRefreshIntervals   = null
        hideTimepicker         = false
        fiscalYearStartMonth   = 0
      }
    }
  }
}
`, genericProviderConfig(t), suffix, suffix, suffix, suffix)

	terraformresource.Test(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckGenericDashboardDestroy,
		Steps: []terraformresource.TestStep{
			{
				Config: config,
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(genericResourceName, "allow_ui_updates", "false"),
				),
			},
			{
				ResourceName:      genericResourceName,
				ImportState:       true,
				ImportStateIdFunc: genericResourceImportIDFunc(genericResourceName),
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if len(states) != 1 {
						return fmt.Errorf("expected one imported state, got %d", len(states))
					}
					if states[0].Attributes["allow_ui_updates"] != "false" {
						return fmt.Errorf("expected imported allow_ui_updates=false, got %q", states[0].Attributes["allow_ui_updates"])
					}
					return nil
				},
			},
			// Re-apply after import — should be idempotent.
			{
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func testAccGenericDashboardConfig(t *testing.T, suffix string) string {
	t.Helper()

	return renderGenericDashboardConfig(t, genericDashboardConfig{
		HomeFolderTitle: "Generic Dashboard Folder " + suffix,
		HomeFolderUID:   "generic-dashboard-home-" + suffix,
		ExtraResources: fmt.Sprintf(`
resource "grafana_folder" "drift" {
  title = "Generic Dashboard Drift Folder %s"
  uid   = "generic-dashboard-drift-%s"
}
`, suffix, suffix),
		ResourceUID: "generic-dashboard-" + suffix,
		Title:       "Generic Dashboard " + suffix,
		MetadataExtra: fmt.Sprintf(`      labels = {
        %q = "desired"
      }
`, genericDashboardManagedLabel),
	})
}

func testAccGenericDashboardFinalizerConfig(t *testing.T, suffix string, finalizers []string) string {
	t.Helper()

	return renderGenericDashboardConfig(t, genericDashboardConfig{
		HomeFolderTitle: "Generic Dashboard Finalizer Folder " + suffix,
		HomeFolderUID:   "gen-dash-fin-home-" + suffix,
		ResourceUID:     "gen-dash-finalizers-" + suffix,
		Title:           "Generic Dashboard Finalizers " + suffix,
		MetadataExtra:   fmt.Sprintf("      finalizers = %s\n", renderHCLStringList(finalizers)),
	})
}

func testAccGenericDashboardManagedMetadataConfig(t *testing.T, suffix string, includeManagedMetadata bool) string {
	t.Helper()

	managedAnnotationBlock := ""
	managedLabelsBlock := "labels = {}"
	if includeManagedMetadata {
		managedAnnotationBlock = fmt.Sprintf("\n        %q = \"true\"", genericDashboardManagedAnnotation)
		managedLabelsBlock = fmt.Sprintf("labels = {\n        %q = \"desired\"\n      }", genericDashboardManagedLabel)
	}

	return renderGenericDashboardConfig(t, genericDashboardConfig{
		HomeFolderTitle: "Generic Dashboard Managed Metadata Folder " + suffix,
		HomeFolderUID:   "generic-dashboard-managed-home-" + suffix,
		ResourceUID:     "generic-dashboard-managed-" + suffix,
		Title:           "Generic Dashboard Managed Metadata " + suffix,
		AnnotationExtra: managedAnnotationBlock,
		MetadataExtra:   "      " + managedLabelsBlock + "\n",
	})
}

type genericDashboardConfig struct {
	HomeFolderTitle string
	HomeFolderUID   string
	ExtraResources  string
	ResourceUID     string
	Title           string
	AnnotationExtra string
	MetadataExtra   string
}

func renderGenericDashboardConfig(t *testing.T, cfg genericDashboardConfig) string {
	t.Helper()

	return fmt.Sprintf(`
%s

resource "grafana_folder" "home" {
  title = %q
  uid   = %q
}
%s
resource "grafana_apps_generic_resource" "test" {
  manifest = {
    apiVersion = "dashboard.grafana.app/v2"
    kind       = "Dashboard"
    metadata = {
      name = %q
      annotations = {
        "grafana.app/folder" = grafana_folder.home.uid%s
      }
%s    }
    spec = {
      title       = %q
      cursorSync  = "Off"
      elements    = {}
      layout      = { kind = "GridLayout", spec = { items = [] } }
      links       = []
      preload     = false
      tags        = null
      annotations = []
      variables   = []
      timeSettings = {
        timezone               = "browser"
        from                   = "now-6h"
        to                     = "now"
        autoRefresh            = ""
        autoRefreshIntervals   = null
        hideTimepicker         = false
        fiscalYearStartMonth   = 0
      }
    }
  }
}
`, genericProviderConfig(t), cfg.HomeFolderTitle, cfg.HomeFolderUID, cfg.ExtraResources, cfg.ResourceUID, cfg.AnnotationExtra, cfg.MetadataExtra, cfg.Title)
}

func getGenericDashboardV2(ctx context.Context, client *common.Client, uid string) (*dashboardv2.Dashboard, error) {
	rcli, err := client.GrafanaAppPlatformAPI.ClientFor(dashboardv2.DashboardKind())
	if err != nil {
		return nil, fmt.Errorf("failed to create dashboard client: %w", err)
	}

	namespacedClient := sdkresource.NewNamespaced(
		sdkresource.NewTypedClient[*dashboardv2.Dashboard, *dashboardv2.DashboardList](rcli, dashboardv2.DashboardKind()),
		genericConfiguredOrgNamespace(client),
	)

	return namespacedClient.Get(ctx, uid)
}

func testAccMutateGenericDashboard(resourceName, title, folderUID string) terraformresource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testutils.Provider.Meta().(*common.Client)
		name, err := stateResourceAttribute(s, resourceName, "manifest.metadata.name")
		if err != nil {
			return err
		}

		dashboard, err := getGenericDashboardV2(context.Background(), client, name)
		if err != nil {
			return err
		}

		meta, err := utils.MetaAccessor(dashboard)
		if err != nil {
			return err
		}
		meta.SetFolder(folderUID)

		labels := dashboard.GetLabels()
		if labels == nil {
			labels = map[string]string{}
		}
		labels[genericDashboardManagedLabel] = "drifted"
		labels[genericDashboardUIOnlyLabel] = "true"
		dashboard.SetLabels(labels)
		dashboard.Spec.Title = title
		description := "UI only description"
		dashboard.Spec.Description = &description

		if err := updateGenericDashboard(context.Background(), client, dashboard); err != nil {
			return err
		}

		return genericEventually(resourceName, getGenericDashboardV2, func(current *dashboardv2.Dashboard) error {
			currentMeta, err := utils.MetaAccessor(current)
			if err != nil {
				return err
			}
			if current.Spec.Title != title {
				return fmt.Errorf("expected drifted dashboard title %q, got %q", title, current.Spec.Title)
			}
			if currentMeta.GetFolder() != folderUID {
				return fmt.Errorf("expected drifted dashboard folder %q, got %q", folderUID, currentMeta.GetFolder())
			}
			if current.GetLabels()[genericDashboardManagedLabel] != "drifted" {
				return fmt.Errorf("expected drifted configured metadata label %q, got %q", genericDashboardManagedLabel, current.GetLabels()[genericDashboardManagedLabel])
			}
			if current.GetLabels()[genericDashboardUIOnlyLabel] != "true" {
				return fmt.Errorf("expected drifted dashboard label %q to be set", genericDashboardUIOnlyLabel)
			}
			if current.Spec.Description == nil || *current.Spec.Description != description {
				return fmt.Errorf("expected drifted dashboard description %q, got %#v", description, current.Spec.Description)
			}
			return nil
		})(s)
	}
}

func updateGenericDashboard(ctx context.Context, client *common.Client, dashboard *dashboardv2.Dashboard) error {
	rcli, err := client.GrafanaAppPlatformAPI.ClientFor(dashboardv2.DashboardKind())
	if err != nil {
		return fmt.Errorf("failed to create dashboard client: %w", err)
	}

	namespacedClient := sdkresource.NewNamespaced(
		sdkresource.NewTypedClient[*dashboardv2.Dashboard, *dashboardv2.DashboardList](rcli, dashboardv2.DashboardKind()),
		genericConfiguredOrgNamespace(client),
	)

	_, err = namespacedClient.Update(ctx, dashboard, sdkresource.UpdateOptions{
		ResourceVersion: dashboard.GetResourceVersion(),
	})
	return err
}

func renderHCLStringList(values []string) string {
	if len(values) == 0 {
		return "[]"
	}

	items := make([]string, 0, len(values))
	for _, value := range values {
		items = append(items, fmt.Sprintf("%q", value))
	}
	return "[" + strings.Join(items, ", ") + "]"
}

func testAccCheckGenericDashboardDestroy(s *terraform.State) error {
	return genericCheckDestroyWithNotFound(s, "grafana_apps_generic_resource", "dashboard", getGenericDashboardV2, apierrors.IsNotFound)
}
