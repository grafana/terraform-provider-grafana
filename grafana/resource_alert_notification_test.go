package grafana

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"

	gapi "github.com/nytm/go-grafana-api"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

func TestAccAlertNotification_basic(t *testing.T) {
	var alertNotification gapi.AlertNotification

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccAlertNotificationCheckDestroy(&alertNotification),
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
						"grafana_alert_notification.test", "id", regexp.MustCompile(`\d+`),
					),
					resource.TestCheckResourceAttr(
						"grafana_alert_notification.test", "send_reminder", "true",
					),
					resource.TestCheckResourceAttr(
						"grafana_alert_notification.test", "frequency", "12h",
					),
					resource.TestCheckResourceAttr(
						"grafana_alert_notification.test", "settings.addresses", "foo@bar.test",
					),
				),
			},
		},
	})
}

func TestAccAlertNotification_invalid_frequence(t *testing.T) {
	var alertNotification gapi.AlertNotification

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccAlertNotificationCheckDestroy(&alertNotification),
		Steps: []resource.TestStep{
			{
				ExpectError: regexp.MustCompile("invalid duration hi"),
				Config:      testAccAlertNotificationConfig_invalid_frequency,
			},
		},
	})
}

func TestAccAlertNotification_reminder_no_frequence(t *testing.T) {
	var alertNotification gapi.AlertNotification

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccAlertNotificationCheckDestroy(&alertNotification),
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

		client := testAccProvider.Meta().(*gapi.Client)
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
		client := testAccProvider.Meta().(*gapi.Client)
		alert, err := client.AlertNotification(a.Id)
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
