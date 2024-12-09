package grafana_test

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"

	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
)

func TestAccContactPoint_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.0.0")

	var points models.ContactPoints

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		// Implicitly tests deletion.
		CheckDestroy: alertingContactPointCheckExists.destroyed(&points, nil),
		Steps: []resource.TestStep{
			// Test creation.
			{
				Config: testutils.TestAccExample(t, "resources/grafana_contact_point/resource.tf"),
				Check: resource.ComposeTestCheckFunc(
					checkAlertingContactPointExistsWithLength("grafana_contact_point.my_contact_point", &points, 1),
					resource.TestCheckResourceAttr("grafana_contact_point.my_contact_point", "name", "My Contact Point"),
					resource.TestCheckResourceAttr("grafana_contact_point.my_contact_point", "email.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.my_contact_point", "email.0.disable_resolve_message", "false"),
					resource.TestCheckResourceAttr("grafana_contact_point.my_contact_point", "email.0.addresses.0", "one@company.org"),
					resource.TestCheckResourceAttr("grafana_contact_point.my_contact_point", "email.0.addresses.1", "two@company.org"),
				),
			},
			// Test import.
			{
				ResourceName:      "grafana_contact_point.my_contact_point",
				ImportState:       true,
				ImportStateId:     "My Contact Point",
				ImportStateVerify: true,
			},
			// Test update content.
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_contact_point/resource.tf", map[string]string{
					"company.org": "user.net",
				}),
				Check: resource.ComposeTestCheckFunc(
					checkAlertingContactPointExistsWithLength("grafana_contact_point.my_contact_point", &points, 1),
					resource.TestCheckResourceAttr("grafana_contact_point.my_contact_point", "email.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.my_contact_point", "email.0.addresses.0", "one@user.net"),
					resource.TestCheckResourceAttr("grafana_contact_point.my_contact_point", "email.0.addresses.1", "two@user.net"),
				),
			},
			// Test rename.
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_contact_point/resource.tf", map[string]string{
					"My Contact Point": "A Different Contact Point",
				}),
				Check: resource.ComposeTestCheckFunc(
					checkAlertingContactPointExistsWithLength("grafana_contact_point.my_contact_point", &points, 1),
					resource.TestCheckResourceAttr("grafana_contact_point.my_contact_point", "name", "A Different Contact Point"),
					resource.TestCheckResourceAttr("grafana_contact_point.my_contact_point", "email.#", "1"),
				),
			},
		},
	})
}

func TestAccContactPoint_compound(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.0.0")

	var points models.ContactPoints

	// TODO: Make parallelizable
	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		// Implicitly tests deletion.
		CheckDestroy: alertingContactPointCheckExists.destroyed(&points, nil),
		Steps: []resource.TestStep{
			// Test creation.
			{
				Config: testutils.TestAccExample(t, "resources/grafana_contact_point/_acc_compound_receiver.tf"),
				Check: resource.ComposeTestCheckFunc(
					checkAlertingContactPointExistsWithLength("grafana_contact_point.compound_contact_point", &points, 2),
					resource.TestCheckResourceAttr("grafana_contact_point.compound_contact_point", "name", "Compound Contact Point"),
					resource.TestCheckResourceAttr("grafana_contact_point.compound_contact_point", "email.#", "2"),
				),
			},
			// Test import by name
			{
				ResourceName:      "grafana_contact_point.compound_contact_point",
				ImportState:       true,
				ImportStateId:     "Compound Contact Point",
				ImportStateVerify: true,
			},
			// Test update.
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_contact_point/_acc_compound_receiver.tf", map[string]string{
					"one": "asdf",
				}),
				Check: resource.ComposeTestCheckFunc(
					checkAlertingContactPointExistsWithLength("grafana_contact_point.compound_contact_point", &points, 2),
					resource.TestCheckResourceAttr("grafana_contact_point.compound_contact_point", "email.#", "2"),
					resource.TestCheckResourceAttr("grafana_contact_point.compound_contact_point", "email.0.addresses.0", "asdf@company.org"),
				),
			},
			// Test addition of a contact point to an existing compound one.
			{
				Config: testutils.TestAccExample(t, "resources/grafana_contact_point/_acc_compound_receiver_added.tf"),
				Check: resource.ComposeTestCheckFunc(
					checkAlertingContactPointExistsWithLength("grafana_contact_point.compound_contact_point", &points, 3),
					resource.TestCheckResourceAttr("grafana_contact_point.compound_contact_point", "email.#", "3"),
					resource.TestCheckResourceAttr("grafana_contact_point.compound_contact_point", "email.0.addresses.0", "five@company.org"),
					resource.TestCheckResourceAttr("grafana_contact_point.compound_contact_point", "email.1.addresses.0", "one@company.org"),
					resource.TestCheckResourceAttr("grafana_contact_point.compound_contact_point", "email.2.addresses.0", "three@company.org"),
				),
			},
			// Test removal of a point from a compound one does not leak.
			{
				Config: testutils.TestAccExample(t, "resources/grafana_contact_point/_acc_compound_receiver_subtracted.tf"),
				Check: resource.ComposeTestCheckFunc(
					checkAlertingContactPointExistsWithLength("grafana_contact_point.compound_contact_point", &points, 1),
					resource.TestCheckResourceAttr("grafana_contact_point.compound_contact_point", "email.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.compound_contact_point", "email.0.addresses.0", "one@company.org"),
				),
			},
			// Test rename.
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_contact_point/_acc_compound_receiver.tf", map[string]string{
					"Compound Contact Point": "A Different Contact Point",
				}),
				Check: resource.ComposeTestCheckFunc(
					checkAlertingContactPointExistsWithLength("grafana_contact_point.compound_contact_point", &points, 2),
					resource.TestCheckResourceAttr("grafana_contact_point.compound_contact_point", "name", "A Different Contact Point"),
					resource.TestCheckResourceAttr("grafana_contact_point.compound_contact_point", "email.#", "2"),
				),
			},
		},
	})
}

func TestAccContactPoint_notifiers(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.1.0")

	var points models.ContactPoints

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		// Implicitly tests deletion.
		CheckDestroy: alertingContactPointCheckExists.destroyed(&points, nil),
		Steps: []resource.TestStep{
			// Test creation.
			{
				Config: testutils.TestAccExample(t, "resources/grafana_contact_point/_acc_receiver_types.tf"),
				Check: resource.ComposeTestCheckFunc(
					checkAlertingContactPointExistsWithLength("grafana_contact_point.receiver_types", &points, 19),
					// alertmanager
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "alertmanager.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "alertmanager.0.url", "http://my-am"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "alertmanager.0.basic_auth_user", "user"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "alertmanager.0.basic_auth_password", "password"),
					// dingding
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "dingding.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "dingding.0.url", "http://dingding-url"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "dingding.0.message_type", "link"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "dingding.0.message", "message"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "dingding.0.title", "title"),
					// discord
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "discord.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "discord.0.url", "http://discord-url"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "discord.0.title", "title"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "discord.0.message", "message"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "discord.0.avatar_url", "avatar_url"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "discord.0.use_discord_username", "true"),
					// email
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "email.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "email.0.addresses.0", "one@company.org"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "email.0.message", "message"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "email.0.subject", "subject"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "email.0.single_email", "true"),
					// googlechat
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "googlechat.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "googlechat.0.url", "http://googlechat-url"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "googlechat.0.title", "title"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "googlechat.0.message", "message"),
					// kafka
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "kafka.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "kafka.0.rest_proxy_url", "http://kafka-rest-proxy-url"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "kafka.0.topic", "mytopic"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "kafka.0.description", "description"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "kafka.0.details", "details"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "kafka.0.username", "username"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "kafka.0.password", "password"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "kafka.0.api_version", "v3"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "kafka.0.cluster_id", "cluster_id"),
					// line
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "line.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "line.0.token", "token"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "line.0.title", "title"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "line.0.description", "description"),
					// opsgenie
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "opsgenie.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "opsgenie.0.url", "http://opsgenie-api"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "opsgenie.0.api_key", "token"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "opsgenie.0.message", "message"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "opsgenie.0.description", "description"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "opsgenie.0.auto_close", "true"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "opsgenie.0.override_priority", "true"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "opsgenie.0.send_tags_as", "both"),
					// pagerduty
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "pagerduty.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "pagerduty.0.integration_key", "token"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "pagerduty.0.severity", "critical"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "pagerduty.0.class", "ping failure"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "pagerduty.0.component", "mysql"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "pagerduty.0.group", "my service"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "pagerduty.0.summary", "message"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "pagerduty.0.source", "source"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "pagerduty.0.client", "client"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "pagerduty.0.client_url", "http://pagerduty"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "pagerduty.0.details.one", "two"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "pagerduty.0.details.three", "four"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "pagerduty.0.url", "http://pagerduty-url"),
					// pushover
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "pushover.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "pushover.0.user_key", "userkey"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "pushover.0.api_token", "token"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "pushover.0.priority", "0"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "pushover.0.ok_priority", "0"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "pushover.0.retry", "45"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "pushover.0.expire", "80000"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "pushover.0.device", "device"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "pushover.0.sound", "bugle"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "pushover.0.ok_sound", "cashregister"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "pushover.0.title", "title"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "pushover.0.message", "message"),
					// sensugo
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "sensugo.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "sensugo.0.url", "http://sensugo-url"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "sensugo.0.api_key", "key"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "sensugo.0.entity", "entity"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "sensugo.0.check", "check"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "sensugo.0.namespace", "namespace"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "sensugo.0.handler", "handler"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "sensugo.0.message", "message"),
					// slack
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "slack.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "slack.0.endpoint_url", "http://custom-slack-url"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "slack.0.token", "xoxb-token"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "slack.0.recipient", "#channel"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "slack.0.text", "message"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "slack.0.title", "title"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "slack.0.username", "bot"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "slack.0.icon_emoji", ":icon:"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "slack.0.icon_url", "http://domain/icon.png"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "slack.0.mention_channel", "here"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "slack.0.mention_users", "user"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "slack.0.mention_groups", "group"),
					// teams
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "teams.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "teams.0.url", "http://teams-webhook"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "teams.0.message", "message"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "teams.0.title", "title"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "teams.0.section_title", "section"),
					// telegram
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "telegram.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "telegram.0.token", "token"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "telegram.0.chat_id", "chat-id"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "telegram.0.message_thread_id", "5"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "telegram.0.message", "message"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "telegram.0.parse_mode", "Markdown"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "telegram.0.disable_web_page_preview", "true"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "telegram.0.protect_content", "true"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "telegram.0.disable_notifications", "true"),
					// threema
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "threema.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "threema.0.gateway_id", "*gateway"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "threema.0.recipient_id", "*target1"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "threema.0.api_secret", "secret"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "threema.0.title", "title"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "threema.0.description", "description"),
					// victorops
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "victorops.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "victorops.0.url", "http://victor-ops-url"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "victorops.0.message_type", "CRITICAL"),
					// webex
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webex.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webex.0.token", "token"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webex.0.api_url", "http://localhost"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webex.0.message", "message"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webex.0.room_id", "room_id"),
					// webhook
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.url", "http://my-url"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.http_method", "POST"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.basic_auth_user", "user"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.basic_auth_password", "password"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.max_alerts", "100"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.message", "Custom message"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.title", "Custom title"),
					// wecom
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "wecom.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "wecom.0.url", "http://wecom-url"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "wecom.0.message", "message"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "wecom.0.title", "title"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "wecom.0.secret", "secret"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "wecom.0.corp_id", "corp_id"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "wecom.0.agent_id", "agent_id"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "wecom.0.msg_type", "text"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "wecom.0.to_user", "to_user"),
				),
			},
			// Test blank fields in settings should be omitted.
			{
				Config: testutils.TestAccExample(t, "resources/grafana_contact_point/_acc_default_settings.tf"),
				Check: resource.ComposeTestCheckFunc(
					checkAlertingContactPointExistsWithLength("grafana_contact_point.default_settings", &points, 1),
					resource.TestCheckResourceAttr("grafana_contact_point.default_settings", "slack.#", "1"),
					resource.TestCheckNoResourceAttr("grafana_contact_point.default_settings", "slack.endpoint_url"),
					func(s *terraform.State) error {
						if val, ok := points[0].Settings.(map[string]interface{})["endpointUrl"]; ok {
							return fmt.Errorf("endpointUrl was still present in the settings when it should have been omitted. value: %#v", val)
						}

						return nil
					},
				),
			},
		},
	})
}

func TestAccContactPoint_notifiers10_2(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=10.2.0")

	var points models.ContactPoints

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		// Implicitly tests deletion.
		CheckDestroy: alertingContactPointCheckExists.destroyed(&points, nil),
		Steps: []resource.TestStep{
			// Test creation.
			{
				Config: testutils.TestAccExample(t, "resources/grafana_contact_point/_acc_receiver_types_10_2.tf"),
				Check: resource.ComposeTestCheckFunc(
					checkAlertingContactPointExistsWithLength("grafana_contact_point.receiver_types", &points, 1),
					// oncall
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "oncall.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "oncall.0.url", "http://my-url"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "oncall.0.http_method", "POST"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "oncall.0.basic_auth_user", "user"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "oncall.0.basic_auth_password", "password"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "oncall.0.max_alerts", "100"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "oncall.0.message", "Custom message"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "oncall.0.title", "Custom title"),
				),
			},
		},
	})
}

func TestAccContactPoint_notifiers10_3(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=10.3.0")

	var points models.ContactPoints

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		// Implicitly tests deletion.
		CheckDestroy: alertingContactPointCheckExists.destroyed(&points, nil),
		Steps: []resource.TestStep{
			// Test creation.
			{
				Config: testutils.TestAccExample(t, "resources/grafana_contact_point/_acc_receiver_types_10_3.tf"),
				Check: resource.ComposeTestCheckFunc(
					checkAlertingContactPointExistsWithLength("grafana_contact_point.receiver_types", &points, 1),
					// opsgenie
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "opsgenie.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "opsgenie.0.url", "http://opsgenie-api"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "opsgenie.0.api_key", "token"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "opsgenie.0.message", "message"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "opsgenie.0.description", "description"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "opsgenie.0.auto_close", "true"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "opsgenie.0.override_priority", "true"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "opsgenie.0.send_tags_as", "both"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "opsgenie.0.responders.0.type", "user"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "opsgenie.0.responders.0.id", "803f87e1a7f848b0a0779810bee5d1d3"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "opsgenie.0.responders.1.type", "team"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "opsgenie.0.responders.1.name", "Test team"),
				),
			},
		},
	})
}

func TestAccContactPoint_notifiers11_3(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=11.3.0")

	var points models.ContactPoints

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		// Implicitly tests deletion.
		CheckDestroy: alertingContactPointCheckExists.destroyed(&points, nil),
		Steps: []resource.TestStep{
			// Test creation.
			{
				Config: testutils.TestAccExample(t, "resources/grafana_contact_point/_acc_receiver_types_11_3.tf"),
				Check: resource.ComposeTestCheckFunc(
					checkAlertingContactPointExistsWithLength("grafana_contact_point.receiver_types", &points, 1),
					// mqtt
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "mqtt.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "mqtt.0.broker_url", "tcp://localhost:1883"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "mqtt.0.topic", "grafana/alerts"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "mqtt.0.client_id", "grafana"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "mqtt.0.message_format", "json"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "mqtt.0.username", "user"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "mqtt.0.password", "password123"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "mqtt.0.qos", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "mqtt.0.tls_config.insecure_skip_verify", "true"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "mqtt.0.tls_config.ca_certificate", "ca_cer"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "mqtt.0.tls_config.client_certificate", "client_cert"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "mqtt.0.tls_config.client_key", "client_key"),
				),
			},
		},
	})
}

func TestAccContactPoint_sensitiveData(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.1.0")

	var points models.ContactPoints
	name := acctest.RandString(10)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             alertingContactPointCheckExists.destroyed(&points, nil),
		Steps: []resource.TestStep{
			{
				Config: testAccContactPointWithSensitiveData(name, "https://api.eu.opsgenie.com/v2/alerts", "mykey"),
				Check: resource.ComposeTestCheckFunc(
					checkAlertingContactPointExistsWithLength("grafana_contact_point.test", &points, 1),
					resource.TestCheckResourceAttr("grafana_contact_point.test", "name", name),
					resource.TestCheckResourceAttr("grafana_contact_point.test", "opsgenie.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.test", "opsgenie.0.url", "https://api.eu.opsgenie.com/v2/alerts"),
					resource.TestCheckResourceAttr("grafana_contact_point.test", "opsgenie.0.api_key", "mykey"),
				),
			},
			// Update non-sensitive data
			{
				Config: testAccContactPointWithSensitiveData(name, "http://my-url", "mykey"),
				Check: resource.ComposeTestCheckFunc(
					checkAlertingContactPointExistsWithLength("grafana_contact_point.test", &points, 1),
					resource.TestCheckResourceAttr("grafana_contact_point.test", "name", name),
					resource.TestCheckResourceAttr("grafana_contact_point.test", "opsgenie.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.test", "opsgenie.0.url", "http://my-url"),
					resource.TestCheckResourceAttr("grafana_contact_point.test", "opsgenie.0.api_key", "mykey"),
				),
			},
			// Update sensitive data
			{
				Config: testAccContactPointWithSensitiveData(name, "http://my-url", "mykey2"),
				Check: resource.ComposeTestCheckFunc(
					checkAlertingContactPointExistsWithLength("grafana_contact_point.test", &points, 1),
					resource.TestCheckResourceAttr("grafana_contact_point.test", "name", name),
					resource.TestCheckResourceAttr("grafana_contact_point.test", "opsgenie.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.test", "opsgenie.0.url", "http://my-url"),
					resource.TestCheckResourceAttr("grafana_contact_point.test", "opsgenie.0.api_key", "mykey2"),
				),
			},
		},
	})
}

func TestAccContactPoint_inOrg(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.1.0")

	var points models.ContactPoints
	var org models.OrgDetailsDTO
	name := acctest.RandString(10)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             orgCheckExists.destroyed(&org, nil),
		Steps: []resource.TestStep{
			// Creation
			{
				Config: testAccContactPointInOrg(name),
				Check: resource.ComposeTestCheckFunc(
					orgCheckExists.exists("grafana_organization.test", &org),
					checkAlertingContactPointExistsWithLength("grafana_contact_point.test", &points, 1),
					resource.TestCheckResourceAttr("grafana_contact_point.test", "name", name),
					resource.TestCheckResourceAttr("grafana_contact_point.test", "email.#", "1"),
					checkResourceIsInOrg("grafana_contact_point.test", "grafana_organization.test"),
				),
			},
			// Import
			{
				ResourceName:      "grafana_contact_point.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Deletion
			{
				Config: testutils.WithoutResource(t, testAccContactPointInOrg(name), "grafana_contact_point.test"),
				Check: resource.ComposeTestCheckFunc(
					orgCheckExists.exists("grafana_organization.test", &org),
					alertingContactPointCheckExists.destroyed(&points, nil),
				),
			},
		},
	})
}

func TestAccContactPoint_recreate(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.1.0")

	var points models.ContactPoints
	name := acctest.RandString(10)
	config := testutils.TestAccExampleWithReplace(t, "resources/grafana_contact_point/resource.tf", map[string]string{
		"My Contact Point": name,
	})

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			// Creation
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					checkAlertingContactPointExistsWithLength("grafana_contact_point.my_contact_point", &points, 1),
					resource.TestCheckResourceAttr("grafana_contact_point.my_contact_point", "name", name),
					resource.TestCheckResourceAttr("grafana_contact_point.my_contact_point", "email.#", "1"),
				),
			},
			// Delete the contact point and check that it is missing
			{
				PreConfig: func() {
					client := grafanaTestClient()
					for _, point := range points {
						_, err := client.Provisioning.DeleteContactpoints(point.UID)
						require.NoError(t, err)
					}
				},
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			// Recreate the contact point
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					checkAlertingContactPointExistsWithLength("grafana_contact_point.my_contact_point", &points, 1),
					resource.TestCheckResourceAttr("grafana_contact_point.my_contact_point", "name", name),
					resource.TestCheckResourceAttr("grafana_contact_point.my_contact_point", "email.#", "1"),
				),
			},
		},
	})
}

func TestAccContactPoint_empty(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.1.0")

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			// Test creation.
			{
				Config: `
				resource "grafana_contact_point" "dev_null" {
					name = "empty-test"
				}
				`,
				ExpectError: regexp.MustCompile(`Missing required argument`),
			},
		},
	})
}

func TestAccContactPoint_disableProvenance(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.1.0")

	var points models.ContactPoints
	name := acctest.RandString(10)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             alertingContactPointCheckExists.destroyed(&points, nil),
		Steps: []resource.TestStep{
			// Create
			{
				Config: testContactPointDisableProvenance(name, false),
				Check: resource.ComposeTestCheckFunc(
					checkAlertingContactPointExistsWithLength("grafana_contact_point.my_contact_point", &points, 1),
					resource.TestCheckResourceAttr("grafana_contact_point.my_contact_point", "name", name),
					resource.TestCheckResourceAttr("grafana_contact_point.my_contact_point", "disable_provenance", "false"),
				),
			},
			// Import (tests that disable_provenance is fetched from API)
			{
				ResourceName:      "grafana_contact_point.my_contact_point",
				ImportState:       true,
				ImportStateId:     name,
				ImportStateVerify: true,
			},
			// Disable provenance
			{
				Config: testContactPointDisableProvenance(name, true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_contact_point.my_contact_point", "name", name),
					resource.TestCheckResourceAttr("grafana_contact_point.my_contact_point", "disable_provenance", "true"),
				),
			},
			// Import (tests that disable_provenance is fetched from API)
			{
				ResourceName:      "grafana_contact_point.my_contact_point",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Re-enable provenance
			{
				Config: testContactPointDisableProvenance(name, false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_contact_point.my_contact_point", "name", name),
					resource.TestCheckResourceAttr("grafana_contact_point.my_contact_point", "disable_provenance", "false"),
				),
			},
		},
	})
}

func checkAlertingContactPointExistsWithLength(rn string, v *models.ContactPoints, expectedLength int) resource.TestCheckFunc {
	return resource.ComposeTestCheckFunc(
		alertingContactPointCheckExists.exists(rn, v),
		func(s *terraform.State) error {
			if len(*v) != expectedLength {
				receivers := make([]string, len(*v))
				for i, v := range *v {
					receivers[i] = fmt.Sprintf("%+v", v)
				}
				return fmt.Errorf("expected %d contact points, got %d. Receivers:\n%s", expectedLength, len(*v), strings.Join(receivers, "\n"))
			}
			return nil
		},
	)
}

func testContactPointDisableProvenance(name string, disableProvenance bool) string {
	return fmt.Sprintf(`
	resource "grafana_contact_point" "my_contact_point" {
		name      = "%s"
		disable_provenance = %t
		email {
			addresses = [ "hello@example.com" ]
		}
	  }
	`, name, disableProvenance)
}

func testAccContactPointInOrg(name string) string {
	return fmt.Sprintf(`
	resource "grafana_organization" "test" {
		name = "%[1]s"
	}

	resource "grafana_contact_point" "test" {
		org_id = grafana_organization.test.id
		name = "%[1]s"
		email {
			addresses = [ "hello@example.com" ]
		}
	}
	`, name)
}

func testAccContactPointWithSensitiveData(name, url, apiKey string) string {
	return fmt.Sprintf(`
	resource "grafana_contact_point" "test" {
		name = "%[1]s"
		opsgenie {
			url               = "%[2]s"
			api_key           = "%[3]s"
			message           = "{{ .CommonAnnotations.summary }}"
			send_tags_as      = "tags"
			override_priority = true
			settings = {
			  tags        = <<EOT
		{{- range .Alerts -}}
		  {{- range .Labels.SortedPairs -}}
			{{- if and (ne .Name "severity") (ne .Name "destination") -}}
			  {{ .Name }}={{ .Value }},
			{{- end -}}
		  {{- end -}}
		{{- end -}}
		EOT
			  og_priority = <<EOT
		{{- range .Alerts -}}
		  {{- range .Labels.SortedPairs -}}
			{{- if eq .Name "severity" -}}
			  {{- if eq .Value "warning" -}}
				P5
			  {{- else if eq .Value "critical" -}}
				P3
			  {{- else -}}
				{{ .Value }}
			  {{- end -}}
			{{- end -}}
		  {{- end -}}
		{{- end -}}
		EOT
			}
		  }
	}`, name, url, apiKey)
}
