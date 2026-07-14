package grafana_test

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/resources/grafana"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
)

func TestAccServiceAccountRotatingToken_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.1.0")

	oldNow := grafana.ServiceAccountRotatingTokenNow
	currentStaticTime := time.Now().UTC()

	namePrefix := "test-rotating-sa-token-terraform-" + acctest.RandString(10)
	var sa models.ServiceAccountDTO
	var token models.TokenDTO
	var tokenAfterRotation models.TokenDTO

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			serviceAccountCheckExists.destroyed(&sa, &models.OrgDetailsDTO{ID: sa.OrgID}),
			testServiceAccountTokenCheckDestroy(&sa, &token),
		),
		Steps: []resource.TestStep{
			{
				PreConfig: setTestServiceAccountRotatingTokenTime(currentStaticTime.Format(time.RFC3339)),
				Config:    testAccServiceAccountRotatingTokenConfig(namePrefix, "Editor", 3600, 600, false),
				Check: resource.ComposeTestCheckFunc(
					// We can't adhere to the interface of checkExistsHelper for SA tokens because we do not have
					// an API that returns a token given its ID. Hence, we need to create special helpers for
					// this particular scenario.
					serviceAccountCheckExists.exists("grafana_service_account.test", &sa),
					checkServiceAccountTokenExists(&sa, testServiceAccountRotatingTokenComputedName(namePrefix, 3600), &token),
					resource.TestCheckResourceAttr("grafana_service_account.test", "name", namePrefix),
					resource.TestCheckResourceAttr("grafana_service_account.test", "role", "Editor"),
					resource.TestCheckResourceAttr("grafana_service_account_rotating_token.test", "name_prefix", namePrefix),
					resource.TestCheckResourceAttr("grafana_service_account_rotating_token.test", "name", testServiceAccountRotatingTokenComputedName(namePrefix, 3600)),
					resource.TestCheckResourceAttr("grafana_service_account_rotating_token.test", "seconds_to_live", "3600"),
					resource.TestCheckResourceAttr("grafana_service_account_rotating_token.test", "early_rotation_window_seconds", "600"),
					resource.TestCheckResourceAttr("grafana_service_account_rotating_token.test", "delete_on_destroy", "false"),
					resource.TestCheckResourceAttrSet("grafana_service_account_rotating_token.test", "expiration"),
				),
			},
			// Test that rotation is not triggered before time by running a plan
			{
				PreConfig: func() {
					setTestServiceAccountRotatingTokenTime(currentStaticTime.Add(5 * time.Second).Format(time.RFC3339))()
				},
				Config:             testAccServiceAccountRotatingTokenConfig(namePrefix, "Editor", 3600, 600, false),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Test that rotation is triggered if running a plan within the early rotation window
			{
				PreConfig: func() {
					setTestServiceAccountRotatingTokenTime(time.Time(token.Expiration).Add(-599 * time.Second).Format(time.RFC3339))()
				},
				Config:             testAccServiceAccountRotatingTokenConfig(namePrefix, "Editor", 3600, 600, false),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			// Test that rotation is triggered if running a plan after the token's expiration date
			{
				PreConfig: func() {
					setTestServiceAccountRotatingTokenTime(time.Time(token.Expiration).Add(10 * time.Second).Format(time.RFC3339))()
				},
				Config:             testAccServiceAccountRotatingTokenConfig(namePrefix, "Editor", 3600, 600, false),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			// Test that early_rotation_window cannot be greater than rotate_after
			{
				PreConfig:   setTestServiceAccountRotatingTokenTime(currentStaticTime.Format(time.RFC3339)),
				Config:      testAccServiceAccountRotatingTokenConfig(namePrefix, "Editor", 10, 20, false),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile("`early_rotation_window_seconds` cannot be greater than `seconds_to_live`"),
			},
			// Test that Terraform-only attributes can be updated without re-creating the token, by updating early_rotation_window and delete_on_destroy
			{
				PreConfig: setTestServiceAccountRotatingTokenTime(currentStaticTime.Format(time.RFC3339)),
				Config:    testAccServiceAccountRotatingTokenConfig(namePrefix, "Editor", 3600, 700, true),
				Check: resource.ComposeTestCheckFunc(
					func(s *terraform.State) error {
						beforeID := token.ID
						err := checkServiceAccountTokenExists(&sa, testServiceAccountRotatingTokenComputedName(namePrefix, 3600), &token)(s)
						if err != nil {
							return err
						}
						if beforeID != token.ID {
							return fmt.Errorf("expected token not to be re-created when updating Terraform-only attributes")
						}
						return nil
					},
					resource.TestCheckResourceAttr("grafana_service_account_rotating_token.test", "seconds_to_live", "3600"),
					resource.TestCheckResourceAttr("grafana_service_account_rotating_token.test", "early_rotation_window_seconds", "700"),
					resource.TestCheckResourceAttr("grafana_service_account_rotating_token.test", "delete_on_destroy", "true"),
				),
			},
			// Test seconds_to_live change should force recreation
			{
				PreConfig: setTestServiceAccountRotatingTokenTime(currentStaticTime.Format(time.RFC3339)),
				Config:    testAccServiceAccountRotatingTokenConfig(namePrefix, "Editor", 3700, 700, false),
				Check: resource.ComposeTestCheckFunc(
					checkServiceAccountTokenExists(&sa, testServiceAccountRotatingTokenComputedName(namePrefix, 3700), &tokenAfterRotation),
					resource.TestCheckResourceAttr("grafana_service_account.test", "name", namePrefix),
					resource.TestCheckResourceAttr("grafana_service_account.test", "role", "Editor"),
					resource.TestCheckResourceAttr("grafana_service_account_rotating_token.test", "name_prefix", namePrefix),
					resource.TestCheckResourceAttr("grafana_service_account_rotating_token.test", "name", testServiceAccountRotatingTokenComputedName(namePrefix, 3700)),
					resource.TestCheckResourceAttr("grafana_service_account_rotating_token.test", "seconds_to_live", "3700"),
					resource.TestCheckResourceAttr("grafana_service_account_rotating_token.test", "early_rotation_window_seconds", "700"),
					resource.TestCheckResourceAttrSet("grafana_service_account_rotating_token.test", "expiration"),

					func(s *terraform.State) error {
						if token.Name == tokenAfterRotation.Name {
							return fmt.Errorf("expected token to be recreated, but Name remained the same: %s", token.Name)
						}
						if token.ID == 0 || tokenAfterRotation.ID == 0 {
							return fmt.Errorf("expected token to be recreated, but ID is empty")
						}
						if token.ID == tokenAfterRotation.ID {
							return fmt.Errorf("expected token to be recreated, but ID remained the same: %d", token.ID)
						}
						return nil
					},
				),
			},
			// Make sure token exists
			{
				Config: testAccServiceAccountRotatingTokenConfig(namePrefix, "Editor", 3700, 700, false),
				Check:  checkServiceAccountTokenExists(&sa, testServiceAccountRotatingTokenComputedName(namePrefix, 3700), &token),
			},
			// Test that destroy does not actually delete the token (it should only show a warning instead)
			{
				Config: testutils.WithoutResource(t, testAccServiceAccountRotatingTokenConfig(namePrefix, "Editor", 3700, 700, false), "grafana_service_account_rotating_token.test"),
				Check: resource.ComposeTestCheckFunc(
					serviceAccountCheckExists.exists("grafana_service_account.test", &sa),
					func(s *terraform.State) error {
						// Check that the token resource is no longer in the state
						_, exists := s.RootModule().Resources["grafana_service_account_rotating_token.test"]
						if exists {
							return fmt.Errorf("expected token resource to be removed from state after destroy, but it still exists")
						}

						// Verify that the token still exists in Grafana
						var tokenFromGrafana models.TokenDTO
						err := checkServiceAccountTokenExists(&sa, token.Name, &tokenFromGrafana)(s)
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
			// Test that the token exists and can be manually deleted through the API
			{
				Config: testutils.WithoutResource(t, testAccServiceAccountRotatingTokenConfig(namePrefix, "Editor", 3700, 700, false), "grafana_service_account_rotating_token.test"),
				PreConfig: func() {
					client := testutils.Provider.Meta().(*common.Client).GrafanaAPI.WithOrgID(sa.OrgID)
					_, err := client.ServiceAccounts.DeleteToken(token.ID, sa.ID)
					if err != nil {
						t.Fatalf("error deleting service account token: %s", err)
					}
				},
			},
			// Create new token to test deletion, setting `delete_on_destroy = true`
			{
				PreConfig: setTestServiceAccountRotatingTokenTime(currentStaticTime.Format(time.RFC3339)),
				Config:    testAccServiceAccountRotatingTokenConfig(namePrefix+"-to-be-deleted", "Editor", 3600, 600, true),
				Check: resource.ComposeTestCheckFunc(
					serviceAccountCheckExists.exists("grafana_service_account.test", &sa),
					checkServiceAccountTokenExists(&sa, testServiceAccountRotatingTokenComputedName(namePrefix+"-to-be-deleted", 3600), &token),
				),
			},
			// Test that destroy deletes the token both in Terraform and in Grafana
			{
				Config: testutils.WithoutResource(t, testAccServiceAccountRotatingTokenConfig(namePrefix+"-to-be-deleted", "Editor", 3600, 600, true), "grafana_service_account_rotating_token.test"),
				Check: resource.ComposeTestCheckFunc(
					serviceAccountCheckExists.exists("grafana_service_account.test", &sa),
					func(s *terraform.State) error {
						// Check that the token resource is no longer in the state
						_, exists := s.RootModule().Resources["grafana_service_account_rotating_token.test"]
						if exists {
							return fmt.Errorf("expected token resource to be removed from state after destroy, but it still exists")
						}

						// Verify that the token is deleted in Grafana too
						var tokenFromGrafana models.TokenDTO
						err := checkServiceAccountTokenExists(&sa, token.Name, &tokenFromGrafana)(s)
						if err == nil {
							return fmt.Errorf("expected token with name '%s' to have been deleted in Grafana, but it still exists", tokenFromGrafana.Name)
						}

						return nil
					},
				),
			},
		},
	})
	grafana.ServiceAccountRotatingTokenNow = oldNow
}

func testAccServiceAccountRotatingTokenConfig(namePrefix, role string, secondsToLive, earlyRotationWindowSeconds int, deleteOnDestroy bool) string {
	var deleteStr string
	if deleteOnDestroy {
		deleteStr = `delete_on_destroy = true`
	}

	return fmt.Sprintf(`
resource "grafana_service_account" "test" {
	name     = "%[1]s"
	role     = "%[2]s"
}

resource "grafana_service_account_rotating_token" "test" {
	name_prefix = "%[1]s"
	service_account_id = grafana_service_account.test.id
	seconds_to_live = %[3]d
	early_rotation_window_seconds = %[4]d
    %[5]s
}
`, namePrefix, role, secondsToLive, earlyRotationWindowSeconds, deleteStr)
}

func setTestServiceAccountRotatingTokenTime(t string) func() {
	return func() {
		grafana.ServiceAccountRotatingTokenNow = func() time.Time {
			parsedT, _ := time.Parse(time.RFC3339, t)
			return parsedT
		}
	}
}

func testServiceAccountRotatingTokenComputedName(namePrefix string, secondsToLive int) string {
	expiration := grafana.ServiceAccountRotatingTokenNow().Add(time.Duration(secondsToLive) * time.Second)
	return fmt.Sprintf("%s-%d", namePrefix, expiration.Unix())
}
