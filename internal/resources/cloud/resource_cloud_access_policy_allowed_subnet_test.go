package cloud_test

import (
	"fmt"
	"os"
	"strings"

	"testing"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestResourceAccessPolicy_AllowedSubnets(t *testing.T) {

	t.Parallel()
	testutils.CheckCloudAPITestsEnabled(t)

	var policy gcom.AuthAccessPolicy

	scopes := []string{
		"accesspolicies:read",
	}

	initialAllowedSubnets := []string{
		"10.0.0.29/32",
	}
	updatedAllowedSubnets := []string{
		"10.0.0.29/32",
		"10.0.0.20/32",
	}

	randomName := acctest.RandStringFromCharSet(6, acctest.CharSetAlpha)

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCloudAccessPolicyCheckDestroy("us", &policy),
		Steps: []resource.TestStep{
			{
				Config: testAccCloudAccessPolicyConfigAllowedSubnets(randomName, "display name", "us", scopes, initialAllowedSubnets),
				Check: resource.ComposeTestCheckFunc(
					testAccCloudAccessPolicyCheckExists("grafana_cloud_access_policy.test", &policy),

					resource.TestCheckResourceAttr("grafana_cloud_access_policy.test", "conditions.#", "1"),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy.test", "conditions.0.allowed_subnets.#", "1"),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy.test", "conditions.0.allowed_subnets.0", "10.0.0.29/32"),
				),
			},
			{
				Config: testAccCloudAccessPolicyConfigAllowedSubnets(randomName, "display name", "us", scopes, updatedAllowedSubnets),
				Check: resource.ComposeTestCheckFunc(
					testAccCloudAccessPolicyCheckExists("grafana_cloud_access_policy.test", &policy),

					resource.TestCheckResourceAttr("grafana_cloud_access_policy.test", "conditions.#", "1"),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy.test", "conditions.0.allowed_subnets.#", "2"),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy.test", "conditions.0.allowed_subnets.0", "10.0.0.20/32"),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy.test", "conditions.0.allowed_subnets.1", "10.0.0.29/32"),
				),
			},
		},
	})
}

// Returns terraform manifests for an Cloud Access Policy with Allowed Subnets defined.
func testAccCloudAccessPolicyConfigAllowedSubnets(name, displayName, region string, scopes []string, allowedSubnets []string) string {
	if displayName != "" {
		displayName = fmt.Sprintf("display_name = \"%s\"", displayName)
	}

	return fmt.Sprintf(`
	data "grafana_cloud_organization" "current" {
		slug = "%[4]s"
	}

	resource "grafana_cloud_access_policy" "test" {
		region       = "%[5]s"
		name         = "%[1]s"
		%[2]s

		scopes = ["%[3]s"]

		realm {
			type       = "org"
			identifier = data.grafana_cloud_organization.current.id
		}

		conditions {
			allowed_subnets = ["%[6]s"]
  		}

	}

	`, name, displayName, strings.Join(scopes, `","`), os.Getenv("GRAFANA_CLOUD_ORG"), region, strings.Join(allowedSubnets, `","`))
}
