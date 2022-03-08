package grafana

import (
	"fmt"
	"testing"
	"time"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccGrafanaAuthKeyFromCloud(t *testing.T) {
	CheckCloudTestsEnabled(t)

	var stack gapi.Stack
	prefix := "tfplugininstallationtest"
	slug := GetRandomStackName(prefix)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccDeleteExistingStacks(t, prefix)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccStackCheckDestroy(&stack),
		Steps: []resource.TestStep{
			{
				Config: testAccGrafanaCloudPluginInstallation(slug, "some-plugin", "1.2.3"),
				Check: resource.ComposeTestCheckFunc(
					testAccStackCheckExists("grafana_cloud_stack.test", &stack),
					testAccGrafanaCloudPluginInstallationCheckFields("grafana_cloud_plugin_installation.management", "slug", "some-plugin", "1.2.3"),

					// TODO: Check how we can remove this sleep
					// Sometimes the stack is not ready to be deleted at the end of the test
					func(s *terraform.State) error {
						time.Sleep(time.Second * 15)
						return nil
					},
				),
			},
			{
				Config: testAccStackConfigBasic(slug, slug),
				Check:  testAccGrafanaAuthKeyCheckDestroyCloud,
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
 		resource "grafana_cloud_plugin_installation" "foo" {
 			stack_slug = "%s"
			name       = "%s"
 			version    = "%s"
 		}
	`, stackSlug, name, version)
}
