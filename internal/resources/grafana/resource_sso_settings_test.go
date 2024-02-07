package grafana_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/models"

	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/grafana/terraform-provider-grafana/internal/testutils"
)

func TestSSOSettings_example(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=10.4.0")

	var settings models.GetProviderSettingsOKBody

	provider := "github"

	api := testutils.Provider.Meta().(*common.Client).GrafanaOAPI.WithOrgID(1)
	defaultSettings, err := api.SsoSettings.GetProviderSettings(provider)
	if err != nil {
		t.Fatalf("failed to fetch the default settings for provider %s: %v", provider, err)
	}

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      checkSsoSettingsReset(api, provider, defaultSettings.Payload),
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "resources/grafana_sso_settings/resource.tf"),

				Check: resource.ComposeTestCheckFunc(
					ssoSettingsCheckExists.exists("grafana_sso_settings.github_sso_settings", &settings),
					resource.TestCheckResourceAttr("grafana_sso_settings.github_sso_settings", "provider_name", provider),
					resource.TestCheckResourceAttr("grafana_sso_settings.github_sso_settings", "oauth2_settings.#", "1"),
					resource.TestCheckResourceAttr("grafana_sso_settings.github_sso_settings", "oauth2_settings.0.client_id", "github_client_id"),
					resource.TestCheckResourceAttr("grafana_sso_settings.github_sso_settings", "oauth2_settings.0.client_secret", "github_client_secret"),
					resource.TestCheckResourceAttr("grafana_sso_settings.github_sso_settings", "oauth2_settings.0.team_ids", "12,50,123"),
					resource.TestCheckResourceAttr("grafana_sso_settings.github_sso_settings", "oauth2_settings.0.allowed_organizations", "organization1,organization2"),
				),
			},
		},
	})
}

func checkSsoSettingsReset(api *client.GrafanaHTTPAPI, provider string, defaultSettings *models.GetProviderSettingsOKBody) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		settings, err := api.SsoSettings.GetProviderSettings(provider)
		if err != nil {
			return err
		}

		if !reflect.DeepEqual(settings.Payload, defaultSettings) {
			return fmt.Errorf("settings not equal to the default settings for provider %s", provider)
		}

		return nil
	}
}
