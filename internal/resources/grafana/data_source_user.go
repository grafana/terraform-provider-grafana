package grafana

import (
	"context"
	"fmt"

	"github.com/grafana/grafana-openapi-client-go/client/users"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DatasourceUser() *schema.Resource {
	return &schema.Resource{
		Description: `
* [Official documentation](https://grafana.com/docs/grafana/latest/administration/user-management/server-user-management/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/user/)

This data source uses Grafana's admin APIs for reading users which
does not currently work with API Tokens. You must use basic auth.
`,
		ReadContext: dataSourceUserRead,
		Schema: map[string]*schema.Schema{
			"user_id": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     -1,
				Description: "The numerical ID of the Grafana user.",
			},
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
				Description: "The username for the Grafana user.",
			},
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The display name for the Grafana user.",
			},
			"is_admin": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the user is an admin.",
			},
		},
	}
}

func dataSourceUserRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := OAPIGlobalClient(meta) // Users are global/org-agnostic

	var resp interface{ GetPayload() *models.UserProfileDTO }
	var err error

	emailOrLogin := d.Get("email").(string)
	if emailOrLogin == "" {
		emailOrLogin = d.Get("login").(string)
	}

	if id := d.Get("user_id").(int); id >= 0 {
		params := users.NewGetUserByIDParams().WithUserID(int64(id))
		resp, err = client.Users.GetUserByID(params, nil)
	} else if emailOrLogin != "" {
		params := users.NewGetUserByLoginOrEmailParams().WithLoginOrEmail(emailOrLogin)
		resp, err = client.Users.GetUserByLoginOrEmail(params, nil)
	} else {
		err = fmt.Errorf("must specify one of user_id, email, or login")
	}

	if err != nil {
		return diag.FromErr(err)
	}

	user := resp.GetPayload()
	d.SetId(fmt.Sprintf("%d", user.ID))
	d.Set("user_id", user.ID)
	d.Set("email", user.Email)
	d.Set("name", user.Name)
	d.Set("login", user.Login)
	d.Set("is_admin", user.IsGrafanaAdmin)

	return nil
}
