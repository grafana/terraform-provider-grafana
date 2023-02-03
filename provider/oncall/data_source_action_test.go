package oncall

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/grafana/terraform-provider-grafana/provider/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceAction_Basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	actionName := fmt.Sprintf("test-acc-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDataSourceActionConfig(actionName),
				ExpectError: regexp.MustCompile(`couldn't find an action`),
			},
		},
	})
}

func testAccDataSourceActionConfig(actionName string) string {
	return fmt.Sprintf(`
data "grafana_oncall_action" "test-acc-action" {
	name = "%s"
}
`, actionName)
}
