package grafana

//
//import (
//	"fmt"
//	"testing"
//
//	gapi "github.com/grafana/grafana-api-golang-client"
//	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
//	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
//)
//
//func TestAccResourceCloudPluginInstallation(t *testing.T) {
//	CheckCloudAPITestsEnabled(t)
//
//	slug := "terraformprovidergrafana"
//	var installation gapi.CloudPluginInstallation
//
//	resource.Test(t, resource.TestCase{
//		ProviderFactories: testAccProviderFactories,
//		Steps: []resource.TestStep{
//			{
//				Config: testAccGrafanaCloudPluginInstallation(slug, "aws-datasource-provisioner-app", "1.7.0"),
//				Check: resource.ComposeTestCheckFunc(
//					testAccCloudPluginInstallationCheckExists("grafana_cloud_plugin_installation.test-installation", &installation),
//					resource.TestCheckResourceAttrSet("grafana_cloud_plugin_installation.test-installation", "id"),
//					resource.TestCheckResourceAttr("grafana_cloud_plugin_installation.test-installation", "stack_slug", slug),
//					resource.TestCheckResourceAttr("grafana_cloud_plugin_installation.test-installation", "slug", "aws-datasource-provisioner-app"),
//					resource.TestCheckResourceAttr("grafana_cloud_plugin_installation.test-installation", "version", "1.7.0")
//			},
//		},
//		CheckDestroy: testAccCloudPluginInstallationDestroy(&installation),
//	})
//}
//
//func testAccCloudPluginInstallationCheckExists(rn string, installation *gapi.CloudPluginInstallation) resource.TestCheckFunc {
//	return func(s *terraform.State) error {
//		rs, ok := s.RootModule().Resources[rn]
//		if !ok {
//			return fmt.Errorf("resource not found: %s\n %#v", rn, s.RootModule().Resources)
//		}
//
//		if rs.Primary.ID == "" {
//			return fmt.Errorf("resource id not set")
//		}
//
//		stackSlug := rs.Primary.Attributes["stack_slug"]
//		pluginSlug := rs.Primary.Attributes["slug"]
//
//		client := testAccProvider.Meta().(*client).gapi
//		actualInstallation, err := client.GetCloudPluginInstallation(stackSlug, pluginSlug)
//		if err != nil {
//			return fmt.Errorf("error getting job: %s", err)
//		}
//
//		installation = actualInstallation
//
//		return nil
//	}
//}
//
//func testAccCloudPluginInstallationDestroy(installation *gapi.CloudPluginInstallation) resource.TestCheckFunc {
//	return func(s *terraform.State) error {
//		client := testAccProvider.Meta().(*client).gapi
//
//		installation, err := client.GetCloudPluginInstallation(installation.InstanceSlug, installation.PluginSlug)
//		if err == nil {
//			return fmt.Errorf("installation `%s` with ID `%d` still exists after destroy", installation.PluginSlug, installation.ID)
//		}
//
//		return nil
//	}
//}
//
//func testAccGrafanaCloudPluginInstallation(stackSlug, name, version string) string {
//	return fmt.Sprintf(`
//		resource "grafana_cloud_plugin_installation" "test-installation" {
//			stack_slug = "%s"
//			slug       = "%s"
//			version    = "%s"
//		}
//	`, stackSlug, name, version)
//}
