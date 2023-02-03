package provider

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/provider/common"
	"github.com/grafana/terraform-provider-grafana/provider/testutils"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccAlertNotification_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	var alertNotification gapi.AlertNotification

	// TODO: Make parallelizable
	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccAlertNotificationCheckDestroy(&alertNotification),
		Steps: []resource.TestStep{
			{
				Config: testAccAlertNotificationConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccAlertNotificationCheckExists("grafana_alert_notification.test", &alertNotification),
					testAccAlertNotificationDefinition(&alertNotification),
					resource.TestCheckResourceAttr(
						"grafana_alert_notification.test", "type", "email",
					),
					resource.TestMatchResourceAttr(
						"grafana_alert_notification.test", "id", idRegexp,
					),
					resource.TestCheckResourceAttr(
						"grafana_alert_notification.test", "send_reminder", "true",
					),
					resource.TestCheckResourceAttr(
						"grafana_alert_notification.test", "frequency", "12h",
					),
					resource.TestCheckResourceAttr(
						"grafana_alert_notification.test", "disable_resolve_message", "false",
					),
					resource.TestCheckResourceAttr(
						"grafana_alert_notification.test", "settings.addresses", "foo@bar.test",
					),
				),
			},
		},
	})
}

func TestAccAlertNotification_disableResolveMessage(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	var alertNotification gapi.AlertNotification

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccAlertNotificationCheckDestroy(&alertNotification),
		Steps: []resource.TestStep{
			{
				Config: testAccAlertNotificationConfig_disable_resolve_message,
				Check: resource.ComposeTestCheckFunc(
					testAccAlertNotificationCheckExists("grafana_alert_notification.test", &alertNotification),
					testAccAlertNotificationDefinition(&alertNotification),
					resource.TestCheckResourceAttr(
						"grafana_alert_notification.test", "type", "email",
					),
					resource.TestMatchResourceAttr(
						"grafana_alert_notification.test", "id", idRegexp,
					),
					resource.TestCheckResourceAttr(
						"grafana_alert_notification.test", "send_reminder", "true",
					),
					resource.TestCheckResourceAttr(
						"grafana_alert_notification.test", "frequency", "12h",
					),
					resource.TestCheckResourceAttr(
						"grafana_alert_notification.test", "disable_resolve_message", "true",
					),
					resource.TestCheckResourceAttr(
						"grafana_alert_notification.test", "settings.addresses", "foo@bar.test",
					),
				),
			},
		},
	})
}

func TestAccAlertNotification_invalid_frequency(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	var alertNotification gapi.AlertNotification

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccAlertNotificationCheckDestroy(&alertNotification),
		Steps: []resource.TestStep{
			{
				ExpectError: regexp.MustCompile("time: invalid duration \"hi\""),
				Config:      testAccAlertNotificationConfig_invalid_frequency,
			},
		},
	})
}

func TestAccAlertNotification_reminder_no_frequency(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	var alertNotification gapi.AlertNotification

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccAlertNotificationCheckDestroy(&alertNotification),
		Steps: []resource.TestStep{
			{
				ExpectError: regexp.MustCompile("frequency must be set when send_reminder is set to 'true'"),
				Config:      testAccAlertNotificationConfig_reminder_no_frequency,
			},
		},
	})
}

func testAccAlertNotificationCheckExists(rn string, a *gapi.AlertNotification) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		id, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
		if err != nil {
			return fmt.Errorf("resource id is malformed")
		}

		client := testAccProvider.Meta().(*common.Client).GrafanaAPI
		gotAlertNotification, err := client.AlertNotification(id)
		if err != nil {
			return fmt.Errorf("error getting data source: %s", err)
		}

		*a = *gotAlertNotification

		return nil
	}
}

func testAccAlertNotificationDefinition(a *gapi.AlertNotification) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if !a.SendReminder {
			return fmt.Errorf("send_reminder is not set properly")
		}

		return nil
	}
}

func testAccAlertNotificationCheckDestroy(a *gapi.AlertNotification) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*common.Client).GrafanaAPI
		alert, err := client.AlertNotification(a.ID)
		if err == nil && alert != nil {
			return fmt.Errorf("alert-notification still exists")
		}
		return nil
	}
}

const testAccAlertNotificationConfig_basic = `
resource "grafana_alert_notification" "test" {
    type = "email"
    name = "terraform-acc-test"
		send_reminder = true
		frequency = "12h"
    settings = {
			"addresses" = "foo@bar.test"
			"uploadImage" = "false"
			"autoResolve" = "true"
		}
	secure_settings = {
		 "foo" = "true"
	}
}
`

const testAccAlertNotificationConfig_disable_resolve_message = `
resource "grafana_alert_notification" "test" {
    type = "email"
    name = "terraform-acc-test"
		send_reminder = true
		frequency = "12h"
		disable_resolve_message = true
    settings = {
			"addresses" = "foo@bar.test"
			"uploadImage" = "false"
			"autoResolve" = "true"
		}
}
`

const testAccAlertNotificationConfig_invalid_frequency = `
resource "grafana_alert_notification" "test" {
    type = "email"
    name = "terraform-acc-test"
		send_reminder = true
		frequency = "hi"
    settings = {
			"addresses" = "foo@bar.test"
			"uploadImage" = "false"
			"autoResolve" = "true"
		}
}
`

const testAccAlertNotificationConfig_reminder_no_frequency = `
resource "grafana_alert_notification" "test" {
    type = "email"
    name = "terraform-acc-test"
		send_reminder = true
    settings = {
			"addresses" = "foo@bar.test"
			"uploadImage" = "false"
			"autoResolve" = "true"
		}
}
`
