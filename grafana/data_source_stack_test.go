package grafana

import (
	"fmt"
	"testing"
	"time"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccDataSourceStack_Basic(t *testing.T) {
	CheckCloudTestsEnabled(t)

	prefix := "tfdatatest"

	resourceName := GetRandomStackName(prefix)
	var stack gapi.Stack
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheckCloudStack(t)
			testAccDeleteExistingStacks(t, prefix)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccStackCheckDestroy(&stack),
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceStackConfig(resourceName),
				Check: resource.ComposeTestCheckFunc(
					testAccStackCheckExists("grafana_cloud_stack.test", &stack),
					resource.TestCheckResourceAttrSet("data.grafana_cloud_stack.test", "id"),
					resource.TestCheckResourceAttr("data.grafana_cloud_stack.test", "name", resourceName),
					resource.TestCheckResourceAttr("data.grafana_cloud_stack.test", "slug", resourceName),
					resource.TestCheckResourceAttrSet("data.grafana_cloud_stack.test", "prometheus_url"),
					resource.TestCheckResourceAttrSet("data.grafana_cloud_stack.test", "prometheus_user_id"),
					resource.TestCheckResourceAttrSet("data.grafana_cloud_stack.test", "alertmanager_user_id"),

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

func testAccDataSourceStackConfig(resourceName string) string {
	return fmt.Sprintf(`
resource "grafana_cloud_stack" "test" {
  name = "%s"
  slug = "%s"
  region_slug = "eu"
}
data "grafana_cloud_stack" "test" {
  slug = grafana_cloud_stack.test.slug
  depends_on = [grafana_cloud_stack.test]
}
`, resourceName, resourceName)
}
