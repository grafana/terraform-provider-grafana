package grafana

import (
	"context"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceOnCallUserGroup() *schema.Resource {
	return &schema.Resource{
		Description: `
* [HTTP API](https://grafana.com/docs/grafana-cloud/oncall/oncall-api-reference/user_groups/)
`,
		ReadContext: dataSourceOnCallUserGroupRead,
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

func dataSourceOnCallUserGroupRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*client).onCallAPI
	options := &onCallAPI.ListUserGroupOptions{}
	slackHandleData := d.Get("slack_handle").(string)

	options.SlackHandle = slackHandleData

	userGroupsResponse, _, err := client.UserGroups.ListUserGroups(options)
	if err != nil {
		return diag.FromErr(err)
	}

	if len(userGroupsResponse.UserGroups) == 0 {
		return diag.Errorf("couldn't find a user group matching: %s", options.SlackHandle)
	} else if len(userGroupsResponse.UserGroups) != 1 {
		return diag.Errorf("couldn't find a user group matching: %s", options.SlackHandle)
	}

	userGroup := userGroupsResponse.UserGroups[0]

	d.SetId(userGroup.ID)
	d.Set("slack_id", userGroup.SlackUserGroup.ID)

	return nil
}
