package provider

import (
	"fmt"
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/provider/common"
	"github.com/grafana/terraform-provider-grafana/provider/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccNotificationPolicy_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)
	testutils.CheckOSSTestsSemver(t, ">=9.1.0")

	// TODO: Make parallizable
	resource.Test(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		// Implicitly tests deletion.
		CheckDestroy: testNotifPolicyCheckDestroy(),
		Steps: []resource.TestStep{
			// Test creation.
			{
				Config: testutils.TestAccExample(t, "resources/grafana_notification_policy/resource.tf"),
				Check: resource.ComposeTestCheckFunc(
					testNotifPolicyCheckExists("grafana_notification_policy.my_notification_policy"),
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "contact_point", "A Contact Point"),
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "group_by.#", "1"),
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "group_by.0", "..."),
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "group_wait", "45s"),
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "group_interval", "6m"),
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "repeat_interval", "3h"),
					// nested
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "policy.#", "2"),
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "policy.0.contact_point", "A Contact Point"),
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "policy.0.matcher.0.label", "mylabel"),
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "policy.0.matcher.0.match", "="),
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "policy.0.matcher.0.value", "myvalue"),
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "policy.0.group_by.0", "alertname"),
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "policy.0.continue", "true"),
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "policy.0.mute_timings.0", "Some Mute Timing"),
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "policy.0.group_wait", "45s"),
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "policy.0.group_interval", "6m"),
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "policy.0.repeat_interval", "3h"),
					// deeply nested
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "policy.0.policy.#", "1"),
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "policy.0.policy.0.contact_point", "A Contact Point"),
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "policy.0.policy.0.matcher.0.label", "sublabel"),
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "policy.0.policy.0.matcher.0.match", "="),
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "policy.0.policy.0.matcher.0.value", "subvalue"),
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "policy.0.policy.0.group_by.0", "..."),
					// nested sibling
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "policy.1.contact_point", "A Contact Point"),
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "policy.1.matcher.0.label", "anotherlabel"),
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "policy.1.matcher.0.match", "=~"),
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "policy.1.matcher.0.value", "another value.*"),
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "policy.1.group_by.0", "..."),
				),
			},
			// Test import.
			{
				ResourceName:      "grafana_notification_policy.my_notification_policy",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Test update.
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_notification_policy/resource.tf", map[string]string{
					"...": "alertname",
				}),
				Check: resource.ComposeTestCheckFunc(
					testNotifPolicyCheckExists("grafana_notification_policy.my_notification_policy"),
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "contact_point", "A Contact Point"),
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "group_by.#", "1"),
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "group_by.0", "alertname"),
				),
			},
		},
	})
}

func testNotifPolicyCheckDestroy() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testutils.Provider.Meta().(*common.Client).GrafanaAPI
		npt, err := client.NotificationPolicyTree()
		if err != nil {
			return fmt.Errorf("failed to get notification policies")
		}

		if !notifPolicyIsDefault(npt) {
			return fmt.Errorf("notification policy tree was not reset back to the default")
		}
		return nil
	}
}

func testNotifPolicyCheckExists(rname string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resource, ok := s.RootModule().Resources[rname]
		if !ok {
			return fmt.Errorf("resource not found: %s, resources: %#v", rname, s.RootModule().Resources)
		}

		if resource.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		client := testutils.Provider.Meta().(*common.Client).GrafanaAPI
		npt, err := client.NotificationPolicyTree()
		if err != nil {
			return fmt.Errorf("failed to get notification policies")
		}

		if notifPolicyIsDefault(npt) {
			return fmt.Errorf("policy tree on the server is still the default one")
		}
		return nil
	}
}

func notifPolicyIsDefault(np gapi.NotificationPolicyTree) bool {
	return np.Receiver == "grafana-default-email"
}
