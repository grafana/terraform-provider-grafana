package cloud_test

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/resources/cloud"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccResourcePluginInstallation(t *testing.T) {
	testutils.CheckCloudAPITestsEnabled(t)

	var stack gcom.FormattedApiInstance
	stackPrefix := "tfplugin"
	stackSlug := GetRandomStackName(stackPrefix)
	pluginSlug := "grafana-googlesheets-datasource" // TODO: Add datasource to find a plugin and use that
	pluginVersion := "1.2.5"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccDeleteExistingStacks(t, stackPrefix) },
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGrafanaCloudPluginInstallation(stackSlug, pluginSlug, pluginVersion),
				Check: resource.ComposeTestCheckFunc(
					testAccStackCheckExists("grafana_cloud_stack.test", &stack),
					testAccCloudPluginInstallationCheckExists(stackSlug, pluginSlug),
					resource.TestCheckResourceAttrSet("grafana_cloud_plugin_installation.test-installation", "id"),
					resource.TestCheckResourceAttr("grafana_cloud_plugin_installation.test-installation", "stack_slug", stackSlug),
					resource.TestCheckResourceAttr("grafana_cloud_plugin_installation.test-installation", "slug", "grafana-googlesheets-datasource"),
					resource.TestCheckResourceAttr("grafana_cloud_plugin_installation.test-installation", "version", "1.2.5")),
			},
			{
				Config: testAccGrafanaCloudPluginInstallationLatest(stackSlug, "grafana-clock-panel", pluginSlug, pluginVersion),
				Check: resource.ComposeTestCheckFunc(
					testAccStackCheckExists("grafana_cloud_stack.test", &stack),
					testAccCloudPluginInstallationCheckExists(stackSlug, "grafana-clock-panel"),
					resource.TestCheckResourceAttrSet("grafana_cloud_plugin_installation.test-installation-no-version", "id"),
					resource.TestCheckResourceAttr("grafana_cloud_plugin_installation.test-installation-no-version", "stack_slug", stackSlug),
					resource.TestCheckResourceAttr("grafana_cloud_plugin_installation.test-installation-no-version", "slug", "grafana-clock-panel"),
					resource.TestCheckResourceAttr("grafana_cloud_plugin_installation.test-installation-no-version", "version", "latest"),
				),
			},
			{
				ResourceName:      "grafana_cloud_plugin_installation.test-installation",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      "grafana_cloud_plugin_installation.test-installation",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     fmt.Sprintf("%s:%s", stackSlug, pluginSlug),
			},
			// Test import with invalid ID
			{
				ResourceName:      "grafana_cloud_plugin_installation.test-installation",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     "noseparator",
				ExpectError:       regexp.MustCompile("Error: id \"noseparator\" does not match expected format. Should be in the format: stackSlug:pluginSlug"),
			},
			// Test deletion (stack must keep existing to really test deletion)
			{
				Config: testutils.WithoutResource(t, testAccGrafanaCloudPluginInstallation(stackSlug, pluginSlug, pluginVersion), "grafana_cloud_plugin_installation.test-installation"),
				Check: resource.ComposeTestCheckFunc(
					testAccStackCheckExists("grafana_cloud_stack.test", &stack),
					testAccCloudPluginInstallationDestroy(stackSlug, pluginSlug),
				),
			},
		},
		CheckDestroy: testAccStackCheckDestroy(&stack),
	})
}

func testAccCloudPluginInstallationCheckExists(stackSlug string, pluginSlug string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testutils.Provider.Meta().(*common.Client).GrafanaCloudAPI
		_, _, err := client.InstancesAPI.GetInstancePlugin(context.Background(), stackSlug, pluginSlug).Execute()
		if err != nil {
			return fmt.Errorf("error getting installation: %s", err)
		}

		return nil
	}
}

func testAccCloudPluginInstallationDestroy(stackSlug string, pluginSlug string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		existsErr := testAccCloudPluginInstallationCheckExists(stackSlug, pluginSlug)(s)
		if existsErr == nil {
			return fmt.Errorf("installation still exists")
		}
		return nil
	}
}

func testAccGrafanaCloudPluginInstallation(stackSlug, name, version string) string {
	return fmt.Sprintf(`
		resource "grafana_cloud_stack" "test" {
			name  = "%[1]s"
			slug  = "%[1]s"
			delete_protection = false
			wait_for_readiness = false
		}

		resource "grafana_cloud_plugin_installation" "test-installation" {
			stack_slug = grafana_cloud_stack.test.slug
			slug       = "%[2]s"
			version    = "%[3]s"
		}
	`, stackSlug, name, version)
}

// testAccGrafanaCloudPluginInstallationLatest preserves test-installation
// (from testAccGrafanaCloudPluginInstallation) so that subsequent ImportState
// steps targeting that resource still find it in state.
func testAccGrafanaCloudPluginInstallationLatest(stackSlug, name, originalSlug, originalVersion string) string {
	return fmt.Sprintf(`
		resource "grafana_cloud_stack" "test" {
			name  = "%[1]s"
			slug  = "%[1]s"
			delete_protection = false
			wait_for_readiness = false
		}

		resource "grafana_cloud_plugin_installation" "test-installation" {
			stack_slug = grafana_cloud_stack.test.slug
			slug       = "%[3]s"
			version    = "%[4]s"
		}

        resource "grafana_cloud_plugin_installation" "test-installation-no-version" {
            stack_slug = grafana_cloud_stack.test.slug
            slug       = "%[2]s"
        }
    `, stackSlug, name, originalSlug, originalVersion)
}

func TestAccResourcePluginInstallation_managed(t *testing.T) {
	testutils.CheckCloudAPITestsEnabled(t)
	if os.Getenv("TF_ACC_CLOUD_MANAGED_PLUGINS") == "" {
		t.Skip("TF_ACC_CLOUD_MANAGED_PLUGINS must be set, with GRAFANA_CLOUD_ACCESS_POLICY_TOKEN for an org that the managed plugin sync manages")
	}

	for _, tt := range []struct {
		strategy      string
		stackPrefix   string
		pluginSlug    string
		pinnedVersion string
	}{
		{strategy: "assigned", stackPrefix: "tfmngassigned", pluginSlug: "grafana-github-datasource", pinnedVersion: "2.9.0"},
		{strategy: "major-aligned", stackPrefix: "tfmngaligned", pluginSlug: "mckn-funnel-panel", pinnedVersion: "2.0.0"},
	} {
		t.Run(tt.strategy, func(t *testing.T) {
			var stack gcom.FormattedApiInstance
			stackSlug := GetRandomStackName(tt.stackPrefix)
			resourceName := "grafana_cloud_plugin_installation.managed"
			config := testAccGrafanaCloudPluginInstallationManaged(stackSlug, tt.pluginSlug, tt.pinnedVersion)
			var targetVersion string

			resource.ParallelTest(t, resource.TestCase{
				PreCheck:                 func() { testAccDeleteExistingStacks(t, tt.stackPrefix) },
				ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: config,
						Check: resource.ComposeTestCheckFunc(
							testAccStackCheckExists("grafana_cloud_stack.test", &stack),
							resource.TestCheckResourceAttr(resourceName, "version", tt.pinnedVersion),
						),
					},
					{
						PreConfig: func() {
							targetVersion = testAccMovePluginToManagedTarget(t, stackSlug, tt.pluginSlug)
						},
						RefreshState: true,
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr(resourceName, "version", tt.pinnedVersion),
							func(s *terraform.State) error {
								return resource.TestCheckResourceAttr(resourceName, "installed_version", targetVersion)(s)
							},
						),
					},
					{
						ResourceName:            resourceName,
						ImportState:             true,
						ImportStateVerify:       true,
						ImportStateVerifyIgnore: []string{"version"},
					},
				},
				CheckDestroy: testAccStackCheckDestroy(&stack),
			})
		})
	}
}

// testAccMovePluginToManagedTarget does what the managed plugin sync would do, without waiting
// for its 10-minute schedule.
func testAccMovePluginToManagedTarget(t *testing.T, stackSlug, pluginSlug string) string {
	t.Helper()
	client := testutils.Provider.Meta().(*common.Client).GrafanaCloudAPI
	installation, _, err := client.InstancesAPI.GetInstancePlugin(context.Background(), stackSlug, pluginSlug).Execute()
	if err != nil {
		t.Fatalf("reading %s: %v", pluginSlug, err)
	}
	managed := installation.GetManaged()
	if !installation.HasManaged() || !managed.GetEnrolled() {
		t.Fatalf("%s on %s is not managed. Use an org that the managed plugin sync enrolls, and a pin at or above the plugin's minimum supported version.", pluginSlug, stackSlug)
	}
	target := managed.GetTargetVersion()
	if target == "" {
		t.Fatalf("%s on %s has no target version", pluginSlug, stackSlug)
	}
	req := gcom.PostInstancePluginRequest{Version: common.Ref(target)}
	_, _, err = client.InstancesAPI.PostInstancePlugin(context.Background(), stackSlug, pluginSlug).
		PostInstancePluginRequest(req).
		XRequestId(cloud.ClientRequestID()).
		Execute()
	if err != nil {
		t.Fatalf("moving %s to %s: %v", pluginSlug, target, err)
	}
	return target
}

func testAccGrafanaCloudPluginInstallationManaged(stackSlug, pluginSlug, version string) string {
	return fmt.Sprintf(`
		resource "grafana_cloud_stack" "test" {
			name  = "%[1]s"
			slug  = "%[1]s"
			delete_protection = false
			wait_for_readiness = false
		}

		resource "grafana_cloud_plugin_installation" "managed" {
			stack_slug = grafana_cloud_stack.test.slug
			slug       = "%[2]s"
			version    = "%[3]s"
		}
	`, stackSlug, pluginSlug, version)
}
