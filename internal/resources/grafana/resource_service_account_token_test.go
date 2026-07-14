package grafana_test

import (
	"fmt"
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
)

func TestAccServiceAccountToken_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.1.0")

	name := acctest.RandString(10)
	var sa models.ServiceAccountDTO
	var token models.TokenDTO

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             serviceAccountCheckExists.destroyed(&sa, nil),
		Steps: []resource.TestStep{
			{
				Config: testAccServiceAccountTokenConfig(name, "Editor", 0, false),
				Check: resource.ComposeTestCheckFunc(
					serviceAccountCheckExists.exists("grafana_service_account.test", &sa),
					checkServiceAccountTokenExists(&sa, name, &token),
					resource.TestCheckResourceAttr("grafana_service_account.test", "name", name),
					resource.TestCheckResourceAttr("grafana_service_account.test", "role", "Editor"),
					resource.TestCheckResourceAttr("grafana_service_account_token.test", "name", name),
					resource.TestCheckNoResourceAttr("grafana_service_account_token.test", "expiration"),
				),
			},
			{
				Config: testAccServiceAccountTokenConfig(name+"-updated", "Viewer", 300, false),
				Check: resource.ComposeTestCheckFunc(
					serviceAccountCheckExists.exists("grafana_service_account.test", &sa),
					checkServiceAccountTokenExists(&sa, name+"-updated", &token),
					resource.TestCheckResourceAttr("grafana_service_account.test", "name", name+"-updated"),
					resource.TestCheckResourceAttr("grafana_service_account.test", "role", "Viewer"),
					resource.TestCheckResourceAttr("grafana_service_account_token.test", "name", name+"-updated"),
					resource.TestCheckResourceAttrSet("grafana_service_account_token.test", "expiration"),
				),
			},
			// Check that the token is deleted when the resource is destroyed
			{
				Config: testutils.WithoutResource(t, testAccServiceAccountTokenConfig(name+"-updated", "Viewer", 300, false), "grafana_service_account_token.test"),
				Check: resource.ComposeTestCheckFunc(
					serviceAccountCheckExists.exists("grafana_service_account.test", &sa),
				),
			},
		},
	})
}

func TestAccServiceAccountToken_inOrg(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.1.0")

	name := acctest.RandString(10)
	var org models.OrgDetailsDTO
	var sa models.ServiceAccountDTO
	var token models.TokenDTO

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             orgCheckExists.destroyed(&org, nil),
		Steps: []resource.TestStep{
			{
				Config: testAccServiceAccountTokenConfig(name, "Editor", 0, true),
				Check: resource.ComposeTestCheckFunc(
					serviceAccountCheckExists.exists("grafana_service_account.test", &sa),
					checkServiceAccountTokenExists(&sa, name, &token),
					resource.TestCheckResourceAttr("grafana_service_account.test", "name", name),
					resource.TestCheckResourceAttr("grafana_service_account.test", "role", "Editor"),
					resource.TestCheckResourceAttr("grafana_service_account_token.test", "name", name),
					resource.TestCheckNoResourceAttr("grafana_service_account_token.test", "expiration"),

					// Check that the service account is in the correct organization
					resource.TestMatchResourceAttr("grafana_service_account.test", "id", nonDefaultOrgIDRegexp),
					orgCheckExists.exists("grafana_organization.test", &org),
					checkResourceIsInOrg("grafana_service_account.test", "grafana_organization.test"),
				),
			},
			{
				Config: testAccServiceAccountTokenConfig(name+"-updated", "Viewer", 300, true),
				Check: resource.ComposeTestCheckFunc(
					serviceAccountCheckExists.exists("grafana_service_account.test", &sa),
					checkServiceAccountTokenExists(&sa, name+"-updated", &token),
					resource.TestCheckResourceAttr("grafana_service_account.test", "name", name+"-updated"),
					resource.TestCheckResourceAttr("grafana_service_account.test", "role", "Viewer"),
					resource.TestCheckResourceAttr("grafana_service_account_token.test", "name", name+"-updated"),
					resource.TestCheckResourceAttrSet("grafana_service_account_token.test", "expiration"),

					// Check that the service account is in the correct organization
					resource.TestMatchResourceAttr("grafana_service_account.test", "id", nonDefaultOrgIDRegexp),
					orgCheckExists.exists("grafana_organization.test", &org),
					checkResourceIsInOrg("grafana_service_account.test", "grafana_organization.test"),
				),
			},
			// Check that the token is deleted when the resource is destroyed
			{
				Config: testutils.WithoutResource(t, testAccServiceAccountTokenConfig(name+"-updated", "Viewer", 300, true), "grafana_service_account_token.test"),
				Check: resource.ComposeTestCheckFunc(
					serviceAccountCheckExists.exists("grafana_service_account.test", &sa),
				),
			},
		},
	})
}

func checkServiceAccountTokenExists(sa *models.ServiceAccountDTO, tokenName string, t *models.TokenDTO) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := grafanaTestClient().WithOrgID(sa.OrgID)
		resp, err := client.ServiceAccounts.ListTokens(sa.ID)
		if err != nil {
			return err
		}

		for _, token := range resp.Payload {
			if token.Name == tokenName {
				*t = *token
				return nil
			}
		}
		return fmt.Errorf("expected token %s not found", tokenName)
	}
}

func testAccServiceAccountTokenConfig(name, role string, secondsToLive int, inOrg bool) string {
	config := ""

	secondsToLiveAttr := ""
	if secondsToLive > 0 {
		secondsToLiveAttr = fmt.Sprintf("seconds_to_live = %d", secondsToLive)
	}

	orgIDAttr := ""
	if inOrg {
		config = fmt.Sprintf(`
resource "grafana_organization" "test" {
	name = "%s"
}
`, name)
		orgIDAttr = "org_id = grafana_organization.test.id"
	}

	return config + fmt.Sprintf(`
resource "grafana_service_account" "test" {
	name     = "%[1]s"
	role     = "%[2]s"
	%[4]s
}

resource "grafana_service_account_token" "test" {
	name = "%[1]s"
	service_account_id = grafana_service_account.test.id
	%[3]s
}
`, name, role, secondsToLiveAttr, orgIDAttr)
}

func testServiceAccountTokenCheckDestroy(sa *models.ServiceAccountDTO, t *models.TokenDTO) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if sa == nil || sa.ID == 0 {
			return nil
		}
		if t == nil || t.ID == 0 {
			return nil
		}
		client := testutils.Provider.Meta().(*common.Client).GrafanaAPI.WithOrgID(sa.OrgID)
		resp, err := client.ServiceAccounts.ListTokens(sa.ID)
		if err != nil {
			return err
		}

		for _, key := range resp.Payload {
			if t.ID == key.ID {
				return fmt.Errorf("grafana service account token `%d` with name `%s` still exists after destroy", t.ID, t.Name)
			}
		}

		return nil
	}
}
