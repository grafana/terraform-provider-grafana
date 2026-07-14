package cloud_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

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
	testAccDeleteExistingAccessPolicies(t, "us", accessPolicyPrefix)
	accessPolicyName := accessPolicyPrefix + acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccStackCheckDestroy(&stack),
		Steps: []resource.TestStep{
			{
				Config: testAccK6Installation(stackSlug, accessPolicyName),
				Check: resource.ComposeTestCheckFunc(
					testAccStackCheckExists("grafana_cloud_stack.test", &stack),
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

func testAccK6Installation(stackSlug, apiKeyName string) string {
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
		`
	resource "grafana_k6_installation" "test" {
		cloud_access_policy_token = grafana_cloud_access_policy_token.test.token
		stack_id                  = grafana_cloud_stack.test.id
		grafana_sa_token          = grafana_cloud_stack_service_account_token.tfk6installtest_sa_token.key
		grafana_user              = "admin"
	}
	`
}
