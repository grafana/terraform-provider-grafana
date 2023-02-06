package grafana_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/grafana/terraform-provider-grafana/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceOnCallAction_Basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	actionName := fmt.Sprintf("test-acc-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDataSourceOnCallActionConfig(actionName),
				ExpectError: regexp.MustCompile(`couldn't find an action`),
			},
		},
	})
}

func testAccDataSourceOnCallActionConfig(actionName string) string {
	return fmt.Sprintf(`
data "grafana_oncall_action" "test-acc-action" {
	name = "%s"
}
`, actionName)
}
