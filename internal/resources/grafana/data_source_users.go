package grafana

import (
	"context"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DatasourceUsers() *schema.Resource {
	return &schema.Resource{
		ReadContext: readUsers,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Description: `
* [Official documentation](https://grafana.com/docs/grafana/latest/administration/user-management/server-user-management/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/user/)
		
This data source uses Grafana's admin APIs for reading users which
does not currently work with API Tokens. You must use basic auth.
		`,

		Schema: map[string]*schema.Schema{
			"users": {
				Type:        schema.TypeSet,
				Computed:    true,
				Description: "The Grafana instance's users.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The user ID.",
						},
						"login": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The user's login.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The user's name.",
						},
						"email": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The user's email.",
						},
						"is_admin": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether the user is admin or not.",
						},
					},
				},
			},
		},
	}
}

func readUsers(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).GrafanaAPI
	users, err := client.Users()
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("grafana_users")

	if err := d.Set("users", flattenUsers(users)); err != nil {
		return diag.Errorf("error setting item: %v", err)
	}

	return nil
}

func flattenUsers(items []gapi.UserSearch) []interface{} {
	userItems := make([]interface{}, 0)
	for _, user := range items {
		f := map[string]interface{}{
			"id":       user.ID,
			"login":    user.Login,
			"name":     user.Name,
			"email":    user.Email,
			"is_admin": user.IsAdmin,
		}
		userItems = append(userItems, f)
	}

	return userItems
}
