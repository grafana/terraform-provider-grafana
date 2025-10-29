package cloud_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/resources/cloud"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
)

func TestResourceAccessPolicyRotatingToken_Basic(t *testing.T) {
	t.Parallel()
	testutils.CheckCloudAPITestsEnabled(t)

	oldNow := cloud.Now

	var policy gcom.AuthAccessPolicy
	var policyToken gcom.AuthToken
	var policyTokenAfterRotation gcom.AuthToken

	namePrefix := "test-rotating-token-terraform"
	accessPolicyName := fmt.Sprintf("test-rotating-token-terraform-initial-%s", acctest.RandStringFromCharSet(6, acctest.CharSetAlpha))
	currentStaticTime := time.Now().UTC()

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCloudAccessPolicyCheckDestroy("prod-us-east-0", &policy),
			testAccCloudAccessPolicyTokenCheckDestroy("prod-us-east-0", &policyToken),
			testAccCloudAccessPolicyTokenCheckDestroy("prod-us-east-0", &policyTokenAfterRotation),
		),
		Steps: []resource.TestStep{
			// Test that the cloud access policy and rotating token get created
			{
				PreConfig: setTestTime(currentStaticTime.Format(time.RFC3339)),
				Config:    testAccCloudAccessPolicyRotatingTokenConfigBasic(accessPolicyName, "", "prod-us-east-0", namePrefix, "1h", "10m", true),
				Check: func() resource.TestCheckFunc {
					expectedExpiresAt := currentStaticTime.Add(1 * time.Hour)

					return resource.ComposeTestCheckFunc(
						testAccCloudAccessPolicyCheckExists("grafana_cloud_access_policy.rotating_token_test", &policy),
						testAccCloudAccessPolicyTokenCheckExists("grafana_cloud_access_policy_rotating_token.test", &policyToken),
						// Computed fields
						resource.TestCheckResourceAttr("grafana_cloud_access_policy_rotating_token.test", "name", fmt.Sprintf("%s-%d", namePrefix, expectedExpiresAt.Unix())),
						resource.TestCheckResourceAttr("grafana_cloud_access_policy_rotating_token.test", "expires_at", expectedExpiresAt.Format(time.RFC3339)),
						resource.TestCheckNoResourceAttr("grafana_cloud_access_policy_rotating_token.test", "updated_at"),
						// Input fields
						resource.TestCheckResourceAttr("grafana_cloud_access_policy_rotating_token.test", "name_prefix", namePrefix),
						resource.TestCheckResourceAttr("grafana_cloud_access_policy_rotating_token.test", "expire_after", "1h"),
						resource.TestCheckResourceAttr("grafana_cloud_access_policy_rotating_token.test", "early_rotation_window", "10m"),
						resource.TestCheckResourceAttr("grafana_cloud_access_policy_rotating_token.test", "region", "prod-us-east-0"),
					)
				}(),
			},
			// Test that rotation is not triggered before time by running a plan
			{
				PreConfig:          setTestTime(currentStaticTime.Add(10 * time.Minute).Format(time.RFC3339)),
				Config:             testAccCloudAccessPolicyRotatingTokenConfigBasic(accessPolicyName, "", "prod-us-east-0", namePrefix, "1h", "10m", true),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Test that rotation is triggered if running a plan within the early rotation window
			{
				PreConfig:          setTestTime(currentStaticTime.Add(51 * time.Minute).Format(time.RFC3339)),
				Config:             testAccCloudAccessPolicyRotatingTokenConfigBasic(accessPolicyName, "", "prod-us-east-0", namePrefix, "1h", "10m", true),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			// Test that rotation is triggered if running a plan after the token's expire_after date
			{
				PreConfig:          setTestTime(currentStaticTime.Add(61 * time.Minute).Format(time.RFC3339)),
				Config:             testAccCloudAccessPolicyRotatingTokenConfigBasic(accessPolicyName, "", "prod-us-east-0", namePrefix, "1h", "10m", true),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			// Test that expires_after cannot be a negative duration
			{
				PreConfig:   setTestTime(currentStaticTime.Format(time.RFC3339)),
				Config:      testAccCloudAccessPolicyRotatingTokenConfigBasic(accessPolicyName, "", "prod-us-east-0", namePrefix, "-1h", "10m", true),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile("`expire_after` must be 0 or a positive duration string"),
			},
			// Test that early_rotation_window cannot be a negative duration
			{
				PreConfig:   setTestTime(currentStaticTime.Format(time.RFC3339)),
				Config:      testAccCloudAccessPolicyRotatingTokenConfigBasic(accessPolicyName, "", "prod-us-east-0", namePrefix, "1h", "-10m", true),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile("`early_rotation_window` must be 0 or a positive duration string"),
			},
			// Test that early_rotation_window cannot be bigger than rotate_after
			{
				PreConfig:   setTestTime(currentStaticTime.Format(time.RFC3339)),
				Config:      testAccCloudAccessPolicyRotatingTokenConfigBasic(accessPolicyName, "", "prod-us-east-0", namePrefix, "1h", "1h10m", true),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile("`early_rotation_window` cannot be bigger than `expire_after`"),
			},
			// Test that Terraform-only attributes can be updated without making API calls, by updating early_rotation_window
			{
				PreConfig: setTestTime(currentStaticTime.Format(time.RFC3339)),
				Config:    testAccCloudAccessPolicyRotatingTokenConfigBasic(accessPolicyName, "", "prod-us-east-0", namePrefix, "1h", "15m", true),
				Check: resource.ComposeTestCheckFunc(
					testAccCloudAccessPolicyTokenCheckExists("grafana_cloud_access_policy_rotating_token.test", &policyToken),
					resource.TestCheckResourceAttr("grafana_cloud_access_policy_rotating_token.test", "early_rotation_window", "15m"),
					resource.TestCheckNoResourceAttr("grafana_cloud_access_policy_rotating_token.test", "updated_at"),
				),
			},
			// Test that the token can have its display name updated
			{
				PreConfig: setTestTime(currentStaticTime.Format(time.RFC3339)),
				Config:    testAccCloudAccessPolicyRotatingTokenConfigBasic(accessPolicyName, "updated", "prod-us-east-0", namePrefix, "1h", "15m", true),
				Check: func() resource.TestCheckFunc {
					expectedExpiresAt := currentStaticTime.Add(1 * time.Hour)

					return resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr("grafana_cloud_access_policy_rotating_token.test", "display_name", "updated"),

						testAccCloudAccessPolicyCheckExists("grafana_cloud_access_policy.rotating_token_test", &policy),
						testAccCloudAccessPolicyTokenCheckExists("grafana_cloud_access_policy_rotating_token.test", &policyToken),
						// Computed fields
						resource.TestCheckResourceAttr("grafana_cloud_access_policy_rotating_token.test", "name", fmt.Sprintf("%s-%d", namePrefix, expectedExpiresAt.Unix())),
						resource.TestCheckResourceAttr("grafana_cloud_access_policy_rotating_token.test", "expires_at", expectedExpiresAt.Format(time.RFC3339)),
						resource.TestCheckResourceAttrSet("grafana_cloud_access_policy_rotating_token.test", "updated_at"),
						// Input fields
						resource.TestCheckResourceAttr("grafana_cloud_access_policy_rotating_token.test", "name_prefix", namePrefix),
						resource.TestCheckResourceAttr("grafana_cloud_access_policy_rotating_token.test", "expire_after", "1h"),
						resource.TestCheckResourceAttr("grafana_cloud_access_policy_rotating_token.test", "early_rotation_window", "15m"),
						resource.TestCheckResourceAttr("grafana_cloud_access_policy_rotating_token.test", "region", "prod-us-east-0"),
					)
				}(),
			},
			// Test import
			{
				ResourceName:            "grafana_cloud_access_policy_rotating_token.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token", "name_prefix", "expire_after", "early_rotation_window", "delete_on_destroy"},
			},
			// Test rotation time change should force recreation
			{
				PreConfig: setTestTime(currentStaticTime.Format(time.RFC3339)),
				Config:    testAccCloudAccessPolicyRotatingTokenConfigBasic(accessPolicyName, "updated", "prod-us-east-0", namePrefix, "2h", "15m", true),
				Check: func() resource.TestCheckFunc {
					expectedExpiresAt := currentStaticTime.Add(2 * time.Hour)

					return resource.ComposeTestCheckFunc(
						testAccCloudAccessPolicyCheckExists("grafana_cloud_access_policy.rotating_token_test", &policy),
						testAccCloudAccessPolicyTokenCheckExists("grafana_cloud_access_policy_rotating_token.test", &policyTokenAfterRotation),
						// Computed fields
						resource.TestCheckResourceAttr("grafana_cloud_access_policy_rotating_token.test", "name", fmt.Sprintf("%s-%d", namePrefix, expectedExpiresAt.Unix())),
						resource.TestCheckResourceAttr("grafana_cloud_access_policy_rotating_token.test", "expires_at", expectedExpiresAt.Format(time.RFC3339)),
						resource.TestCheckNoResourceAttr("grafana_cloud_access_policy_rotating_token.test", "updated_at"),
						// Input fields
						resource.TestCheckResourceAttr("grafana_cloud_access_policy_rotating_token.test", "name_prefix", namePrefix),
						resource.TestCheckResourceAttr("grafana_cloud_access_policy_rotating_token.test", "expire_after", "2h"),
						resource.TestCheckResourceAttr("grafana_cloud_access_policy_rotating_token.test", "early_rotation_window", "15m"),
						resource.TestCheckResourceAttr("grafana_cloud_access_policy_rotating_token.test", "region", "prod-us-east-0"),

						func(s *terraform.State) error {
							if policyToken.Name == policyTokenAfterRotation.Name {
								return fmt.Errorf("expected token to be recreated, but Name remained the same: %s", policyToken.Name)
							}
							if policyToken.Id == nil || policyTokenAfterRotation.Id == nil {
								return fmt.Errorf("expected token to be recreated, but ID is nil")
							}
							if *policyToken.Id == *policyTokenAfterRotation.Id {
								return fmt.Errorf("expected token to be recreated, but ID remained the same: %s", *policyToken.Id)
							}
							return nil
						},
					)
				}(),
			},
			// Make sure token exists
			{
				Config: testAccCloudAccessPolicyRotatingTokenConfigBasic(accessPolicyName, "", "prod-us-east-0", "test-no-delete", "10m", "5m", false),
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
						// Check that the token resource is no longer in the state
						_, exists := s.RootModule().Resources["grafana_cloud_access_policy_rotating_token.test"]
						if exists {
							return fmt.Errorf("expected token resource to be removed from state after destroy, but it still exists")
						}

						// Verify that the token still exists in the Grafana Cloud API
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
							return fmt.Errorf("expected token to still exist after destroy, but API call failed: %w", err)
						}
						if token == nil || token.Id == nil || policyToken.Id == nil {
							return fmt.Errorf("expected token to still exist after destroy, but API response is empty")
						}
						if *token.Id != *policyToken.Id {
							return fmt.Errorf("expected token IDs to be the same (%s) (%s)", *token.Id, *policyToken.Id)
						}
						if token.ExpiresAt == nil {
							return fmt.Errorf("expected token to have an expiration date, but it does not have one")
						}
						if token.ExpiresAt.Before(time.Now()) {
							return fmt.Errorf("expected token not to be expired, but it expired on %s", token.ExpiresAt.Format(time.RFC3339))
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
			// Create new token to test deletion
			{
				Config: testAccCloudAccessPolicyRotatingTokenConfigBasic(accessPolicyName, "", "prod-us-east-0", "test-delete", "10m", "5m", true),
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
						// Check that the token resource is no longer in the state
						_, exists := s.RootModule().Resources["grafana_cloud_access_policy_rotating_token.test"]
						if exists {
							return fmt.Errorf("expected token resource to be removed from state after destroy, but it still exists")
						}

						// Verify that the token is removed in the Grafana Cloud API too
						client := testutils.Provider.Meta().(*common.Client).GrafanaCloudAPI
						orgID, err := strconv.ParseInt(*policy.OrgId, 10, 32)
						if err != nil {
							return err
						}
						token, resp, _ := client.TokensAPI.GetToken(context.Background(), *policyToken.Id).
							Region("prod-us-east-0").
							OrgId(int32(orgID)).
							Execute()
						if resp == nil {
							return fmt.Errorf("Expected API response not to be empty")
						}
						if resp.StatusCode != http.StatusNotFound {
							return fmt.Errorf("expected API to return 404 when fetching the deleted token, but instead got (%d)", resp.StatusCode)
						}
						if token != nil {
							return fmt.Errorf("expected token to be deleted after destroy, but the token still exists in Grafana Cloud")
						}

						return nil
					},
				),
			},
		},
	})
	cloud.Now = oldNow
}

func testAccCloudAccessPolicyRotatingTokenConfigBasic(name, displayName, region, namePrefix, expireAfter, earlyRotationWindow string, deleteOnDestroy bool) string {
	if displayName != "" {
		displayName = fmt.Sprintf("display_name = \"%s\"", displayName)
	}
	var deleteStr string
	if deleteOnDestroy {
		deleteStr = `delete_on_destroy = true`
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
		expire_after            = "%[6]s"
		early_rotation_window   = "%[8]s"
		%[2]s
        %[9]s
	}
	`, name, displayName, strings.Join(scopes, `","`), os.Getenv("GRAFANA_CLOUD_ORG"), namePrefix, expireAfter, region, earlyRotationWindow, deleteStr)
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

func setTestTime(t string) func() {
	return func() {
		cloud.Now = func() time.Time {
			parsedT, _ := time.Parse(time.RFC3339, t)
			return parsedT
		}
	}
}
