package oncall_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceSlackChannel_Basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	slackChannelName := fmt.Sprintf("test-acc-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
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
