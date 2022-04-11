package grafana

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceAmixrUserGroup_Basic(t *testing.T) {
	CheckCloudInstanceTestsEnabled(t)

	slackHandle := fmt.Sprintf("test-acc-%s", acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDataSourceAmixrUserGroupConfig(slackHandle),
				ExpectError: regexp.MustCompile(`couldn't find a user group`),
			},
		},
	})
}

func testAccDataSourceAmixrUserGroupConfig(slackHandle string) string {
	return fmt.Sprintf(`
data "amixr_user_group" "test-acc-user-group" {
	slack_handle = "%s"
}
`, slackHandle)
}
