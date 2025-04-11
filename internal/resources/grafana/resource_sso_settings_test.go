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

	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
)

func TestSSOSettings_basic_oauth2(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t) // TODO: Fix the tests to run on local instances

	providers := []string{"gitlab", "google", "generic_oauth", "azuread", "okta"}

	api := grafanaTestClient()

	for _, provider := range providers {
		defaultSettings, err := api.SsoSettings.GetProviderSettings(provider)
		if err != nil {
			t.Fatalf("failed to fetch the default settings for provider %s: %v", provider, err)
		}

		resourceName := fmt.Sprintf("grafana_sso_settings.%s_sso_settings", provider)

		resource.Test(t, resource.TestCase{
			ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
			CheckDestroy:             checkSsoSettingsReset(api, provider, defaultSettings.Payload),
			Steps: []resource.TestStep{
				{
					Config: testConfigForOAuth2Provider(provider, "new"),
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr(resourceName, "provider_name", provider),
						resource.TestCheckResourceAttr(resourceName, "oauth2_settings.#", "1"),
						resource.TestCheckResourceAttr(resourceName, "oauth2_settings.0.client_id", fmt.Sprintf("new_%s_client_id", provider)),
						resource.TestCheckResourceAttr(resourceName, "oauth2_settings.0.client_secret", fmt.Sprintf("new_%s_client_secret", provider)),
					),
				},
				{
					Config: testConfigForOAuth2Provider(provider, "updated"),
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

func TestSSOSettings_basic_saml(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=11.1")

	provider := "saml"

	api := grafanaTestClient()

	defaultSettings, err := api.SsoSettings.GetProviderSettings(provider)
	if err != nil {
		t.Fatalf("failed to fetch the default settings for provider %s: %v", provider, err)
	}

	resourceName := "grafana_sso_settings.saml_sso_settings"

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             checkSsoSettingsReset(api, provider, defaultSettings.Payload),
		Steps: []resource.TestStep{
			{
				Config: testConfigForSamlProvider,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "provider_name", provider),
					resource.TestCheckResourceAttr(resourceName, "saml_settings.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "saml_settings.0.certificate_path", "/certs/saml.crt"),
					resource.TestCheckResourceAttr(resourceName, "saml_settings.0.private_key_path", "/certs/saml.key"),
					resource.TestCheckResourceAttr(resourceName, "saml_settings.0.idp_metadata_url", "https://nexus.microsoftonline-p.com/federationmetadata/saml20/federationmetadata.xml"),
					resource.TestCheckResourceAttr(resourceName, "saml_settings.0.signature_algorithm", "rsa-sha256"),
					resource.TestCheckResourceAttr(resourceName, "saml_settings.0.metadata_valid_duration", "24h"),
					resource.TestCheckResourceAttr(resourceName, "saml_settings.0.entity_id", "https://custom-entity-id.com"),
				),
			},
			{
				Config: testConfigForSamlProviderUpdated,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "provider_name", provider),
					resource.TestCheckResourceAttr(resourceName, "saml_settings.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "saml_settings.0.certificate_path", "/certs/saml.crt"),
					resource.TestCheckResourceAttr(resourceName, "saml_settings.0.private_key_path", "/certs/saml.key"),
					resource.TestCheckResourceAttr(resourceName, "saml_settings.0.idp_metadata_url", "https://nexus.microsoftonline-p.com/federationmetadata/saml20/federationmetadata.xml"),
					resource.TestCheckResourceAttr(resourceName, "saml_settings.0.signature_algorithm", "rsa-sha512"),
					resource.TestCheckResourceAttr(resourceName, "saml_settings.0.metadata_valid_duration", "48h"),
					resource.TestCheckResourceAttr(resourceName, "saml_settings.0.assertion_attribute_email", "email"),
					resource.TestCheckResourceAttr(resourceName, "saml_settings.0.allow_sign_up", "true"),
				),
			},
			{
				Config: testConfigForSAMLProviderWithAzureAD,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "provider_name", provider),
					resource.TestCheckResourceAttr(resourceName, "saml_settings.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "saml_settings.0.certificate_path", "/certs/saml.crt"),
					resource.TestCheckResourceAttr(resourceName, "saml_settings.0.private_key_path", "/certs/saml.key"),
					resource.TestCheckResourceAttr(resourceName, "saml_settings.0.idp_metadata_url", "https://nexus.microsoftonline-p.com/federationmetadata/saml20/federationmetadata.xml"),
					resource.TestCheckResourceAttr(resourceName, "saml_settings.0.signature_algorithm", "rsa-sha256"),
					resource.TestCheckResourceAttr(resourceName, "saml_settings.0.metadata_valid_duration", "24h"),
					resource.TestCheckResourceAttr(resourceName, "saml_settings.0.assertion_attribute_email", "email"),
					resource.TestCheckResourceAttr(resourceName, "saml_settings.0.client_id", "client_id"),
					resource.TestCheckResourceAttr(resourceName, "saml_settings.0.client_secret", "client_secret"),
					resource.TestCheckResourceAttr(resourceName, "saml_settings.0.token_url", "https://myidp.com/oauth/token"),
					resource.TestCheckResourceAttr(resourceName, "saml_settings.0.force_use_graph_api", "true"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"saml_settings.0.private_key_path", "saml_settings.0.certificate_path", "saml_settings.0.client_secret", "saml_settings.0.token_url"},
			},
		},
	})
}

func TestSSOSettings_basic_ldap(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=11.3")

	provider := "ldap"

	api := grafanaTestClient()

	defaultSettings, err := api.SsoSettings.GetProviderSettings(provider)
	if err != nil {
		t.Fatalf("failed to fetch the default settings for provider %s: %v", provider, err)
	}

	resourceName := "grafana_sso_settings.ldap_sso_settings"

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             checkSsoSettingsReset(api, provider, defaultSettings.Payload),
		Steps: []resource.TestStep{
			{
				Config: testConfigForLdapProvider,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "provider_name", provider),
					resource.TestCheckResourceAttr(resourceName, "ldap_settings.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "ldap_settings.0.config.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "ldap_settings.0.config.0.servers.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "ldap_settings.0.config.0.servers.0.host", "127.0.0.1"),
					resource.TestCheckResourceAttr(resourceName, "ldap_settings.0.config.0.servers.0.search_filter", "(cn=%s)"),
					resource.TestCheckResourceAttr(resourceName, "ldap_settings.0.config.0.servers.0.search_base_dns.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "ldap_settings.0.config.0.servers.0.search_base_dns.0", "dc=grafana,dc=org"),
				),
			},
			{
				Config: testConfigForLdapProviderUpdated,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "provider_name", provider),
					resource.TestCheckResourceAttr(resourceName, "ldap_settings.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "ldap_settings.0.config.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "ldap_settings.0.config.0.servers.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "ldap_settings.0.config.0.servers.0.host", "127.0.0.5"),
					resource.TestCheckResourceAttr(resourceName, "ldap_settings.0.config.0.servers.0.bind_password", "password"),
					resource.TestCheckResourceAttr(resourceName, "ldap_settings.0.config.0.servers.0.search_filter", "(cn=%s)"),
					resource.TestCheckResourceAttr(resourceName, "ldap_settings.0.config.0.servers.0.search_base_dns.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "ldap_settings.0.config.0.servers.0.search_base_dns.0", "dc=grafana,dc=org"),
					resource.TestCheckResourceAttr(resourceName, "ldap_settings.0.config.0.servers.0.attributes.email", "email"),
					resource.TestCheckResourceAttr(resourceName, "ldap_settings.0.config.0.servers.0.attributes.name", "name"),
					resource.TestCheckResourceAttr(resourceName, "ldap_settings.0.config.0.servers.0.group_mappings.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "ldap_settings.0.config.0.servers.0.group_mappings.0.group_dn", "cn=superadmins,dc=grafana,dc=org"),
					resource.TestCheckResourceAttr(resourceName, "ldap_settings.0.config.0.servers.0.group_mappings.0.org_role", "Admin"),
					resource.TestCheckResourceAttr(resourceName, "ldap_settings.0.config.0.servers.0.group_mappings.0.org_id", "1"),
					resource.TestCheckResourceAttr(resourceName, "ldap_settings.0.config.0.servers.0.group_mappings.0.grafana_admin", "true"),
					resource.TestCheckResourceAttr(resourceName, "ldap_settings.0.config.0.servers.0.group_mappings.1.group_dn", "cn=users,dc=grafana,dc=org"),
					resource.TestCheckResourceAttr(resourceName, "ldap_settings.0.config.0.servers.0.group_mappings.1.org_role", "Editor"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"ldap_settings.0.config.0.servers.0.bind_password"},
			},
		},
	})
}

func TestSSOSettings_customFields(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t) // TODO: Fix the tests to run on local instances

	api := grafanaTestClient()

	provider := "github"

	defaultSettings, err := api.SsoSettings.GetProviderSettings(provider)
	if err != nil {
		t.Fatalf("failed to fetch the default settings for provider %s: %v", provider, err)
	}

	resourceName := "grafana_sso_settings.sso_settings"

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             checkSsoSettingsReset(api, provider, defaultSettings.Payload),
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
				Config: testConfigWithCustomFieldsUpdated,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "provider_name", provider),
					resource.TestCheckResourceAttr(resourceName, "oauth2_settings.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "oauth2_settings.0.client_id", "client_id_updated"),
					resource.TestCheckResourceAttr(resourceName, "oauth2_settings.0.client_secret", "client_secret"),
					resource.TestCheckResourceAttr(resourceName, "oauth2_settings.0.scopes", "email profile"),
					resource.TestCheckResourceAttr(resourceName, "oauth2_settings.0.custom.custom_field", "custom1_updated"),
					resource.TestCheckResourceAttr(resourceName, "oauth2_settings.0.custom.another_custom_field", "custom2_updated"),
					resource.TestCheckResourceAttr(resourceName, "oauth2_settings.0.custom.one_more_custom_field", "custom4"),
					// check that all custom fields are returned by the API
					func(s *terraform.State) error {
						resp, err := api.SsoSettings.GetProviderSettings(provider)
						if err != nil {
							return err
						}

						payload := resp.GetPayload()
						settings := payload.Settings.(map[string]any)

						// the API returns the settings names in camelCase
						if settings["clientId"] != "client_id_updated" {
							t.Fatalf("expected value for client_id is not equal to the actual value: %s", settings["clientId"])
						}
						if settings["scopes"] != "email profile" {
							t.Fatalf("expected value for scopes is not equal to the actual value: %s", settings["scopes"])
						}
						if settings["customField"] != "custom1_updated" {
							t.Fatalf("expected value for custom_field is not equal to the actual value: %s", settings["customField"])
						}
						if settings["anotherCustomField"] != "custom2_updated" {
							t.Fatalf("expected value for another_custom_field is not equal to the actual value: %s", settings["anotherCustomField"])
						}
						if settings["oneMoreCustomField"] != "custom4" {
							t.Fatalf("expected value for one_more_custom_field is not equal to the actual value: %s", settings["oneMoreCustomField"])
						}
						if _, ok := settings["camelCaseField"]; ok {
							t.Fatalf("camelCaseField custom field is not expected to exist")
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
	testutils.CheckOSSTestsEnabled(t)

	provider := "invalid_provider"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testConfigForOAuth2Provider(provider, "new"),
				ExpectError: regexp.MustCompile("expected provider_name to be one of"),
			},
		},
	})
}

func TestSSOSettings_resourceWithNoSettings(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	for _, config := range testConfigsWithNoSettings {
		resource.Test(t, resource.TestCase{
			ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      config,
					ExpectError: regexp.MustCompile("no settings found"),
				},
			},
		})
	}
}

func TestSSOSettings_resourceWithEmptySettings(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testConfigWithEmptySettings,
				ExpectError: regexp.MustCompile("Missing required argument"),
			},
		},
	})
}

func TestSSOSettings_resourceWithManySettings(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testConfigWithManySettings,
				ExpectError: regexp.MustCompile("Too many oauth2_settings blocks"),
			},
		},
	})
}

func TestSSOSettings_resourceWithInvalidCustomField(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testConfigWithInvalidCustomField,
				ExpectError: regexp.MustCompile("Invalid custom field"),
			},
		},
	})
}

func TestSSOSettings_resourceWithValidationErrors(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	for _, config := range testConfigsWithValidationErrors {
		resource.Test(t, resource.TestCase{
			ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
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

func testConfigForOAuth2Provider(provider string, prefix string) string {
	urls := ""
	switch provider {
	case "generic_oauth", "okta":
		urls = `auth_url = "https://myidp.com/oauth/authorize"
    token_url = "https://myidp.com/oauth/token"
	api_url = "https://myidp.com/oauth/userinfo"`
	case "azuread":
		urls = `auth_url = "https://myidp.com/oauth/authorize"
    token_url = "https://myidp.com/oauth/token"`
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

// the SAML configuration needs a valid certificate, private_key and idp_metadata to be accepted by Grafana API
const testConfigForSamlProvider = `resource "grafana_sso_settings" "saml_sso_settings" {
  provider_name = "saml"
  saml_settings {
    certificate_path        = "/certs/saml.crt"
    private_key_path        = "/certs/saml.key"
    idp_metadata_url        = "https://nexus.microsoftonline-p.com/federationmetadata/saml20/federationmetadata.xml"
    signature_algorithm     = "rsa-sha256"
    metadata_valid_duration = "24h"
	entity_id               = "https://custom-entity-id.com"
  }
}`

const testConfigForSamlProviderUpdated = `resource "grafana_sso_settings" "saml_sso_settings" {
  provider_name = "saml"
  saml_settings {
    certificate_path          = "/certs/saml.crt"
    private_key_path          = "/certs/saml.key"
    idp_metadata_url          = "https://nexus.microsoftonline-p.com/federationmetadata/saml20/federationmetadata.xml"
    allow_sign_up             = true
    signature_algorithm       = "rsa-sha512"
    metadata_valid_duration   = "48h"
    assertion_attribute_email = "email"
  }
}`

const testConfigForSAMLProviderWithAzureAD = `resource "grafana_sso_settings" "saml_sso_settings" {
	provider_name = "saml"
	saml_settings {
		certificate_path          = "/certs/saml.crt"
		private_key_path          = "/certs/saml.key"
		idp_metadata_url          = "https://nexus.microsoftonline-p.com/federationmetadata/saml20/federationmetadata.xml"
		signature_algorithm       = "rsa-sha256"
		metadata_valid_duration   = "24h"
		assertion_attribute_email = "email"
		client_id                 = "client_id"
		client_secret             = "client_secret"
		token_url                 = "https://myidp.com/oauth/token"
		force_use_graph_api       = true
	}
}`

const testConfigForLdapProvider = `resource "grafana_sso_settings" "ldap_sso_settings" {
  provider_name = "ldap"
  ldap_settings {
    config {
      servers {
        host = "127.0.0.1"
        search_filter = "(cn=%s)"
        search_base_dns = [
          "dc=grafana,dc=org",
        ]
      }
    }
  }
}`

const testConfigForLdapProviderUpdated = `resource "grafana_sso_settings" "ldap_sso_settings" {
  provider_name = "ldap"
  ldap_settings {
    config {
      servers {
        host = "127.0.0.5"
        search_filter = "(cn=%s)"
		bind_password = "password"
        search_base_dns = [
          "dc=grafana,dc=org",
        ]
		attributes = {
          email = "email"
          name = "name"
        }
        group_mappings {
          group_dn = "cn=superadmins,dc=grafana,dc=org"
          org_role = "Admin"
          org_id = 1
          grafana_admin = true
        }
        group_mappings {
          group_dn = "cn=users,dc=grafana,dc=org"
          org_role = "Editor"
        }
      }
    }
  }
}`

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

const testConfigWithCustomFieldsUpdated = `resource "grafana_sso_settings" "sso_settings" {
  provider_name = "github"
  oauth2_settings {
    client_id     = "client_id_updated"
    client_secret = "client_secret"
    scopes        = "email profile"
    custom = {
      custom_field = "custom1_updated"
      another_custom_field = "custom2_updated"
      one_more_custom_field = "custom4"
    }
  }
}`

const testConfigWithEmptySettings = `resource "grafana_sso_settings" "sso_settings" {
  provider_name = "okta"
  oauth2_settings {
  }
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

var testConfigsWithNoSettings = []string{
	// no oauth2_settings for gitlab
	`resource "grafana_sso_settings" "sso_settings" {
  provider_name = "gitlab"
}`,
	// saml_settings instead of oauth2_settings for gitlab
	`resource "grafana_sso_settings" "sso_settings" {
  provider_name = "gitlab"
  saml_settings {
    certificate_path  = "/var/certificate_%[1]s"
    private_key_path  = "/var/private_key_%[1]s"
  }
}`,
	// no saml_settings for saml
	`resource "grafana_sso_settings" "sso_settings" {
  provider_name = "saml"
}`,
	// oauth2_settings instead of saml_settings for saml
	`resource "grafana_sso_settings" "sso_settings" {
  provider_name = "saml"
  oauth2_settings {
    client_id     = "client_id"
    client_secret = "client_secret"
  }
}`,
}

var testConfigsWithValidationErrors = []string{
	// no token_url provided for azuread
	`resource "grafana_sso_settings" "azure_sso_settings" {
	  provider_name = "azuread"
	  oauth2_settings {
	    client_id = "client_id"
	    auth_url  = "https://login.microsoftonline.com/12345/oauth2/v2.0/authorize"
	  }
	}`,
	// api_url is not empty for azuread
	`resource "grafana_sso_settings" "azure_sso_settings" {
	provider_name = "azuread"
	oauth2_settings {
		client_id = "client_id"
	  	auth_url  = "https://login.microsoftonline.com/12345/oauth2/v2.0/authorize"
	  	token_url = "https://login.microsoftonline.com/12345/oauth2/v2.0/token"
		api_url   = "https://login.microsoftonline.com/12345/oauth2/v2.0/user"
	}
	}`,
	// token_url is not a valid url for azuread
	`resource "grafana_sso_settings" "azure_sso_settings" {
	provider_name = "azuread"
	oauth2_settings {
		client_id = "client_id"
	  	auth_url  = "https://login.microsoftonline.com/12345/oauth2/v2.0/authorize"
	  	token_url = "this-is-an-invalid-url"
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
	// mixed path and value are configured for saml for certificate and private_key
	`resource "grafana_sso_settings" "saml_sso_settings" {
  provider_name = "saml"
  saml_settings {
    certificate_path = "/valid/certificate/path"
    private_key = "this-is-a-valid-private-key"
    idp_metadata_path = "/path/to/metadata"
  }
}`,
	// missing idp_metadata for saml
	`resource "grafana_sso_settings" "saml_sso_settings" {
  provider_name = "saml"
  saml_settings {
    certificate = "this-is-a-valid-certificate"
    private_key = "this-is-a-valid-private-key"
  }
}`,
	// missing value for client_secret
	`resource "grafana_sso_settings" "saml_sso_settings" {
	provider_name = "saml"
	saml_settings {
		certificate = "this-is-a-valid-certificate"
		private_key = "this-is-a-valid-private-key"
		idp_metadata_path = "/path/to/metadata"
		client_id = "client_id"
		client_secret = ""
		token_url = "https://myidp.com/oauth/token"
	}
}`,
	// org_attribute_path is not empty for AzureAD
	`resource "grafana_sso_settings" "azure_sso_settings" {
		provider_name = "azuread"
		oauth2_settings {
			client_id = "client_id"
			auth_url  = "https://login.microsoftonline.com/12345/oauth2/v2.0/authorize"
			token_url = "https://login.microsoftonline.com/12345/oauth2/v2.0/token"
			org_attribute_path = "org"
		}
	}`,
	// org_mapping is configured but org_attribute_path is missing for Okta
	`resource "grafana_sso_settings" "okta_sso_settings" {
  		provider_name = "okta"
  		oauth2_settings {
    		client_id = "client_id"
    		auth_url  = "https://tenantid123.okta.com/oauth2/v1/auth"
    		token_url = "https://tenantid123.okta.com/oauth2/v1/token"
			api_url = "https://tenantid123.okta.com/oauth2/v1/userinfo"
			org_mapping = "[\"Group A:1:Editor\",\"Group A:2:Admin\"]"
  		}
	}`,
	// org_attribute_path is configured but org_mapping is missing for Generic OAuth
	`resource "grafana_sso_settings" "generic_oauth_sso_settings" {
		provider_name = "generic_oauth"
		oauth2_settings {
		  client_id = "client_id"
		  auth_url  = "https://tenantid123.okta.com/oauth2/v1/auth"
		  token_url = "https://tenantid123.okta.com/oauth2/v1/token"
		  api_url = "https://tenantid123.okta.com/oauth2/v1/userinfo"
		  org_attribute_path = "groups"
		}
  }`,
}
