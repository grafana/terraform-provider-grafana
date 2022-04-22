package grafana

import (
	"errors"
	"fmt"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceOnCallSlackChannel() *schema.Resource {
	return &schema.Resource{
		Description: `
* [HTTP API](https://grafana.com/docs/grafana-cloud/oncall/oncall-api-reference/slack_channels/)
`,
		Read: dataSourceOnCallSlackChannelRead,
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
}

func dataSourceOnCallSlackChannelRead(d *schema.ResourceData, m interface{}) error {
	client := m.(*client).onCallAPI
	if client == nil {
		return errors.New("Grafana OnCall api client is not configured")
	}
	options := &onCallAPI.ListSlackChannelOptions{}
	nameData := d.Get("name").(string)

	options.ChannelName = nameData

	slackChannelsResponse, _, err := client.SlackChannels.ListSlackChannels(options)
	if err != nil {
		return err
	}

	if len(slackChannelsResponse.SlackChannels) == 0 {
		return fmt.Errorf("couldn't find a slack_channel matching: %s", options.ChannelName)
	} else if len(slackChannelsResponse.SlackChannels) != 1 {
		return fmt.Errorf("more than one slack_channel found matching: %s", options.ChannelName)
	}

	slack_channel := slackChannelsResponse.SlackChannels[0]

	d.SetId(slack_channel.SlackId)
	d.Set("name", slack_channel.Name)
	d.Set("slack_id", slack_channel.SlackId)

	return nil
}
