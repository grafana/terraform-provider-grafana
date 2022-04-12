package grafana

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceAmixrSlackChannel_Basic(t *testing.T) {
	CheckCloudInstanceTestsEnabled(t)

	slackChannelName := fmt.Sprintf("test-acc-%s", acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDataSourceAmixrSlackChannelConfig(slackChannelName),
				ExpectError: regexp.MustCompile(`couldn't find a slack_channel`),
			},
		},
	})
}

func testAccDataSourceAmixrSlackChannelConfig(slackChannelName string) string {
	return fmt.Sprintf(`
data "grafanaamixr_slack_channel" "test-acc-slack-channel" {
	name = "%s"
}
`, slackChannelName)
}
