package grafana

import (
	"context"
	"fmt"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DatasourceUser() *schema.Resource {
	return &schema.Resource{
		Description: `
* [Official documentation](https://grafana.com/docs/grafana/latest/manage-users/server-admin/server-admin-manage-users/)
* [HTTP API](https://grafana.com/docs/grafana/latest/http_api/user/)

This resource uses Grafana's admin APIs for creating and updating users which
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
	client := meta.(*client).gapi

	var user gapi.User
	var err error
	if id := d.Get("user_id").(int); id >= 0 {
		user, err = client.User(int64(id))
	} else if email := d.Get("email").(string); email != "" {
		user, err = client.UserByEmail(email)
	} else if login := d.Get("login").(string); login != "" {
		user, err = client.UserByEmail(login)
	} else {
		err = fmt.Errorf("must specify one of user_id, email, or login")
	}

	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%d", user.ID))
	d.Set("user_id", user.ID)
	d.Set("email", user.Email)
	d.Set("name", user.Name)
	d.Set("login", user.Login)
	d.Set("is_admin", user.IsAdmin)

	return nil
}
