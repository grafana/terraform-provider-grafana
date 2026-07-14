package oncall

import (
	"context"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceSlackChannel() *common.DataSource {
	schema := &schema.Resource{
		Description: `
* [HTTP API](https://grafana.com/docs/oncall/latest/oncall-api-reference/slack_channels/)
`,
		ReadContext: withClient[schema.ReadContextFunc](dataSourceSlackChannelRead),
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The Slack channel name.",
			},
			"slack_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The Slack ID of the channel.",
			},
		},
	}
	return common.NewLegacySDKDataSource(common.CategoryOnCall, "grafana_oncall_slack_channel", schema)
}

func dataSourceSlackChannelRead(ctx context.Context, d *schema.ResourceData, client *onCallAPI.Client) diag.Diagnostics {
	options := &onCallAPI.ListSlackChannelOptions{}
	nameData := d.Get("name").(string)

	options.ChannelName = nameData

	slackChannelsResponse, _, err := client.SlackChannels.ListSlackChannels(options)
	if err != nil {
		return diag.FromErr(err)
	}

	if len(slackChannelsResponse.SlackChannels) == 0 {
		return diag.Errorf("couldn't find a slack_channel matching: %s", options.ChannelName)
	} else if len(slackChannelsResponse.SlackChannels) != 1 {
		return diag.Errorf("more than one slack_channel found matching: %s", options.ChannelName)
	}

	slackChannel := slackChannelsResponse.SlackChannels[0]

	d.SetId(slackChannel.SlackId)
	d.Set("name", slackChannel.Name)
	d.Set("slack_id", slackChannel.SlackId)

	return nil
}
