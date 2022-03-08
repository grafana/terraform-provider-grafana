package grafana

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccCloudApiKey_Basic(t *testing.T) {
	CheckCloudAPITestsEnabled(t)

	var tests = []struct {
		role string
	}{
		{"Viewer"},
		{"Editor"},
		{"Admin"},
		{"MetricsPublisher"},
		{"PluginPublisher"},
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			resourceName := "zzztest-" + acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

			resource.Test(t, resource.TestCase{
				ProviderFactories: testAccProviderFactories,
				CheckDestroy:      testAccCheckCloudAPIKeyDestroy,
				Steps: []resource.TestStep{
					{
						Config: testAccCloudAPIKeyConfig(resourceName, tt.role),
						Check: resource.ComposeTestCheckFunc(
							testAccCheckCloudAPIKeyExists("grafana_cloud_api_key.test"),
							resource.TestCheckResourceAttrSet("grafana_cloud_api_key.test", "id"),
							resource.TestCheckResourceAttrSet("grafana_cloud_api_key.test", "key"),
							resource.TestCheckResourceAttr("grafana_cloud_api_key.test", "name", resourceName),
							resource.TestCheckResourceAttr("grafana_cloud_api_key.test", "role", tt.role),
						),
					},
					{
						ResourceName:      "grafana_cloud_api_key.test",
						ImportState:       true,
						ImportStateVerify: true,
					},
				},
			})
		})
	}
}

func testAccCheckCloudAPIKeyExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource `%s` not found", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource `%s` has no ID set", resourceName)
		}

		client := testAccProvider.Meta().(*client).gcloudapi
		res, err := client.ListCloudAPIKeys(rs.Primary.Attributes["cloud_org_slug"])
		if err != nil {
			return err
		}

		for _, apiKey := range res.Items {
			if apiKey.Name == rs.Primary.Attributes["name"] {
				return nil
			}
		}

		return fmt.Errorf("resource `%s` not found via API", resourceName)
	}
}

func testAccCheckCloudAPIKeyDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*client).gcloudapi

	for name, rs := range s.RootModule().Resources {
		if rs.Type != "grafana_cloud_api_key" {
			continue
		}

		res, err := client.ListCloudAPIKeys(rs.Primary.Attributes["cloud_org_slug"])
		if err != nil {
			return err
		}

		for _, apiKey := range res.Items {
			if apiKey.Name == rs.Primary.Attributes["name"] {
				return fmt.Errorf("resource `%s` still exists via API", name)
			}
		}
	}

	return nil
}

func testAccCloudAPIKeyConfig(resourceName, role string) string {
	// GRAFANA_CLOUD_ORG is required from the `CheckCloudAPITestsEnabled` function
	return fmt.Sprintf(`
resource "grafana_cloud_api_key" "test" {
  cloud_org_slug = "%s"
  name           = "%s"
  role           = "%s"
}
`, os.Getenv("GRAFANA_CLOUD_ORG"), resourceName, role)
}
