package grafana

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceOnCallOutgoingWebhook_Basic(t *testing.T) {
	CheckCloudInstanceTestsEnabled(t)

	outgoingWebhookName := fmt.Sprintf("test-acc-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDataSourceOnCallOutgoingWebhookConfig(outgoingWebhookName),
				ExpectError: regexp.MustCompile(`couldn't find an outgoing webhook`),
			},
		},
	})
}

func testAccDataSourceOnCallOutgoingWebhookConfig(outgoingWebhookName string) string {
	return fmt.Sprintf(`
data "grafana_oncall_outgoing_webhook" "test-acc-outgoing_webhook" {
	name = "%s"
}
`, outgoingWebhookName)
}
