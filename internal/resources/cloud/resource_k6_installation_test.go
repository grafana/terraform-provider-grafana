package cloud_test

import (
	"fmt"
	"regexp"
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
				// The publisher token is required for new installations.
				Config:      testAccK6Installation(stackSlug, accessPolicyName, ""),
				ExpectError: regexp.MustCompile("publisher_token is required when creating a new k6 installation"),
			},
			{
				Config: testAccK6Installation(stackSlug, accessPolicyName, "publisher"),
				Check: resource.ComposeTestCheckFunc(
					testAccStackCheckExists("grafana_cloud_stack.test", &stack),
					resource.TestCheckResourceAttrSet("grafana_k6_installation.test", "k6_access_token"),
					resource.TestCheckResourceAttrSet("grafana_k6_installation.test", "k6_organization"),
					resource.TestCheckResourceAttrSet("grafana_k6_installation.test", "publisher_token"),
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
				// Rotating the publisher token is an in-place update, not a recreation.
				Config: testAccK6Installation(stackSlug, accessPolicyName, "publisher2"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_k6_installation.test", "publisher_token"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["grafana_k6_installation.test"]
						if !ok {
							return fmt.Errorf("grafana_k6_installation.test not found in state")
						}
						if rs.Primary.ID != installationID {
							return fmt.Errorf("installation was recreated on publisher_token change: id %q != %q", rs.Primary.ID, installationID)
						}
						return nil
					},
				),
			},
		},
	})
}

func testAccK6InstallationBase(stackSlug, accessPolicyName string) string {
	return testAccStackConfigBasic(stackSlug, stackSlug, "description") +
		testAccCloudAccessPolicyTokenConfigBasic(accessPolicyName, accessPolicyName, "eu", []string{"stacks:read", "stacks:write", "subscriptions:read", "orgs:read"}, "", false) +
		testAccK6InstallationPublisherPolicy(accessPolicyName)
}

func testAccK6InstallationPublisherPolicy(accessPolicyName string) string {
	return fmt.Sprintf(`
	resource "grafana_cloud_access_policy" "publisher" {
		region = "eu"
		name   = "%[1]s-publisher"
		scopes = ["metrics:read", "metrics:write", "rules:read", "rules:write"]

		realm {
			type       = "stack"
			identifier = grafana_cloud_stack.test.id
		}
	}

	resource "grafana_cloud_access_policy_token" "publisher" {
		region           = "eu"
		access_policy_id = grafana_cloud_access_policy.publisher.policy_id
		name             = "%[1]s-publisher"
	}

	resource "grafana_cloud_access_policy_token" "publisher2" {
		region           = "eu"
		access_policy_id = grafana_cloud_access_policy.publisher.policy_id
		name             = "%[1]s-publisher2"
	}
	`, accessPolicyName)
}

func testAccK6Installation(stackSlug, apiKeyName, publisherTokenResource string) string {
	publisherToken := ""
	if publisherTokenResource != "" {
		publisherToken = fmt.Sprintf("publisher_token  = grafana_cloud_access_policy_token.%s.token", publisherTokenResource)
	}

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
	` +
		fmt.Sprintf(`
	resource "grafana_k6_installation" "test" {
		stack_id         = grafana_cloud_stack.test.id
		grafana_sa_token = grafana_cloud_stack_service_account_token.tfk6installtest_sa_token.key
		grafana_user     = "admin"
		%s
	}
	`, publisherToken)
}
