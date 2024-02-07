package grafana_test

import (
	"fmt"
	"reflect"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/models"

	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/grafana/terraform-provider-grafana/internal/testutils"
)

func TestSSOSettings_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=10.4.0")

	providers := []string{"github", "gitlab", "google", "generic_oauth", "azuread", "okta"}

	api := testutils.Provider.Meta().(*common.Client).GrafanaOAPI.WithOrgID(1)

	for _, provider := range providers {
		defaultSettings, err := api.SsoSettings.GetProviderSettings(provider)
		if err != nil {
			t.Fatalf("failed to fetch the default settings for provider %s: %v", provider, err)
		}

		resourceName := fmt.Sprintf("grafana_sso_settings.%s_sso_settings", provider)

		resource.Test(t, resource.TestCase{
			ProviderFactories: testutils.ProviderFactories,
			CheckDestroy:      checkSsoSettingsReset(api, provider, defaultSettings.Payload),
			Steps: []resource.TestStep{
				{
					Config: testConfigForProvider(provider, "new"),
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr(resourceName, "provider_name", provider),
						resource.TestCheckResourceAttr(resourceName, "oauth2_settings.#", "1"),
						resource.TestCheckResourceAttr(resourceName, "oauth2_settings.0.client_id", fmt.Sprintf("new_%s_client_id", provider)),
						resource.TestCheckResourceAttr(resourceName, "oauth2_settings.0.client_secret", fmt.Sprintf("new_%s_client_secret", provider)),
					),
				},
				{
					Config: testConfigForProvider(provider, "updated"),
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr(resourceName, "provider_name", provider),
						resource.TestCheckResourceAttr(resourceName, "oauth2_settings.#", "1"),
						resource.TestCheckResourceAttr(resourceName, "oauth2_settings.0.client_id", fmt.Sprintf("updated_%s_client_id", provider)),
						resource.TestCheckResourceAttr(resourceName, "oauth2_settings.0.client_secret", fmt.Sprintf("updated_%s_client_secret", provider)),
					),
				},
			},
		})
	}
}

func TestSSOSettings_applyExample(t *testing.T) {
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
			{
				ResourceName:            "grafana_sso_settings.github_sso_settings",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"oauth2_settings.0.client_secret"},
			},
		},
	})
}

// multiple SSO settings resources having the same provider are sharing a single entry in the DB
func TestSSOSettings_twoResourcesForOneProvider(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=10.4.0")

	provider := "azuread"

	resource.Test(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testConfigWithTwoResourcesForOneProvider,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_sso_settings.first_sso_settings", "provider_name", provider),
					resource.TestCheckResourceAttr("grafana_sso_settings.first_sso_settings", "oauth2_settings.#", "1"),
					resource.TestCheckResourceAttr("grafana_sso_settings.first_sso_settings", "oauth2_settings.0.client_id", "first_gitlab_client_id"),
					resource.TestCheckResourceAttr("grafana_sso_settings.second_sso_settings", "provider_name", provider),
					resource.TestCheckResourceAttr("grafana_sso_settings.second_sso_settings", "oauth2_settings.#", "1"),
					resource.TestCheckResourceAttr("grafana_sso_settings.second_sso_settings", "oauth2_settings.0.client_id", "second_gitlab_client_id"),
				),
				ExpectNonEmptyPlan: true,
			},
			{
				ResourceName:            "grafana_sso_settings.first_sso_settings",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"oauth2_settings.0.client_secret"},
			},
			{
				ResourceName:            "grafana_sso_settings.second_sso_settings",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"oauth2_settings.0.client_secret"},
			},
			{
				RefreshState: true,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_sso_settings.first_sso_settings", "provider_name", provider),
					resource.TestCheckResourceAttr("grafana_sso_settings.first_sso_settings", "oauth2_settings.#", "1"),
					resource.TestCheckResourceAttr("grafana_sso_settings.second_sso_settings", "provider_name", provider),
					resource.TestCheckResourceAttr("grafana_sso_settings.second_sso_settings", "oauth2_settings.#", "1"),
					func(s *terraform.State) error {
						first, err := getPrimaryInstanceState(s, "grafana_sso_settings.first_sso_settings")
						if err != nil {
							return err
						}

						second, err := getPrimaryInstanceState(s, "grafana_sso_settings.second_sso_settings")
						if err != nil {
							return err
						}

						firstClientID, ok := first.Attributes["oauth2_settings.0.client_id"]
						if !ok {
							return fmt.Errorf("client_id not found in settings")
						}

						secondClientID, ok := second.Attributes["oauth2_settings.0.client_id"]
						if !ok {
							return fmt.Errorf("client_id not found in settings")
						}

						if firstClientID != secondClientID {
							return fmt.Errorf("client_id is not the same: %s vs. %s", firstClientID, secondClientID)
						}

						return nil
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestSSOSettings_resourceWithNoSettings(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=10.4.0")

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testConfigWithNoSettings,
				ExpectError: regexp.MustCompile("Insufficient oauth2_settings blocks"),
			},
		},
	})
}

func TestSSOSettings_resourceWithEmptySettings(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=10.4.0")

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testConfigWithEmptySettings,
				ExpectError: regexp.MustCompile("Missing required argument"),
			},
		},
	})
}

func TestSSOSettings_resourceWithManySettings(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=10.4.0")

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testConfigWithManySettings,
				ExpectError: regexp.MustCompile("Too many oauth2_settings blocks"),
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

func getPrimaryInstanceState(s *terraform.State, name string) (*terraform.InstanceState, error) {
	ms := s.RootModule()

	res, ok := ms.Resources[name]
	if !ok {
		return nil, fmt.Errorf("resource %s not found", name)
	}

	primary := res.Primary
	if primary == nil {
		return nil, fmt.Errorf("no primary instance for resource %s", name)
	}

	return primary, nil
}

func testConfigForProvider(provider string, prefix string) string {
	return fmt.Sprintf(`resource "grafana_sso_settings" "%[2]s_sso_settings" {
  provider_name = "%[2]s"
  oauth2_settings {
    client_id             = "%[1]s_%[2]s_client_id"
    client_secret         = "%[1]s_%[2]s_client_secret"
  }
}`, prefix, provider)
}

const testConfigWithTwoResourcesForOneProvider = `resource "grafana_sso_settings" "first_sso_settings" {
  provider_name = "azuread"
  oauth2_settings {
    client_id             = "first_gitlab_client_id"
  }
}

resource "grafana_sso_settings" "second_sso_settings" {
  provider_name = "azuread"
  oauth2_settings {
    client_id             = "second_gitlab_client_id"
  }
}`

const testConfigWithEmptySettings = `resource "grafana_sso_settings" "sso_settings" {
  provider_name = "okta"
  oauth2_settings {
  }
}`

const testConfigWithNoSettings = `resource "grafana_sso_settings" "sso_settings" {
  provider_name = "gitlab"
}`

const testConfigWithManySettings = `resource "grafana_sso_settings" "sso_settings" {
  provider_name = "gitlab"
  oauth2_settings {
    client_id             = "first_gitlab_client_id"
    client_secret         = "first_gitlab_client_secret"
  }

  oauth2_settings {
    client_id             = "second_gitlab_client_id"
    client_secret         = "second_gitlab_client_secret"
  }
}`
