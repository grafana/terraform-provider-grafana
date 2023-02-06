package oncall

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/grafana/terraform-provider-grafana/provider/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceSlackChannel_Basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	slackChannelName := fmt.Sprintf("test-acc-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.GetProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      testAccDataSourceSlackChannelConfig(slackChannelName),
				ExpectError: regexp.MustCompile(`couldn't find a slack_channel`),
			},
		},
	})
}

func testAccDataSourceSlackChannelConfig(slackChannelName string) string {
	return fmt.Sprintf(`
data "grafana_oncall_slack_channel" "test-acc-slack-channel" {
	name = "%s"
}
`, slackChannelName)
}
