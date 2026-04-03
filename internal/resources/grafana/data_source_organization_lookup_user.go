package grafana

import (
	"context"

	"github.com/grafana/grafana-openapi-client-go/client/org"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func datasourceOrganizationLookupUser() *common.DataSource {
	schema := &schema.Resource{
		Description: `
* [Official documentation](https://grafana.com/docs/grafana/latest/administration/user-management/server-user-management/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/org/#get-all-users-within-the-current-organization-lookup)
`,
		ReadContext: dataSourceOrganizationLookupUserRead,
		Schema: map[string]*schema.Schema{
			"org_id": orgIDAttribute(),
			"login": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The username of the Grafana user.",
			},
			"user_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The numerical ID of the Grafana user.",
			},
		},
	}
	return common.NewLegacySDKDataSource(common.CategoryGrafanaOSS, "grafana_organization_lookup_user", schema)
}

func dataSourceOrganizationLookupUserRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client, orgID := OAPIClientFromNewOrgResource(meta, d)

	var resp interface {
		GetPayload() []*models.UserLookupDTO
	}

	login := d.Get("login").(string)

	params := org.NewGetOrgUsersForCurrentOrgLookupParams().WithQuery(&login)
	resp, err := client.Org.GetOrgUsersForCurrentOrgLookup(params)
	if err != nil {
		return diag.FromErr(err)
	}

	if len(resp.GetPayload()) == 0 {
		return diag.Errorf("organization user not found with query: %q", login)
	}

	for _, user := range resp.GetPayload() {
		if user.Login == login {
			d.Set("user_id", user.UserID)
			d.Set("login", user.Login)
			d.SetId(MakeOrgResourceID(orgID, user.UserID))
			return nil
		}
	}

	return diag.Errorf("no organization user found with login: %q (users returned: %d)", login, len(resp.GetPayload()))
}
