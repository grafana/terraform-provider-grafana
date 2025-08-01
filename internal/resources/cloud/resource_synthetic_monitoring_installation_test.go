package cloud_test

import (
	"testing"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccSyntheticMonitoringInstallation(t *testing.T) {
	testutils.CheckCloudAPITestsEnabled(t)

	for region, expectedURL := range map[string]string{
		"prod-ca-east-0": "https://synthetic-monitoring-api-ca-east-0.grafana.net",
		"eu":             "https://synthetic-monitoring-api-eu-west.grafana.net",
	} {
		t.Run(region, func(t *testing.T) {
			var stack gcom.FormattedApiInstance
			stackPrefix := "tfsminstalltest"
			testAccDeleteExistingStacks(t, stackPrefix)
			stackSlug := GetRandomStackName(stackPrefix)

			accessPolicyPrefix := "testsminstall-"
			testAccDeleteExistingAccessPolicies(t, region, accessPolicyPrefix)
			accessPolicyName := accessPolicyPrefix + acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

			resource.ParallelTest(t, resource.TestCase{
				ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
				CheckDestroy:             testAccStackCheckDestroy(&stack),
				Steps: []resource.TestStep{
					{
						Config: testAccSyntheticMonitoringInstallation(stackSlug, accessPolicyName, region),
						Check: resource.ComposeTestCheckFunc(
							testAccStackCheckExists("grafana_cloud_stack.test", &stack),
							resource.TestCheckResourceAttrSet("grafana_synthetic_monitoring_installation.test", "sm_access_token"),
							resource.TestCheckResourceAttr("grafana_synthetic_monitoring_installation.test", "stack_sm_api_url", expectedURL),
						),
					},
					// Test deletion
					{
						Config: testAccSyntheticMonitoringInstallation_Base(stackSlug, accessPolicyName, region),
					},
				},
			})
		})
	}
}

func testAccSyntheticMonitoringInstallation_Base(stackSlug, accessPolicyName, region string) string {
	return testAccStackConfigBasicWithCustomResourceName(stackSlug, stackSlug, region, "test", "description") +
		testAccCloudAccessPolicyTokenConfigBasic(accessPolicyName, accessPolicyName, region, []string{"metrics:write", "stacks:read", "logs:write", "traces:write"}, "")
}

func testAccSyntheticMonitoringInstallation(stackSlug, apiKeyName, region string) string {
	return testAccSyntheticMonitoringInstallation_Base(stackSlug, apiKeyName, region) +
		`
	resource "grafana_synthetic_monitoring_installation" "test" {
		stack_id              = grafana_cloud_stack.test.id
		metrics_publisher_key = grafana_cloud_access_policy_token.test.token
	}
	`
}
