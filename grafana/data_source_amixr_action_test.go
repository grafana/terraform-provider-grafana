package grafana

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceAmixrAction_Basic(t *testing.T) {
	CheckCloudInstanceTestsEnabled(t)

	actionName := fmt.Sprintf("test-acc-%s", acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDataSourceAmixrActionConfig(actionName),
				ExpectError: regexp.MustCompile(`couldn't find an action`),
			},
		},
	})
}

func testAccDataSourceAmixrActionConfig(actionName string) string {
	return fmt.Sprintf(`
data "grafana_amixr_action" "test-acc-action" {
	name = "%s"
}
`, actionName)
}
