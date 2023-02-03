package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceOnCallUser_Basic(t *testing.T) {
	CheckCloudInstanceTestsEnabled(t)

	username := fmt.Sprintf("test-acc-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDataSourceOnCallUserConfig(username),
				ExpectError: regexp.MustCompile(`couldn't find a user`),
			},
		},
	})
}

func testAccDataSourceOnCallUserConfig(username string) string {
	return fmt.Sprintf(`
data "grafana_oncall_user" "test-acc-user" {
	username = "%s"
}
`, username)
}
