package oncall

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/grafana/terraform-provider-grafana/provider/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceUser_Basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	username := fmt.Sprintf("test-acc-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.GetProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      testAccDataSourceUserConfig(username),
				ExpectError: regexp.MustCompile(`couldn't find a user`),
			},
		},
	})
}

func testAccDataSourceUserConfig(username string) string {
	return fmt.Sprintf(`
data "grafana_oncall_user" "test-acc-user" {
	username = "%s"
}
`, username)
}
