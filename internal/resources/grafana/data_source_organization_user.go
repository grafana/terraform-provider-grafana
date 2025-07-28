package grafana

import (
	"context"

	"github.com/grafana/grafana-openapi-client-go/client/org"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func datasourceOrganizationUser() *common.DataSource {
	schema := &schema.Resource{
		Description: `
* [Official documentation](https://grafana.com/docs/grafana/latest/administration/user-management/server-user-management/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/org/#get-all-users-within-the-current-organization-lookup)
`,
		ReadContext: dataSourceOrganizationUserRead,
		Schema: map[string]*schema.Schema{
			"org_id": orgIDAttribute(),
			"email": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "The email address of the Grafana user.",
			},
			"login": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "The username of the Grafana user.",
			},
			"user_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The numerical ID of the Grafana user.",
			},
		},
	}
	return common.NewLegacySDKDataSource(common.CategoryGrafanaOSS, "grafana_organization_user", schema)
}

func dataSourceOrganizationUserRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID := OAPIClientFromNewOrgResource(meta, d)

	var resp interface {
		GetPayload() []*models.UserLookupDTO
	}

	emailOrLogin := d.Get("email").(string)
	if emailOrLogin == "" {
		emailOrLogin = d.Get("login").(string)
	}
	if emailOrLogin == "" {
		return diag.Errorf("must specify one of email or login")
	}

	params := org.NewGetOrgUsersForCurrentOrgLookupParams().WithQuery(&emailOrLogin)
	resp, err := client.Org.GetOrgUsersForCurrentOrgLookup(params)
	if err != nil {
		return diag.FromErr(err)
	}

	// Make sure that exactly 1 user was returned
	if len(resp.GetPayload()) > 1 {
		return diag.Errorf("ambiguous query when reading organization user, multiple users returned by query: %q", emailOrLogin)
	} else if len(resp.GetPayload()) == 0 {
		return diag.Errorf("organization user not found with query: %q", emailOrLogin)
	}

	user := resp.GetPayload()[0]
	d.Set("user_id", user.UserID)
	d.Set("login", user.Login)

	d.SetId(MakeOrgResourceID(orgID, user.UserID))
	return nil
}
