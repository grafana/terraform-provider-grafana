package grafana

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceAmixrUser_Basic(t *testing.T) {
	CheckCloudInstanceTestsEnabled(t)

	username := fmt.Sprintf("test-acc-%s", acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDataSourceAmixrUserConfig(username),
				ExpectError: regexp.MustCompile(`couldn't find a user`),
			},
		},
	})
}

func testAccDataSourceAmixrUserConfig(username string) string {
	return fmt.Sprintf(`
data "amixr_user" "test-acc-user" {
	username = "%s"
}
`, username)
}
