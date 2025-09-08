package grafana_test

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
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
					// Check that uid, disable_resolve_message and leftover "settings" and not in grafana payload.
					func(s *terraform.State) error {
						if val, ok := points[0].Settings.(map[string]any)["uid"]; ok {
							return fmt.Errorf("uid was present in the settings when it should have been omitted. value: %#v", val)
						}
						if val, ok := points[0].Settings.(map[string]any)["disable_resolve_message"]; ok {
							return fmt.Errorf("disable_resolve_message was present in the settings when it should have been omitted. value: %#v", val)
						}
						if val, ok := points[0].Settings.(map[string]any)["settings"]; ok {
							return fmt.Errorf("leftover settings was present in the settings when it should have been omitted. value: %#v", val)
						}
						return nil
					},
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
					checkAlertingContactPointExistsWithLength("grafana_contact_point.receiver_types", &points, 20),
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
					// slack url
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "slack.#", "2"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "slack.0.endpoint_url", "http://custom-slack-endpoint"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "slack.0.url", "http://custom-slack-url"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "slack.0.recipient", "#channel"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "slack.0.text", "message"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "slack.0.title", "title"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "slack.0.username", "bot"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "slack.0.icon_emoji", ":icon:"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "slack.0.icon_url", "http://domain/icon.png"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "slack.0.mention_channel", "here"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "slack.0.mention_users", "user"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "slack.0.mention_groups", "group"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "slack.0.color", "color"),
					// slack token
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "slack.1.endpoint_url", "http://custom-slack-url"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "slack.1.token", "xoxb-token"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "slack.1.recipient", "#channel"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "slack.1.text", "message"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "slack.1.title", "title"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "slack.1.username", "bot"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "slack.1.icon_emoji", ":icon:"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "slack.1.icon_url", "http://domain/icon.png"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "slack.1.mention_channel", "here"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "slack.1.mention_users", "user"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "slack.1.mention_groups", "group"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "slack.1.color", "color"),
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
			// The following test ensures that the plan remains empty for notifiers when updates are made to
			// other notifiers in the contact point. This is a regression test to catch certain notifiers without
			// required fields being deleted on update.
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_contact_point/_acc_receiver_types.tf", map[string]string{
					`victor-ops-url`: `updated-victor-ops-url`, // VictorOps is a "safe" update as it's not being tested.
				}),
				ExpectNonEmptyPlan: false,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "victorops.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "victorops.0.url", "http://updated-victor-ops-url"),
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
						if val, ok := points[0].Settings.(map[string]any)["endpointUrl"]; ok {
							return fmt.Errorf("endpointUrl was still present in the settings when it should have been omitted. value: %#v", val)
						}

						return nil
					},
				),
			},
		},
	})
}

func TestAccContactPoint_notifiers9_3(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.3.0")

	var points models.ContactPoints

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		// Implicitly tests deletion.
		CheckDestroy: alertingContactPointCheckExists.destroyed(&points, nil),
		Steps: []resource.TestStep{
			// Test creation.
			{
				Config: testutils.TestAccExample(t, "resources/grafana_contact_point/_acc_receiver_types_9_3.tf"),
				Check: resource.ComposeTestCheckFunc(
					checkAlertingContactPointExistsWithLength("grafana_contact_point.receiver_types", &points, 1),
					// oncall
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "victorops.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "victorops.0.url", "http://victor-ops-url"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "victorops.0.message_type", "CRITICAL"),

					// New
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "victorops.0.title", "title"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "victorops.0.description", "description"),
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
					checkAlertingContactPointExistsWithLength("grafana_contact_point.receiver_types", &points, 2),
					// oncall
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "oncall.#", "2"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "oncall.0.url", "http://my-url"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "oncall.0.http_method", "POST"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "oncall.0.basic_auth_user", "user"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "oncall.0.basic_auth_password", "password"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "oncall.0.max_alerts", "100"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "oncall.0.message", "Custom message"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "oncall.0.title", "Custom title"),

					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "oncall.1.url", "http://my-url"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "oncall.1.http_method", "POST"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "oncall.1.authorization_scheme", "Basic"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "oncall.1.authorization_credentials", "dXNlcjpwYXNzd29yZA=="),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "oncall.1.max_alerts", "100"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "oncall.1.message", "Custom message"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "oncall.1.title", "Custom title"),
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

func TestAccContactPoint_notifiers11_4(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=11.4.0")

	var points models.ContactPoints

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		// Implicitly tests deletion.
		CheckDestroy: alertingContactPointCheckExists.destroyed(&points, nil),
		Steps: []resource.TestStep{
			// Test creation.
			{
				Config: testutils.TestAccExample(t, "resources/grafana_contact_point/_acc_receiver_types_11_4.tf"),
				Check: resource.ComposeTestCheckFunc(
					checkAlertingContactPointExistsWithLength("grafana_contact_point.receiver_types", &points, 2),
					// webhook basic
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.#", "2"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.url", "http://my-url"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.http_method", "POST"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.basic_auth_user", "user"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.basic_auth_password", "password"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.max_alerts", "100"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.message", "Custom message"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.title", "Custom title"),

					// New
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.tls_config.insecure_skip_verify", "true"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.tls_config.ca_certificate", "ca.crt"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.tls_config.client_certificate", "client.crt"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.tls_config.client_key", "client.key"),

					// webhook auth creds
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.1.url", "http://my-url"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.1.http_method", "POST"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.1.authorization_scheme", "Basic"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.1.authorization_credentials", "dXNlcjpwYXNzd29yZA=="),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.1.max_alerts", "100"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.1.message", "Custom message"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.1.title", "Custom title"),

					// New
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.1.tls_config.insecure_skip_verify", "true"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.1.tls_config.ca_certificate", "ca.crt"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.1.tls_config.client_certificate", "client.crt"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.1.tls_config.client_key", "client.key"),
				),
			},
		},
	})
}

func TestAccContactPoint_notifiers11_6(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=11.6.0")

	var points models.ContactPoints

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		// Implicitly tests deletion.
		CheckDestroy: alertingContactPointCheckExists.destroyed(&points, nil),
		Steps: []resource.TestStep{
			// Multiple hmac_config blocks are not allowed.
			{
				Config: `
				resource "grafana_contact_point" "receiver_types" {
				  name = "Receiver Types since v11.6"
				
				  webhook {
					url = "http://my-url"
					hmac_config {
					  secret = "test-secret1"
					}
					hmac_config {
					  secret = "test-secret2"
					}
				  }
				}
				`,
				ExpectError: regexp.MustCompile(`Too many hmac_config blocks`),
			},
			// Secret field required.
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_contact_point/_acc_receiver_types_11_6.tf", map[string]string{
					`secret           = "test-hmac-secret"`: ``,
				}),
				ExpectError: regexp.MustCompile(`Missing required argument`),
			},
			// Test creation.
			{
				Config: testutils.TestAccExample(t, "resources/grafana_contact_point/_acc_receiver_types_11_6.tf"),
				Check: resource.ComposeTestCheckFunc(
					checkAlertingContactPointExistsWithLength("grafana_contact_point.receiver_types", &points, 2),
					// webhook basic
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.#", "2"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.url", "http://hmac-minimal-webhook-url"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.1.url", "http://hmac-webhook-url"),

					// New
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.hmac_config.0.secret", "test-hmac-minimal-secret"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.hmac_config.0.header", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.hmac_config.0.timestamp_header", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.1.hmac_config.0.secret", "test-hmac-secret"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.1.hmac_config.0.header", "X-Grafana-Alerting-Signature"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.1.hmac_config.0.timestamp_header", "X-Grafana-Alerting-Timestamp"),

					// Ensure that we're sending the grafana-style fiel names to the API by checking the GET response.
					func(s *terraform.State) error {
						found := false
						for _, p := range points {
							hmacConfig, ok := p.Settings.(map[string]interface{})["hmacConfig"]
							if !ok {
								return fmt.Errorf("hmacConfig was not present in the settings when it should have been. value: %#v", p.Settings)
							}
							if _, ok := hmacConfig.(map[string]interface{})["timestampHeader"]; ok {
								found = true
							}
						}
						if !found {
							return fmt.Errorf("timestampHeader was not present in any hmacConfig when it should have been. Settings: [%v, %v]", points[0].Settings, points[1].Settings)
						}
						return nil
					},
				),
			},
		},
	})
}

func TestAccContactPoint_notifiers12_0(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=12.0.0")

	var points models.ContactPoints

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		// Implicitly tests deletion.
		CheckDestroy: alertingContactPointCheckExists.destroyed(&points, nil),
		Steps: []resource.TestStep{
			// Multiple payload blocks are not allowed.
			{
				Config: `
				resource "grafana_contact_point" "receiver_types" {
				  name = "Receiver Types since v12.0"
				
				  webhook {
					url                 = "http://my-url"
					headers = {
					  Content-Type  = "test-content-type"
					  X-Test-Header = "test-header-value"
					}
					payload {
					  template = "{{ .Receiver }}: {{ .Vars.var1 }}"
					  vars = {
						var1 = "variable value"
					  }
					}
					payload {
					  template = "{{ .Receiver }}: {{ .Vars.var1 }} 2"
					  vars = {
						var1 = "variable value2"
					  }
					}
				  }
				}
				`,
				ExpectError: regexp.MustCompile(`Too many payload blocks`),
			},
			// Template field required.
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_contact_point/_acc_receiver_types_12_0.tf", map[string]string{
					`template = "{{ .Receiver }}: {{ .Vars.var1 }}"`: ``,
				}),
				ExpectError: regexp.MustCompile(`Missing required argument`),
			},
			// Empty payload and header are omitted.
			{
				Config: `
				resource "grafana_contact_point" "receiver_types" {
				  name = "Receiver Types since v12.0"
				
				  webhook {
					url = "http://my-url"
				  }
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					checkAlertingContactPointExistsWithLength("grafana_contact_point.receiver_types", &points, 1),
					// webhook
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.url", "http://my-url"),

					// Sanity check to ensure empty TLS config is omitted instead of being set to an empty map.
					func(s *terraform.State) error {
						if val, ok := points[0].Settings.(map[string]interface{})["headers"]; ok {
							return fmt.Errorf("headers was still present in the settings when it should have been omitted. value: %#v", val)
						}
						if val, ok := points[0].Settings.(map[string]interface{})["payload"]; ok {
							return fmt.Errorf("payload was still present in the settings when it should have been omitted. value: %#v", val)
						}

						return nil
					},
				),
			},
			// Test creation.
			{
				Config: testutils.TestAccExample(t, "resources/grafana_contact_point/_acc_receiver_types_12_0.tf"),
				Check: resource.ComposeTestCheckFunc(
					checkAlertingContactPointExistsWithLength("grafana_contact_point.receiver_types", &points, 1),
					// webhook
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.url", "http://my-url"),

					// Since we are using schema.TypeSet for nested types, we need ".0" for this notation to correctly reference the nested element.
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.headers.%", "2"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.headers.Content-Type", "test-content-type"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.headers.X-Test-Header", "test-header-value"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.payload.0.template", "{{ .Receiver }}: {{ .Vars.var1 }}"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.payload.0.vars.%", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.payload.0.vars.var1", "variable value"),

					// Sanity check to ensure empty TLS config is omitted instead of being set to an empty map.
					func(s *terraform.State) error {
						if val, ok := points[0].Settings.(map[string]interface{})["tlsConfig"]; ok {
							return fmt.Errorf("tlsConfig was still present in the settings when it should have been omitted. value: %#v", val)
						}

						return nil
					},
				),
			},
			// Update non-sensitive data.
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_contact_point/_acc_receiver_types_12_0.tf", map[string]string{
					"\"test-header-value\"":                 "\"updated-test-header-value\"",
					"\"variable value\"":                    "\"updated-variable value\"",
					"\"{{ .Receiver }}: {{ .Vars.var1 }}\"": "\"Updated {{ .Receiver }}: {{ .Vars.var1 }}\"",
				}),

				Check: resource.ComposeTestCheckFunc(
					checkAlertingContactPointExistsWithLength("grafana_contact_point.receiver_types", &points, 1),
					// webhook
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.url", "http://my-url"),

					// Since we are using schema.TypeSet for nested types, we need ".0" for this notation to correctly reference the nested element.
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.headers.%", "2"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.headers.Content-Type", "test-content-type"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.headers.X-Test-Header", "updated-test-header-value"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.payload.0.template", "Updated {{ .Receiver }}: {{ .Vars.var1 }}"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.payload.0.vars.%", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.payload.0.vars.var1", "updated-variable value"),

					// Sanity check to ensure empty TLS config is omitted instead of being set to an empty map.
					func(s *terraform.State) error {
						if val, ok := points[0].Settings.(map[string]interface{})["tlsConfig"]; ok {
							return fmt.Errorf("tlsConfig was still present in the settings when it should have been omitted. value: %#v", val)
						}

						return nil
					},
				),
			},
		},
	})
}

func TestAccContactPoint_notifiers12_1(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=12.1.0")

	var points models.ContactPoints

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		// Implicitly tests deletion.
		CheckDestroy: alertingContactPointCheckExists.destroyed(&points, nil),
		Steps: []resource.TestStep{
			// Test creation.
			{
				Config: testutils.TestAccExample(t, "resources/grafana_contact_point/_acc_receiver_types_12_1.tf"),
				Check: resource.ComposeTestCheckFunc(
					checkAlertingContactPointExistsWithLength("grafana_contact_point.receiver_types", &points, 1),
					// webhook
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.url", "http://my-url"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.http_method", "POST"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.basic_auth_user", "user"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.basic_auth_password", "password"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.max_alerts", "100"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.message", "Custom message"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.title", "Custom title"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.tls_config.insecure_skip_verify", "true"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.tls_config.ca_certificate", "ca.crt"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.tls_config.client_certificate", "client.crt"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.tls_config.client_key", "client.key"),

					// Since we are using schema.TypeSet for nested types, we need ".0" for this notation to correctly reference the nested element.
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.http_config.0.oauth2.0.client_id", "client_id"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.http_config.0.oauth2.0.client_secret", "client_secret"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.http_config.0.oauth2.0.token_url", "http://oauth2-token-url"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.http_config.0.oauth2.0.scopes.#", "2"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.http_config.0.oauth2.0.scopes.0", "scope1"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.http_config.0.oauth2.0.scopes.1", "scope2"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.http_config.0.oauth2.0.endpoint_params.param1", "value1"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.http_config.0.oauth2.0.endpoint_params.param2", "value2"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.http_config.0.oauth2.0.proxy_config.0.proxy_url", "http://proxy-url"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.http_config.0.oauth2.0.proxy_config.0.proxy_from_environment", "false"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.http_config.0.oauth2.0.proxy_config.0.no_proxy", "localhost"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.http_config.0.oauth2.0.proxy_config.0.proxy_connect_header.X-Proxy-Header", "proxy-value"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.http_config.0.oauth2.0.tls_config.0.insecure_skip_verify", "true"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.http_config.0.oauth2.0.tls_config.0.ca_certificate", "-----BEGIN CERTIFICATE-----\nMIGrMF+gAwIBAgIBATAFBgMrZXAwADAeFw0yNDExMTYxMDI4MzNaFw0yNTExMTYx\nMDI4MzNaMAAwKjAFBgMrZXADIQCf30GvRnHbs9gukA3DLXDK6W5JVgYw6mERU/60\n2M8+rjAFBgMrZXADQQCGmeaRp/AcjeqmJrF5Yh4d7aqsMSqVZvfGNDc0ppXyUgS3\nWMQ1+3T+/pkhU612HR0vFd3vyFhmB4yqFoNV8RML\n-----END CERTIFICATE-----\n"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.http_config.0.oauth2.0.tls_config.0.client_certificate", "-----BEGIN CERTIFICATE-----\nMIIBhTCCASugAwIBAgIQIRi6zePL6mKjOipn+dNuaTAKBggqhkjOPQQDAjASMRAw\nDgYDVQQKEwdBY21lIENvMB4XDTE3MTAyMDE5NDMwNloXDTE4MTAyMDE5NDMwNlow\nEjEQMA4GA1UEChMHQWNtZSBDbzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABD0d\n7VNhbWvZLWPuj/RtHFjvtJBEwOkhbN/BnnE8rnZR8+sbwnc/KhCk3FhnpHZnQz7B\n5aETbbIgmuvewdjvSBSjYzBhMA4GA1UdDwEB/wQEAwICpDATBgNVHSUEDDAKBggr\nBgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MCkGA1UdEQQiMCCCDmxvY2FsaG9zdDo1\nNDUzgg4xMjcuMC4wLjE6NTQ1MzAKBggqhkjOPQQDAgNIADBFAiEA2zpJEPQyz6/l\nWf86aX6PepsntZv2GYlA5UpabfT2EZICICpJ5h/iI+i341gBmLiAFQOyTDT+/wQc\n6MF9+Yw1Yy0t\n-----END CERTIFICATE-----\n"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.http_config.0.oauth2.0.tls_config.0.client_key", "-----BEGIN EC PRIVATE KEY-----\nMHcCAQEEIIrYSSNQFaA2Hwf1duRSxKtLYX5CB04fSeQ6tF1aY/PuoAoGCCqGSM49\nAwEHoUQDQgAEPR3tU2Fta9ktY+6P9G0cWO+0kETA6SFs38GecTyudlHz6xvCdz8q\nEKTcWGekdmdDPsHloRNtsiCa697B2O9IFA==\n-----END EC PRIVATE KEY-----\n"),
				),
			},
			// Update sensitive data.
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_contact_point/_acc_receiver_types_12_1.tf", map[string]string{
					"\"client_secret\"": "\"updated_client_secret\"",
				}),

				Check: resource.ComposeTestCheckFunc(
					checkAlertingContactPointExistsWithLength("grafana_contact_point.receiver_types", &points, 1),

					// Updated
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.http_config.0.oauth2.0.client_secret", "updated_client_secret"),

					// Unchanged
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.url", "http://my-url"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.http_method", "POST"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.basic_auth_user", "user"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.basic_auth_password", "password"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.max_alerts", "100"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.message", "Custom message"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.title", "Custom title"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.tls_config.insecure_skip_verify", "true"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.tls_config.ca_certificate", "ca.crt"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.tls_config.client_certificate", "client.crt"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.tls_config.client_key", "client.key"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.http_config.0.oauth2.0.client_id", "client_id"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.http_config.0.oauth2.0.token_url", "http://oauth2-token-url"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.http_config.0.oauth2.0.scopes.#", "2"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.http_config.0.oauth2.0.scopes.0", "scope1"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.http_config.0.oauth2.0.scopes.1", "scope2"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.http_config.0.oauth2.0.endpoint_params.param1", "value1"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.http_config.0.oauth2.0.endpoint_params.param2", "value2"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.http_config.0.oauth2.0.proxy_config.0.proxy_url", "http://proxy-url"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.http_config.0.oauth2.0.proxy_config.0.proxy_from_environment", "false"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.http_config.0.oauth2.0.proxy_config.0.no_proxy", "localhost"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.http_config.0.oauth2.0.proxy_config.0.proxy_connect_header.X-Proxy-Header", "proxy-value"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.http_config.0.oauth2.0.tls_config.0.insecure_skip_verify", "true"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.http_config.0.oauth2.0.tls_config.0.ca_certificate", "-----BEGIN CERTIFICATE-----\nMIGrMF+gAwIBAgIBATAFBgMrZXAwADAeFw0yNDExMTYxMDI4MzNaFw0yNTExMTYx\nMDI4MzNaMAAwKjAFBgMrZXADIQCf30GvRnHbs9gukA3DLXDK6W5JVgYw6mERU/60\n2M8+rjAFBgMrZXADQQCGmeaRp/AcjeqmJrF5Yh4d7aqsMSqVZvfGNDc0ppXyUgS3\nWMQ1+3T+/pkhU612HR0vFd3vyFhmB4yqFoNV8RML\n-----END CERTIFICATE-----\n"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.http_config.0.oauth2.0.tls_config.0.client_certificate", "-----BEGIN CERTIFICATE-----\nMIIBhTCCASugAwIBAgIQIRi6zePL6mKjOipn+dNuaTAKBggqhkjOPQQDAjASMRAw\nDgYDVQQKEwdBY21lIENvMB4XDTE3MTAyMDE5NDMwNloXDTE4MTAyMDE5NDMwNlow\nEjEQMA4GA1UEChMHQWNtZSBDbzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABD0d\n7VNhbWvZLWPuj/RtHFjvtJBEwOkhbN/BnnE8rnZR8+sbwnc/KhCk3FhnpHZnQz7B\n5aETbbIgmuvewdjvSBSjYzBhMA4GA1UdDwEB/wQEAwICpDATBgNVHSUEDDAKBggr\nBgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MCkGA1UdEQQiMCCCDmxvY2FsaG9zdDo1\nNDUzgg4xMjcuMC4wLjE6NTQ1MzAKBggqhkjOPQQDAgNIADBFAiEA2zpJEPQyz6/l\nWf86aX6PepsntZv2GYlA5UpabfT2EZICICpJ5h/iI+i341gBmLiAFQOyTDT+/wQc\n6MF9+Yw1Yy0t\n-----END CERTIFICATE-----\n"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.http_config.0.oauth2.0.tls_config.0.client_key", "-----BEGIN EC PRIVATE KEY-----\nMHcCAQEEIIrYSSNQFaA2Hwf1duRSxKtLYX5CB04fSeQ6tF1aY/PuoAoGCCqGSM49\nAwEHoUQDQgAEPR3tU2Fta9ktY+6P9G0cWO+0kETA6SFs38GecTyudlHz6xvCdz8q\nEKTcWGekdmdDPsHloRNtsiCa697B2O9IFA==\n-----END EC PRIVATE KEY-----\n"),
				),
			},
			// Update non-sensitive data.
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_contact_point/_acc_receiver_types_12_1.tf", map[string]string{
					"\"client_secret\"": "\"updated_client_secret\"",
					"http://proxy-url":  "http://updated-proxy-url",
				}),

				Check: resource.ComposeTestCheckFunc(
					checkAlertingContactPointExistsWithLength("grafana_contact_point.receiver_types", &points, 1),

					// Updated
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.http_config.0.oauth2.0.client_secret", "updated_client_secret"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.http_config.0.oauth2.0.proxy_config.0.proxy_url", "http://updated-proxy-url"),

					// Unchanged
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.url", "http://my-url"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.http_method", "POST"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.basic_auth_user", "user"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.basic_auth_password", "password"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.max_alerts", "100"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.message", "Custom message"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.title", "Custom title"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.tls_config.insecure_skip_verify", "true"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.tls_config.ca_certificate", "ca.crt"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.tls_config.client_certificate", "client.crt"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.tls_config.client_key", "client.key"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.http_config.0.oauth2.0.client_id", "client_id"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.http_config.0.oauth2.0.token_url", "http://oauth2-token-url"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.http_config.0.oauth2.0.scopes.#", "2"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.http_config.0.oauth2.0.scopes.0", "scope1"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.http_config.0.oauth2.0.scopes.1", "scope2"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.http_config.0.oauth2.0.endpoint_params.param1", "value1"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.http_config.0.oauth2.0.endpoint_params.param2", "value2"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.http_config.0.oauth2.0.proxy_config.0.proxy_from_environment", "false"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.http_config.0.oauth2.0.proxy_config.0.no_proxy", "localhost"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.http_config.0.oauth2.0.proxy_config.0.proxy_connect_header.X-Proxy-Header", "proxy-value"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.http_config.0.oauth2.0.tls_config.0.insecure_skip_verify", "true"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.http_config.0.oauth2.0.tls_config.0.ca_certificate", "-----BEGIN CERTIFICATE-----\nMIGrMF+gAwIBAgIBATAFBgMrZXAwADAeFw0yNDExMTYxMDI4MzNaFw0yNTExMTYx\nMDI4MzNaMAAwKjAFBgMrZXADIQCf30GvRnHbs9gukA3DLXDK6W5JVgYw6mERU/60\n2M8+rjAFBgMrZXADQQCGmeaRp/AcjeqmJrF5Yh4d7aqsMSqVZvfGNDc0ppXyUgS3\nWMQ1+3T+/pkhU612HR0vFd3vyFhmB4yqFoNV8RML\n-----END CERTIFICATE-----\n"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.http_config.0.oauth2.0.tls_config.0.client_certificate", "-----BEGIN CERTIFICATE-----\nMIIBhTCCASugAwIBAgIQIRi6zePL6mKjOipn+dNuaTAKBggqhkjOPQQDAjASMRAw\nDgYDVQQKEwdBY21lIENvMB4XDTE3MTAyMDE5NDMwNloXDTE4MTAyMDE5NDMwNlow\nEjEQMA4GA1UEChMHQWNtZSBDbzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABD0d\n7VNhbWvZLWPuj/RtHFjvtJBEwOkhbN/BnnE8rnZR8+sbwnc/KhCk3FhnpHZnQz7B\n5aETbbIgmuvewdjvSBSjYzBhMA4GA1UdDwEB/wQEAwICpDATBgNVHSUEDDAKBggr\nBgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MCkGA1UdEQQiMCCCDmxvY2FsaG9zdDo1\nNDUzgg4xMjcuMC4wLjE6NTQ1MzAKBggqhkjOPQQDAgNIADBFAiEA2zpJEPQyz6/l\nWf86aX6PepsntZv2GYlA5UpabfT2EZICICpJ5h/iI+i341gBmLiAFQOyTDT+/wQc\n6MF9+Yw1Yy0t\n-----END CERTIFICATE-----\n"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "webhook.0.http_config.0.oauth2.0.tls_config.0.client_key", "-----BEGIN EC PRIVATE KEY-----\nMHcCAQEEIIrYSSNQFaA2Hwf1duRSxKtLYX5CB04fSeQ6tF1aY/PuoAoGCCqGSM49\nAwEHoUQDQgAEPR3tU2Fta9ktY+6P9G0cWO+0kETA6SFs38GecTyudlHz6xvCdz8q\nEKTcWGekdmdDPsHloRNtsiCa697B2O9IFA==\n-----END EC PRIVATE KEY-----\n"),
				),
			},
		},
	})
}

func TestAccContactPoint_TypeSet_MaxItems(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=12.1.0")

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			// Multiple http_config blocks are not allowed.
			{
				Config: `
				resource "grafana_contact_point" "typset_max_items" {
					name = "typset_max_items"
					webhook {
						url                 = "http://my-url"
						http_config {
							oauth2 {
								client_id     = "client_id"
								client_secret = "client_secret"
								token_url = "http://oauth2-token-url"
							}
						}
						http_config {
							oauth2 {
								client_id     = "client_id2"
								client_secret = "client_secret2"
								token_url = "http://oauth2-token-url2"
							}
						}
					}
				}
				`,
				ExpectError: regexp.MustCompile(`Too many http_config blocks`),
			},
			// Multiple oauth2 blocks are not allowed.
			{
				Config: `
				resource "grafana_contact_point" "typset_max_items" {
					name = "typset_max_items"
					webhook {
						url                 = "http://my-url"
						http_config {
							oauth2 {
								client_id     = "client_id"
								client_secret = "client_secret"
								token_url = "http://oauth2-token-url"
							}
							oauth2 {
								client_id     = "client_id2"
								client_secret = "client_secret2"
								token_url = "http://oauth2-token-url2"
							}
						}
					}
				}
				`,
				ExpectError: regexp.MustCompile(`Too many oauth2 blocks`),
			},
			// Multiple proxy_config blocks are not allowed.
			{
				Config: `
				resource "grafana_contact_point" "typset_max_items" {
					name = "typset_max_items"
					webhook {
						url                 = "http://my-url"
						http_config {
							oauth2 {
								client_id     = "client_id"
								client_secret = "client_secret"
								token_url = "http://oauth2-token-url"
								proxy_config {
									proxy_url = "http://proxy-url"
								}
								proxy_config {
									proxy_url = "http://proxy-url"
								}
							}
						}
					}
				}
				`,
				ExpectError: regexp.MustCompile(`Too many proxy_config blocks`),
			},
			// Multiple tls_config blocks are not allowed.
			{
				Config: `
				resource "grafana_contact_point" "typset_max_items" {
					name = "typset_max_items"
					webhook {
						url                 = "http://my-url"
						http_config {
							oauth2 {
								client_id     = "client_id"
								client_secret = "client_secret"
								token_url = "http://oauth2-token-url"
								tls_config {
									insecure_skip_verify = true
								}
								tls_config {
									insecure_skip_verify = true
								}
							}
						}
					}
				}
				`,
				ExpectError: regexp.MustCompile(`Too many tls_config blocks`),
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

// Tests minimal definitions for all contact point types. This is useful to ensure that:
// 1) The Terraform schema is correct about what is required vs optional.
// 2) The provider correctly omits optional fields that are not set, so any API-side defaults are applied and don't contain redundant/blank values.
func TestAccContactPoint_minimalDefinitions(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=12.1.0")

	var points models.ContactPoints

	getContactPoint := func(points *models.ContactPoints, uid string) *models.EmbeddedContactPoint {
		for _, pt := range *points {
			if pt.UID == uid {
				return pt
			}
		}
		return nil
	}

	getResourceUID := func(s *terraform.State, prefix string) string {
		return s.RootModule().Resources["grafana_contact_point.minimal_receivers"].Primary.Attributes[prefix+".uid"]
	}

	checkOtherAttrsOmittedInResponse := func(points *models.ContactPoints, key string, existingAttrs ...string) resource.TestCheckFunc {
		return func(s *terraform.State) error {
			uid := getResourceUID(s, key)
			pt := getContactPoint(points, uid)
			if pt == nil {
				return fmt.Errorf("contact point %s with uid %s not found in the API response", key, uid)
			}
			expectedAttrs := make(map[string]struct{})
			for _, attr := range existingAttrs {
				expectedAttrs[attr] = struct{}{}
			}
			for attr := range pt.Settings.(map[string]any) {
				if val, ok := expectedAttrs[attr]; !ok {
					return fmt.Errorf("contact point %s attribute %s should not exist in the contact point settings, but was found: %v", key, attr, val)
				}
				delete(expectedAttrs, attr)
			}
			if len(expectedAttrs) > 0 {
				keys := maps.Keys(expectedAttrs)
				return fmt.Errorf("contact point %s attributes %q should exist in the contact point settings, but were not found", key, strings.Join(slices.Sorted(keys), ", "))
			}
			return nil
		}
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		// Implicitly tests deletion.
		CheckDestroy: alertingContactPointCheckExists.destroyed(&points, nil),
		Steps: []resource.TestStep{
			// Test creation.
			{
				Config: testutils.TestAccExample(t, "resources/grafana_contact_point/_acc_receiver_types_minimal.tf"),
				Check: resource.ComposeTestCheckFunc(
					checkAlertingContactPointExistsWithLength("grafana_contact_point.minimal_receivers", &points, 22),
					// alertmanager
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "alertmanager.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "alertmanager.0.url", "http://my-am"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "alertmanager.0.basic_auth_user", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "alertmanager.0.basic_auth_password", ""),
					checkOtherAttrsOmittedInResponse(&points, "alertmanager.0",
						"url",
					),
					// dingding
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "dingding.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "dingding.0.url", "http://dingding-url"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "dingding.0.message_type", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "dingding.0.message", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "dingding.0.title", ""),
					checkOtherAttrsOmittedInResponse(&points, "dingding.0",
						"url",
					),
					// discord
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "discord.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "discord.0.url", "http://discord-url"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "discord.0.title", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "discord.0.message", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "discord.0.avatar_url", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "discord.0.use_discord_username", "false"),
					checkOtherAttrsOmittedInResponse(&points, "discord.0",
						"url",
						"use_discord_username", // TODO: This would be better omitted.
					),
					// email
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "email.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "email.0.addresses.0", "one@company.org"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "email.0.addresses.1", "two@company.org"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "email.0.message", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "email.0.subject", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "email.0.single_email", "false"),
					checkOtherAttrsOmittedInResponse(&points, "email.0",
						"addresses",
						"singleEmail", // TODO: This would be better omitted.
					),
					// googlechat
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "googlechat.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "googlechat.0.url", "http://googlechat-url"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "googlechat.0.title", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "googlechat.0.message", ""),
					checkOtherAttrsOmittedInResponse(&points, "googlechat.0",
						"url",
					),
					// kafka
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "kafka.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "kafka.0.rest_proxy_url", "http://kafka-rest-proxy-url"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "kafka.0.topic", "mytopic"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "kafka.0.description", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "kafka.0.details", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "kafka.0.username", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "kafka.0.password", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "kafka.0.api_version", "v2"), // Default value.
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "kafka.0.cluster_id", ""),
					checkOtherAttrsOmittedInResponse(&points, "kafka.0",
						"kafkaRestProxy",
						"kafkaTopic",
						"apiVersion",
					),
					// line
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "line.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "line.0.token", "token"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "line.0.title", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "line.0.description", ""),
					checkOtherAttrsOmittedInResponse(&points, "line.0",
						"token",
					),
					// oncall
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "oncall.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "oncall.0.url", "http://oncall-url"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "oncall.0.http_method", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "oncall.0.basic_auth_user", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "oncall.0.basic_auth_password", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "oncall.0.max_alerts", "0"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "oncall.0.message", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "oncall.0.title", ""),
					checkOtherAttrsOmittedInResponse(&points, "oncall.0",
						"url",
						"maxAlerts", // TODO: This would be better omitted.
					),
					// opsgenie
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "opsgenie.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "opsgenie.0.api_key", "token"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "opsgenie.0.url", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "opsgenie.0.message", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "opsgenie.0.description", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "opsgenie.0.auto_close", "false"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "opsgenie.0.override_priority", "false"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "opsgenie.0.send_tags_as", ""),
					checkOtherAttrsOmittedInResponse(&points, "opsgenie.0",
						"apiKey",
						"autoClose",        // TODO: This would be better omitted.
						"overridePriority", // TODO: This would be better omitted.
						"responders",       // TODO: This would be better omitted.
					),
					// pagerduty
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "pagerduty.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "pagerduty.0.integration_key", "token"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "pagerduty.0.severity", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "pagerduty.0.class", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "pagerduty.0.component", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "pagerduty.0.group", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "pagerduty.0.summary", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "pagerduty.0.source", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "pagerduty.0.client", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "pagerduty.0.client_url", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "pagerduty.0.details.%", "0"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "pagerduty.0.url", ""),
					checkOtherAttrsOmittedInResponse(&points, "pagerduty.0",
						"integrationKey",
						"details", // TODO: This would be better omitted.
					),
					// pushover
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "pushover.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "pushover.0.user_key", "userkey"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "pushover.0.api_token", "token"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "pushover.0.priority", "0"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "pushover.0.ok_priority", "0"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "pushover.0.retry", "0"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "pushover.0.expire", "0"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "pushover.0.device", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "pushover.0.sound", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "pushover.0.ok_sound", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "pushover.0.title", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "pushover.0.message", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "pushover.0.upload_image", "false"),
					checkOtherAttrsOmittedInResponse(&points, "pushover.0",
						"userKey",
						"apiToken",
						"priority",    // TODO: This would be better omitted.
						"okPriority",  // TODO: This would be better omitted.
						"retry",       // TODO: This would be better omitted.
						"expire",      // TODO: This would be better omitted.
						"uploadImage", // TODO: This would be better omitted.
					),
					// sensugo
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "sensugo.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "sensugo.0.url", "http://sensugo-url"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "sensugo.0.api_key", "key"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "sensugo.0.entity", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "sensugo.0.check", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "sensugo.0.namespace", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "sensugo.0.handler", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "sensugo.0.message", ""),
					checkOtherAttrsOmittedInResponse(&points, "sensugo.0",
						"url",
						"apikey",
					),
					// slack url
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "slack.#", "2"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "slack.0.url", "http://custom-slack-url"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "slack.0.endpoint_url", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "slack.0.recipient", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "slack.0.text", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "slack.0.title", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "slack.0.username", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "slack.0.icon_emoji", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "slack.0.icon_url", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "slack.0.mention_channel", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "slack.0.mention_users", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "slack.0.mention_groups", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "slack.0.color", ""),
					checkOtherAttrsOmittedInResponse(&points, "slack.0",
						"url",
					),
					// slack token
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "slack.1.token", "xoxb-token"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "slack.1.recipient", "#channel"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "slack.1.endpoint_url", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "slack.1.text", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "slack.1.title", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "slack.1.username", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "slack.1.icon_emoji", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "slack.1.icon_url", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "slack.1.mention_channel", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "slack.1.mention_users", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "slack.1.mention_groups", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "slack.1.color", ""),
					checkOtherAttrsOmittedInResponse(&points, "slack.1",
						"token",
						"recipient",
					),
					// teams
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "teams.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "teams.0.url", "http://teams-webhook"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "teams.0.message", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "teams.0.title", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "teams.0.section_title", ""),
					checkOtherAttrsOmittedInResponse(&points, "teams.0",
						"url",
					),
					// telegram
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "telegram.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "telegram.0.token", "token"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "telegram.0.chat_id", "chat-id"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "telegram.0.message_thread_id", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "telegram.0.message", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "telegram.0.parse_mode", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "telegram.0.disable_web_page_preview", "false"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "telegram.0.protect_content", "false"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "telegram.0.disable_notifications", "false"),
					checkOtherAttrsOmittedInResponse(&points, "telegram.0",
						"bottoken",
						"chatid",
						"disable_web_page_preview", // TODO: This would be better omitted.
						"protect_content",          // TODO: This would be better omitted.
						"disable_notifications",    // TODO: This would be better omitted.
					),
					// threema
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "threema.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "threema.0.gateway_id", "*gateway"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "threema.0.recipient_id", "*target1"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "threema.0.api_secret", "secret"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "threema.0.title", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "threema.0.description", ""),
					checkOtherAttrsOmittedInResponse(&points, "threema.0",
						"gateway_id",
						"recipient_id",
						"api_secret",
					),
					// victorops
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "victorops.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "victorops.0.url", "http://victor-ops-url"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "victorops.0.message_type", ""),
					checkOtherAttrsOmittedInResponse(&points, "victorops.0",
						"url",
					),
					// webex
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "webex.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "webex.0.token", "token"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "webex.0.room_id", "room_id"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "webex.0.api_url", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "webex.0.message", ""),
					checkOtherAttrsOmittedInResponse(&points, "webex.0",
						"bot_token",
						"room_id",
					),
					// webhook
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "webhook.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "webhook.0.url", "http://webhook-url"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "webhook.0.http_method", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "webhook.0.basic_auth_user", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "webhook.0.basic_auth_password", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "webhook.0.max_alerts", "0"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "webhook.0.message", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "webhook.0.title", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "webhook.0.headers.%", "0"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "webhook.0.payload.#", "0"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "webhook.0.http_config.#", "0"),
					checkOtherAttrsOmittedInResponse(&points, "webhook.0",
						"url",
						"maxAlerts", // TODO: This would be better omitted.
					),
					// wecom url
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "wecom.#", "2"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "wecom.0.url", "http://wecom-url"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "wecom.0.message", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "wecom.0.title", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "wecom.0.secret", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "wecom.0.corp_id", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "wecom.0.agent_id", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "wecom.0.msg_type", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "wecom.0.to_user", ""),
					checkOtherAttrsOmittedInResponse(&points, "wecom.0",
						"url",
					),
					// wecom secret
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "wecom.1.secret", "secret"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "wecom.1.corp_id", "corp_id"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "wecom.1.agent_id", "agent_id"),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "wecom.1.url", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "wecom.1.message", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "wecom.1.title", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "wecom.1.msg_type", ""),
					resource.TestCheckResourceAttr("grafana_contact_point.minimal_receivers", "wecom.1.to_user", ""),
					checkOtherAttrsOmittedInResponse(&points, "wecom.1",
						"secret",
						"corp_id",
						"agent_id",
					),
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
