package cloud_test

import (
	"fmt"
	"testing"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceStack_Basic(t *testing.T) {
	testutils.CheckCloudAPITestsEnabled(t)

	prefix := "tfdatatest"

	resourceName := GetRandomStackName(prefix)
	var stack gcom.FormattedApiInstance
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccDeleteExistingStacks(t, prefix)
		},
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccStackCheckDestroy(&stack),
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceStackConfig(resourceName),
				Check: resource.ComposeTestCheckFunc(
					testAccStackCheckExists("grafana_cloud_stack.test", &stack),
					resource.TestCheckResourceAttrSet("data.grafana_cloud_stack.test", "id"),
					resource.TestCheckResourceAttr("data.grafana_cloud_stack.test", "name", resourceName),
					resource.TestCheckResourceAttr("data.grafana_cloud_stack.test", "slug", resourceName),
					resource.TestCheckResourceAttr("data.grafana_cloud_stack.test", "prometheus_remote_endpoint", "https://prometheus-prod-01-eu-west-0.grafana.net/api/prom"),
					resource.TestCheckResourceAttr("data.grafana_cloud_stack.test", "prometheus_remote_write_endpoint", "https://prometheus-prod-01-eu-west-0.grafana.net/api/prom/push"),
					resource.TestCheckResourceAttrSet("data.grafana_cloud_stack.test", "prometheus_url"),
					resource.TestCheckResourceAttrSet("data.grafana_cloud_stack.test", "prometheus_user_id"),
					resource.TestCheckResourceAttrSet("data.grafana_cloud_stack.test", "alertmanager_user_id"),
					resource.TestCheckResourceAttrSet("data.grafana_cloud_stack.test", "fleet_management_user_id"),
					resource.TestCheckResourceAttrSet("data.grafana_cloud_stack.test", "fleet_management_name"),
					resource.TestCheckResourceAttrSet("data.grafana_cloud_stack.test", "fleet_management_url"),
					resource.TestCheckResourceAttrSet("data.grafana_cloud_stack.test", "fleet_management_status"),
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
