package grafana

import (
	"context"

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
				Type:        schema.TypeMap,
				Required:    true,
				Description: "The SSO settings set.",
			},
		},
	}
}

func ReadSSOSettings(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := OAPIGlobalClient(meta)

	provider := d.Get(providerKey).(string)

	resp, err := client.SsoSettings.GetProviderSettings(provider)
	if err != nil {
		return diag.Errorf("failed to get the SSO settings for provider %s: %v", provider, err)
	}

	settings := resp.GetPayload()
	d.Set(providerKey, settings.Provider)
	d.Set(settingsKey, settings.Settings)

	return nil
}

func UpdateSSOSettings(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := OAPIGlobalClient(meta)

	provider := d.Get(providerKey).(string)
	settings := d.Get(settingsKey).(map[string]any)

	ssoSettings := models.SSOSettings{
		Provider: provider,
		Settings: settings,
	}

	_, err := client.SsoSettings.UpdateProviderSettings(provider, &ssoSettings)
	if err != nil {
		return diag.Errorf("failed to create the SSO settings for provider %s: %v", provider, err)
	}

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
