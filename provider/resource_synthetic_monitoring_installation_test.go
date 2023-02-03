package provider

import (
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccSyntheticMonitoringInstallation(t *testing.T) {
	CheckCloudAPITestsEnabled(t)

	var stack gapi.Stack
	stackPrefix := "tfsminstalltest"
	testAccDeleteExistingStacks(t, stackPrefix)
	stackSlug := GetRandomStackName(stackPrefix)

	apiKeyPrefix := "testsminstall-"
	testAccDeleteExistingCloudAPIKeys(t, apiKeyPrefix)
	apiKeyName := apiKeyPrefix + acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccStackCheckDestroy(&stack),
		Steps: []resource.TestStep{
			{
				Config: testAccSyntheticMonitoringInstallation(stackSlug, apiKeyName),
				Check: resource.ComposeTestCheckFunc(
					testAccStackCheckExists("grafana_cloud_stack.test", &stack),
					resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_installation.test", "sm_access_token"),
				),
			},
			// Test deletion
			{
				Config: testAccSyntheticMonitoringInstallation(stackSlug, apiKeyName),
			},
		},
	})
}

func testAccSyntheticMonitoringInstallation_Base(stackSlug, apiKeyName string) string {
	return testAccStackConfigBasic(stackSlug, stackSlug) +
		testAccCloudAPIKeyConfig(apiKeyName, "MetricsPublisher")
}

func testAccSyntheticMonitoringInstallation(stackSlug, apiKeyName string) string {
	return testAccSyntheticMonitoringInstallation_Base(stackSlug, apiKeyName) +
		`
	resource "grafana_synthetic_monitoring_installation" "test" {
		stack_id              = grafana_cloud_stack.test.id
		metrics_instance_id   = grafana_cloud_stack.test.prometheus_user_id
		logs_instance_id      = grafana_cloud_stack.test.logs_user_id
		metrics_publisher_key = grafana_cloud_api_key.test.key
	}
	`
}
