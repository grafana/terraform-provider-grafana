package grafana

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v2/internal/common"
)

const (
	providerKey       = "provider_name"
	oauth2SettingsKey = "oauth2_settings"
	customFieldsKey   = "custom"
)

func resourceSSOSettings() *common.Resource {
	schema := &schema.Resource{

		Description: `
Manages Grafana SSO Settings for OAuth2. SAML support will be added soon.

* [Official documentation](https://grafana.com/docs/grafana/latest/setup-grafana/configure-security/configure-authentication/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/sso-settings/)
`,

		CreateContext: UpdateSSOSettings,
		ReadContext:   ReadSSOSettings,
		UpdateContext: UpdateSSOSettings,
		DeleteContext: DeleteSSOSettings,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			providerKey: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "The name of the SSO provider. Supported values: github, gitlab, google, azuread, okta, generic_oauth.",
				ValidateFunc: validation.StringInSlice([]string{"github", "gitlab", "google", "azuread", "okta", "generic_oauth"}, false),
			},
			oauth2SettingsKey: {
				Type:        schema.TypeSet,
				Required:    true,
				MaxItems:    1,
				MinItems:    1,
				Description: "The SSO settings set.",
				Elem:        oauth2SettingsSchema,
			},
		},
	}

	return common.NewLegacySDKResource(
		"grafana_sso_settings",
		orgResourceIDString("provider"),
		schema,
	)
}

var oauth2SettingsSchema = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"enabled": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     true,
			Description: "Define whether this configuration is enabled for the specified provider.",
		},
		"client_id": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "The client Id of your OAuth2 app.",
		},
		"client_secret": {
			Type:        schema.TypeString,
			Optional:    true,
			Sensitive:   true,
			Description: "The client secret of your OAuth2 app.",
		},
		"allowed_organizations": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "List of comma- or space-separated organizations. The user should be a member of at least one organization to log in.",
		},
		"allowed_domains": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "List of comma- or space-separated domains. The user should belong to at least one domain to log in.",
		},
		"auth_url": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The authorization endpoint of your OAuth2 provider. Required for azuread, okta and generic_oauth providers.",
		},
		"auth_style": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "It determines how client_id and client_secret are sent to Oauth2 provider. Possible values are AutoDetect, InParams, InHeader. Default is AutoDetect.",
		},
		"token_url": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The token endpoint of your OAuth2 provider. Required for azuread, okta and generic_oauth providers.",
		},
		"scopes": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "List of comma- or space-separated OAuth2 scopes.",
		},
		"empty_scopes": {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "If enabled, no scopes will be sent to the OAuth2 provider.",
		},
		"allowed_groups": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "List of comma- or space-separated groups. The user should be a member of at least one group to log in. For Generic OAuth, if you configure allowed_groups, you must also configure groups_attribute_path.",
		},
		"api_url": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The user information endpoint of your OAuth2 provider. Required for azuread, okta and generic_oauth providers.",
		},
		"role_attribute_path": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "JMESPath expression to use for Grafana role lookup.",
		},
		"name": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Helpful if you use more than one identity providers or SSO protocols.",
		},
		"allow_sign_up": {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "If not enabled, only existing Grafana users can log in using OAuth.",
		},
		"auto_login": {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Log in automatically, skipping the login screen.",
		},
		"signout_redirect_url": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The URL to redirect the user to after signing out from Grafana.",
		},
		"email_attribute_name": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Name of the key to use for user email lookup within the attributes map of OAuth2 ID token. Only applicable to Generic OAuth.",
		},
		"email_attribute_path": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "JMESPath expression to use for user email lookup from the user information. Only applicable to Generic OAuth.",
		},
		"name_attribute_path": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "JMESPath expression to use for user name lookup from the user ID token. This name will be used as the user’s display name. Only applicable to Generic OAuth.",
		},
		"login_attribute_path": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "JMESPath expression to use for user login lookup from the user ID token. Only applicable to Generic OAuth.",
		},
		"id_token_attribute_name": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The name of the key used to extract the ID token from the returned OAuth2 token. Only applicable to Generic OAuth.",
		},
		"role_attribute_strict": {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "If enabled, denies user login if the Grafana role cannot be extracted using Role attribute path.",
		},
		"allow_assign_grafana_admin": {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "If enabled, it will automatically sync the Grafana server administrator role.",
		},
		"skip_org_role_sync": {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Prevent synchronizing users’ organization roles from your IdP.",
		},
		"define_allowed_groups": {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Define allowed groups.",
		},
		"define_allowed_teams_ids": {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Define allowed teams ids.",
		},
		"use_pkce": {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "If enabled, Grafana will use Proof Key for Code Exchange (PKCE) with the OAuth2 Authorization Code Grant.",
		},
		"use_refresh_token": {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "If enabled, Grafana will fetch a new access token using the refresh token provided by the OAuth2 provider.",
		},
		"tls_client_ca": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The path to the trusted certificate authority list. Is not applicable on Grafana Cloud.",
		},
		"tls_client_cert": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The path to the certificate. Is not applicable on Grafana Cloud.",
		},
		"tls_client_key": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The path to the key. Is not applicable on Grafana Cloud.",
		},
		"tls_skip_verify_insecure": {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "If enabled, the client accepts any certificate presented by the server and any host name in that certificate. You should only use this for testing, because this mode leaves SSL/TLS susceptible to man-in-the-middle attacks.",
		},
		"groups_attribute_path": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "JMESPath expression to use for user group lookup. If you configure allowed_groups, you must also configure groups_attribute_path.",
		},
		"teams_url": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The URL used to query for Team Ids. If not set, the default value is /teams. If you configure teams_url, you must also configure team_ids_attribute_path. Only applicable to Generic OAuth.",
		},
		"team_ids_attribute_path": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The JMESPath expression to use for Grafana Team Id lookup within the results returned by the teams_url endpoint. Only applicable to Generic OAuth.",
		},
		"team_ids": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "String list of Team Ids. If set, the user must be a member of one of the given teams to log in. If you configure team_ids, you must also configure teams_url and team_ids_attribute_path.",
		},
		customFieldsKey: {
			Type:        schema.TypeMap,
			Optional:    true,
			Description: "Custom fields to configure for OAuth2 such as the [force_use_graph_api](https://grafana.com/docs/grafana/latest/setup-grafana/configure-security/configure-authentication/azuread/#force-fetching-groups-from-microsoft-graph-api) field.",
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
		},
	},
}

func ReadSSOSettings(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _ := OAPIGlobalClient(meta) // TODO: Check error. This resource works with a token. Is it org-scoped?

	provider := d.Id()

	// only one of oauth2, saml, ldap settings can be provided in a resource
	// currently we implemented only the oauth2 settings
	settingsKey := oauth2SettingsKey
	settingsSchema := oauth2SettingsSchema

	resp, err := client.SsoSettings.GetProviderSettings(provider)
	if err != nil {
		return diag.Errorf("failed to get the SSO settings for provider %s: %v", provider, err)
	}

	payload := resp.GetPayload()

	var settingsFromTfState map[string]any
	settingsFromTfStateList := d.Get(settingsKey).(*schema.Set).List()
	if len(settingsFromTfStateList) > 0 {
		settingsFromTfState = settingsFromTfStateList[0].(map[string]any)
	}

	customFieldsFromTfState := make(map[string]any)
	if settingsFromTfState[customFieldsKey] != nil {
		customFieldsFromTfState = settingsFromTfState[customFieldsKey].(map[string]any)
	}

	settingsSnake := make(map[string]any)

	if _, ok := settingsSnake[customFieldsKey]; !ok {
		settingsSnake[customFieldsKey] = make(map[string]any)
	}

	for k, v := range payload.Settings.(map[string]any) {
		key := toSnake(k)

		if _, ok := settingsSchema.Schema[key]; ok {
			if isSecret(key) {
				// secrets are not exposed by the SSO Settings API, we get them from the terraform state
				if val, ok := settingsFromTfState[key]; ok {
					settingsSnake[key] = val
				}
			} else if !isIgnored(provider, key) {
				// some fields cannot be updated, but they are returned by the API, so we ignore them
				settingsSnake[key] = v
			}
		} else if _, ok := customFieldsFromTfState[key]; ok {
			settingsSnake[customFieldsKey].(map[string]any)[key] = v
		} else if _, ok := customFieldsFromTfState[k]; ok {
			// for covering the case when a custom field name is in camelCase
			settingsSnake[customFieldsKey].(map[string]any)[k] = v
		}
	}

	var settings []interface{}
	settings = append(settings, settingsSnake)

	d.Set(providerKey, payload.Provider)
	d.Set(settingsKey, settings)

	return nil
}

func UpdateSSOSettings(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _ := OAPIGlobalClient(meta) // TODO: Check error. This resource works with a token. Is it org-scoped?

	provider := d.Get(providerKey).(string)

	// only one of oauth2, saml, ldap settings can be provided in a resource
	// currently we implemented only the oauth2 settings
	settingsKey := oauth2SettingsKey

	settings, err := getSettingsFromResourceData(d, settingsKey)
	if err != nil {
		return diag.FromErr(err)
	}

	diags := validateCustomFields(settings)
	if diags != nil {
		return diags
	}

	settings = mergeCustomFields(settings)

	err = validateOAuth2Settings(provider, settings)
	if err != nil {
		return diag.FromErr(err)
	}

	ssoSettings := models.UpdateProviderSettingsParamsBody{
		Provider: provider,
		Settings: settings,
	}

	_, err = client.SsoSettings.UpdateProviderSettings(provider, &ssoSettings)
	if err != nil {
		return diag.Errorf("failed to create the SSO settings for provider %s: %v", provider, err)
	}

	d.SetId(provider)

	return ReadSSOSettings(ctx, d, meta)
}

func DeleteSSOSettings(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _ := OAPIGlobalClient(meta) // TODO: Check error. This resource works with a token. Is it org-scoped?

	provider := d.Get(providerKey).(string)

	if _, err := client.SsoSettings.RemoveProviderSettings(provider); err != nil {
		return diag.Errorf("failed to remove the SSO settings for provider %s: %v", provider, err)
	}

	return nil
}

func getSettingsFromResourceData(d *schema.ResourceData, settingsKey string) (map[string]any, error) {
	settingsList := d.Get(settingsKey).(*schema.Set).List()

	if len(settingsList) == 0 {
		return nil, fmt.Errorf("no settings found for the provider %s", d.Get(providerKey).(string))
	}

	// TODO investigate why we need this
	// sometimes the settings set contains some empty items that we want to ignore
	// we are only interested in the settings that have the client_id set because the client_id is a required field
	for _, item := range settingsList {
		settings := item.(map[string]any)
		if settings["client_id"] != "" {
			return settings, nil
		}
	}

	return nil, fmt.Errorf("no valid settings found for the provider %s", d.Get(providerKey).(string))
}

type validateFunc func(settingsMap map[string]any, provider string) error

var validationsByProvider = map[string][]validateFunc{
	"azuread": {
		validateNotEmpty("auth_url"),
		validateUrl("auth_url"),
		validateNotEmpty("token_url"),
		validateUrl("token_url"),
		validateEmpty("api_url"),
		validateUrl("token_url"),
	},
	"generic_oauth": {
		validateNotEmpty("auth_url"),
		validateNotEmpty("token_url"),
		validateNotEmpty("api_url"),
		validateUrl("auth_url"),
		validateUrl("token_url"),
		validateUrl("api_url"),
	},
	"okta": {
		validateNotEmpty("auth_url"),
		validateNotEmpty("token_url"),
		validateNotEmpty("api_url"),
		validateUrl("auth_url"),
		validateUrl("token_url"),
		validateUrl("api_url"),
	},
	"github": {
		validateEmpty("auth_url"),
		validateEmpty("token_url"),
		validateEmpty("api_url"),
	},
	"gitlab": {
		validateEmpty("auth_url"),
		validateEmpty("token_url"),
		validateEmpty("api_url"),
	},
	"google": {
		validateEmpty("auth_url"),
		validateEmpty("token_url"),
		validateEmpty("api_url"),
	},
}

func validateOAuth2Settings(provider string, settings map[string]any) error {
	validators := validationsByProvider[provider]
	for _, validateF := range validators {
		err := validateF(settings, provider)
		if err != nil {
			return err
		}
	}

	// authURL := settings["auth_url"].(string)
	// tokenURL := settings["token_url"].(string)
	// apiURL := settings["api_url"].(string)

	// switch provider {
	// case "github", "gitlab", "google":
	// 	if authURL != "" {
	// 		return fmt.Errorf("auth_url must be empty for the provider %s", provider)
	// 	}
	// 	if tokenURL != "" {
	// 		return fmt.Errorf("token_url must be empty for the provider %s", provider)
	// 	}
	// 	if apiURL != "" {
	// 		return fmt.Errorf("api_url must be empty for the provider %s", provider)
	// 	}
	// case "azuread", "generic_oauth", "okta":
	// 	if authURL == "" {
	// 		return fmt.Errorf("auth_url must be set for the provider %s", provider)
	// 	}
	// 	if !isValidURL(authURL) {
	// 		return fmt.Errorf("auth_url must be a valid http/https URL")
	// 	}
	// 	if tokenURL == "" {
	// 		return fmt.Errorf("token_url must be set for the provider %s", provider)
	// 	}
	// 	if !isValidURL(tokenURL) {
	// 		return fmt.Errorf("token_url must be a valid http/https URL")
	// 	}
	// 	if apiURL == "" {
	// 		return fmt.Errorf("api_url must be set for the provider %s", provider)
	// 	}
	// 	if !isValidURL(apiURL) {
	// 		return fmt.Errorf("api_url must be a valid http/https URL")
	// 	}
	// }

	return nil
}

// copied and adapted from https://github.com/grafana/grafana/blob/main/pkg/services/featuremgmt/strcase/snake.go#L70
//
//nolint:gocyclo
func toSnake(s string) string {
	delimiter := byte('_')

	s = strings.TrimSpace(s)
	n := strings.Builder{}
	n.Grow(len(s) + 2) // nominal 2 bytes of extra space for inserted delimiters
	for i, v := range []byte(s) {
		vIsCap := v >= 'A' && v <= 'Z'
		vIsLow := v >= 'a' && v <= 'z'
		if vIsCap {
			v += 'a'
			v -= 'A'
		}

		// treat acronyms as words, eg for JSONData -> JSON is a whole word
		if i+1 < len(s) {
			next := s[i+1]
			vIsNum := v >= '0' && v <= '9'
			nextIsCap := next >= 'A' && next <= 'Z'
			nextIsLow := next >= 'a' && next <= 'z'
			nextIsNum := next >= '0' && next <= '9'
			// add underscore if next letter case type is changed
			if (vIsCap && (nextIsLow || nextIsNum)) || (vIsLow && (nextIsCap || nextIsNum)) || (vIsNum && (nextIsCap || nextIsLow)) {
				if vIsCap && nextIsLow {
					if prevIsCap := i > 0 && s[i-1] >= 'A' && s[i-1] <= 'Z'; prevIsCap {
						n.WriteByte(delimiter)
					}
				}
				n.WriteByte(v)
				if vIsLow || vIsNum || nextIsNum {
					n.WriteByte(delimiter)
				}
				continue
			}
		}

		if v == ' ' || v == '_' || v == '-' || v == '.' {
			// replace space/underscore/hyphen/dot with delimiter
			n.WriteByte(delimiter)
		} else {
			n.WriteByte(v)
		}
	}

	return n.String()
}

func isSecret(fieldName string) bool {
	secretFieldPatterns := []string{"secret"}

	for _, v := range secretFieldPatterns {
		if strings.Contains(strings.ToLower(fieldName), strings.ToLower(v)) {
			return true
		}
	}
	return false
}

func validateCustomFields(settings map[string]any) diag.Diagnostics {
	for key := range settings[customFieldsKey].(map[string]any) {
		if _, ok := oauth2SettingsSchema.Schema[key]; ok {
			return diag.Errorf("Invalid custom field %s, the field is already defined in the settings schema", key)
		}
	}

	return nil
}

func mergeCustomFields(settings map[string]any) map[string]any {
	merged := make(map[string]any)

	for key, val := range settings {
		if key != customFieldsKey {
			merged[key] = val
		}
	}

	for key, val := range settings[customFieldsKey].(map[string]any) {
		merged[key] = val
	}

	return merged
}

func isIgnored(provider string, fieldName string) bool {
	switch provider {
	case "github", "gitlab", "google":
		switch fieldName {
		case "auth_url", "token_url", "api_url":
			return true
		}
	}
	return false
}

func isValidURL(actual string) bool {
	parsed, err := url.ParseRequestURI(actual)
	if err != nil {
		return false
	}
	return strings.HasPrefix(parsed.Scheme, "http") && parsed.Host != ""
}

func validateNotEmpty(key string) validateFunc {
	return func(settingsMap map[string]any, provider string) error {
		if settingsMap[key] == "" {
			return fmt.Errorf("%s must be set for the provider %s", key, provider)
		}

		return nil
	}
}

func validateEmpty(key string) validateFunc {
	return func(settingsMap map[string]any, provider string) error {
		if settingsMap[key].(string) != "" {
			return fmt.Errorf("%s must be empty for the provider %s", key, provider)
		}

		return nil
	}
}

func validateUrl(key string) validateFunc {
	return func(settingsMap map[string]any, provider string) error {
		if !isValidURL(settingsMap[key].(string)) {
			return fmt.Errorf("%s must be a valid http/https URL for the provider %s", key, provider)
		}
		return nil
	}
}
