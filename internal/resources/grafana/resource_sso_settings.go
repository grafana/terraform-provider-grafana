package grafana

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/grafana/grafana-openapi-client-go/models"
)

const (
	providerKey = "provider_name"
	settingsKey = "settings"
)

func ResourceSSOSettings() *schema.Resource {
	return &schema.Resource{

		Description: `
Manages Grafana SSO Settings for OAuth2, SAML and LDAP.

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
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the SSO provider.",
			},
			settingsKey: {
				Type:        schema.TypeSet,
				Required:    true,
				MaxItems:    1,
				MinItems:    1,
				Description: "The SSO settings set.",
				Elem:        settingsSchema,
			},
		},
	}
}

var settingsSchema = &schema.Resource{
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
			Computed:    true,
			Description: "The client secret of your OAuth2 app.",
			// suppress this because the API returns this field redacted
			DiffSuppressFunc: func(k, oldValue, newValue string, d *schema.ResourceData) bool {
				return true
			},
			DiffSuppressOnRefresh: true,
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
			Description: "The authorization endpoint of your OAuth2 provider.",
		},
		"auth_style": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "It determines how client_id and client_secret are sent to Oauth2 provider. Possible values are AutoDetect, InParams, InHeader. Default is AutoDetect.",
		},
		"token_url": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The token endpoint of your OAuth2 provider.",
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
			Description: "List of comma- or space-separated groups. The user should be a member of at least one group to log in. If you configure allowed_groups, you must also configure groups_attribute_path.",
		},
		"api_url": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The user information endpoint of your OAuth2 provider.",
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
			Description: "Name of the key to use for user email lookup within the attributes map of OAuth2 ID token.",
		},
		"email_attribute_path": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "JMESPath expression to use for user email lookup from the user information.",
		},
		"name_attribute_path": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "JMESPath expression to use for user name lookup from the user ID token. This name will be used as the user’s display name.",
		},
		"login_attribute_path": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "JMESPath expression to use for user login lookup from the user ID token.",
		},
		"id_token_attribute_name": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The name of the key used to extract the ID token from the returned OAuth2 token.",
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
			Description: "The URL used to query for Team Ids. If not set, the default value is /teams. If you configure teams_url, you must also configure team_ids_attribute_path.",
		},
		"team_ids_attribute_path": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The JMESPath expression to use for Grafana Team Id lookup within the results returned by the teams_url endpoint.",
		},
		"team_ids": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "String list of Team Ids. If set, the user must be a member of one of the given teams to log in. If you configure team_ids, you must also configure teams_url and team_ids_attribute_path.",
		},
	},
}

func ReadSSOSettings(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := OAPIGlobalClient(meta)

	provider := d.Get(providerKey).(string)

	resp, err := client.SsoSettings.GetProviderSettings(provider)
	if err != nil {
		return diag.Errorf("failed to get the SSO settings for provider %s: %v", provider, err)
	}

	payload := resp.GetPayload()

	settingsSnake := make(map[string]any)
	for k, v := range payload.Settings.(map[string]any) {
		key := toSnake(k)
		if _, ok := settingsSchema.Schema[key]; ok {
			settingsSnake[key] = v
		}
	}

	var settings []interface{}
	settings = append(settings, settingsSnake)

	d.Set(providerKey, payload.Provider)
	d.Set(settingsKey, settings)

	return nil
}

func UpdateSSOSettings(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := OAPIGlobalClient(meta)

	provider := d.Get(providerKey).(string)

	settings := make(map[string]any)
	settingsList := d.Get(settingsKey).(*schema.Set).List()
	if len(settingsList) > 0 {
		settings = settingsList[0].(map[string]any)
	}

	ssoSettings := models.UpdateProviderSettingsParamsBody{
		Provider: provider,
		Settings: settings,
	}

	_, err := client.SsoSettings.UpdateProviderSettings(provider, &ssoSettings)
	if err != nil {
		return diag.Errorf("failed to create the SSO settings for provider %s: %v", provider, err)
	}

	d.SetId(provider)

	return ReadSSOSettings(ctx, d, meta)
}

func DeleteSSOSettings(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := OAPIGlobalClient(meta)

	provider := d.Get(providerKey).(string)

	_, err := client.SsoSettings.RemoveProviderSettings(provider)
	if err != nil {
		return diag.Errorf("failed to remove the SSO settings for provider %s: %v", provider, err)
	}

	return nil
}

// copied and adapted from https://github.com/grafana/grafana/blob/main/pkg/services/featuremgmt/strcase/snake.go#L70
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
