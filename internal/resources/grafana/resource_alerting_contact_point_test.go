package grafana_test

import (
	"fmt"
	"regexp"
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/grafana/terraform-provider-grafana/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccContactPoint_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)
	testutils.CheckOSSTestsSemver(t, ">=9.0.0")

	var points []gapi.ContactPoint

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		// Implicitly tests deletion.
		CheckDestroy: testContactPointCheckDestroy(points),
		Steps: []resource.TestStep{
			// Test creation.
			{
				Config: testutils.TestAccExample(t, "resources/grafana_contact_point/resource.tf"),
				Check: resource.ComposeTestCheckFunc(
					testContactPointCheckExists("grafana_contact_point.my_contact_point", &points, 1),
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
					testContactPointCheckExists("grafana_contact_point.my_contact_point", &points, 1),
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
					testContactPointCheckExists("grafana_contact_point.my_contact_point", &points, 1),
					resource.TestCheckResourceAttr("grafana_contact_point.my_contact_point", "name", "A Different Contact Point"),
					resource.TestCheckResourceAttr("grafana_contact_point.my_contact_point", "email.#", "1"),
					testContactPointCheckAllDestroy("My Contact Point"),
				),
			},
		},
	})
}

func TestAccContactPoint_compound(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)
	testutils.CheckOSSTestsSemver(t, ">=9.0.0")

	var points []gapi.ContactPoint

	// TODO: Make parallelizable
	// Error: wrong number of contact points on the server, expected 2 but got []{..., ..., ...} (len=3)
	resource.Test(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		// Implicitly tests deletion.
		CheckDestroy: testContactPointCheckDestroy(points),
		Steps: []resource.TestStep{
			// Test creation.
			{
				Config: testutils.TestAccExample(t, "resources/grafana_contact_point/_acc_compound_receiver.tf"),
				Check: resource.ComposeTestCheckFunc(
					testContactPointCheckExists("grafana_contact_point.compound_contact_point", &points, 2),
					resource.TestCheckResourceAttr("grafana_contact_point.compound_contact_point", "name", "Compound Contact Point"),
					resource.TestCheckResourceAttr("grafana_contact_point.compound_contact_point", "email.#", "2"),
				),
			},
			// Test update.
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_contact_point/_acc_compound_receiver.tf", map[string]string{
					"one": "asdf",
				}),
				Check: resource.ComposeTestCheckFunc(
					testContactPointCheckExists("grafana_contact_point.compound_contact_point", &points, 2),
					resource.TestCheckResourceAttr("grafana_contact_point.compound_contact_point", "email.#", "2"),
					resource.TestCheckResourceAttr("grafana_contact_point.compound_contact_point", "email.0.addresses.0", "asdf@company.org"),
				),
			},
			// Test addition of a contact point to an existing compound one.
			{
				Config: testutils.TestAccExample(t, "resources/grafana_contact_point/_acc_compound_receiver_added.tf"),
				Check: resource.ComposeTestCheckFunc(
					testContactPointCheckExists("grafana_contact_point.compound_contact_point", &points, 3),
					resource.TestCheckResourceAttr("grafana_contact_point.compound_contact_point", "email.#", "3"),
					resource.TestCheckResourceAttr("grafana_contact_point.compound_contact_point", "email.0.addresses.0", "one@company.org"),
					resource.TestCheckResourceAttr("grafana_contact_point.compound_contact_point", "email.1.addresses.0", "three@company.org"),
					resource.TestCheckResourceAttr("grafana_contact_point.compound_contact_point", "email.2.addresses.0", "five@company.org"),
				),
			},
			// Test removal of a point from a compound one does not leak.
			{
				Config: testutils.TestAccExample(t, "resources/grafana_contact_point/_acc_compound_receiver_subtracted.tf"),
				Check: resource.ComposeTestCheckFunc(
					testContactPointCheckExists("grafana_contact_point.compound_contact_point", &points, 1),
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
					testContactPointCheckExists("grafana_contact_point.compound_contact_point", &points, 2),
					resource.TestCheckResourceAttr("grafana_contact_point.compound_contact_point", "name", "A Different Contact Point"),
					resource.TestCheckResourceAttr("grafana_contact_point.compound_contact_point", "email.#", "2"),
					testContactPointCheckAllDestroy("Compound Contact Point"),
				),
			},
		},
	})
}

func TestAccContactPoint_notifiers(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)
	testutils.CheckOSSTestsSemver(t, ">=9.1.0")

	var points []gapi.ContactPoint

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		// Implicitly tests deletion.
		CheckDestroy: testContactPointCheckDestroy(points),
		Steps: []resource.TestStep{
			// Test creation.
			{
				Config: testutils.TestAccExample(t, "resources/grafana_contact_point/_acc_receiver_types.tf"),
				Check: resource.ComposeTestCheckFunc(
					testContactPointCheckExists("grafana_contact_point.receiver_types", &points, 17),
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
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "googlechat.0.message", "message"),
					// kafka
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "kafka.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "kafka.0.rest_proxy_url", "http://kafka-rest-proxy-url"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "kafka.0.topic", "mytopic"),
					// oncall
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "oncall.#", "1"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "oncall.0.url", "http://oncall-url"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "oncall.0.message", "message"),
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "oncall.0.title", "title"),
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
					resource.TestCheckResourceAttr("grafana_contact_point.receiver_types", "telegram.0.message", "message"),
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
				),
			},
			// Test blank fields in settings should be omitted.
			{
				Config: testutils.TestAccExample(t, "resources/grafana_contact_point/_acc_default_settings.tf"),
				Check: resource.ComposeTestCheckFunc(
					testContactPointCheckExists("grafana_contact_point.default_settings", &points, 1),
					resource.TestCheckResourceAttr("grafana_contact_point.default_settings", "slack.#", "1"),
					resource.TestCheckNoResourceAttr("grafana_contact_point.default_settings", "slack.endpoint_url"),
					func(s *terraform.State) error {
						rname := "grafana_contact_point.default_settings"
						rs, ok := s.RootModule().Resources[rname]
						if !ok {
							return fmt.Errorf("resource not found: %s, resources: %#v", rname, s.RootModule().Resources)
						}
						uid := rs.Primary.ID

						client := testutils.Provider.Meta().(*common.Client).GrafanaAPI
						pt, err := client.ContactPoint(uid)
						if err != nil {
							return fmt.Errorf("error getting resource: %w", err)
						}

						if val, ok := pt.Settings["endpointUrl"]; ok {
							return fmt.Errorf("endpointUrl was still present in the settings when it should have been omitted. value: %#v", val)
						}

						return nil
					},
				),
			},
		},
	})
}

func TestAccContactPoint_empty(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)
	testutils.CheckOSSTestsSemver(t, ">=9.1.0")

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		Steps: []resource.TestStep{
			// Test creation.
			{
				Config:      testAccEmptyContactPoint,
				ExpectError: regexp.MustCompile(`Missing required argument`),
			},
		},
	})
}

func testContactPointCheckExists(rname string, pts *[]gapi.ContactPoint, expCount int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resource, ok := s.RootModule().Resources[rname]
		if !ok {
			return fmt.Errorf("resource not found: %s, resources: %#v", rname, s.RootModule().Resources)
		}

		name, ok := resource.Primary.Attributes["name"]
		if !ok {
			return fmt.Errorf("resource name not set")
		}

		client := testutils.Provider.Meta().(*common.Client).GrafanaAPI
		points, err := client.ContactPointsByName(name)
		if err != nil {
			return fmt.Errorf("error getting resource: %w", err)
		}

		// Work around query parameters not being supported in some older patch versions.
		filtered := make([]gapi.ContactPoint, 0, len(points))
		for i := range points {
			if points[i].Name == name {
				filtered = append(filtered, points[i])
			}
		}

		if len(filtered) != expCount {
			return fmt.Errorf("wrong number of contact points on the server, expected %d but got %#v", expCount, filtered)
		}

		*pts = points
		return nil
	}
}

func testContactPointCheckDestroy(points []gapi.ContactPoint) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testutils.Provider.Meta().(*common.Client).GrafanaAPI
		for _, p := range points {
			_, err := client.ContactPoint(p.UID)
			if err == nil {
				return fmt.Errorf("contact point still exists on the server")
			}
		}
		points = []gapi.ContactPoint{}
		return nil
	}
}

func testContactPointCheckAllDestroy(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testutils.Provider.Meta().(*common.Client).GrafanaAPI
		points, err := client.ContactPointsByName(name)
		if err != nil {
			return fmt.Errorf("error getting resource: %w", err)
		}

		// Work around query parameters not being supported in some older patch versions.
		filtered := make([]gapi.ContactPoint, 0, len(points))
		for i := range points {
			if points[i].Name == name {
				filtered = append(filtered, points[i])
			}
		}

		if len(filtered) > 0 {
			return fmt.Errorf("contact points still exist on the server: %#v", filtered)
		}
		return nil
	}
}

const testAccEmptyContactPoint = `
resource "grafana_contact_point" "dev_null" {
	name = "empty-test"
}
`
