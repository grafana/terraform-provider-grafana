package grafana

import (
	"context"
	"strconv"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceUser() *schema.Resource {
	return &schema.Resource{
		CreateContext: CreateUser,
		ReadContext:   ReadUser,
		UpdateContext: UpdateUser,
		DeleteContext: DeleteUser,
		Exists:        ExistsUser,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"email": {
				Type:     schema.TypeString,
				Required: true,
			},
			"name": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"login": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"password": {
				Type:      schema.TypeString,
				Required:  true,
				Sensitive: true,
			},
			"is_admin": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
		},
	}
}

func CreateUser(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gapi.Client)
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
	client := meta.(*gapi.Client)
	id, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}
	user, err := client.User(id)
	if err != nil {
		return diag.FromErr(err)
	}
	d.Set("email", user.Email)
	d.Set("name", user.Name)
	d.Set("login", user.Login)
	d.Set("is_admin", user.IsAdmin)
	return nil
}

func UpdateUser(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gapi.Client)
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
	client := meta.(*gapi.Client)
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
	client := meta.(*gapi.Client)
	userId, _ := strconv.ParseInt(d.Id(), 10, 64)
	_, err := client.User(userId)
	if err != nil {
		return false, err
	}
	return true, nil
}
