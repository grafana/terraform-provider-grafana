package grafana_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/internal/testutils"
)

func TestAccNotificationPolicy_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.1.0")

	var policy models.Route

	// TODO: Make parallizable
	resource.Test(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		// Implicitly tests deletion.
		CheckDestroy: alertingNotificationPolicyCheckExists.destroyed(&policy, nil),
		Steps: []resource.TestStep{
			// Test creation.
			{
				Config: testutils.TestAccExample(t, "resources/grafana_notification_policy/resource.tf"),
				Check: resource.ComposeTestCheckFunc(
					alertingNotificationPolicyCheckExists.exists("grafana_notification_policy.my_notification_policy", &policy),
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "contact_point", "A Contact Point"),
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "group_by.#", "1"),
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "group_by.0", "..."),
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "group_wait", "45s"),
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "group_interval", "6m"),
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "repeat_interval", "3h"),
					// nested
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "policy.#", "2"),
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "policy.0.contact_point", "A Contact Point"),
					// Matchers are reordered by the API
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "policy.0.matcher.0.label", "Name"),
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "policy.0.matcher.0.match", "=~"),
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "policy.0.matcher.0.value", "host.*|host-b.*"),
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "policy.0.matcher.1.label", "alertname"),
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "policy.0.matcher.1.match", "="),
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "policy.0.matcher.1.value", "CPU Usage"),
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "policy.0.matcher.2.label", "mylabel"),
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "policy.0.matcher.2.match", "="),
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "policy.0.matcher.2.value", "myvalue"),
					resource.TestCheckNoResourceAttr("grafana_notification_policy.my_notification_policy", "policy.0.group_by"),
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
					alertingNotificationPolicyCheckExists.exists("grafana_notification_policy.my_notification_policy", &policy),
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "contact_point", "A Contact Point"),
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "group_by.#", "1"),
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "group_by.0", "alertname"),
				),
			},
		},
	})
}

func TestAccNotificationPolicy_disableProvenance(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.1.0")

	var policy models.Route

	// TODO: Make parallizable
	resource.Test(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		// Implicitly tests deletion.
		CheckDestroy: alertingNotificationPolicyCheckExists.destroyed(&policy, nil),
		Steps: []resource.TestStep{
			// Create
			{
				Config: testAccNotificationPolicyDisableProvenance(false),
				Check: resource.ComposeTestCheckFunc(
					alertingNotificationPolicyCheckExists.exists("grafana_notification_policy.test", &policy),
					resource.TestCheckResourceAttr("grafana_notification_policy.test", "disable_provenance", "false"),
				),
			},
			// Import (tests that disable_provenance is fetched from API)
			{
				ResourceName:      "grafana_notification_policy.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Disable provenance
			{
				Config: testAccNotificationPolicyDisableProvenance(true),
				Check: resource.ComposeTestCheckFunc(
					alertingNotificationPolicyCheckExists.exists("grafana_notification_policy.test", &policy),
					resource.TestCheckResourceAttr("grafana_notification_policy.test", "disable_provenance", "true"),
				),
			},
			// Import (tests that disable_provenance is fetched from API)
			{
				ResourceName:      "grafana_notification_policy.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Re-enable provenance
			{
				Config: testAccNotificationPolicyDisableProvenance(false),
				Check: resource.ComposeTestCheckFunc(
					alertingNotificationPolicyCheckExists.exists("grafana_notification_policy.test", &policy),
					resource.TestCheckResourceAttr("grafana_notification_policy.test", "disable_provenance", "false"),
				),
			},
		},
	})
}

func TestAccNotificationPolicy_error(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.1.0")

	resource.Test(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `resource "grafana_notification_policy" "test" {
					group_by      = ["..."]
					contact_point = "invalid"
				  }`,
				// This tests that the API error message is propagated to the user.
				ExpectError: regexp.MustCompile("400.+invalid object specification: receiver 'invalid' does not exist"),
			},
		},
	})
}

func testAccNotificationPolicyDisableProvenance(disableProvenance bool) string {
	return fmt.Sprintf(`
	resource "grafana_contact_point" "a_contact_point" {
		name = "A Contact Point"
	  
		email {
		  addresses = ["one@company.org", "two@company.org"]
		}
	  }

	resource "grafana_notification_policy" "test" {
		group_by      = ["hello"]
		contact_point = grafana_contact_point.a_contact_point.name
		disable_provenance = %t

		policy {
			group_by = ["hello"]
			matcher {
				label = "Name"
				match = "=~"
				value = "host.*|host-b.*"
			}
			contact_point = grafana_contact_point.a_contact_point.name
		}
	  }
	`, disableProvenance)
}
