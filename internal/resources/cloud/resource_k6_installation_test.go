package cloud_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/grafana/grafana-com-public-clients/go/gcom"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
)

func TestAccK6Installation(t *testing.T) {
	testutils.CheckCloudAPITestsEnabled(t)

	var stack gcom.FormattedApiInstance
	stackPrefix := "tfk6installtest"
	testAccDeleteExistingStacks(t, stackPrefix)
	stackSlug := GetRandomStackName(stackPrefix)

	accessPolicyPrefix := "testk6install-"
	testAccDeleteExistingAccessPolicies(t, "eu", accessPolicyPrefix)
	accessPolicyName := accessPolicyPrefix + acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	var installationID string

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccStackCheckDestroy(&stack),
		Steps: []resource.TestStep{
			{
				Config: testAccK6Installation(stackSlug, accessPolicyName, "tfk6installtest_sa_token", "admin"),
				Check: resource.ComposeTestCheckFunc(
					testAccStackCheckExists("grafana_cloud_stack.test", &stack),
					resource.TestCheckResourceAttrSet("grafana_k6_installation.test", "k6_access_token"),
					resource.TestCheckResourceAttrSet("grafana_k6_installation.test", "k6_organization"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["grafana_k6_installation.test"]
						if !ok {
							return fmt.Errorf("grafana_k6_installation.test not found in state")
						}
						installationID = rs.Primary.ID
						return nil
					},
				),
			},
			{
				// grafana_sa_token is not ForceNew: changing it is an in-place
				// update, so the installation is not recreated.
				Config: testAccK6Installation(stackSlug, accessPolicyName, "tfk6installtest_sa_token2", "admin"),
				Check: resource.ComposeTestCheckFunc(
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["grafana_k6_installation.test"]
						if !ok {
							return fmt.Errorf("grafana_k6_installation.test not found in state")
						}
						if rs.Primary.ID != installationID {
							return fmt.Errorf("installation was recreated on grafana_sa_token change: id %q != %q", rs.Primary.ID, installationID)
						}
						return nil
					},
				),
			},
			{
				// grafana_user is still ForceNew, so this replaces the installation.
				// /start returns the existing organization, so the id is unchanged.
				Config: testAccK6Installation(stackSlug, accessPolicyName, "tfk6installtest_sa_token2", "someone-else"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_k6_installation.test", "k6_access_token"),
					resource.TestCheckResourceAttrSet("grafana_k6_installation.test", "k6_organization"),
				),
			},
		},
	})
}

func testAccK6InstallationBase(stackSlug, accessPolicyName string) string {
	return testAccStackConfigBasic(stackSlug, stackSlug, "description") +
		testAccCloudAccessPolicyTokenConfigBasic(accessPolicyName, accessPolicyName, "eu", []string{"stacks:read", "stacks:write", "subscriptions:read", "orgs:read"}, "", false)
}

// testAccK6Installation builds an installation config. saTokenResource selects
// which service account token feeds grafana_sa_token, so tests can change it
// and force an in-place update.
func testAccK6Installation(stackSlug, apiKeyName, saTokenResource, grafanaUser string) string {
	return testAccK6InstallationBase(stackSlug, apiKeyName) +
		`
	resource "grafana_cloud_stack_service_account" "tfk6installtest_sa" {
		stack_slug = grafana_cloud_stack.test.slug
		name        = "tfk6installtest-sa"
		role        = "Admin"
		is_disabled = false
	}

	resource "grafana_cloud_stack_service_account_token" "tfk6installtest_sa_token" {
		stack_slug = grafana_cloud_stack.test.slug
		service_account_id = grafana_cloud_stack_service_account.tfk6installtest_sa.id
		name       = "tfk6installtest-sa-token"
	}

	resource "grafana_cloud_stack_service_account_token" "tfk6installtest_sa_token2" {
		stack_slug = grafana_cloud_stack.test.slug
		service_account_id = grafana_cloud_stack_service_account.tfk6installtest_sa.id
		name       = "tfk6installtest-sa-token2"
	}
	` +
		fmt.Sprintf(`
	resource "grafana_k6_installation" "test" {
		stack_id         = grafana_cloud_stack.test.id
		grafana_sa_token = grafana_cloud_stack_service_account_token.%s.key
		grafana_user     = %q
	}
	`, saTokenResource, grafanaUser)
}
