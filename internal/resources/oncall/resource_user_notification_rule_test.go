package oncall_test

import (
	"fmt"
	"testing"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccUserNotificationRule_basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	userId := acctest.RandString(8)
	resourceName := "grafana_oncall_user_notification_rule.test-acc-user_notification_rule"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckOnCallUserNotificationRuleResourceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOnCallUserNotificationRuleWait(userId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOnCallUserNotificationRuleResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "user_id", userId),
					resource.TestCheckResourceAttr(resourceName, "position", "1"),
					resource.TestCheckResourceAttr(resourceName, "duration", "300"),
					resource.TestCheckResourceAttr(resourceName, "type", "wait"),
					resource.TestCheckResourceAttr(resourceName, "important", "false"),
				),
			},
			// TODO: add some more test cases...
		},
	})
}

func testAccCheckOnCallUserNotificationRuleResourceDestroy(s *terraform.State) error {
	client := testutils.Provider.Meta().(*common.Client).OnCallClient
	for _, r := range s.RootModule().Resources {
		if r.Type != "grafana_oncall_user_notification_rule" {
			continue
		}

		if _, _, err := client.UserNotificationRules.GetUserNotificationRule(r.Primary.ID, &onCallAPI.GetUserNotificationRuleOptions{}); err == nil {
			return fmt.Errorf("UserNotificationRule still exists")
		}
	}
	return nil
}

func testAccOnCallUserNotificationRuleWait(userId string) string {
	return fmt.Sprintf(`
resource "grafana_oncall_user_notification_rule" "test-acc-user_notification_rule" {
  user_id  = "%s"
  position = 1
  duration = 300
  type     = "wait"
}
`, userId)
}

func testAccCheckOnCallUserNotificationRuleResourceExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No UserNotificationRule ID is set")
		}

		client := testutils.Provider.Meta().(*common.Client).OnCallClient

		found, _, err := client.UserNotificationRules.GetUserNotificationRule(rs.Primary.ID, &onCallAPI.GetUserNotificationRuleOptions{})
		if err != nil {
			return err
		}
		if found.ID != rs.Primary.ID {
			return fmt.Errorf("UserNotificationRule policy not found: %v - %v", rs.Primary.ID, found)
		}
		return nil
	}
}
