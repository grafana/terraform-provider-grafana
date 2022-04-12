package grafana

import (
	"errors"
	"fmt"

	amixrAPI "github.com/grafana/amixr-api-go-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceAmixrUser() *schema.Resource {
	return &schema.Resource{
		Description: `
* [HTTP API](https://grafana.com/docs/grafana-cloud/oncall/oncall-api-reference/users/)
`,
		Read: dataSourceAmixrUserRead,
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
}

func dataSourceAmixrUserRead(d *schema.ResourceData, m interface{}) error {
	client := m.(*client).amixrAPI
	if client == nil {
		err := errors.New("amixr api client is not configured")
		return err
	}
	options := &amixrAPI.ListUserOptions{}
	usernameData := d.Get("username").(string)

	options.Username = usernameData

	usersResponse, _, err := client.Users.ListUsers(options)

	if err != nil {
		return err
	}

	if len(usersResponse.Users) == 0 {
		return fmt.Errorf("couldn't find a user matching: %s", options.Username)
	} else if len(usersResponse.Users) != 1 {
		return fmt.Errorf("more than one user found matching: %s", options.Username)
	}

	user := usersResponse.Users[0]

	d.Set("email", user.Email)
	d.Set("username", user.Username)
	d.Set("role", user.Role)

	d.SetId(user.ID)

	return nil
}
