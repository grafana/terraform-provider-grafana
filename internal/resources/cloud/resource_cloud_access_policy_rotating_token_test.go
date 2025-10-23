package cloud_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
)

func TestResourceAccessPolicyRotatingToken_Basic(t *testing.T) {
	t.Parallel()
	testutils.CheckCloudAPITestsEnabled(t)

	var policy gcom.AuthAccessPolicy
	var policyToken gcom.AuthToken
	var policyTokenAfterRecreation gcom.AuthToken

	rotateAfter := time.Now().Add(time.Hour * 2).UTC()
	updatedRotateAfter := rotateAfter.Add(time.Hour * 4)
	postRotationLifetime := "24h"
	expectedExpiresAt := rotateAfter.Add(24 * time.Hour).Format(time.RFC3339)
	namePrefix := "test-rotating-token-terraform"
	expectedName := fmt.Sprintf("%s-%d-%s", namePrefix, rotateAfter.Unix(), postRotationLifetime)
	accessPolicyName := fmt.Sprintf("test-rotating-token-terraform-initial-%s", acctest.RandStringFromCharSet(6, acctest.CharSetAlpha))

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCloudAccessPolicyCheckDestroy("prod-us-east-0", &policy),
			testAccCloudAccessPolicyTokenCheckDestroy("prod-us-east-0", &policyToken),
			testAccCloudAccessPolicyTokenCheckDestroy("prod-us-east-0", &policyTokenAfterRecreation),
		),
		Steps: []resource.TestStep{
			// Test that the cloud access policy and rotating token get created
			{
				Config: testAccCloudAccessPolicyRotatingTokenConfigBasic(accessPolicyName, "", "prod-us-east-0", namePrefix, rotateAfter.Unix(), postRotationLifetime),
				Check: resource.ComposeTestCheckFunc(
					testAccCloudAccessPolicyCheckExists("grafana_cloud_access_policy.rotating_token_test", &policy),
					testAccCloudAccessPolicyTokenCheckExists("grafana_cloud_access_policy_rotating_token.test", &policyToken),

					// Computed fields
					resource.TestCheckResourceAttr("grafana_cloud_access_policy_rotating_token.test", "name", expectedName),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy_rotating_token.test", "expires_at", expectedExpiresAt),
					// Input fields
					resource.TestCheckResourceAttr("grafana_cloud_access_policy_rotating_token.test", "name_prefix", namePrefix),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy_rotating_token.test", "rotate_after", strconv.FormatInt(rotateAfter.Unix(), 10)),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy_rotating_token.test", "post_rotation_lifetime", postRotationLifetime),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy_rotating_token.test", "region", "prod-us-east-0"),
				),
			},
			// Test that rotation is not triggered before time by running a plan
			{
				Config:             testAccCloudAccessPolicyRotatingTokenConfigBasic(accessPolicyName, "", "prod-us-east-0", namePrefix, rotateAfter.Unix(), postRotationLifetime),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Test that the token can have its display name updated
			{
				Config: testAccCloudAccessPolicyRotatingTokenConfigBasic(accessPolicyName, "updated", "prod-us-east-0", namePrefix, rotateAfter.Unix(), postRotationLifetime),
				Check: resource.ComposeTestCheckFunc(
					testAccCloudAccessPolicyCheckExists("grafana_cloud_access_policy.rotating_token_test", &policy),
					testAccCloudAccessPolicyTokenCheckExists("grafana_cloud_access_policy_rotating_token.test", &policyToken),

					resource.TestCheckResourceAttr("grafana_cloud_access_policy_rotating_token.test", "display_name", "updated"),

					// Computed fields
					resource.TestCheckResourceAttr("grafana_cloud_access_policy_rotating_token.test", "name", expectedName),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy_rotating_token.test", "expires_at", expectedExpiresAt),
					// Input fields
					resource.TestCheckResourceAttr("grafana_cloud_access_policy_rotating_token.test", "name_prefix", namePrefix),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy_rotating_token.test", "rotate_after", strconv.FormatInt(rotateAfter.Unix(), 10)),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy_rotating_token.test", "post_rotation_lifetime", postRotationLifetime),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy_rotating_token.test", "region", "prod-us-east-0"),
				),
			},
			// Test import
			{
				ResourceName:            "grafana_cloud_access_policy_rotating_token.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token"},
			},
			// Test rotation time change should force recreation
			{
				Config: testAccCloudAccessPolicyRotatingTokenConfigBasic(accessPolicyName, "", "prod-us-east-0", namePrefix, updatedRotateAfter.Unix(), postRotationLifetime),
				Check: resource.ComposeTestCheckFunc(
					testAccCloudAccessPolicyCheckExists("grafana_cloud_access_policy.rotating_token_test", &policy),
					testAccCloudAccessPolicyTokenCheckExists("grafana_cloud_access_policy_rotating_token.test", &policyTokenAfterRecreation),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy_rotating_token.test", "rotate_after", strconv.Itoa(int(updatedRotateAfter.Unix()))),
					func(s *terraform.State) error {
						if policyToken.Id == nil || policyTokenAfterRecreation.Id == nil {
							return fmt.Errorf("expected token to be recreated, but ID is nil")
						}
						if *policyToken.Id == *policyTokenAfterRecreation.Id {
							return fmt.Errorf("expected token to be recreated, but ID remained the same: %s", *policyToken.Id)
						}
						return nil
					},
				),
			},
			// Make sure token exists
			{
				Config: testAccCloudAccessPolicyRotatingTokenConfigBasic(accessPolicyName, "", "prod-us-east-0", "test-no-delete", rotateAfter.Unix(), postRotationLifetime),
				Check: resource.ComposeTestCheckFunc(
					testAccCloudAccessPolicyCheckExists("grafana_cloud_access_policy.rotating_token_test", &policy),
					testAccCloudAccessPolicyTokenCheckExists("grafana_cloud_access_policy_rotating_token.test", &policyToken),
				),
			},
			// Test that destroy does not actually delete the token (it should only show a warning instead)
			{
				Config: testAccCloudAccessPolicyRotatingTokenConfigNoToken(accessPolicyName, "", "prod-us-east-0"),
				Check: resource.ComposeTestCheckFunc(
					testAccCloudAccessPolicyCheckExists("grafana_cloud_access_policy.rotating_token_test", &policy),
					func(s *terraform.State) error {
						// Verify token still exists in Grafana Cloud API
						client := testutils.Provider.Meta().(*common.Client).GrafanaCloudAPI
						orgID, err := strconv.ParseInt(*policy.OrgId, 10, 32)
						if err != nil {
							return err
						}
						token, _, err := client.TokensAPI.GetToken(context.Background(), *policyToken.Id).
							Region("prod-us-east-0").
							OrgId(int32(orgID)).
							Execute()
						if err != nil {
							return fmt.Errorf("expected token to still exist after destroy, but API call failed: %s", err)
						}
						if token == nil || token.Id == nil || policyToken.Id == nil {
							return errors.New("expected token to still exist after destroy, but API response is empty")
						}
						if *token.Id != *policyToken.Id {
							return fmt.Errorf("expected token IDs to be the same (%s) (%s)", *token.Id, *policyToken.Id)
						}

						// Check that the token resource is no longer in state
						_, exists := s.RootModule().Resources["grafana_cloud_access_policy_rotating_token.test"]
						if exists {
							return fmt.Errorf("expected token resource to be removed from state after destroy, but it still exists")
						}

						return nil
					},
				),
			},
			// Test that the token exists and can be manually deleted through the API
			{
				Config: testAccCloudAccessPolicyRotatingTokenConfigNoToken(accessPolicyName, "", "prod-us-east-0"),
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
		},
	})
}

func testAccCloudAccessPolicyRotatingTokenConfigBasic(name, displayName, region string, namePrefix string, rotateAfter int64, postRotationLifetime string) string {
	if displayName != "" {
		displayName = fmt.Sprintf("display_name = \"%s\"", displayName)
	}

	scopes := []string{
		"metrics:read",
		"logs:write",
		"accesspolicies:read",
		"accesspolicies:write",
		"accesspolicies:delete",
		"datadog:validate",
	}

	return fmt.Sprintf(`
	data "grafana_cloud_organization" "current" {
		slug = "%[4]s"
	}

	resource "grafana_cloud_access_policy" "rotating_token_test" {
		region       = "%[7]s"
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

	resource "grafana_cloud_access_policy_rotating_token" "test" {
		region                  = "%[7]s"
		access_policy_id        = grafana_cloud_access_policy.rotating_token_test.policy_id
		name_prefix             = "%[5]s"
		rotate_after            = "%[6]d"
		post_rotation_lifetime  = "%[8]s"
		%[2]s
	}
	`, name, displayName, strings.Join(scopes, `","`), os.Getenv("GRAFANA_CLOUD_ORG"), namePrefix, rotateAfter, region, postRotationLifetime)
}

func testAccCloudAccessPolicyRotatingTokenConfigNoToken(name, displayName, region string) string {
	if displayName != "" {
		displayName = fmt.Sprintf("display_name = \"%s\"", displayName)
	}

	scopes := []string{
		"metrics:read",
		"logs:write",
		"accesspolicies:read",
		"accesspolicies:write",
		"accesspolicies:delete",
		"datadog:validate",
	}

	return fmt.Sprintf(`
	data "grafana_cloud_organization" "current" {
		slug = "%[4]s"
	}

	resource "grafana_cloud_access_policy" "rotating_token_test" {
		region       = "%[5]s"
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
	`, name, displayName, strings.Join(scopes, `","`), os.Getenv("GRAFANA_CLOUD_ORG"), region)
}
