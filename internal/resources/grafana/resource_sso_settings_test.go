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

	"github.com/grafana/terraform-provider-grafana/internal/resources/grafana"
	"github.com/grafana/terraform-provider-grafana/internal/testutils"
)

func TestSSOSettings_basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t) // TODO: Run on v10.4.0 once it's released

	providers := []string{"gitlab", "google", "generic_oauth", "azuread", "okta"}

	api := grafana.OAPIGlobalClient(testutils.Provider.Meta())

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
				{
					ResourceName:            resourceName,
					ImportState:             true,
					ImportStateVerify:       true,
					ImportStateVerifyIgnore: []string{"oauth2_settings.0.client_secret", "oauth2_settings.0.custom"},
				},
			},
		})
	}
}

func TestSSOSettings_customFields(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t) // TODO: Run on v10.4.0 once it's released

	api := grafana.OAPIGlobalClient(testutils.Provider.Meta())

	provider := "github"

	defaultSettings, err := api.SsoSettings.GetProviderSettings(provider)
	if err != nil {
		t.Fatalf("failed to fetch the default settings for provider %s: %v", provider, err)
	}

	resourceName := "grafana_sso_settings.sso_settings"

	resource.Test(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      checkSsoSettingsReset(api, provider, defaultSettings.Payload),
		Steps: []resource.TestStep{
			{
				Config: testConfigWithCustomFields,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "provider_name", provider),
					resource.TestCheckResourceAttr(resourceName, "oauth2_settings.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "oauth2_settings.0.client_id", "client_id"),
					resource.TestCheckResourceAttr(resourceName, "oauth2_settings.0.client_secret", "client_secret"),
					resource.TestCheckResourceAttr(resourceName, "oauth2_settings.0.custom.custom_field", "custom1"),
					resource.TestCheckResourceAttr(resourceName, "oauth2_settings.0.custom.another_custom_field", "custom2"),
					resource.TestCheckResourceAttr(resourceName, "oauth2_settings.0.custom.camelCaseField", "custom3"),
					// check that all custom fields are returned by the API
					func(s *terraform.State) error {
						resp, err := api.SsoSettings.GetProviderSettings(provider)
						if err != nil {
							return err
						}

						payload := resp.GetPayload()
						settings := payload.Settings.(map[string]any)

						// the API returns the settings names in camelCase
						if settings["clientId"] != "client_id" {
							t.Fatalf("expected value for client_id is not equal to the actual value: %s", settings["clientId"])
						}
						if settings["customField"] != "custom1" {
							t.Fatalf("expected value for custom_field is not equal to the actual value: %s", settings["customField"])
						}
						if settings["anotherCustomField"] != "custom2" {
							t.Fatalf("expected value for another_custom_field is not equal to the actual value: %s", settings["anotherCustomField"])
						}
						if settings["camelCaseField"] != "custom3" {
							t.Fatalf("expected value for camelCaseField is not equal to the actual value: %s", settings["camelCaseField"])
						}

						return nil
					},
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"oauth2_settings.0.client_secret", "oauth2_settings.0.custom"},
			},
		},
	})
}

func TestSSOSettings_resourceWithInvalidProvider(t *testing.T) {
	provider := "invalid_provider"

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testConfigForProvider(provider, "new"),
				ExpectError: regexp.MustCompile("expected provider_name to be one of"),
			},
		},
	})
}

func TestSSOSettings_resourceWithNoSettings(t *testing.T) {
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

func TestSSOSettings_resourceWithInvalidCustomField(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testConfigWithInvalidCustomField,
				ExpectError: regexp.MustCompile("Invalid custom field"),
			},
		},
	})
}

func TestSSOSettings_resourceWithValidationErrors(t *testing.T) {
	for _, config := range testConfigsWithValidationErrors {
		resource.Test(t, resource.TestCase{
			ProviderFactories: testutils.ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: config,
					// all validation errors contain the word "must"
					ExpectError: regexp.MustCompile("must"),
				},
			},
		})
	}
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

func testConfigForProvider(provider string, prefix string) string {
	urls := ""
	switch provider {
	case "azuread", "generic_oauth", "okta":
		urls = `auth_url = "https://myidp.com/oauth/authorize"
    token_url = "https://myidp.com/oauth/token"
	api_url = "https://myidp.com/oauth/userinfo"`
	}

	return fmt.Sprintf(`resource "grafana_sso_settings" "%[2]s_sso_settings" {
  provider_name = "%[2]s"
  oauth2_settings {
    client_id     = "%[1]s_%[2]s_client_id"
    client_secret = "%[1]s_%[2]s_client_secret"
    %[3]s
  }
}`, prefix, provider, urls)
}

const testConfigWithCustomFields = `resource "grafana_sso_settings" "sso_settings" {
  provider_name = "github"
  oauth2_settings {
    client_id     = "client_id"
    client_secret = "client_secret"
    custom = {
      custom_field = "custom1"
      another_custom_field = "custom2"
      camelCaseField = "custom3"
    }
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
    client_id     = "first_gitlab_client_id"
    client_secret = "first_gitlab_client_secret"
    auth_url      = "https://gitlab.com/oauth/authorize"
    token_url     = "https://gitlab.com/oauth/token"
  }

  oauth2_settings {
    client_id     = "second_gitlab_client_id"
    client_secret = "second_gitlab_client_secret"
    auth_url      = "https://gitlab.com/oauth/authorize"
    token_url     = "https://gitlab.com/oauth/token"
  }
}`

const testConfigWithInvalidCustomField = `resource "grafana_sso_settings" "sso_settings" {
  provider_name = "gitlab"
  oauth2_settings {
    client_id     = "first_gitlab_client_id"
    client_secret = "first_gitlab_client_secret"
    auth_url      = "https://gitlab.com/oauth/authorize"
    token_url     = "https://gitlab.com/oauth/token"
    custom        = {
      token_url = "https://gitlab-clone.com/oauth/token"
    }
  }
}`

var testConfigsWithValidationErrors = []string{
	// no token_url provided for azuread
	`resource "grafana_sso_settings" "azure_sso_settings" {
  provider_name = "azuread"
  oauth2_settings {
    client_id = "client_id"
    auth_url  = "https://login.microsoftonline.com/12345/oauth2/v2.0/authorize"
  }
}`,
	// invalid auth_url provided for okta
	`resource "grafana_sso_settings" "okta_sso_settings" {
  provider_name = "okta"
  oauth2_settings {
    client_id = "client_id"
    auth_url  = "ftp://login.microsoftonline.com/12345/oauth2/v2.0/authorize"
    token_url = "https://tenantid123.okta.com/oauth2/v1/token"
	api_url = "https://tenantid123.okta.com/oauth2/v1/userinfo"
  }
}`,
	// auth_url is not empty for github
	`resource "grafana_sso_settings" "github_sso_settings" {
  provider_name = "github"
  oauth2_settings {
    client_id = "client_id"
    auth_url  = "https://login.microsoftonline.com/12345/oauth2/v2.0/authorize"
  }
}`,
	// token_url is not empty for gitlab
	`resource "grafana_sso_settings" "gitlab_sso_settings" {
  provider_name = "gitlab"
  oauth2_settings {
    client_id = "client_id"
    token_url  = "https://login.microsoftonline.com/12345/oauth2/v2.0/token"
  }
}`,
	// api_url is not empty for google
	`resource "grafana_sso_settings" "google_sso_settings" {
  provider_name = "google"
  oauth2_settings {
    client_id = "client_id"
    api_url  = "https://login.microsoftonline.com/12345/oauth2/v2.0/userinfo"
  }
}`,
}
