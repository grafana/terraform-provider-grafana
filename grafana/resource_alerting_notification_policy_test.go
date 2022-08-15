package grafana

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccNotificationPolicy_basic(t *testing.T) {
	CheckOSSTestsEnabled(t)
	CheckOSSTestsSemver(t, ">=9.0.0")

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		// Implicitly tests deletion.
		CheckDestroy: testNotifPolicyCheckDestroy(),
		Steps: []resource.TestStep{
			// Test creation.
			{
				Config: testAccExample(t, "resources/grafana_notification_policy/resource.tf"),
				Check:  resource.ComposeTestCheckFunc(),
			},
		},
	})
}

func testNotifPolicyCheckDestroy() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*client).gapi
		npt, err := client.NotificationPolicyTree()
		if err != nil {
			return fmt.Errorf("failed to get notification policies")
		}

		if npt.Receiver != "grafana-default-email" {
			return fmt.Errorf("notification policy tree was not reset back to the default")
		}
		return nil
	}
}
