package grafana

import (
	"context"
	"strconv"

	gapi "github.com/albeego/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceUser() *schema.Resource {
	return &schema.Resource{

		Description: `
* [Official documentation](https://grafana.com/docs/grafana/latest/manage-users/server-admin/server-admin-manage-users/)
* [HTTP API](https://grafana.com/docs/grafana/latest/http_api/user/)

This resource uses Grafana's admin APIs for creating and updating users which
does not currently work with API Tokens. You must use basic auth.
`,

		CreateContext: CreateUser,
		ReadContext:   ReadUser,
		UpdateContext: UpdateUser,
		DeleteContext: DeleteUser,
		Exists:        ExistsUser,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"user_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The numerical ID of the Grafana user.",
			},
			"email": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The email address of the Grafana user.",
			},
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The display name for the Grafana user.",
			},
			"login": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The username for the Grafana user.",
			},
			"password": {
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				Description: "The password for the Grafana user.",
			},
			"is_admin": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether to make user an admin.",
			},
		},
	}
}

func CreateUser(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi
	user := gapi.User{
		Email:    d.Get("email").(string),
		Name:     d.Get("name").(string),
		Login:    d.Get("login").(string),
		Password: d.Get("password").(string),
	}
	id, err := client.CreateUser(user)
	if err != nil {
		return diag.FromErr(err)
	}
	if d.HasChange("is_admin") {
		err = client.UpdateUserPermissions(id, d.Get("is_admin").(bool))
		if err != nil {
			return diag.FromErr(err)
		}
	}
	d.SetId(strconv.FormatInt(id, 10))
	return ReadUser(ctx, d, meta)
}

func ReadUser(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi
	id, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}
	user, err := client.User(id)
	if err != nil {
		return diag.FromErr(err)
	}
	d.Set("user_id", user.ID)
	d.Set("email", user.Email)
	d.Set("name", user.Name)
	d.Set("login", user.Login)
	d.Set("is_admin", user.IsAdmin)
	return nil
}

func UpdateUser(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi
	id, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}
	u := gapi.User{
		ID:    id,
		Email: d.Get("email").(string),
		Name:  d.Get("name").(string),
		Login: d.Get("login").(string),
	}
	err = client.UserUpdate(u)
	if err != nil {
		return diag.FromErr(err)
	}
	if d.HasChange("password") {
		err = client.UpdateUserPassword(id, d.Get("password").(string))
		if err != nil {
			return diag.FromErr(err)
		}
	}
	if d.HasChange("is_admin") {
		err = client.UpdateUserPermissions(id, d.Get("is_admin").(bool))
		if err != nil {
			return diag.FromErr(err)
		}
	}
	return ReadUser(ctx, d, meta)
}

func DeleteUser(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi
	id, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}
	if err = client.DeleteUser(id); err != nil {
		return diag.FromErr(err)
	}

	return diag.Diagnostics{}
}

func ExistsUser(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(*client).gapi
	userID, _ := strconv.ParseInt(d.Id(), 10, 64)
	_, err := client.User(userID)
	if err != nil {
		return false, err
	}
	return true, nil
}
