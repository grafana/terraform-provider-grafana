package grafana

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
)

const (
	providerKey       = "provider_name"
	oauth2SettingsKey = "oauth2_settings"
	samlSettingsKey   = "saml_settings"
	ldapSettingsKey   = "ldap_settings"
	customFieldsKey   = "custom"
)

func resourceSSOSettings() *common.Resource {
	schema := &schema.Resource{

		Description: `
Manages Grafana SSO Settings for OAuth2, SAML and LDAP. Support for LDAP is currently in preview, it will be available in Grafana starting with v11.3.

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
				Description:  "The name of the SSO provider. Supported values: github, gitlab, google, azuread, okta, generic_oauth, saml, ldap.",
				ValidateFunc: validation.StringInSlice([]string{"github", "gitlab", "google", "azuread", "okta", "generic_oauth", "saml", "ldap"}, false),
			},
			oauth2SettingsKey: {
				Type:          schema.TypeSet,
				Optional:      true,
				MaxItems:      1,
				MinItems:      0,
				Description:   "The OAuth2 settings set. Required for github, gitlab, google, azuread, okta, generic_oauth providers.",
				Elem:          oauth2SettingsSchema,
				ConflictsWith: []string{samlSettingsKey, ldapSettingsKey},
			},
			samlSettingsKey: {
				Type:          schema.TypeSet,
				Optional:      true,
				MaxItems:      1,
				MinItems:      0,
				Description:   "The SAML settings set. Required for the saml provider.",
				Elem:          samlSettingsSchema,
				ConflictsWith: []string{oauth2SettingsKey, ldapSettingsKey},
			},
			ldapSettingsKey: {
				Type:          schema.TypeSet,
				Optional:      true,
				MaxItems:      1,
				MinItems:      0,
				Description:   "The LDAP settings set. Required for the ldap provider.",
				Elem:          ldapSettingsSchema,
				ConflictsWith: []string{oauth2SettingsKey, samlSettingsKey},
			},
		},
	}

	return common.NewLegacySDKResource(
		common.CategoryGrafanaOSS,
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
			Description: "The user information endpoint of your OAuth2 provider. Required for okta and generic_oauth providers.",
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
		"org_mapping": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "List of comma- or space-separated Organization:OrgIdOrOrgName:Role mappings. Organization can be * meaning “All users”. Role is optional and can have the following values: None, Viewer, Editor or Admin.",
		},
		"org_attribute_path": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: `JMESPath expression to use for the organization mapping lookup from the user ID token. The extracted list will be used for the organization mapping (to match "Organization" in the "org_mapping"). Only applicable to Generic OAuth and Okta.`,
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
			Description: "Custom fields to configure for OAuth2 such as the `force_use_graph_api` and `domain_hint` for Azure AD.",
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
		},
	},
}

var samlSettingsSchema = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"enabled": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     true,
			Description: "Define whether this configuration is enabled for SAML.",
		},
		"name": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Name used to refer to the SAML authentication.",
		},
		"entity_id": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The entity ID is a globally unique identifier for the service provider. It is used to identify the service provider to the identity provider. Defaults to the URL of the Grafana instance if not set.",
		},
		"single_logout": {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Whether SAML Single Logout is enabled.",
		},
		"allow_sign_up": {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Whether to allow new Grafana user creation through SAML login. If set to false, then only existing Grafana users can log in with SAML.",
		},
		"auto_login": {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Whether SAML auto login is enabled.",
		},
		"allow_idp_initiated": {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Whether SAML IdP-initiated login is allowed.",
		},
		"certificate": {
			Type:        schema.TypeString,
			Optional:    true,
			Sensitive:   true,
			Description: "Base64-encoded string for the SP X.509 certificate.",
		},
		"certificate_path": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Path for the SP X.509 certificate.",
		},
		"private_key": {
			Type:        schema.TypeString,
			Optional:    true,
			Sensitive:   true,
			Description: "Base64-encoded string for the SP private key.",
		},
		"private_key_path": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Path for the SP private key.",
		},
		"signature_algorithm": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Signature algorithm used for signing requests to the IdP. Supported values are rsa-sha1, rsa-sha256, rsa-sha512.",
		},
		"idp_metadata": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Base64-encoded string for the IdP SAML metadata XML.",
		},
		"idp_metadata_path": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Path for the IdP SAML metadata XML.",
		},
		"idp_metadata_url": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "URL for the IdP SAML metadata XML.",
		},
		"max_issue_delay": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Duration, since the IdP issued a response and the SP is allowed to process it. For example: 90s, 1h.",
		},
		"metadata_valid_duration": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Duration, for how long the SP metadata is valid. For example: 48h, 5d.",
		},
		"relay_state": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Relay state for IdP-initiated login. Should match relay state configured in IdP.",
		},
		"assertion_attribute_name": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Friendly name or name of the attribute within the SAML assertion to use as the user name. Alternatively, this can be a template with variables that match the names of attributes within the SAML assertion.",
		},
		"assertion_attribute_login": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Friendly name or name of the attribute within the SAML assertion to use as the user login handle.",
		},
		"assertion_attribute_email": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Friendly name or name of the attribute within the SAML assertion to use as the user email.",
		},
		"assertion_attribute_groups": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Friendly name or name of the attribute within the SAML assertion to use as the user groups.",
		},
		"assertion_attribute_role": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Friendly name or name of the attribute within the SAML assertion to use as the user roles.",
		},
		"assertion_attribute_org": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Friendly name or name of the attribute within the SAML assertion to use as the user organization.",
		},
		"allowed_organizations": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "List of comma- or space-separated organizations. User should be a member of at least one organization to log in.",
		},
		"org_mapping": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "List of comma- or space-separated Organization:OrgId:Role mappings. Organization can be * meaning “All users”. Role is optional and can have the following values: Viewer, Editor or Admin.",
		},
		"role_values_none": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "List of comma- or space-separated roles which will be mapped into the None role.",
		},
		"role_values_viewer": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "List of comma- or space-separated roles which will be mapped into the Viewer role.",
		},
		"role_values_editor": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "List of comma- or space-separated roles which will be mapped into the Editor role.",
		},
		"role_values_admin": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "List of comma- or space-separated roles which will be mapped into the Admin role.",
		},
		"role_values_grafana_admin": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "List of comma- or space-separated roles which will be mapped into the Grafana Admin (Super Admin) role.",
		},
		"name_id_format": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The Name ID Format to request within the SAML assertion. Defaults to urn:oasis:names:tc:SAML:2.0:nameid-format:transient",
		},
		"skip_org_role_sync": {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Prevent synchronizing users’ organization roles from your IdP.",
		},
		"client_id": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The client Id of your OAuth2 app.",
		},
		"client_secret": {
			Type:     schema.TypeString,
			Optional: true,
			// Sensitive:   true,
			Description: "The client secret of your OAuth2 app.",
		},
		"token_url": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The token endpoint of your OAuth2 provider. Required for Azure AD providers.",
		},
		"force_use_graph_api": {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "If enabled, Grafana will fetch groups from Microsoft Graph API instead of using the groups claim from the ID token.",
		},
	},
}

var ldapSettingsSchema = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"enabled": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     true,
			Description: "Define whether this configuration is enabled for LDAP.",
		},
		"allow_sign_up": {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Whether to allow new Grafana user creation through LDAP login. If set to false, then only existing Grafana users can log in with LDAP.",
		},
		"skip_org_role_sync": {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Prevent synchronizing users’ organization roles from LDAP.",
		},
		"config": {
			Type:        schema.TypeList,
			Required:    true,
			MaxItems:    1,
			MinItems:    1,
			Description: "The LDAP configuration.",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"servers": {
						Type:        schema.TypeList,
						Required:    true,
						MinItems:    1,
						Description: "The LDAP servers configuration.",
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"host": {
									Type:        schema.TypeString,
									Required:    true,
									Description: "The LDAP server host.",
								},
								"port": {
									Type:        schema.TypeInt,
									Optional:    true,
									Description: "The LDAP server port.",
								},
								"use_ssl": {
									Type:        schema.TypeBool,
									Optional:    true,
									Description: "Set to true if LDAP server should use an encrypted TLS connection (either with STARTTLS or LDAPS).",
								},
								"start_tls": {
									Type:        schema.TypeBool,
									Optional:    true,
									Description: "If set to true, use LDAP with STARTTLS instead of LDAPS.",
								},
								"tls_ciphers": {
									Type:        schema.TypeList,
									Optional:    true,
									Description: "Accepted TLS ciphers. For a complete list of supported ciphers, refer to: https://go.dev/src/crypto/tls/cipher_suites.go.",
									Elem: &schema.Schema{
										Type: schema.TypeString,
									},
								},
								"min_tls_version": {
									Type:        schema.TypeString,
									Optional:    true,
									Description: "Minimum TLS version allowed. Accepted values are: TLS1.2, TLS1.3.",
								},
								"ssl_skip_verify": {
									Type:        schema.TypeBool,
									Optional:    true,
									Description: "If set to true, the SSL cert validation will be skipped.",
								},
								"root_ca_cert": {
									Type:        schema.TypeString,
									Optional:    true,
									Description: "The path to the root CA certificate.",
								},
								"root_ca_cert_value": {
									Type:        schema.TypeList,
									Optional:    true,
									Description: "The Base64 encoded values of the root CA certificates.",
									Elem: &schema.Schema{
										Type: schema.TypeString,
									},
								},
								"client_cert": {
									Type:        schema.TypeString,
									Optional:    true,
									Description: "The path to the client certificate.",
								},
								"client_cert_value": {
									Type:        schema.TypeString,
									Optional:    true,
									Description: "The Base64 encoded value of the client certificate.",
								},
								"client_key": {
									Type:        schema.TypeString,
									Optional:    true,
									Sensitive:   true,
									Description: "The path to the client private key.",
								},
								"client_key_value": {
									Type:        schema.TypeString,
									Optional:    true,
									Sensitive:   true,
									Description: "The Base64 encoded value of the client private key.",
								},
								"bind_dn": {
									Type:        schema.TypeString,
									Optional:    true,
									Description: "The search user bind DN.",
								},
								"bind_password": {
									Type:        schema.TypeString,
									Optional:    true,
									Sensitive:   true,
									Description: "The search user bind password.",
								},
								"timeout": {
									Type:        schema.TypeInt,
									Optional:    true,
									Description: "The timeout in seconds for connecting to the LDAP host.",
								},
								"search_filter": {
									Type:        schema.TypeString,
									Required:    true,
									Description: "The user search filter, for example \"(cn=%s)\" or \"(sAMAccountName=%s)\" or \"(uid=%s)\".",
								},
								"search_base_dns": {
									Type:        schema.TypeList,
									Required:    true,
									MinItems:    1,
									Description: "An array of base DNs to search through.",
									Elem: &schema.Schema{
										Type: schema.TypeString,
									},
								},
								"group_search_filter": {
									Type:        schema.TypeString,
									Optional:    true,
									Description: "Group search filter, to retrieve the groups of which the user is a member (only set if memberOf attribute is not available).",
								},
								"group_search_base_dns": {
									Type:        schema.TypeList,
									Optional:    true,
									Description: "An array of the base DNs to search through for groups. Typically uses ou=groups.",
									Elem: &schema.Schema{
										Type: schema.TypeString,
									},
								},
								"group_search_filter_user_attribute": {
									Type:        schema.TypeString,
									Optional:    true,
									Description: "The %s in the search filter will be replaced with the attribute defined in this field.",
								},
								"attributes": {
									Type:        schema.TypeMap,
									Optional:    true,
									Description: "The LDAP server attributes. The following attributes can be configured: email, member_of, name, surname, username.",
									Elem: &schema.Schema{
										Type: schema.TypeString,
									},
								},
								"group_mappings": {
									Type:        schema.TypeList,
									Optional:    true,
									Description: "For mapping an LDAP group to a Grafana organization and role.",
									Elem: &schema.Resource{
										Schema: map[string]*schema.Schema{
											"group_dn": {
												Type:        schema.TypeString,
												Required:    true,
												Description: "LDAP distinguished name (DN) of LDAP group. If you want to match all (or no LDAP groups) then you can use wildcard (\"*\").",
											},
											"org_role": {
												Type:        schema.TypeString,
												Required:    true,
												Description: "Assign users of group_dn the organization role Admin, Editor, or Viewer.",
											},
											"org_id": {
												Type:        schema.TypeInt,
												Optional:    true,
												Description: "The Grafana organization database id.",
											},
											"grafana_admin": {
												Type:        schema.TypeBool,
												Optional:    true,
												Description: "If set to true, it makes the user of group_dn Grafana server admin.",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	},
}

func ReadSSOSettings(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _ := OAPIGlobalClient(meta) // TODO: Check error. This resource works with a token. Is it org-scoped?

	provider := d.Id()

	settingsKey, err := getSettingsKey(provider)
	if err != nil {
		return diag.FromErr(err)
	}
	settingsSchema, err := getSettingsSchema(provider)
	if err != nil {
		return diag.FromErr(err)
	}

	resp, err := client.SsoSettings.GetProviderSettings(provider)
	if err != nil {
		return diag.Errorf("failed to get the SSO settings for provider %s: %v", provider, err)
	}

	payload := resp.GetPayload()

	settingsFromAPI, err := getSettingsForTF(payload)
	if err != nil {
		return diag.FromErr(err)
	}

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

	if isOAuth2Provider(provider) {
		if _, ok := settingsSnake[customFieldsKey]; !ok {
			settingsSnake[customFieldsKey] = make(map[string]any)
		}
	}

	for k, v := range settingsFromAPI {
		key := toSnake(k)

		if _, ok := settingsSchema.Schema[key]; ok {
			val, ok := getSettingOk(key, settingsFromTfState)

			// If the terraform state is empty we return all settings from the API, this is useful when
			// importing existing sso settings into terraform. Otherwise, the API response may return fields
			// that don't exist in the terraform state. We ignore them because they are not managed by terraform.
			if ok || len(settingsFromTfState) == 0 {
				switch {
				case provider == "ldap" && key == "config":
					// special case for LDAP as the settings are nested
					settingsSnake[key] = getSettingsWithSecretsForLdap(val, v)
				case isSecret(key):
					// secrets are not exposed by the SSO Settings API, we get them from the terraform state
					settingsSnake[key] = val
				case !isIgnored(provider, key):
					// some fields are returned by the API, but they are read only, so we ignore them
					settingsSnake[key] = v
				}
			}
		} else if isOAuth2Provider(provider) {
			if _, ok := customFieldsFromTfState[key]; ok {
				settingsSnake[customFieldsKey].(map[string]any)[key] = v
			} else if _, ok := customFieldsFromTfState[k]; ok {
				// for covering the case when a custom field name is in camelCase
				settingsSnake[customFieldsKey].(map[string]any)[k] = v
			}
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

	settingsKey, err := getSettingsKey(provider)
	if err != nil {
		return diag.FromErr(err)
	}

	settings, err := getSettingsFromResourceData(d, settingsKey)
	if err != nil {
		return diag.FromErr(err)
	}

	settings = getSettingsForAPI(provider, settings)

	if isOAuth2Provider(provider) {
		diags := validateOAuth2CustomFields(settings)
		if diags != nil {
			return diags
		}

		settings = mergeCustomFields(settings)
	}

	err = validateSSOSettings(provider, settings)
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

func isOAuth2Provider(provider string) bool {
	switch provider {
	case "github", "gitlab", "google", "azuread", "okta", "generic_oauth":
		return true
	}
	return false
}

func isSamlProvider(provider string) bool {
	return provider == "saml"
}

func isLdapProvider(provider string) bool {
	return provider == "ldap"
}

func getSettingsKey(provider string) (string, error) {
	if isOAuth2Provider(provider) {
		return oauth2SettingsKey, nil
	}
	if isSamlProvider(provider) {
		return samlSettingsKey, nil
	}
	if isLdapProvider(provider) {
		return ldapSettingsKey, nil
	}

	return "", fmt.Errorf("no settings key found for provider %s", provider)
}

func getSettingsSchema(provider string) (*schema.Resource, error) {
	if isOAuth2Provider(provider) {
		return oauth2SettingsSchema, nil
	}
	if isSamlProvider(provider) {
		return samlSettingsSchema, nil
	}
	if isLdapProvider(provider) {
		return ldapSettingsSchema, nil
	}

	return nil, fmt.Errorf("no settings schema found for provider %s", provider)
}

// getSettingOk mimics the terraform function schema.ResourceData.GetOk but for the nested fields inside settings
// it assumes that any empty string value from settings is not part of the tf configuration
func getSettingOk(key string, settings map[string]any) (any, bool) {
	val, ok := settings[key]
	if ok {
		stringVal, stringOk := val.(string)
		if stringOk && stringVal == "" {
			return val, false
		}
	}

	return val, ok
}

func getSettingsWithSecretsForLdap(state any, config any) any {
	secretFields := []string{"client_key", "client_key_value", "bind_password"}

	stateSlice, ok := state.([]any)
	if !ok {
		return config
	}

	configSlice, ok := config.([]any)
	if !ok {
		return config
	}

	if len(stateSlice) == 0 || len(configSlice) == 0 {
		return config
	}

	stateServers, ok := stateSlice[0].(map[string]any)["servers"].([]any)
	if !ok {
		return config
	}

	configServers, ok := configSlice[0].(map[string]any)["servers"].([]any)
	if !ok {
		return config
	}

	for i, serverRaw := range configServers {
		server := serverRaw.(map[string]any)
		for _, field := range secretFields {
			if len(stateServers) < i+1 {
				continue
			}

			secret, ok := stateServers[i].(map[string]any)[field].(string)
			if ok {
				server[field] = secret
			}
		}
	}

	return config
}

func getSettingsForTF(payload *models.GetProviderSettingsOKBody) (map[string]any, error) {
	settings, ok := payload.Settings.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid settings format: %v", payload.Settings)
	}

	if payload.Provider == "ldap" {
		// config is represented as an array in terraform
		settings["config"] = []any{settings["config"]}
	}

	if payload.Provider == "saml" {
		settings = convertNumericDurationsToStrings(settings)
	}

	return settings, nil
}

func getSettingsForAPI(provider string, settings map[string]any) map[string]any {
	if provider == "ldap" {
		config := settings["config"].([]any)

		if len(config) > 0 {
			// config is an object in API
			settings["config"] = config[0]
		}
	}

	return settings
}

func getSettingsFromResourceData(d *schema.ResourceData, settingsKey string) (map[string]any, error) {
	settingsList := d.Get(settingsKey).(*schema.Set).List()

	if len(settingsList) == 0 {
		return nil, fmt.Errorf("no settings found for the provider %s", d.Get(providerKey).(string))
	}

	if len(settingsList) == 1 {
		return settingsList[0].(map[string]any), nil
	}

	// sometimes the settings set contains some empty items that we want to ignore
	// we are only interested in the settings that have one of the following:
	// - the client_id set because the client_id is a required field for OAuth2 providers
	// - a non-empty config for LDAP
	// - the private_key or private_key_path set because those are required fields for SAML
	for _, item := range settingsList {
		settings := item.(map[string]any)

		clientID, ok := settings["client_id"]
		if ok && clientID != "" {
			return settings, nil
		}

		config, okConfig := settings["config"].([]any)
		if okConfig && len(config) > 0 {
			return settings, nil
		}

		privateKey, okPrivateKey := settings["private_key"]
		privateKeyPath, okPrivateKeyPath := settings["private_key_path"]
		if (okPrivateKey && privateKey != "") || (okPrivateKeyPath && privateKeyPath != "") {
			return settings, nil
		}
	}

	return nil, fmt.Errorf("no valid settings found for the provider %s", d.Get(providerKey).(string))
}

type validateFunc func(settingsMap map[string]any, provider string) error

var validationsByProvider = map[string][]validateFunc{
	"azuread": {
		ssoValidateNotEmpty("auth_url"),
		ssoValidateNotEmpty("token_url"),
		ssoValidateEmpty("api_url"),
		ssoValidateEmpty("org_attribute_path"),
		ssoValidateURL("auth_url"),
		ssoValidateURL("token_url"),
	},
	"generic_oauth": {
		ssoValidateNotEmpty("auth_url"),
		ssoValidateNotEmpty("token_url"),
		ssoValidateNotEmpty("api_url"),
		ssoValidateURL("auth_url"),
		ssoValidateURL("token_url"),
		ssoValidateURL("api_url"),
		ssoValidateInterdependencyXOR("org_attribute_path", "org_mapping"),
	},
	"okta": {
		ssoValidateNotEmpty("auth_url"),
		ssoValidateNotEmpty("token_url"),
		ssoValidateNotEmpty("api_url"),
		ssoValidateURL("auth_url"),
		ssoValidateURL("token_url"),
		ssoValidateURL("api_url"),
		ssoValidateInterdependencyXOR("org_attribute_path", "org_mapping"),
	},
	"github": {
		ssoValidateEmpty("auth_url"),
		ssoValidateEmpty("token_url"),
		ssoValidateEmpty("api_url"),
		ssoValidateEmpty("org_attribute_path"),
	},
	"gitlab": {
		ssoValidateEmpty("auth_url"),
		ssoValidateEmpty("token_url"),
		ssoValidateEmpty("api_url"),
		ssoValidateEmpty("org_attribute_path"),
	},
	"google": {
		ssoValidateEmpty("auth_url"),
		ssoValidateEmpty("token_url"),
		ssoValidateEmpty("api_url"),
		ssoValidateEmpty("org_attribute_path"),
	},
	"saml": {
		ssoValidateInterdependencyXOR("certificate", "private_key"),
		ssoValidateInterdependencyXOR("certificate_path", "private_key_path"),
		ssoValidateOnlyOneOf("idp_metadata", "idp_metadata_path", "idp_metadata_url"),
		ssoValidateURL("idp_metadata_url"),
		ssoValidateInterdependencyXOR("client_id", "client_secret", "token_url"),
		ssoValidateURL("token_url"),
	},
	// no client side validations for LDAP because the settings are nested
	"ldap": {},
}

func validateSSOSettings(provider string, settings map[string]any) error {
	validators := validationsByProvider[provider]
	for _, validateF := range validators {
		err := validateF(settings, provider)
		if err != nil {
			return err
		}
	}

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
	secretFieldPatterns := []string{"secret", "certificate", "private"}

	for _, v := range secretFieldPatterns {
		if strings.Contains(strings.ToLower(fieldName), strings.ToLower(v)) {
			return true
		}
	}
	return false
}

func convertNumericDurationsToStrings(settings map[string]any) map[string]any {
	newSettings := make(map[string]any)

	for key, value := range settings {
		if isDuration(key) {
			switch value := value.(type) {
			case float64:
				newSettings[key] = strconv.FormatFloat(value, 'f', -1, 64)
			default:
				newSettings[key] = value
			}
		} else {
			newSettings[key] = value
		}
	}

	return newSettings
}

func isDuration(fieldName string) bool {
	durationFieldPatterns := []string{
		"max_issue_delay", "maxIssueDelay",
		"metadata_valid_duration", "metadataValidDuration",
	}

	for _, v := range durationFieldPatterns {
		if strings.Contains(strings.ToLower(fieldName), strings.ToLower(v)) {
			return true
		}
	}
	return false
}

func validateOAuth2CustomFields(settings map[string]any) diag.Diagnostics {
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

func ssoValidateNotEmpty(key string) validateFunc {
	return func(settingsMap map[string]any, provider string) error {
		if settingsMap[key] == "" {
			return fmt.Errorf("%s must be set for the provider %s", key, provider)
		}

		return nil
	}
}

func ssoValidateEmpty(key string) validateFunc {
	return func(settingsMap map[string]any, provider string) error {
		if settingsMap[key].(string) != "" {
			return fmt.Errorf("%s must be empty for the provider %s", key, provider)
		}

		return nil
	}
}

func ssoValidateURL(key string) validateFunc {
	return func(settingsMap map[string]any, provider string) error {
		if settingsMap[key].(string) != "" && !isValidURL(settingsMap[key].(string)) {
			return fmt.Errorf("%s must be a valid http/https URL for the provider %s", key, provider)
		}
		return nil
	}
}

func ssoValidateOnlyOneOf(keys ...string) validateFunc {
	return func(settingsMap map[string]any, provider string) error {
		configuredKeys := 0

		for _, key := range keys {
			if settingsMap[key].(string) != "" {
				configuredKeys++
			}
		}

		if configuredKeys != 1 {
			return fmt.Errorf("exactly one of %v must be configured for provider %s", keys, provider)
		}

		return nil
	}
}

// XOR validation of variables
func ssoValidateInterdependencyXOR(keys ...string) validateFunc {
	return func(settingsMap map[string]any, provider string) error {
		configuredKeys := 0
		nonConfiguredKeys := 0

		for _, key := range keys {
			if settingsMap[key].(string) != "" {
				configuredKeys++
			} else {
				nonConfiguredKeys++
			}
		}

		if configuredKeys != len(keys) && nonConfiguredKeys != len(keys) {
			return fmt.Errorf("all variables in %v must be configured or empty for provider %s", keys, provider)
		}

		return nil
	}
}
