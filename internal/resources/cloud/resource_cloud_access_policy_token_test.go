package cloud_test

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"testing"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/resources/cloud"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// This test covers both the cloud_access_policy and cloud_access_policy_token resources.
func TestResourceAccessPolicyToken_Basic(t *testing.T) {
	t.Parallel()
	testutils.CheckCloudAPITestsEnabled(t)

	var policy gcom.AuthAccessPolicy
	var policyToken gcom.AuthToken

	expiresAt := time.Now().Add(time.Hour * 24).UTC().Format(time.RFC3339)
	initialScopes := []string{
		"metrics:read",
		"logs:write",
		"accesspolicies:read",
		"accesspolicies:write",
		"accesspolicies:delete",
		"datadog:validate",
	}
	updatedScopes := []string{
		"metrics:write",
	}

	randomName := acctest.RandStringFromCharSet(6, acctest.CharSetAlpha)
	initialName := fmt.Sprintf("initial-%s", randomName)
	initialToken := fmt.Sprintf("token-%s", initialName)
	updatedName := fmt.Sprintf("updated-%s", randomName)

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCloudAccessPolicyCheckDestroy("prod-us-east-0", &policy),
			testAccCloudAccessPolicyTokenCheckDestroy("prod-us-east-0", &policyToken),
		),
		Steps: []resource.TestStep{
			{
				Config: testAccCloudAccessPolicyTokenConfigBasic(initialName, "", "prod-us-east-0", initialScopes, expiresAt),
				Check: resource.ComposeTestCheckFunc(
					testAccCloudAccessPolicyCheckExists("grafana_cloud_access_policy.test", &policy),
					testAccCloudAccessPolicyTokenCheckExists("grafana_cloud_access_policy_token.test", &policyToken),

					resource.TestCheckResourceAttr("grafana_cloud_access_policy.test", "name", initialName),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy.test", "display_name", initialName),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy.test", "scopes.#", "6"),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy.test", "scopes.0", "accesspolicies:delete"),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy.test", "scopes.1", "accesspolicies:read"),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy.test", "scopes.2", "accesspolicies:write"),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy.test", "scopes.3", "datadog:validate"),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy.test", "scopes.4", "logs:write"),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy.test", "scopes.5", "metrics:read"),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy.test", "realm.#", "1"),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy.test", "realm.0.type", "org"),

					resource.TestCheckResourceAttr("grafana_cloud_access_policy_token.test", "name", initialToken),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy_token.test", "display_name", initialToken),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy_token.test", "expires_at", expiresAt),
				),
			},
			{
				Config: testAccCloudAccessPolicyTokenConfigBasic(initialName, "", "prod-us-east-0", initialScopes, expiresAt),
				PreConfig: func() {
					orgID, err := strconv.ParseInt(*policy.OrgId, 10, 32)
					if err != nil {
						t.Fatal(err)
					}
					client := testutils.Provider.Meta().(*common.Client).GrafanaCloudAPI
					_, _, err = client.TokensAPI.DeleteToken(context.Background(), *policyToken.Id).
						Region("prod-us-east-0").
						OrgId(int32(orgID)).
						XRequestId("deleting-token").Execute()
					if err != nil {
						t.Fatalf("error getting cloud access policy: %s", err)
					}
				},
			},
			{
				Config: testAccCloudAccessPolicyTokenConfigBasic(initialName, "updated", "prod-us-east-0", updatedScopes, expiresAt),
				Check: resource.ComposeTestCheckFunc(
					testAccCloudAccessPolicyCheckExists("grafana_cloud_access_policy.test", &policy),
					testAccCloudAccessPolicyTokenCheckExists("grafana_cloud_access_policy_token.test", &policyToken),

					resource.TestCheckResourceAttr("grafana_cloud_access_policy.test", "name", initialName),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy.test", "display_name", "updated"),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy.test", "scopes.#", "1"),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy.test", "scopes.0", "metrics:write"),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy.test", "realm.#", "1"),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy.test", "realm.0.type", "org"),

					resource.TestCheckResourceAttr("grafana_cloud_access_policy_token.test", "name", initialToken),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy_token.test", "display_name", "updated"),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy_token.test", "expires_at", expiresAt),
				),
			},
			// Recreate
			{
				Config: testAccCloudAccessPolicyTokenConfigBasic(updatedName, "updated", "prod-us-east-0", updatedScopes, expiresAt),
				Check: resource.ComposeTestCheckFunc(
					testAccCloudAccessPolicyCheckExists("grafana_cloud_access_policy.test", &policy),
					testAccCloudAccessPolicyTokenCheckExists("grafana_cloud_access_policy_token.test", &policyToken),

					resource.TestCheckResourceAttr("grafana_cloud_access_policy.test", "name", updatedName),
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

func TestResourceAccessPolicyToken_NoExpiration(t *testing.T) {
	t.Parallel()
	testutils.CheckCloudAPITestsEnabled(t)

	var policy gcom.AuthAccessPolicy
	var policyToken gcom.AuthToken

	randomName := fmt.Sprintf("initial-no-expiration-%s", acctest.RandStringFromCharSet(6, acctest.CharSetAlpha))
	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCloudAccessPolicyTokenConfigBasic(randomName, "", "prod-us-east-0", []string{"metrics:read"}, ""),
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

func testAccCloudAccessPolicyCheckExists(rn string, a *gcom.AuthAccessPolicy) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		region, id, _ := strings.Cut(rs.Primary.ID, ":")

		client := testutils.Provider.Meta().(*common.Client).GrafanaCloudAPI
		policy, _, err := client.AccesspoliciesAPI.GetAccessPolicy(context.Background(), id).Region(region).Execute()
		if err != nil {
			return fmt.Errorf("error getting cloud access policy: %s", err)
		}

		*a = *policy

		return nil
	}
}

func testAccCloudAccessPolicyTokenCheckExists(rn string, a *gcom.AuthToken) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		region, id, _ := strings.Cut(rs.Primary.ID, ":")

		client := testutils.Provider.Meta().(*common.Client).GrafanaCloudAPI
		token, _, err := client.TokensAPI.GetToken(context.Background(), id).Region(region).Execute()
		if err != nil {
			return fmt.Errorf("error getting cloud access policy token: %s", err)
		}

		*a = *token

		return nil
	}
}

func testAccCloudAccessPolicyCheckDestroy(region string, a *gcom.AuthAccessPolicy) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if a == nil {
			return nil
		}
		client := testutils.Provider.Meta().(*common.Client).GrafanaCloudAPI
		policy, _, err := client.AccesspoliciesAPI.GetAccessPolicy(context.Background(), *a.Id).Region(region).Execute()
		if err == nil && policy.Name != "" {
			return fmt.Errorf("cloud access policy `%s` with ID `%s` still exists after destroy", policy.Name, *policy.Id)
		}

		return nil
	}
}

func testAccCloudAccessPolicyTokenCheckDestroy(region string, a *gcom.AuthToken) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if a == nil {
			return nil
		}
		client := testutils.Provider.Meta().(*common.Client).GrafanaCloudAPI
		token, _, err := client.TokensAPI.GetToken(context.Background(), *a.Id).Region(region).Execute()
		if err == nil && token.Name != "" {
			return fmt.Errorf("cloud access policy token `%s` with ID `%s` still exists after destroy", token.Name, *token.Id)
		}

		return nil
	}
}

func testAccDeleteExistingAccessPolicies(t *testing.T, region, prefix string) {
	client := testutils.Provider.Meta().(*common.Client).GrafanaCloudAPI
	resp, _, err := client.AccesspoliciesAPI.GetAccessPolicies(context.Background()).Region(region).Execute()
	if err != nil {
		t.Error(err)
	}

	for _, ap := range resp.Items {
		if strings.HasPrefix(ap.Name, prefix) {
			_, _, err := client.AccesspoliciesAPI.DeleteAccessPolicy(context.Background(), *ap.Id).XRequestId(cloud.ClientRequestID()).Region(region).Execute()
			if err != nil {
				t.Error(err)
			}
		}
	}
}

func testAccCloudAccessPolicyTokenConfigBasic(name, displayName, region string, scopes []string, expiresAt string) string {
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
		region       = "%[6]s"
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
		region           = "%[6]s"
		access_policy_id = grafana_cloud_access_policy.test.policy_id
		name             = "token-%[1]s"
		%[2]s
		%[5]s
	}
	`, name, displayName, strings.Join(scopes, `","`), os.Getenv("GRAFANA_CLOUD_ORG"), expiresAt, region)
}
