package grafana

import (
	"fmt"
	"testing"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccOnCallOutgoingWebhook_basic(t *testing.T) {
	CheckCloudInstanceTestsEnabled(t)

	webhookName := fmt.Sprintf("name-%s", acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckOnCallOutgoingWebhookResourceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOnCallOutgoingWebhookConfig(webhookName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOnCallOutgoingWebhookResourceExists("grafana_oncall_outgoing_webhook.test-acc-outgoing_webhook"),
				),
			},
		},
	})
}

func testAccCheckOnCallOutgoingWebhookResourceDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*client).onCallAPI
	for _, r := range s.RootModule().Resources {
		if r.Type != "grafana_oncall_outgoing_webhook" {
			continue
		}

		if _, _, err := client.CustomActions.GetCustomAction(r.Primary.ID, &onCallAPI.GetCustomActionOptions{}); err == nil {
			return fmt.Errorf("OutgoingWebhook still exists")
		}
	}
	return nil
}

func testAccOnCallOutgoingWebhookConfig(webhookName string) string {
	return fmt.Sprintf(`
resource "grafana_oncall_outgoing_webhook" "test-acc-outgoing_webhook" {
	name = "%s"
	url = "https://example.com"
	data = "\"test\""
	user = "test"
	password = "test"
	authorization_header = "Authorization"
	forward_whole_payload = false
}
`, webhookName)
}

func testAccCheckOnCallOutgoingWebhookResourceExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No OutgoingWebhook ID is set")
		}

		client := testAccProvider.Meta().(*client).onCallAPI

		found, _, err := client.CustomActions.GetCustomAction(rs.Primary.ID, &onCallAPI.GetCustomActionOptions{})
		if err != nil {
			return err
		}
		if found.ID != rs.Primary.ID {
			return fmt.Errorf("OutgoingWebhook policy not found: %v - %v", rs.Primary.ID, found)
		}
		return nil
	}
}
