package grafana_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/grafana/terraform-provider-grafana/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceOnCallUserGroup_Basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	slackHandle := fmt.Sprintf("test-acc-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
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
