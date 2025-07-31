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

	email := d.Get("email").(string)
	login := d.Get("login").(string)

	if email == "" && login == "" {
		return diag.Errorf("must specify one of email or login")
	}

	// Use email if provided, otherwise use login
	query := email
	if query == "" {
		query = login
	}

	params := org.NewGetOrgUsersForCurrentOrgLookupParams().WithQuery(&query)
	resp, err := client.Org.GetOrgUsersForCurrentOrgLookup(params)
	if err != nil {
		return diag.FromErr(err)
	}

	users := resp.GetPayload()

	// If no users found, return error
	if len(users) == 0 {
		return diag.Errorf("organization user not found with query: %q", query)
	}

	// If exactly one user found, use it
	if len(users) == 1 {
		user := users[0]
		d.Set("user_id", user.UserID)
		d.Set("login", user.Login)
		d.SetId(MakeOrgResourceID(orgID, user.UserID))
		return nil
	}

	// Multiple users found - try to find exact match
	var exactMatch *models.UserLookupDTO

	if login != "" {
		// Look for exact login match
		for _, user := range users {
			if user.Login == login {
				if exactMatch != nil {
					// Multiple exact matches found (shouldn't happen with login)
					return diag.Errorf("ambiguous query when reading organization user, multiple users with exact login match: %q", login)
				}
				exactMatch = user
			}
		}
	} else if email != "" {
		// For email queries, we can't do exact matching since UserLookupDTO doesn't have Email field, return error
		return diag.Errorf("ambiguous query when reading organization user, multiple users returned by query: %q", query)
	}

	if exactMatch != nil {
		d.Set("user_id", exactMatch.UserID)
		d.Set("login", exactMatch.Login)
		d.SetId(MakeOrgResourceID(orgID, exactMatch.UserID))
		return nil
	}

	// No exact match found, return error
	return diag.Errorf("ambiguous query when reading organization user, multiple users returned by query: %q", query)
}
