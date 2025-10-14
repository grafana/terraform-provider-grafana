package grafana_test

import (
	"fmt"
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
)

func TestAccMuteTiming_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">9.0.0")

	var mt models.MuteTimeInterval

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		// Implicitly tests deletion.
		CheckDestroy: alertingMuteTimingCheckExists.destroyed(&mt, nil),
		Steps: []resource.TestStep{
			// Test creation.
			{
				Config: testutils.TestAccExample(t, "resources/grafana_mute_timing/resource.tf"),
				Check: resource.ComposeTestCheckFunc(
					alertingMuteTimingCheckExists.exists("grafana_mute_timing.my_mute_timing", &mt),
					resource.TestCheckResourceAttr("grafana_mute_timing.my_mute_timing", "name", "My Mute Timing"),
					resource.TestCheckResourceAttr("grafana_mute_timing.my_mute_timing", "intervals.0.times.0.start", "04:56"),
					resource.TestCheckResourceAttr("grafana_mute_timing.my_mute_timing", "intervals.0.times.0.end", "14:17"),
					resource.TestCheckResourceAttr("grafana_mute_timing.my_mute_timing", "intervals.0.weekdays.0", "monday"),
					resource.TestCheckResourceAttr("grafana_mute_timing.my_mute_timing", "intervals.0.weekdays.1", "tuesday:thursday"),
					resource.TestCheckResourceAttr("grafana_mute_timing.my_mute_timing", "intervals.0.days_of_month.0", "1:7"),
					resource.TestCheckResourceAttr("grafana_mute_timing.my_mute_timing", "intervals.0.days_of_month.1", "-1"),
					resource.TestCheckResourceAttr("grafana_mute_timing.my_mute_timing", "intervals.0.months.0", "1:3"),
					resource.TestCheckResourceAttr("grafana_mute_timing.my_mute_timing", "intervals.0.months.1", "12"),
					resource.TestCheckResourceAttr("grafana_mute_timing.my_mute_timing", "intervals.0.years.0", "2030"),
					resource.TestCheckResourceAttr("grafana_mute_timing.my_mute_timing", "intervals.0.years.1", "2025:2026"),
					resource.TestCheckResourceAttr("grafana_mute_timing.my_mute_timing", "intervals.0.location", "America/New_York"),
					testutils.CheckLister("grafana_mute_timing.my_mute_timing"),
				),
			},
			// Test import.
			{
				ResourceName:            "grafana_mute_timing.my_mute_timing",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"disable_provenance"},
			},
			// Test plan (should be empty)
			{
				Config:   testutils.TestAccExample(t, "resources/grafana_mute_timing/resource.tf"),
				PlanOnly: true,
			},
			// Test update content.
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_mute_timing/resource.tf", map[string]string{
					"monday": "friday",
				}),
				Check: resource.ComposeTestCheckFunc(
					alertingMuteTimingCheckExists.exists("grafana_mute_timing.my_mute_timing", &mt),
					resource.TestCheckResourceAttr("grafana_mute_timing.my_mute_timing", "intervals.0.weekdays.0", "friday"),
					resource.TestCheckResourceAttr("grafana_mute_timing.my_mute_timing", "intervals.0.weekdays.1", "tuesday:thursday"),
				),
			},
			// Test rename.
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_mute_timing/resource.tf", map[string]string{
					"My Mute Timing": "A Different Mute Timing",
				}),
				Check: resource.ComposeTestCheckFunc(
					alertingMuteTimingCheckExists.exists("grafana_mute_timing.my_mute_timing", &mt),
					resource.TestCheckResourceAttr("grafana_mute_timing.my_mute_timing", "name", "A Different Mute Timing"),
					alertingMuteTimingCheckExists.destroyed(&models.MuteTimeInterval{Name: "My Mute Timing"}, nil),
				),
			},
		},
	})
}

func TestAccMuteTiming_AllTime(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">9.0.0")

	var mt models.MuteTimeInterval
	name := "My-Mute-Timing"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             alertingMuteTimingCheckExists.destroyed(&mt, nil),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "grafana_mute_timing" "my_mute_timing" {
	  name = "%s"
	  intervals {}
}`, name),
				Check: resource.ComposeTestCheckFunc(
					alertingMuteTimingCheckExists.exists("grafana_mute_timing.my_mute_timing", &mt),
					resource.TestCheckResourceAttr("grafana_mute_timing.my_mute_timing", "name", name),
					resource.TestCheckResourceAttr("grafana_mute_timing.my_mute_timing", "intervals.0.times.#", "0"),
					resource.TestCheckResourceAttr("grafana_mute_timing.my_mute_timing", "intervals.0.weekdays.#", "0"),
					resource.TestCheckResourceAttr("grafana_mute_timing.my_mute_timing", "intervals.0.days_of_month.#", "0"),
					resource.TestCheckResourceAttr("grafana_mute_timing.my_mute_timing", "intervals.0.months.#", "0"),
					resource.TestCheckResourceAttr("grafana_mute_timing.my_mute_timing", "intervals.0.years.#", "0"),
					resource.TestCheckNoResourceAttr("grafana_mute_timing.my_mute_timing", "intervals.0.location"),
				),
			},
		},
	})
}

func TestAccMuteTiming_RemoveInUse(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">9.0.0")

	config := func(mute bool) string {
		return fmt.Sprintf(`
		locals {
			use_mute = %t
		}

		resource "grafana_organization" "my_org" {
			name = "mute-timing-test"
		}

		resource "grafana_contact_point" "default_policy" {
			org_id = grafana_organization.my_org.id
			name   = "default-policy"
			email {
				addresses = ["test@example.com"]
			}
		}

		resource "grafana_notification_policy" "org_policy" {
			org_id             = grafana_organization.my_org.id
			group_by           = ["..."]
			group_wait         = "45s"
			group_interval     = "6m"
			repeat_interval    = "3h"
			contact_point      = grafana_contact_point.default_policy.name

			policy {
				mute_timings = local.use_mute ? [grafana_mute_timing.test[0].name] : []
				contact_point = grafana_contact_point.default_policy.name
			}
		}

		resource "grafana_mute_timing" "test" {
			count = local.use_mute ? 1 : 0
			org_id = grafana_organization.my_org.id
			name = "test-mute-timing"
			intervals {}
		}`, mute)
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config(true),
			},
			{
				Config: config(false),
			},
		},
	})
}

func TestAccMuteTiming_RemoveInUseInAlertRule(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=11.2.0")

	randomStr := acctest.RandString(6)

	config := func(mute bool) string {
		return fmt.Sprintf(`
		locals {
			use_mute = %[2]t
		}

		resource "grafana_folder" "rule_folder" {
			title  = "%[1]s"
		}

		resource "grafana_contact_point" "default_policy" {
			name   = "%[1]s"
			email {
				addresses = ["test@example.com"]
			}
		}

		resource "grafana_rule_group" "this" {
			name               = "%[1]s"
			folder_uid         = grafana_folder.rule_folder.uid
			interval_seconds   = 60

			rule {
				name = "%[1]s"
				condition      = "B"
				notification_settings {
					contact_point = grafana_contact_point.default_policy.name
					group_by      = ["..."]
					mute_timings  = local.use_mute ? [grafana_mute_timing.test[0].name] : []
				}
				data {
					ref_id     = "A"
					query_type = ""
					relative_time_range {
						from = 600
						to   = 0
					}
					datasource_uid = "PD8C576611E62080A"
					model = jsonencode({
						hide          = false
						intervalMs    = 1000
						maxDataPoints = 43200
						refId         = "A"
					})
				}
			}
		}


		resource "grafana_mute_timing" "test" {
			count = local.use_mute ? 1 : 0
			name = "%[1]s"
			intervals {}
		}`, randomStr, mute)
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config(true),
			},
			{
				Config: config(false),
			},
		},
	})
}
