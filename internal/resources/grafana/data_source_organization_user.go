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
		GetPayload() []*models.OrgUserDTO
	}

	matchBy := matchByEmail
	emailOrLogin := d.Get("email").(string)
	if emailOrLogin == "" {
		emailOrLogin = d.Get("login").(string)
		matchBy = matchByLogin
	}
	if emailOrLogin == "" {
		return diag.Errorf("must specify one of email or login")
	}

	params := org.NewGetOrgUsersForCurrentOrgParams().WithQuery(&emailOrLogin)
	resp, err := client.Org.GetOrgUsersForCurrentOrg(params)
	if err != nil {
		return diag.FromErr(err)
	}

	if len(resp.GetPayload()) == 0 {
		return diag.Errorf("organization user not found with query: %q", emailOrLogin)
	}

	for _, user := range resp.GetPayload() {
		if matchBy(user, emailOrLogin) {
			d.Set("user_id", user.UserID)
			d.Set("login", user.Login)
			d.Set("email", user.Email)
			d.SetId(MakeOrgResourceID(orgID, user.UserID))
			return nil
		}
	}

	return diag.Errorf("ambiguous query when reading organization user, multiple users returned by query: %q", emailOrLogin)
}

func matchByEmail(user *models.OrgUserDTO, email string) bool {
	return user.Email == email
}

func matchByLogin(user *models.OrgUserDTO, login string) bool {
	return user.Login == login
}
