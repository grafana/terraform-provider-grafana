package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceOnCallUserGroup_Basic(t *testing.T) {
	CheckCloudInstanceTestsEnabled(t)

	slackHandle := fmt.Sprintf("test-acc-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDataSourceOnCallUserGroupConfig(slackHandle),
				ExpectError: regexp.MustCompile(`couldn't find a user group`),
			},
		},
	})
}

func testAccDataSourceOnCallUserGroupConfig(slackHandle string) string {
	return fmt.Sprintf(`
data "grafana_oncall_user_group" "test-acc-user-group" {
	slack_handle = "%s"
}
`, slackHandle)
}
