package grafana

import (
	"fmt"
	"os"
	"strings"
	"time"

	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// This test covers both the cloud_access_policy and cloud_access_policy_token resources.
func TestResourceCloudAccessPolicyToken_Basic(t *testing.T) {
	t.Parallel()
	CheckCloudAPITestsEnabled(t)

	var policy gapi.CloudAccessPolicy
	var policyToken gapi.CloudAccessPolicyToken

	expiresAt := time.Now().Add(time.Hour * 24).UTC().Format(time.RFC3339)
	initialScopes := []string{
		"metrics:read",
		"logs:write",
		"accesspolicies:read",
		"accesspolicies:write",
		"accesspolicies:delete",
	}
	updatedScopes := []string{
		"metrics:write",
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCloudAccessPolicyCheckDestroy("us", &policy),
			testAccCloudAccessPolicyTokenCheckDestroy("us", &policyToken),
		),
		Steps: []resource.TestStep{
			{
				Config: testAccCloudAccessPolicyTokenConfigBasic("initial", "", initialScopes, expiresAt),
				Check: resource.ComposeTestCheckFunc(
					testAccCloudAccessPolicyCheckExists("grafana_cloud_access_policy.test", &policy),
					testAccCloudAccessPolicyTokenCheckExists("grafana_cloud_access_policy_token.test", &policyToken),

					resource.TestCheckResourceAttr("grafana_cloud_access_policy.test", "name", "initial"),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy.test", "display_name", "initial"),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy.test", "scopes.#", "5"),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy.test", "scopes.0", "accesspolicies:delete"),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy.test", "scopes.1", "accesspolicies:read"),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy.test", "scopes.2", "accesspolicies:write"),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy.test", "scopes.3", "logs:write"),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy.test", "scopes.4", "metrics:read"),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy.test", "realm.#", "1"),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy.test", "realm.0.type", "org"),

					resource.TestCheckResourceAttr("grafana_cloud_access_policy_token.test", "name", "token-initial"),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy_token.test", "display_name", "token-initial"),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy_token.test", "expires_at", expiresAt),
				),
			},
			{
				Config: testAccCloudAccessPolicyTokenConfigBasic("initial", "updated", updatedScopes, expiresAt),
				Check: resource.ComposeTestCheckFunc(
					testAccCloudAccessPolicyCheckExists("grafana_cloud_access_policy.test", &policy),
					testAccCloudAccessPolicyTokenCheckExists("grafana_cloud_access_policy_token.test", &policyToken),

					resource.TestCheckResourceAttr("grafana_cloud_access_policy.test", "name", "initial"),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy.test", "display_name", "updated"),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy.test", "scopes.#", "1"),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy.test", "scopes.0", "metrics:write"),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy.test", "realm.#", "1"),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy.test", "realm.0.type", "org"),

					resource.TestCheckResourceAttr("grafana_cloud_access_policy_token.test", "name", "token-initial"),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy_token.test", "display_name", "updated"),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy_token.test", "expires_at", expiresAt),
				),
			},
			// Recreate
			{
				Config: testAccCloudAccessPolicyTokenConfigBasic("updated", "updated", updatedScopes, expiresAt),
				Check: resource.ComposeTestCheckFunc(
					testAccCloudAccessPolicyCheckExists("grafana_cloud_access_policy.test", &policy),
					testAccCloudAccessPolicyTokenCheckExists("grafana_cloud_access_policy_token.test", &policyToken),

					resource.TestCheckResourceAttr("grafana_cloud_access_policy.test", "name", "updated"),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy.test", "display_name", "updated"),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy.test", "scopes.#", "1"),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy.test", "scopes.0", "metrics:write"),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy.test", "realm.#", "1"),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy.test", "realm.0.type", "org"),
				),
			},
			{
				ResourceName:      "grafana_cloud_access_policy.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:            "grafana_cloud_access_policy_token.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token"},
			},
		},
	})
}

func TestResourceCloudAccessPolicyToken_NoExpiration(t *testing.T) {
	t.Parallel()
	CheckCloudAPITestsEnabled(t)

	var policy gapi.CloudAccessPolicy
	var policyToken gapi.CloudAccessPolicyToken

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCloudAccessPolicyTokenConfigBasic("initial-no-expiration", "", []string{"metrics:read"}, ""),
				Check: resource.ComposeTestCheckFunc(
					testAccCloudAccessPolicyCheckExists("grafana_cloud_access_policy.test", &policy),
					testAccCloudAccessPolicyTokenCheckExists("grafana_cloud_access_policy_token.test", &policyToken),
					resource.TestCheckNoResourceAttr("grafana_cloud_access_policy_token.test", "expires_at"),
				),
			},
			{
				ResourceName:            "grafana_cloud_access_policy_token.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token"},
			},
		},
	})
}

func testAccCloudAccessPolicyCheckExists(rn string, a *gapi.CloudAccessPolicy) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		region, id, _ := strings.Cut(rs.Primary.ID, "/")

		client := testAccProvider.Meta().(*client).gcloudapi
		policy, err := client.CloudAccessPolicyByID(region, id)
		if err != nil {
			return fmt.Errorf("error getting cloud access policy: %s", err)
		}

		*a = policy

		return nil
	}
}

func testAccCloudAccessPolicyTokenCheckExists(rn string, a *gapi.CloudAccessPolicyToken) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		region, id, _ := strings.Cut(rs.Primary.ID, "/")

		client := testAccProvider.Meta().(*client).gcloudapi
		token, err := client.CloudAccessPolicyTokenByID(region, id)
		if err != nil {
			return fmt.Errorf("error getting cloud access policy token: %s", err)
		}

		*a = token

		return nil
	}
}

func testAccCloudAccessPolicyCheckDestroy(region string, a *gapi.CloudAccessPolicy) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*client).gcloudapi
		policy, err := client.CloudAccessPolicyByID(region, a.ID)
		if err == nil && policy.Name != "" {
			return fmt.Errorf("cloud access policy `%s` with ID `%s` still exists after destroy", policy.Name, policy.ID)
		}

		return nil
	}
}

func testAccCloudAccessPolicyTokenCheckDestroy(region string, a *gapi.CloudAccessPolicyToken) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*client).gcloudapi
		token, err := client.CloudAccessPolicyTokenByID(region, a.ID)
		if err == nil && token.Name != "" {
			return fmt.Errorf("cloud access policy token `%s` with ID `%s` still exists after destroy", token.Name, token.ID)
		}

		return nil
	}
}

func testAccCloudAccessPolicyTokenConfigBasic(name, displayName string, scopes []string, expiresAt string) string {
	if displayName != "" {
		displayName = fmt.Sprintf("display_name = \"%s\"", displayName)
	}

	if expiresAt != "" {
		expiresAt = fmt.Sprintf("expires_at = \"%s\"", expiresAt)
	}

	return fmt.Sprintf(`
	data "grafana_cloud_organization" "current" {
		slug = "%[4]s"
	}

	resource "grafana_cloud_access_policy" "test" {
		region       = "us"
		name         = "%[1]s"
		%[2]s

		scopes = ["%[3]s"]

		realm {
			type       = "org"
			identifier = data.grafana_cloud_organization.current.id

			label_policy {
				selector = "{namespace=\"default\"}"
			}
		}
	}

	resource "grafana_cloud_access_policy_token" "test" {
		region           = "us"
		access_policy_id = grafana_cloud_access_policy.test.policy_id
		name             = "token-%[1]s"
		%[2]s
		%[5]s
	}
	`, name, displayName, strings.Join(scopes, `","`), os.Getenv("GRAFANA_CLOUD_ORG"), expiresAt)
}
