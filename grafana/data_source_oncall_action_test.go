package grafana

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceOnCallAction_Basic(t *testing.T) {
	CheckCloudInstanceTestsEnabled(t)

	actionName := fmt.Sprintf("test-acc-%s", acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDataSourceOnCallActionConfig(actionName),
				ExpectError: regexp.MustCompile(`couldn't find an outgoing webhook`),
			},
		},
	})
}

func testAccDataSourceOnCallActionConfig(actionName string) string {
	return fmt.Sprintf(`
data "grafana_oncall_outgoing_webhook" "test-acc-action" {
	name = "%s"
}
`, actionName)
}
