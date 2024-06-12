package oncall

import (
	"context"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceUser() *common.DataSource {
	schema := &schema.Resource{
		Description: `
* [HTTP API](https://grafana.com/docs/oncall/latest/oncall-api-reference/users/)
`,
		ReadContext: withClient[schema.ReadContextFunc](dataSourceUserRead),
		Schema: map[string]*schema.Schema{
			"username": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The username of the user.",
			},
			"email": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The email of the user.",
			},
			"role": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The role of the user.",
			},
		},
	}
	return common.NewLegacySDKDataSource(common.CategoryOnCall, "grafana_oncall_user", schema)
}

func dataSourceUserRead(ctx context.Context, d *schema.ResourceData, client *onCallAPI.Client) diag.Diagnostics {
	options := &onCallAPI.ListUserOptions{}
	usernameData := d.Get("username").(string)

	options.Username = usernameData

	usersResponse, _, err := client.Users.ListUsers(options)
	if err != nil {
		return diag.FromErr(err)
	}

	if len(usersResponse.Users) == 0 {
		return diag.Errorf("couldn't find a user matching: %s", options.Username)
	} else if len(usersResponse.Users) != 1 {
		return diag.Errorf("more than one user found matching: %s", options.Username)
	}

	user := usersResponse.Users[0]

	d.Set("email", user.Email)
	d.Set("username", user.Username)
	d.Set("role", user.Role)

	d.SetId(user.ID)

	return nil
}
