package cloud_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
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
				Config: testAccGrafanaCloudPluginInstallationNoVersion(stackSlug, pluginSlug),
				Check: resource.ComposeTestCheckFunc(
					testAccStackCheckExists("grafana_cloud_stack.test", &stack),
					testAccCloudPluginInstallationCheckExists(stackSlug, pluginSlug),
					resource.TestCheckResourceAttrSet("grafana_cloud_plugin_installation.test-installation-no-version", "id"),
					resource.TestCheckResourceAttr("grafana_cloud_plugin_installation.test-installation-no-version", "stack_slug", stackSlug),
					resource.TestCheckResourceAttr("grafana_cloud_plugin_installation.test-installation-no-version", "slug", pluginSlug),
					// Don't check version attribute since it's not specified in config
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
			wait_for_readiness = false
		}

		resource "grafana_cloud_plugin_installation" "test-installation" {
			stack_slug = grafana_cloud_stack.test.slug
			slug       = "%[2]s"
			version    = "%[3]s"
		}
	`, stackSlug, name, version)
}

func testAccGrafanaCloudPluginInstallationNoVersion(stackSlug, name string) string {
	return fmt.Sprintf(`
        resource "grafana_cloud_stack" "test" {
            name  = "%[1]s"
            slug  = "%[1]s"
            wait_for_readiness = false
        }
        resource "grafana_cloud_plugin_installation" "test-installation-no-version" {
            stack_slug = grafana_cloud_stack.test.slug
            slug       = "%[2]s"
            # version omitted - should install latest
        }
    `, stackSlug, name)
}
