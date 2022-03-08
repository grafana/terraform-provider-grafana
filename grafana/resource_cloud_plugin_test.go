package grafana

import (
	"fmt"
	"testing"
	"time"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccResourceCloudPluginInstallation(t *testing.T) {
	CheckCloudAPITestsEnabled(t)

	var stack gapi.Stack
	prefix := "tfpl"
	slug := GetRandomStackName(prefix)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccDeleteExistingStacks(t, prefix)
		},
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStackConfigBasic(slug, slug),
				Check: resource.ComposeTestCheckFunc(
					testAccStackCheckExists("grafana_cloud_stack.test", &stack),
				),
			},
			{
				Config: testAccGrafanaCloudPluginInstallation(slug, "aws-datasource-provisioner-app", "1.7.0"),
				Check: resource.ComposeTestCheckFunc(
					testAccStackCheckExists("grafana_cloud_stack.test", &stack),
					testAccGrafanaCloudPluginInstallationCheckFields("grafana_cloud_plugin_installation.management", slug, "aws-datasource-provisioner-app", "1.7.0"),

					// TODO: Check how we can remove this sleep
					// Sometimes the stack is not ready to be deleted at the end of the test
					func(s *terraform.State) error {
						time.Sleep(time.Second * 15)
						return nil
					},
				),
			},
		},
	})
}

func testAccGrafanaCloudPluginInstallationCheckFields(n string, stackSlug string, slug string, version string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		if rs.Primary.Attributes["stack_slug"] != stackSlug {
			return fmt.Errorf("incorrect stack slug field found: %s", rs.Primary.Attributes["stack_slug"])
		}

		if rs.Primary.Attributes["slug"] != slug {
			return fmt.Errorf("incorrect slug field found: %s", rs.Primary.Attributes["slug"])
		}

		if rs.Primary.Attributes["version"] != version {
			return fmt.Errorf("incorrect version field found: %s", rs.Primary.Attributes["version"])
		}

		return nil
	}
}

func testAccGrafanaAuthKeyCheckDestroyCloud(s *terraform.State) error {
	return nil //errors.New("")
}

func testAccGrafanaCloudPluginInstallation(stackSlug, name, version string) string {
	return fmt.Sprintf(`
 		resource "grafana_cloud_plugin_installation" "installation" {
 			stack_slug = "%s"
			slug       = "%s"
 			version    = "%s"
 		}
	`, stackSlug, name, version)
}
