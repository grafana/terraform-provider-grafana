package cloud_test

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/grafana/terraform-provider-grafana/v4/internal/resources/cloud"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
)

func TestAccGrafanaServiceAccountRotatingTokenFromCloud(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	var stack gcom.FormattedApiInstance
	var sa models.ServiceAccountDTO
	var token models.TokenDTO
	prefix := "tf-sa-rotating-token-test"
	slug := GetRandomStackName(prefix)

	oldNow := cloud.Now
	currentStaticTime := time.Now().UTC()

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccDeleteExistingStacks(t, prefix)
		},
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccStackCheckDestroy(&stack),
		Steps: []resource.TestStep{
			{
				PreConfig: setTestTime(currentStaticTime.Format(time.RFC3339)),
				Config:    testAccGrafanaServiceAccountRotatingTokenFromCloud(slug, slug, slug, 120, 60, false),
				Check: resource.ComposeTestCheckFunc(
					testAccStackCheckExists("grafana_cloud_stack.test", &stack),
					testAccGrafanaAuthCheckServiceAccounts(slug, slug+"-sa", &sa),
					testAccGrafanaAuthCheckServiceAccountToken(slug, &sa, computedName(slug, currentStaticTime.Add(120*time.Second)), &token),
					resource.TestCheckResourceAttr("grafana_cloud_stack_service_account.management", "name", fmt.Sprintf("%s-sa", slug)),
					resource.TestCheckResourceAttr("grafana_cloud_stack_service_account_rotating_token.management_token", "name", computedName(slug, currentStaticTime.Add(120*time.Second))),
					resource.TestCheckResourceAttr("grafana_cloud_stack_service_account_rotating_token.management_token", "seconds_to_live", "120"),
					resource.TestCheckResourceAttr("grafana_cloud_stack_service_account_rotating_token.management_token", "early_rotation_window_seconds", "60"),
					resource.TestCheckResourceAttrSet("grafana_cloud_stack_service_account_rotating_token.management_token", "expiration"),
					resource.TestCheckResourceAttrSet("grafana_cloud_stack_service_account_rotating_token.management_token", "key"),
				),
			},
			// Test that rotation is not triggered before time by running a plan
			{
				PreConfig: func() {
					setTestTime(time.Time(token.Expiration).Add(-61 * time.Second).Format(time.RFC3339))()
				},
				Config:             testAccGrafanaServiceAccountRotatingTokenFromCloud(slug, slug, slug, 120, 60, false),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Test that rotation is triggered if running a plan within the early rotation window
			{
				PreConfig: func() {
					setTestTime(time.Time(token.Expiration).Add(-59 * time.Second).Format(time.RFC3339))()
				},
				Config:             testAccGrafanaServiceAccountRotatingTokenFromCloud(slug, slug, slug, 120, 60, false),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			// Test that rotation is triggered if running a plan after the token's expiration date
			{
				PreConfig: func() {
					setTestTime(time.Time(token.Expiration).Add(1 * time.Second).Format(time.RFC3339))()
				},
				Config:             testAccGrafanaServiceAccountRotatingTokenFromCloud(slug, slug, slug, 120, 60, false),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			// Test that early_rotation_window cannot be greater than rotate_after
			{
				PreConfig:   setTestTime(currentStaticTime.Format(time.RFC3339)),
				Config:      testAccGrafanaServiceAccountRotatingTokenFromCloud(slug, slug, slug, 1, 22, false),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile("`early_rotation_window_seconds` cannot be greater than `seconds_to_live`"),
			},
			// Test that deletion only deletes the token from the TF state leaves it to expire in Grafana
			{
				Config: testutils.WithoutResource(t, testAccGrafanaServiceAccountRotatingTokenFromCloud(slug, slug, slug, 120, 60, false), "grafana_cloud_stack_service_account_rotating_token.management_token"),
				Check: resource.ComposeTestCheckFunc(
					testAccGrafanaAuthCheckServiceAccounts(slug, slug+"-sa", &sa),
					func(s *terraform.State) error {
						// Check that the token resource is no longer in the state
						_, exists := s.RootModule().Resources["grafana_cloud_stack_service_account_rotating_token.management_token"]
						if exists {
							return fmt.Errorf("expected token resource to be removed from state after destroy, but it still exists")
						}

						// Verify that the token still exists in Grafana
						var tokenFromGrafana models.TokenDTO
						err := testAccGrafanaAuthCheckServiceAccountToken(slug, &sa, computedName(slug, currentStaticTime.Add(120*time.Second)), &tokenFromGrafana)(s)
						if err != nil {
							return err
						}
						if tokenFromGrafana.ID == 0 || token.ID == 0 {
							return fmt.Errorf("expected token to still exist after destroy, but API response is empty")
						}
						if token.ID != tokenFromGrafana.ID {
							return fmt.Errorf("expected token IDs to be the same (%d) (%d)", token.ID, token.ID)
						}
						if token.Expiration.IsZero() {
							return fmt.Errorf("expected token to have an expiration date, but it does not have one")
						}
						if token.HasExpired {
							return fmt.Errorf("expected token not to be expired, but it expired on %s", token.Expiration)
						}

						return nil
					},
				),
			},
			// Re-create token with delete_on_destroy = true
			{
				PreConfig: setTestTime(currentStaticTime.Format(time.RFC3339)),
				Config:    testAccGrafanaServiceAccountRotatingTokenFromCloud(slug+"-recreated", slug+"-recreated", slug+"-recreated", 120, 60, true),
				Check: resource.ComposeTestCheckFunc(
					testAccGrafanaAuthCheckServiceAccounts(slug, slug+"-recreated"+"-sa", &sa),
					testAccGrafanaAuthCheckServiceAccountToken(slug, &sa, computedName(slug+"-recreated", currentStaticTime.Add(120*time.Second)), &token),
				),
			},
			// Delete and check that it has been deleted from both Terraform and Grafana
			{
				Config: testutils.WithoutResource(t, testAccGrafanaServiceAccountRotatingTokenFromCloud(slug, slug, slug, 120, 60, true), "grafana_cloud_stack_service_account_rotating_token.management_token"),
				Check: func(s *terraform.State) error {
					// Check that the token resource is no longer in the state
					_, exists := s.RootModule().Resources["grafana_cloud_stack_service_account_rotating_token.management_token"]
					if exists {
						return fmt.Errorf("expected token resource to be removed from state after destroy, but it still exists")
					}

					// Verify that the token does not exist in Grafana
					var tokenFromGrafana models.TokenDTO
					err := testAccGrafanaAuthCheckServiceAccountToken(slug, &sa, computedName(slug, currentStaticTime.Add(120*time.Second)), &tokenFromGrafana)(s)
					if err == nil {
						return fmt.Errorf("expected token to have been deleted in Grafana, but it still exists")
					}

					return nil
				},
			},
		},
	})
	cloud.Now = oldNow
}

func testAccGrafanaServiceAccountRotatingTokenFromCloud(name, slug, namePrefix string, secondsToLive, earlyRotationWindowSeconds int, deleteOnDestroy bool) string {
	var deleteOnDestroyStr string
	if deleteOnDestroy {
		deleteOnDestroyStr = `delete_on_destroy = true`
	}
	return testAccStackConfigBasic(name, slug, "description") +
		fmt.Sprintf(`
	resource "grafana_cloud_stack_service_account" "management" {
		stack_slug = grafana_cloud_stack.test.slug
		name        = "%[1]s-sa"
		role        = "Viewer"
		is_disabled = false
	}

	resource "grafana_cloud_stack_service_account_rotating_token" "management_token" {
		stack_slug = grafana_cloud_stack.test.slug
		service_account_id = grafana_cloud_stack_service_account.management.id
		name_prefix       = "%[1]s"
		seconds_to_live   = %[2]d
        early_rotation_window_seconds = %[3]d
		%[4]s
	}
	`, namePrefix, secondsToLive, earlyRotationWindowSeconds, deleteOnDestroyStr)
}

func computedName(prefix string, suffix time.Time) string {
	return fmt.Sprintf("%s-%d", prefix, suffix.Unix())
}
