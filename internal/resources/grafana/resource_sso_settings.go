package grafana

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceSSOSettings() *schema.Resource {
	return &schema.Resource{

		Description: `
Manages Grafana SSO Settings for OAuth2, SAML and LDAP.

* [Official documentation](https://grafana.com/docs/grafana/latest/setup-grafana/configure-security/configure-authentication/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/sso-settings/)
`,

		CreateContext: CreateSSOSettings,
		ReadContext:   ReadSSOSettings,
		UpdateContext: UpdateSSOSettings,
		DeleteContext: DeleteSSOSettings,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"provider": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the SSO provider.",
			},
			"settings": {
				Type:        schema.TypeMap,
				Required:    true,
				Description: "The SSO settings set.",
			},
		},
	}
}

func CreateSSOSettings(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return nil
}

func ReadSSOSettings(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return nil
}

func UpdateSSOSettings(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return nil
}

func DeleteSSOSettings(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return nil
}
