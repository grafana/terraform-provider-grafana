package grafana

import (
	"errors"
	"fmt"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceOnCallUserGroup() *schema.Resource {
	return &schema.Resource{
		Description: `
* [HTTP API](https://grafana.com/docs/grafana-cloud/oncall/oncall-api-reference/user_groups/)
`,
		Read: dataSourceOnCallUserGroupRead,
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

func dataSourceOnCallUserGroupRead(d *schema.ResourceData, m interface{}) error {
	client := m.(*client).onCallAPI
	if client == nil {
		return errors.New("grafana OnCall api client is not configured")
	}
	options := &onCallAPI.ListUserGroupOptions{}
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

	userGroup := userGroupsResponse.UserGroups[0]

	d.SetId(userGroup.ID)
	d.Set("slack_id", userGroup.SlackUserGroup.ID)

	return nil
}
