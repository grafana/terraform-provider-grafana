package oncall_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceUserGroup_Basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	slackHandle := fmt.Sprintf("test-acc-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDataSourceUserGroupConfig(slackHandle),
				ExpectError: regexp.MustCompile(`couldn't find a user group`),
			},
		},
	})
}

func testAccDataSourceUserGroupConfig(slackHandle string) string {
	return fmt.Sprintf(`
data "grafana_oncall_user_group" "test-acc-user-group" {
	slack_handle = "%s"
}
`, slackHandle)
}
