package grafana

import (
	"errors"
	"fmt"
	"log"

	amixrAPI "github.com/grafana/amixr-api-go-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceAmixrUserGroup() *schema.Resource {
	return &schema.Resource{
		Description: `
* [HTTP API](https://grafana.com/docs/grafana-cloud/oncall/oncall-api-reference/user_groups/)
`,
		Read: dataSourceAmixrUserGroupRead,
		Schema: map[string]*schema.Schema{
			"slack_handle": {
				Type:     schema.TypeString,
				Required: true,
			},
			"slack_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceAmixrUserGroupRead(d *schema.ResourceData, m interface{}) error {
	log.Printf("[DEBUG] read amixr user group")

	client := m.(*client).amixrAPI
	if client == nil {
		err := errors.New("amixr api client is not configured")
		return err
	}
	options := &amixrAPI.ListUserGroupOptions{}
	slackHandleData := d.Get("slack_handle").(string)

	options.SlackHandle = slackHandleData

	userGroupsResponse, _, err := client.UserGroups.ListUserGroups(options)

	if err != nil {
		return err
	}

	if len(userGroupsResponse.UserGroups) == 0 {
		return fmt.Errorf("couldn't find a user group matching: %s", options.SlackHandle)
	} else if len(userGroupsResponse.UserGroups) != 1 {
		return fmt.Errorf("couldn't find a user group matching: %s", options.SlackHandle)
	}

	user_group := userGroupsResponse.UserGroups[0]

	d.SetId(user_group.ID)
	d.Set("slack_id", user_group.SlackUserGroup.ID)

	return nil
}
