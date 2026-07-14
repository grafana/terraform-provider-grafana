package oncall

import (
	"context"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceUserGroup() *common.DataSource {
	schema := &schema.Resource{
		Description: `
* [HTTP API](https://grafana.com/docs/oncall/latest/oncall-api-reference/user_groups/)
`,
		ReadContext: withClient[schema.ReadContextFunc](dataSourceUserGroupRead),
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
	return common.NewLegacySDKDataSource(common.CategoryOnCall, "grafana_oncall_user_group", schema)
}

func dataSourceUserGroupRead(ctx context.Context, d *schema.ResourceData, client *onCallAPI.Client) diag.Diagnostics {
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
