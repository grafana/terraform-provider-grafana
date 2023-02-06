package provider

import (
	"fmt"
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/provider/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDatasourceCloudStack_Basic(t *testing.T) {
	testutils.CheckCloudAPITestsEnabled(t)

	prefix := "tfdatatest"

	resourceName := GetRandomStackName(prefix)
	var stack gapi.Stack
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccDeleteExistingStacks(t, prefix)
		},
		ProviderFactories: testutils.GetProviderFactories(),
		CheckDestroy:      testAccStackCheckDestroy(&stack),
		Steps: []resource.TestStep{
			{
				Config: testAccDatasourceCloudStackConfig(resourceName),
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
				),
			},
		},
	})
}

func testAccDatasourceCloudStackConfig(resourceName string) string {
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
