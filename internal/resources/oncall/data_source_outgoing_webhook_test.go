package oncall_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceOutgoingWebhook_Basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	outgoingWebhookName := fmt.Sprintf("test-acc-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDataSourceOutgoingWebhookConfig(outgoingWebhookName),
				ExpectError: regexp.MustCompile(`couldn't find an outgoing webhook`),
			},
		},
	})
}

func testAccDataSourceOutgoingWebhookConfig(outgoingWebhookName string) string {
	return fmt.Sprintf(`
data "grafana_oncall_outgoing_webhook" "test-acc-outgoing_webhook" {
	name = "%s"
}
`, outgoingWebhookName)
}
