package grafana

import (
	"context"
	"strconv"
	"strings"

	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/client/users"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var resourceUserID = common.NewResourceID(common.IntIDField("id"))

func resourceUser() *common.Resource {
	schema := &schema.Resource{

		Description: `
* [Official documentation](https://grafana.com/docs/grafana/latest/administration/user-management/server-user-management/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/user/)

This resource represents an instance-scoped resource and uses Grafana's admin APIs.
It does not work with API tokens or service accounts which are org-scoped. 
You must use basic auth.
This resource is also not compatible with Grafana Cloud, as it does not allow basic auth.
`,

		CreateContext: CreateUser,
		ReadContext:   ReadUser,
		UpdateContext: UpdateUser,
		DeleteContext: DeleteUser,
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
				Type:             schema.TypeString,
				Required:         true,
				Description:      "The email address of the Grafana user.",
				DiffSuppressFunc: caseInsensitiveDiff,
			},
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The display name for the Grafana user.",
			},
			"login": {
				Type:             schema.TypeString,
				Optional:         true,
				Description:      "The username for the Grafana user.",
				DiffSuppressFunc: caseInsensitiveDiff,
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

	return common.NewLegacySDKResource(
		common.CategoryGrafanaOSS,
		"grafana_user",
		resourceUserID,
		schema,
	).
		WithLister(listerFunction(listUsers)).
		WithPreferredResourceNameField("login")
}

func listUsers(ctx context.Context, client *goapi.GrafanaHTTPAPI, data *ListerData) ([]string, error) {
	var ids []string
	var page int64 = 1
	for {
		params := users.NewSearchUsersParams().WithPage(&page)
		resp, err := client.Users.SearchUsers(params)
		if err != nil {
			return nil, err
		}

		for _, user := range resp.Payload {
			ids = append(ids, strconv.FormatInt(user.ID, 10))
		}

		if len(resp.Payload) == 0 {
			break
		}

		page++
	}

	return ids, nil
}

func CreateUser(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := OAPIGlobalClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	user := models.AdminCreateUserForm{
		Email:    d.Get("email").(string),
		Name:     d.Get("name").(string),
		Login:    d.Get("login").(string),
		Password: models.Password(d.Get("password").(string)),
	}
	resp, err := client.AdminUsers.AdminCreateUser(&user)
	if err != nil {
		return diag.FromErr(err)
	}
	if d.HasChange("is_admin") {
		perm := models.AdminUpdateUserPermissionsForm{IsGrafanaAdmin: d.Get("is_admin").(bool)}
		if _, err = client.AdminUsers.AdminUpdateUserPermissions(resp.Payload.ID, &perm); err != nil {
			return diag.FromErr(err)
		}
	}
	d.SetId(strconv.FormatInt(resp.Payload.ID, 10))
	return ReadUser(ctx, d, meta)
}

func ReadUser(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := OAPIGlobalClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	id, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}
	resp, err := client.Users.GetUserByID(id)
	if err, shouldReturn := common.CheckReadError("user", d, err); shouldReturn {
		return err
	}
	user := resp.Payload

	d.Set("user_id", user.ID)
	d.Set("email", user.Email)
	d.Set("name", user.Name)
	d.Set("login", user.Login)
	d.Set("is_admin", user.IsGrafanaAdmin)
	return nil
}

func UpdateUser(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := OAPIGlobalClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	id, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}
	u := models.UpdateUserCommand{
		Email: d.Get("email").(string),
		Name:  d.Get("name").(string),
		Login: d.Get("login").(string),
	}
	if _, err = client.Users.UpdateUser(id, &u); err != nil {
		return diag.FromErr(err)
	}
	if d.HasChange("password") {
		f := models.AdminUpdateUserPasswordForm{Password: models.Password(d.Get("password").(string))}
		if _, err = client.AdminUsers.AdminUpdateUserPassword(id, &f); err != nil {
			return diag.FromErr(err)
		}
	}
	if d.HasChange("is_admin") {
		f := models.AdminUpdateUserPermissionsForm{IsGrafanaAdmin: d.Get("is_admin").(bool)}
		if _, err = client.AdminUsers.AdminUpdateUserPermissions(id, &f); err != nil {
			return diag.FromErr(err)
		}
	}
	return ReadUser(ctx, d, meta)
}

func DeleteUser(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := OAPIGlobalClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	id, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}
	_, err = client.AdminUsers.AdminDeleteUser(id)
	diag, _ := common.CheckReadError("user", d, err)
	return diag
}

func caseInsensitiveDiff(k, oldValue, newValue string, d *schema.ResourceData) bool {
	return strings.EqualFold(oldValue, newValue)
}
