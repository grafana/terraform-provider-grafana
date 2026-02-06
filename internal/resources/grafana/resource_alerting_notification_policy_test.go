package grafana_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/grafana/grafana-openapi-client-go/models"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
)

func TestAccNotificationPolicy_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.1.0")

	var policy models.Route

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		// Implicitly tests deletion.
		CheckDestroy: alertingNotificationPolicyCheckExists.destroyed(&policy, nil),
		Steps: []resource.TestStep{
			// Test creation.
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_notification_policy/resource.tf", map[string]string{
					"active_timings = [grafana_mute_timing.working_hours.name]": "", // old versions of Grafana do not support this field
				}),
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
					testutils.CheckLister("grafana_notification_policy.my_notification_policy"),
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
					"active_timings = [grafana_mute_timing.working_hours.name]": "", // old versions of Grafana do not support this field
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

func TestAccNotificationPolicy_activeTimings(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=12.1.0")

	var policy models.Route

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		// Implicitly tests deletion.
		CheckDestroy: alertingNotificationPolicyCheckExists.destroyed(&policy, nil),
		Steps: []resource.TestStep{
			// Test creation.
			{
				Config: testutils.TestAccExample(t, "resources/grafana_notification_policy/resource.tf"),
				Check: resource.ComposeTestCheckFunc(
					alertingNotificationPolicyCheckExists.exists("grafana_notification_policy.my_notification_policy", &policy),
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "policy.0.active_timings.0", "Working Hours"),
				),
			},
		},
	})
}

func TestAccNotificationPolicy_inheritContactPoint(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=11.0.0")
	var policy models.Route

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		// Implicitly tests deletion.
		CheckDestroy: alertingNotificationPolicyCheckExists.destroyed(&policy, nil),
		Steps: []resource.TestStep{
			// Test creation.
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_notification_policy/resource.tf", map[string]string{
					"contact_point  = grafana_contact_point.a_contact_point.name // This can be omitted to inherit from the parent":              "",
					"contact_point = grafana_contact_point.a_contact_point.name // This can also be omitted to inherit from the parent's parent": "",
					"active_timings = [grafana_mute_timing.working_hours.name]":                                                                  "",
				}),
				Check: resource.ComposeTestCheckFunc(
					alertingNotificationPolicyCheckExists.exists("grafana_notification_policy.my_notification_policy", &policy),
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "contact_point", "A Contact Point"),
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "policy.0.contact_point", ""),
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "policy.0.policy.0.contact_point", ""),
					resource.TestCheckResourceAttr("grafana_notification_policy.my_notification_policy", "policy.1.contact_point", "A Contact Point"),
				),
			},
			// Test import.
			{
				ResourceName:      "grafana_notification_policy.my_notification_policy",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccNotificationPolicy_disableProvenance(t *testing.T) {
	t.Run("fetch disable_provenance", func(t *testing.T) {
		testutils.CheckOSSTestsEnabled(t, ">=11.3.0")

		var policy models.Route

		resource.Test(t, resource.TestCase{
			ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
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
			},
		})
	})

	t.Run("disable_provenance", func(t *testing.T) {
		testutils.CheckOSSTestsEnabled(t, ">=9.1.0,<=11.1.0")

		var policy models.Route

		resource.Test(t, resource.TestCase{
			ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
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
	})
}

func TestAccNotificationPolicy_error(t *testing.T) {
	testCases := []struct {
		versionConstraint string
		errorMessage      string
	}{
		{
			versionConstraint: ">=9.1.0,<11.4.0",
			errorMessage:      "400.+invalid object specification: receiver 'invalid' does not exist",
		},
		{
			versionConstraint: ">=11.4.0",
			errorMessage:      "400.+Invalid format of the submitted route: receiver 'invalid' does not exist",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.versionConstraint, func(t *testing.T) {
			testutils.CheckOSSTestsEnabled(t, tc.versionConstraint)

			resource.Test(t, resource.TestCase{
				ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: `resource "grafana_notification_policy" "test" {
					group_by      = ["..."]
					contact_point = "invalid"
				  }`,
						// This tests that the API error message is propagated to the user.
						ExpectError: regexp.MustCompile(tc.errorMessage),
					},
				},
			})
		})
	}
}

func TestAccNotificationPolicy_inOrg(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.1.0")

	var policy models.Route
	var org models.OrgDetailsDTO

	name := acctest.RandString(10)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             orgCheckExists.destroyed(&org, nil),
		Steps: []resource.TestStep{
			{
				Config: testAccNotificationPolicyInOrg(name, "my-key"),
				Check: resource.ComposeTestCheckFunc(
					orgCheckExists.exists("grafana_organization.test", &org),
					alertingNotificationPolicyCheckExists.exists("grafana_notification_policy.test", &policy),
					checkResourceIsInOrg("grafana_notification_policy.test", "grafana_organization.test"),
				),
			},
			// Change contact point config
			{
				Config: testAccNotificationPolicyInOrg(name, "my-key2"),
				Check: resource.ComposeTestCheckFunc(
					orgCheckExists.exists("grafana_organization.test", &org),
					alertingNotificationPolicyCheckExists.exists("grafana_notification_policy.test", &policy),
					checkResourceIsInOrg("grafana_notification_policy.test", "grafana_organization.test"),
				),
			},
			{
				Config: testutils.WithoutResource(t, testAccNotificationPolicyInOrg(name, "my-key2"), "grafana_notification_policy.test"),
				Check: resource.ComposeTestCheckFunc(
					orgCheckExists.exists("grafana_organization.test", &org),
					alertingNotificationPolicyCheckExists.destroyed(&policy, &org),
				),
			},
		},
	})
}

func testAccNotificationPolicyInOrg(name, key string) string {
	return fmt.Sprintf(`
	resource "grafana_organization" "test" {
		name = "%[1]s"
	}

	resource "grafana_contact_point" "a_contact_point" {
		org_id = grafana_organization.test.id
		name = "A Contact Point"
		pagerduty {
			integration_key = "%[2]s"
			details = {
				"key" = "%[2]s"
			}
		}
	}

	resource "grafana_notification_policy" "test" {
		org_id = grafana_organization.test.id
		group_by      = ["hello"]
		contact_point = grafana_contact_point.a_contact_point.name

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
	`, name, key)
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

func TestAccNotificationPolicy_amConfig(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.1.0")

	name := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			// Test creation with AM Config API
			{
				Config: testAccNotificationPolicyAMConfig(name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_notification_policy.test", "alertmanager_uid", "grafana"),
					resource.TestCheckResourceAttr("grafana_notification_policy.test", "contact_point", name),
					resource.TestCheckResourceAttr("grafana_notification_policy.test", "group_by.#", "2"),
					resource.TestCheckResourceAttr("grafana_notification_policy.test", "group_by.0", "alertname"),
					resource.TestCheckResourceAttr("grafana_notification_policy.test", "group_by.1", "cluster"),
					resource.TestCheckResourceAttr("grafana_notification_policy.test", "group_wait", "10s"),
					resource.TestCheckResourceAttr("grafana_notification_policy.test", "group_interval", "1m"),
					resource.TestCheckResourceAttr("grafana_notification_policy.test", "repeat_interval", "5m"),
				),
			},
			// Test update — change group_by and intervals
			{
				Config: testAccNotificationPolicyAMConfigUpdated(name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_notification_policy.test", "alertmanager_uid", "grafana"),
					resource.TestCheckResourceAttr("grafana_notification_policy.test", "contact_point", name),
					resource.TestCheckResourceAttr("grafana_notification_policy.test", "group_by.#", "1"),
					resource.TestCheckResourceAttr("grafana_notification_policy.test", "group_by.0", "..."),
					resource.TestCheckResourceAttr("grafana_notification_policy.test", "group_wait", "30s"),
					resource.TestCheckResourceAttr("grafana_notification_policy.test", "group_interval", "5m"),
					resource.TestCheckResourceAttr("grafana_notification_policy.test", "repeat_interval", "4h"),
				),
			},
		},
	})
}

func testAccNotificationPolicyAMConfig(name string) string {
	return fmt.Sprintf(`
resource "grafana_contact_point" "test" {
  name = "%[1]s"
  email {
    addresses = ["test@example.com"]
  }
}

resource "grafana_notification_policy" "test" {
  alertmanager_uid = "grafana"
  contact_point    = grafana_contact_point.test.name
  group_by         = ["alertname", "cluster"]
  group_wait       = "10s"
  group_interval   = "1m"
  repeat_interval  = "5m"
}
`, name)
}

func testAccNotificationPolicyAMConfigUpdated(name string) string {
	return fmt.Sprintf(`
resource "grafana_contact_point" "test" {
  name = "%[1]s"
  email {
    addresses = ["test@example.com"]
  }
}

resource "grafana_notification_policy" "test" {
  alertmanager_uid = "grafana"
  contact_point    = grafana_contact_point.test.name
  group_by         = ["..."]
  group_wait       = "30s"
  group_interval   = "5m"
  repeat_interval  = "4h"
}
`, name)
}

// TestAccNotificationPolicy_amConfigNativeAlertmanager tests notification policies on a native (non-Grafana-managed)
// alertmanager. This exercises the native AM format conversion code (routeModelToAMConfig, etc.).
func TestAccNotificationPolicy_amConfigNativeAlertmanager(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	name := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNotificationPolicyNativeAMConfig(name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_notification_policy.test", "alertmanager_uid", "grafanacloud-ngalertmanager"),
					resource.TestCheckResourceAttr("grafana_notification_policy.test", "contact_point", name),
					resource.TestCheckResourceAttr("grafana_notification_policy.test", "group_by.#", "2"),
					resource.TestCheckResourceAttr("grafana_notification_policy.test", "group_by.0", "alertname"),
					resource.TestCheckResourceAttr("grafana_notification_policy.test", "group_by.1", "cluster"),
					resource.TestCheckResourceAttr("grafana_notification_policy.test", "group_wait", "10s"),
					resource.TestCheckResourceAttr("grafana_notification_policy.test", "group_interval", "1m"),
					resource.TestCheckResourceAttr("grafana_notification_policy.test", "repeat_interval", "5m"),
				),
			},
			{
				Config: testAccNotificationPolicyNativeAMConfigUpdated(name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_notification_policy.test", "alertmanager_uid", "grafanacloud-ngalertmanager"),
					resource.TestCheckResourceAttr("grafana_notification_policy.test", "contact_point", name),
					resource.TestCheckResourceAttr("grafana_notification_policy.test", "group_by.#", "1"),
					resource.TestCheckResourceAttr("grafana_notification_policy.test", "group_by.0", "..."),
					resource.TestCheckResourceAttr("grafana_notification_policy.test", "group_wait", "30s"),
					resource.TestCheckResourceAttr("grafana_notification_policy.test", "group_interval", "5m"),
					resource.TestCheckResourceAttr("grafana_notification_policy.test", "repeat_interval", "4h"),
				),
			},
		},
	})
}

func testAccNotificationPolicyNativeAMConfig(name string) string {
	return fmt.Sprintf(`
resource "grafana_contact_point" "test" {
  alertmanager_uid = "grafanacloud-ngalertmanager"
  name             = "%[1]s"
  email {
    addresses = ["test@example.com"]
  }
}

resource "grafana_notification_policy" "test" {
  alertmanager_uid = "grafanacloud-ngalertmanager"
  contact_point    = grafana_contact_point.test.name
  group_by         = ["alertname", "cluster"]
  group_wait       = "10s"
  group_interval   = "1m"
  repeat_interval  = "5m"
}
`, name)
}

func testAccNotificationPolicyNativeAMConfigUpdated(name string) string {
	return fmt.Sprintf(`
resource "grafana_contact_point" "test" {
  alertmanager_uid = "grafanacloud-ngalertmanager"
  name             = "%[1]s"
  email {
    addresses = ["test@example.com"]
  }
}

resource "grafana_notification_policy" "test" {
  alertmanager_uid = "grafanacloud-ngalertmanager"
  contact_point    = grafana_contact_point.test.name
  group_by         = ["..."]
  group_wait       = "30s"
  group_interval   = "5m"
  repeat_interval  = "4h"
}
`, name)
}
