package grafana

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccResourceCloudPluginInstallation(t *testing.T) {
	CheckCloudAPITestsEnabled(t)

	slug := "terraformprovidergrafana"
	pluginSlug := "aws-datasource-provisioner-app"
	pluginVersion := "1.7.0"

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGrafanaCloudPluginInstallation(slug, pluginSlug, pluginVersion),
				Check: resource.ComposeTestCheckFunc(
					testAccCloudPluginInstallationCheckExists("grafana_cloud_plugin_installation.test-installation", slug, pluginSlug),
					resource.TestCheckResourceAttrSet("grafana_cloud_plugin_installation.test-installation", "id"),
					resource.TestCheckResourceAttr("grafana_cloud_plugin_installation.test-installation", "stack_slug", slug),
					resource.TestCheckResourceAttr("grafana_cloud_plugin_installation.test-installation", "slug", "aws-datasource-provisioner-app"),
					resource.TestCheckResourceAttr("grafana_cloud_plugin_installation.test-installation", "version", "1.7.0")),
			},
		},
		CheckDestroy: testAccCloudPluginInstallationDestroy(pluginSlug, pluginVersion),
	})
}

func testAccCloudPluginInstallationCheckExists(rn string, stackSlug string, pluginSlug string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s\n %#v", rn, s.RootModule().Resources)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		client := testAccProvider.Meta().(*client).gcloudapi
		actualInstallation, err := client.GetCloudPluginInstallation(stackSlug, pluginSlug)
		if err != nil {
			return fmt.Errorf("error getting installation: %s", err)
		}

		if strconv.Itoa(actualInstallation.ID) != rs.Primary.ID {
			return fmt.Errorf("resource id does not match")
		}

		return nil
	}
}

func testAccCloudPluginInstallationDestroy(stackSlug string, pluginSlug string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*client).gcloudapi

		installation, err := client.GetCloudPluginInstallation(stackSlug, pluginSlug)
		if err == nil {
			return fmt.Errorf("installation `%s` with ID `%d` still exists after destroy", installation.PluginSlug, installation.ID)
		}

		return nil
	}
}

func testAccGrafanaCloudPluginInstallation(stackSlug, name, version string) string {
	return fmt.Sprintf(`
		resource "grafana_cloud_plugin_installation" "test-installation" {
			stack_slug = "%s"
			slug       = "%s"
			version    = "%s"
		}
	`, stackSlug, name, version)
}
