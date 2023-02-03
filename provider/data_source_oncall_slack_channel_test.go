package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceOnCallSlackChannel_Basic(t *testing.T) {
	CheckCloudInstanceTestsEnabled(t)

	slackChannelName := fmt.Sprintf("test-acc-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDataSourceOnCallSlackChannelConfig(slackChannelName),
				ExpectError: regexp.MustCompile(`couldn't find a slack_channel`),
			},
		},
	})
}

func testAccDataSourceOnCallSlackChannelConfig(slackChannelName string) string {
	return fmt.Sprintf(`
data "grafana_oncall_slack_channel" "test-acc-slack-channel" {
	name = "%s"
}
`, slackChannelName)
}
