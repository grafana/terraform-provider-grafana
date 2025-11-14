package cloud_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
)

func TestDataSourceAccessPolicy_Basic(t *testing.T) {
	testutils.CheckCloudAPITestsEnabled(t)

	var policy gcom.AuthAccessPolicy

	expiresAt := time.Now().Add(time.Hour * 24).UTC().Format(time.RFC3339)
	randomName := acctest.RandStringFromCharSet(6, acctest.CharSetAlpha)
	scopes := []string{
		"metrics:read",
		"logs:write",
		"accesspolicies:read",
		"accesspolicies:write",
		"accesspolicies:delete",
		"datadog:validate",
	}

	accessPolicyConfig := testAccCloudAccessPolicyTokenConfigBasic(randomName, randomName+"display", "prod-us-east-0", scopes, expiresAt, false)
	setItemMatcher := func(s *terraform.State) error {
		return resource.TestCheckTypeSetElemNestedAttrs("data.grafana_cloud_access_policies.test", "access_policies.*", map[string]string{
			"id":           *policy.Id,
			"region":       "prod-us-east-0",
			"name":         randomName,
			"display_name": randomName + "display",
			"status":       *policy.Status,
		})(s)
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCloudAccessPolicyCheckDestroy("prod-us-east-0", &policy),
		Steps: []resource.TestStep{
			// Test without filters
			{
				Config: accessPolicyConfig + testAccDataSourceAccessPoliciesConfigBasic(nil, nil),
				Check: resource.ComposeTestCheckFunc(
					testAccCloudAccessPolicyCheckExists("grafana_cloud_access_policy.test", &policy),

					resource.TestCheckNoResourceAttr("data.grafana_cloud_access_policies.test", "name_filter"),
					resource.TestCheckNoResourceAttr("data.grafana_cloud_access_policies.test", "region_filter"),
					setItemMatcher,
				),
			},
			// Test with name filter
			{
				Config: accessPolicyConfig + testAccDataSourceAccessPoliciesConfigBasic(&randomName, nil),
				Check: resource.ComposeTestCheckFunc(
					testAccCloudAccessPolicyCheckExists("grafana_cloud_access_policy.test", &policy),

					resource.TestCheckResourceAttr("data.grafana_cloud_access_policies.test", "name_filter", randomName),
					resource.TestCheckNoResourceAttr("data.grafana_cloud_access_policies.test", "region_filter"),
					resource.TestCheckResourceAttr("data.grafana_cloud_access_policies.test", "access_policies.#", "1"),
					setItemMatcher,
				),
			},
			// Test with region filter
			{
				Config: accessPolicyConfig + testAccDataSourceAccessPoliciesConfigBasic(nil, common.Ref("prod-us-east-0")),
				Check: resource.ComposeTestCheckFunc(
					testAccCloudAccessPolicyCheckExists("grafana_cloud_access_policy.test", &policy),

					resource.TestCheckNoResourceAttr("data.grafana_cloud_access_policies.test", "name_filter"),
					resource.TestCheckResourceAttr("data.grafana_cloud_access_policies.test", "region_filter", "prod-us-east-0"),
					setItemMatcher,
				),
			},
			// Test with name and region filter
			{
				Config: accessPolicyConfig + testAccDataSourceAccessPoliciesConfigBasic(&randomName, common.Ref("prod-us-east-0")),
				Check: resource.ComposeTestCheckFunc(
					testAccCloudAccessPolicyCheckExists("grafana_cloud_access_policy.test", &policy),

					resource.TestCheckResourceAttr("data.grafana_cloud_access_policies.test", "name_filter", randomName),
					resource.TestCheckResourceAttr("data.grafana_cloud_access_policies.test", "region_filter", "prod-us-east-0"),
					resource.TestCheckResourceAttr("data.grafana_cloud_access_policies.test", "access_policies.#", "1"),
					setItemMatcher,
				),
			},
			// Test with non-matching name filter
			{
				Config: accessPolicyConfig + testAccDataSourceAccessPoliciesConfigBasic(common.Ref("nonexistent"), nil),
				Check: resource.ComposeTestCheckFunc(
					testAccCloudAccessPolicyCheckExists("grafana_cloud_access_policy.test", &policy),

					resource.TestCheckResourceAttr("data.grafana_cloud_access_policies.test", "name_filter", "nonexistent"),
					resource.TestCheckNoResourceAttr("data.grafana_cloud_access_policies.test", "region_filter"),
					resource.TestCheckResourceAttr("data.grafana_cloud_access_policies.test", "access_policies.#", "0"),
				),
			},
		},
	})
}

func testAccDataSourceAccessPoliciesConfigBasic(name *string, region *string) string {
	regionAttr := ""
	if region != nil {
		regionAttr = fmt.Sprintf("region_filter = %q", *region)
	}

	nameAttr := ""
	if name != nil {
		nameAttr = fmt.Sprintf("name_filter = %q", *name)
	}

	return fmt.Sprintf(`
data "grafana_cloud_access_policies" "test" {
  depends_on = [grafana_cloud_access_policy.test]
  %s
  %s
}
`, regionAttr, nameAttr)
}
