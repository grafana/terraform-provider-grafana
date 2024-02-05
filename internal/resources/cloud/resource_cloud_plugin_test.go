package cloud_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/grafana/terraform-provider-grafana/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccResourcePluginInstallation(t *testing.T) {
	testutils.CheckCloudAPITestsEnabled(t)

	var stack gcom.FormattedApiInstance
	stackPrefix := "tfplugin"
	stackSlug := GetRandomStackName(stackPrefix)
	pluginSlug := "aws-datasource-provisioner-app"
	pluginVersion := "1.7.0"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccDeleteExistingStacks(t, stackPrefix) },
		ProviderFactories: testutils.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGrafanaCloudPluginInstallation(stackSlug, pluginSlug, pluginVersion),
				Check: resource.ComposeTestCheckFunc(
					testAccStackCheckExists("grafana_cloud_stack.test", &stack),
					testAccCloudPluginInstallationCheckExists(stackSlug, pluginSlug),
					resource.TestCheckResourceAttrSet("grafana_cloud_plugin_installation.test-installation", "id"),
					resource.TestCheckResourceAttr("grafana_cloud_plugin_installation.test-installation", "stack_slug", stackSlug),
					resource.TestCheckResourceAttr("grafana_cloud_plugin_installation.test-installation", "slug", "aws-datasource-provisioner-app"),
					resource.TestCheckResourceAttr("grafana_cloud_plugin_installation.test-installation", "version", "1.7.0")),
			},
			{
				ResourceName:      "grafana_cloud_plugin_installation.test-installation",
				ImportState:       true,
				ImportStateVerify: true,
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
