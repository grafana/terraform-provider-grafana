package appplatform_test

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/grafana/authlib/claims"
	sdkresource "github.com/grafana/grafana-app-sdk/resource"
	"github.com/grafana/grafana/apps/alerting/notifications/pkg/apis/alertingnotifications/v1beta1"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/resources/appplatform/integrationschemas"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	terraformresource "github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	receiverResourceType = "grafana_apps_notifications_receiver_v1beta1"
	receiverResourceName = receiverResourceType + ".test"
)

// The tests below verify what Grafana persisted (integration UIDs, types, secure fields), not only
// Terraform state: the shared engine populates state from the plan and doesn't refresh the spec.
// They use terraformresource.Test and not ParallelTest: receivers live in the org's Alertmanager
// config, which uses optimistic locking.

func TestAccReceiver(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=13.4.0")

	// Credentials are not returned by the API, so imports can't recover them.
	importIgnore := []string{
		"options.%",
		"options.overwrite",
		"spec.alertmanager.0.basic_auth_password",
		"spec.slack.0.url",
		"spec.slack.1.url",
	}
	importStep := terraformresource.TestStep{
		ResourceName:            receiverResourceName,
		ImportState:             true,
		ImportStateVerify:       true,
		ImportStateVerifyIgnore: importIgnore,
		ImportStateIdFunc:       importStateIDFunc(receiverResourceName),
	}

	t.Run("basic", func(t *testing.T) {
		suffix := acctest.RandString(6)
		uid := "test-receiver-" + suffix
		title := "Receiver " + suffix
		slackUID := "slack-" + suffix
		emailUID := "email-" + suffix

		terraformresource.Test(t, terraformresource.TestCase{
			ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
			CheckDestroy:             testAccCheckReceiverDestroy,
			Steps: []terraformresource.TestStep{
				{
					Config: testAccReceiverConfig(uid, fmt.Sprintf(`
    title = %[1]q

    slack {
      uid   = %[2]q
      url   = "https://example.com/hook"
      title = "Notification"
    }

    email {
      uid       = %[3]q
      addresses = ["a@example.com", "b@example.com"]
    }
`, title, slackUID, emailUID)),
					Check: terraformresource.ComposeTestCheckFunc(
						terraformresource.TestCheckResourceAttr(receiverResourceName, "metadata.uid", uid),
						terraformresource.TestCheckResourceAttr(receiverResourceName, "spec.title", title),
						terraformresource.TestCheckResourceAttr(receiverResourceName, "spec.disable_provenance", "false"),
						terraformresource.TestCheckResourceAttr(receiverResourceName, "spec.slack.0.uid", slackUID),
						terraformresource.TestCheckResourceAttr(receiverResourceName, "spec.slack.0.version", "v1"),
						terraformresource.TestCheckResourceAttr(receiverResourceName, "spec.email.0.addresses.#", "2"),
						testAccCheckReceiverBackend(uid, title, map[string]testAccReceiverIntegration{
							slackUID: {typ: "slack", secureFields: []string{"url"}, settings: map[string]any{"title": "Notification"}},
							emailUID: {typ: "email", settings: map[string]any{"addresses": "a@example.com;b@example.com"}},
						}),
					),
				},
				importStep,
			},
		})
	})

	t.Run("integrations keep their identity when edited, added, removed and reordered", func(t *testing.T) {
		suffix := acctest.RandString(6)
		uid := "test-receiver-" + suffix
		title := "Receiver " + suffix
		a, b, c := "int-a-"+suffix, "int-b-"+suffix, "int-c-"+suffix

		slack := func(uid, text string) string {
			return fmt.Sprintf(`
    slack {
      uid  = %q
      url  = "https://example.com/%s"
      text = %q
    }
`, uid, uid, text)
		}

		terraformresource.Test(t, terraformresource.TestCase{
			ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
			CheckDestroy:             testAccCheckReceiverDestroy,
			Steps: []terraformresource.TestStep{
				{
					Config: testAccReceiverConfig(uid, fmt.Sprintf("title = %q\n%s%s", title, slack(a, "one"), slack(b, "two"))),
					Check: testAccCheckReceiverBackend(uid, title, map[string]testAccReceiverIntegration{
						a: {typ: "slack", secureFields: []string{"url"}, settings: map[string]any{"text": "one"}},
						b: {typ: "slack", secureFields: []string{"url"}, settings: map[string]any{"text": "two"}},
					}),
				},
				{
					// Edit one integration, and insert one in the middle.
					Config: testAccReceiverConfig(uid, fmt.Sprintf("title = %q\n%s%s%s", title, slack(a, "one"), slack(c, "three"), slack(b, "two edited"))),
					Check: testAccCheckReceiverBackend(uid, title, map[string]testAccReceiverIntegration{
						a: {typ: "slack", secureFields: []string{"url"}, settings: map[string]any{"text": "one"}},
						b: {typ: "slack", secureFields: []string{"url"}, settings: map[string]any{"text": "two edited"}},
						c: {typ: "slack", secureFields: []string{"url"}, settings: map[string]any{"text": "three"}},
					}),
				},
				{
					// Remove the first integration: the others keep their own UIDs and credentials.
					Config: testAccReceiverConfig(uid, fmt.Sprintf("title = %q\n%s%s", title, slack(c, "three"), slack(b, "two edited"))),
					Check: testAccCheckReceiverBackend(uid, title, map[string]testAccReceiverIntegration{
						b: {typ: "slack", secureFields: []string{"url"}, settings: map[string]any{"text": "two edited"}},
						c: {typ: "slack", secureFields: []string{"url"}, settings: map[string]any{"text": "three"}},
					}),
				},
				{
					ResourceName:            receiverResourceName,
					ImportState:             true,
					ImportStateVerify:       true,
					ImportStateVerifyIgnore: importIgnore,
					ImportStateIdFunc:       importStateIDFunc(receiverResourceName),
				},
			},
		})
	})

	t.Run("credentials are set, rotated and cleared", func(t *testing.T) {
		suffix := acctest.RandString(6)
		uid := "test-receiver-" + suffix
		title := "Receiver " + suffix
		intUID := "am-" + suffix

		config := func(user, password string) string {
			body := fmt.Sprintf("title = %q\n\n    alertmanager {\n      uid = %q\n      url = \"http://localhost:9093\"\n", title, intUID)
			if user != "" {
				body += fmt.Sprintf("      basic_auth_user = %q\n", user)
			}
			if password != "" {
				body += fmt.Sprintf("      basic_auth_password = %q\n", password)
			}
			return testAccReceiverConfig(uid, body+"    }\n")
		}

		terraformresource.Test(t, terraformresource.TestCase{
			ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
			CheckDestroy:             testAccCheckReceiverDestroy,
			Steps: []terraformresource.TestStep{
				{
					Config: config("user", "secretpass"),
					Check: testAccCheckReceiverBackend(uid, title, map[string]testAccReceiverIntegration{
						intUID: {typ: "prometheus-alertmanager", secureFields: []string{"basicAuthPassword"}, settings: map[string]any{"basicAuthUser": "user"}},
					}),
				},
				{
					// Unrelated change: the credential is sent again and stays set.
					Config: config("user-updated", "secretpass"),
					Check: testAccCheckReceiverBackend(uid, title, map[string]testAccReceiverIntegration{
						intUID: {typ: "prometheus-alertmanager", secureFields: []string{"basicAuthPassword"}, settings: map[string]any{"basicAuthUser": "user-updated"}},
					}),
				},
				{
					// Rotation.
					Config: config("user-updated", "secretpass-rotated"),
					Check: testAccCheckReceiverBackend(uid, title, map[string]testAccReceiverIntegration{
						intUID: {typ: "prometheus-alertmanager", secureFields: []string{"basicAuthPassword"}, settings: map[string]any{"basicAuthUser": "user-updated"}},
					}),
				},
				{
					// Removing the credential from the configuration clears it.
					Config: config("user-updated", ""),
					Check: testAccCheckReceiverBackend(uid, title, map[string]testAccReceiverIntegration{
						intUID: {typ: "prometheus-alertmanager", settings: map[string]any{"basicAuthUser": "user-updated"}},
					}),
				},
			},
		})
	})

	t.Run("rename keeps the receiver", func(t *testing.T) {
		suffix := acctest.RandString(6)
		uid := "test-receiver-" + suffix
		intUID := "email-" + suffix
		var uuid string

		config := func(title string) string {
			return testAccReceiverConfig(uid, fmt.Sprintf(`
    title = %q

    email {
      uid       = %q
      addresses = ["a@example.com"]
    }
`, title, intUID))
		}

		terraformresource.Test(t, terraformresource.TestCase{
			ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
			CheckDestroy:             testAccCheckReceiverDestroy,
			Steps: []terraformresource.TestStep{
				{
					Config: config("Before " + suffix),
					Check: terraformresource.ComposeTestCheckFunc(
						captureAttr(receiverResourceName, "metadata.uuid", &uuid),
						testAccCheckReceiverBackend(uid, "Before "+suffix, map[string]testAccReceiverIntegration{intUID: {typ: "email"}}),
					),
				},
				{
					// metadata.uuid is derived from the identity, so it only shows the identity is stable.
					// The title has no RequiresReplace, so a rename is an in-place update by schema.
					Config: config("After " + suffix),
					Check: terraformresource.ComposeTestCheckFunc(
						terraformresource.TestCheckResourceAttr(receiverResourceName, "metadata.uid", uid),
						func(s *terraform.State) error {
							if got := s.RootModule().Resources[receiverResourceName].Primary.Attributes["metadata.uuid"]; got != uuid {
								return fmt.Errorf("renaming changed the resource identity: metadata.uuid changed from %q to %q", uuid, got)
							}
							return nil
						},
						testAccCheckReceiverBackend(uid, "After "+suffix, map[string]testAccReceiverIntegration{intUID: {typ: "email"}}),
					),
				},
				importStep,
			},
		})
	})

	t.Run("provenance", func(t *testing.T) {
		suffix := acctest.RandString(6)
		uid := "test-receiver-" + suffix
		title := "Receiver " + suffix
		intUID := "email-" + suffix

		config := func(extra string) string {
			return testAccReceiverConfig(uid, fmt.Sprintf(`
    title = %q
    %s

    email {
      uid       = %q
      addresses = ["a@example.com"]
    }
`, title, extra, intUID))
		}

		terraformresource.Test(t, terraformresource.TestCase{
			ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
			CheckDestroy:             testAccCheckReceiverDestroy,
			Steps: []terraformresource.TestStep{
				{
					Config: config(""), // defaults to false
					Check:  terraformresource.TestCheckResourceAttr(receiverResourceName, "spec.disable_provenance", "false"),
				},
				importStep,
				{
					Config: config("disable_provenance = true"),
					Check:  terraformresource.TestCheckResourceAttr(receiverResourceName, "spec.disable_provenance", "true"),
				},
				importStep,
				{
					Config: config("disable_provenance = false"),
					Check:  terraformresource.TestCheckResourceAttr(receiverResourceName, "spec.disable_provenance", "false"),
				},
				importStep,
			},
		})
	})

	t.Run("a receiver without integrations", func(t *testing.T) {
		suffix := acctest.RandString(6)
		uid := "test-receiver-" + suffix
		title := "Empty " + suffix

		terraformresource.Test(t, terraformresource.TestCase{
			ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
			CheckDestroy:             testAccCheckReceiverDestroy,
			Steps: []terraformresource.TestStep{
				{
					// Grafana stores provenance on integrations, so a receiver without any is never locked.
					Config: testAccReceiverConfig(uid, fmt.Sprintf("title = %q\n    disable_provenance = true", title)),
					Check: terraformresource.ComposeTestCheckFunc(
						terraformresource.TestCheckResourceAttr(receiverResourceName, "spec.title", title),
						testAccCheckReceiverBackend(uid, title, nil),
					),
				},
				importStep,
			},
		})
	})

	t.Run("a receiver deleted outside Terraform is created again", func(t *testing.T) {
		suffix := acctest.RandString(6)
		uid := "test-receiver-" + suffix
		title := "Recreate " + suffix
		emailUID := "email-" + suffix
		config := testAccReceiverConfig(uid, fmt.Sprintf(`
    title = %q

    email {
      uid       = %q
      addresses = ["a@example.com"]
    }
`, title, emailUID))
		check := testAccCheckReceiverBackend(uid, title, map[string]testAccReceiverIntegration{
			emailUID: {typ: "email", settings: map[string]any{"addresses": "a@example.com"}},
		})

		terraformresource.Test(t, terraformresource.TestCase{
			ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
			CheckDestroy:             testAccCheckReceiverDestroy,
			Steps: []terraformresource.TestStep{
				{Config: config, Check: check},
				{
					PreConfig: func() {
						cli, err := testAccReceiverClient()
						if err != nil {
							t.Fatal(err)
						}
						if err := cli.Delete(context.Background(), uid, sdkresource.DeleteOptions{}); err != nil {
							t.Fatal(err)
						}
					},
					Config:             config,
					PlanOnly:           true,
					ExpectNonEmptyPlan: true,
				},
				{Config: config, Check: check},
			},
		})
	})

	t.Run("the example", func(t *testing.T) {
		const exampleResourceName = receiverResourceType + ".operations"
		terraformresource.Test(t, terraformresource.TestCase{
			ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
			CheckDestroy:             testAccCheckReceiverDestroy,
			Steps: []terraformresource.TestStep{
				{
					Config: testutils.TestAccExample(t, "resources/grafana_apps_notifications_receiver_v1beta1/resource.tf"),
					Check: testAccCheckReceiverBackend("operations", "Operations", map[string]testAccReceiverIntegration{
						"operations-slack-primary":   {typ: "slack", secureFields: []string{"url"}, settings: map[string]any{"title": "Primary notifications"}},
						"operations-slack-secondary": {typ: "slack", secureFields: []string{"url"}},
						"operations-email":           {typ: "email", settings: map[string]any{"addresses": "operations@example.com"}},
						"operations-webhook":         {typ: "webhook", secureFields: []string{"http_config.oauth2.client_secret"}, settings: map[string]any{"httpMethod": "POST"}},
					}),
				},
				{
					ResourceName:      exampleResourceName,
					ImportState:       true,
					ImportStateVerify: true,
					ImportStateVerifyIgnore: append([]string{
						"spec.webhook.0.http_config.oauth2.client_secret",
					}, importIgnore...),
					ImportStateIdFunc: importStateIDFunc(exampleResourceName),
				},
			},
		})
	})

	t.Run("duplicate integration uids are rejected at plan time", func(t *testing.T) {
		suffix := acctest.RandString(6)
		uid := "test-receiver-" + suffix
		dup := "dup-" + suffix

		terraformresource.Test(t, terraformresource.TestCase{
			ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
			CheckDestroy:             testAccCheckReceiverDestroy,
			Steps: []terraformresource.TestStep{
				{
					Config: testAccReceiverConfig(uid, fmt.Sprintf(`
    title = "Duplicate uids %[1]s"

    email {
      uid       = %[2]q
      addresses = ["a@example.com"]
    }

    email {
      uid       = %[2]q
      addresses = ["b@example.com"]
    }
`, suffix, dup)),
					ExpectError: regexp.MustCompile("Duplicate integration uid"),
				},
			},
		})
	})
}

// receiverTypeConfig is one integration block, and what Grafana is expected to store for it.
type receiverTypeConfig struct {
	// body is the content of the integration block, besides its uid.
	body string
	// settings are expected stored values, by dot-separated Grafana settings key.
	settings map[string]any
	// credentials are the Terraform paths of the credentials in body (relative to the block); Grafana
	// reports their Grafana keys in secureFields, and import cannot recover them.
	credentials map[string]string // Terraform path -> Grafana key
}

type receiverTypeCase struct {
	name     string
	tfType   string // block name
	apiType  string
	required receiverTypeConfig // only the fields that are required
	all      receiverTypeConfig // every field that can be set together
}

// TestAccReceiver_integrationTypes applies each integration type with only its required
// fields, then with all of its fields, and compares what Grafana stored with what was configured.
func TestAccReceiver_integrationTypes(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=13.4.0")

	for _, c := range receiverTypeCases() {
		t.Run(c.name, func(t *testing.T) {
			suffix := acctest.RandString(6)
			uid := "test-receiver-" + suffix
			title := "Types " + c.name + " " + suffix
			integrationUID := "int-" + suffix

			block := func(cfg receiverTypeConfig) string {
				return fmt.Sprintf("title = %q\n\n%s {\n  uid = %q\n%s\n}\n", title, c.tfType, integrationUID, cfg.body)
			}

			var importIgnore []string
			importIgnore = append(importIgnore, "options.%", "options.overwrite")
			for path := range c.all.credentials {
				importIgnore = append(importIgnore, fmt.Sprintf("spec.%s.0.%s", c.tfType, path))
			}

			terraformresource.Test(t, terraformresource.TestCase{
				ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
				CheckDestroy:             testAccCheckReceiverDestroy,
				Steps: []terraformresource.TestStep{
					{
						Config: testAccReceiverConfig(uid, block(c.required)),
						Check:  testAccCheckReceiverIntegration(uid, title, integrationUID, c.apiType, c.required),
					},
					{
						Config: testAccReceiverConfig(uid, block(c.all)),
						Check:  testAccCheckReceiverIntegration(uid, title, integrationUID, c.apiType, c.all),
					},
					{
						ResourceName:            receiverResourceName,
						ImportState:             true,
						ImportStateVerify:       true,
						ImportStateVerifyIgnore: importIgnore,
						ImportStateIdFunc:       importStateIDFunc(receiverResourceName),
					},
				},
			})
		})
	}
}

// testAccCheckReceiverIntegration reads the receiver from Grafana and checks the integration's type,
// settings and credentials.
func testAccCheckReceiverIntegration(uid, title, integrationUID, apiType string, want receiverTypeConfig) terraformresource.TestCheckFunc {
	return func(_ *terraform.State) error {
		recv, err := testAccGetReceiver(uid)
		if err != nil {
			return err
		}
		if recv.Spec.Title != title {
			return fmt.Errorf("title is %q, want %q", recv.Spec.Title, title)
		}
		if len(recv.Spec.Integrations) != 1 {
			return fmt.Errorf("has %d integrations, want 1", len(recv.Spec.Integrations))
		}
		in := recv.Spec.Integrations[0]
		if in.Uid == nil || *in.Uid != integrationUID || in.Type != apiType || in.Version != "v1" {
			return fmt.Errorf("integration is %v/%s/%s, want %s/%s/v1", in.Uid, in.Type, in.Version, integrationUID, apiType)
		}

		for key, value := range want.settings {
			got, ok := receiverSettingAt(in.Settings, strings.Split(key, "."))
			if !ok || fmt.Sprint(got) != fmt.Sprint(value) {
				return fmt.Errorf("setting %s is %v, want %v", key, got, value)
			}
		}

		var secure []string
		for k, set := range in.SecureFields {
			if set {
				secure = append(secure, k)
			}
		}
		var wantSecure []string
		for _, key := range want.credentials {
			wantSecure = append(wantSecure, key)
		}
		sort.Strings(secure)
		sort.Strings(wantSecure)
		if !slices.Equal(secure, wantSecure) {
			return fmt.Errorf("secure fields are %v, want %v", secure, wantSecure)
		}
		for _, key := range wantSecure {
			if got, ok := receiverSettingAt(in.Settings, strings.Split(key, ".")); ok {
				return fmt.Errorf("credential %s is in the stored settings: %v", key, got)
			}
		}
		return nil
	}
}

func receiverSettingAt(settings map[string]any, path []string) (any, bool) {
	var current any = settings
	for _, key := range path {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		if current, ok = m[key]; !ok {
			return nil, false
		}
	}
	return current, true
}

func receiverTypeCases() []receiverTypeCase {
	return []receiverTypeCase{
		{
			name: "alertmanager", tfType: "alertmanager", apiType: "prometheus-alertmanager",
			required: receiverTypeConfig{body: `url = "http://alertmanager:9093"`},
			all: receiverTypeConfig{
				body: `
  url                 = "http://alertmanager:9093"
  basic_auth_user     = "user"
  basic_auth_password = "password"`,
				settings: map[string]any{
					"url":           "http://alertmanager:9093",
					"basicAuthUser": "user",
				},
				credentials: map[string]string{"basic_auth_password": "basicAuthPassword"},
			},
		},
		{
			name: "dingding", tfType: "dingding", apiType: "dingding",
			required: receiverTypeConfig{
				body:        `url = "https://example.com/dingding"`,
				credentials: map[string]string{"url": "url"},
			},
			all: receiverTypeConfig{
				body: `
  url          = "https://example.com/dingding"
  message_type = "actionCard"
  title        = "title"
  message      = "message"`,
				settings: map[string]any{
					"msgType": "actionCard",
					"title":   "title",
					"message": "message",
				},
				credentials: map[string]string{"url": "url"},
			},
		},
		{
			name: "discord", tfType: "discord", apiType: "discord",
			required: receiverTypeConfig{
				body:        `url = "https://example.com/discord"`,
				credentials: map[string]string{"url": "url"},
			},
			all: receiverTypeConfig{
				body: `
  url                   = "https://example.com/discord"
  title                 = "title"
  message               = "message"
  avatar_url            = "https://example.com/avatar.png"
  use_discord_username  = true
  use_embed_description = true`,
				settings: map[string]any{
					"title":                 "title",
					"message":               "message",
					"avatar_url":            "https://example.com/avatar.png",
					"use_discord_username":  true,
					"use_embed_description": true,
				},
				credentials: map[string]string{"url": "url"},
			},
		},
		{
			name: "email", tfType: "email", apiType: "email",
			required: receiverTypeConfig{
				body:     `addresses = ["a@example.com", "b@example.com"]`,
				settings: map[string]any{"addresses": "a@example.com;b@example.com"},
			},
			all: receiverTypeConfig{
				body: `
  addresses    = ["a@example.com", "b@example.com"]
  single_email = true
  message      = "message"
  subject      = "subject"`,
				settings: map[string]any{
					"addresses":   "a@example.com;b@example.com",
					"singleEmail": true,
					"message":     "message",
					"subject":     "subject",
				},
			},
		},
		{
			name: "googlechat", tfType: "googlechat", apiType: "googlechat",
			required: receiverTypeConfig{
				body:        `url = "https://example.com/googlechat"`,
				credentials: map[string]string{"url": "url"},
			},
			all: receiverTypeConfig{
				body: `
  url               = "https://example.com/googlechat"
  title             = "title"
  message           = "message"
  hide_open_button  = true
  hide_version_info = true`,
				settings: map[string]any{
					"title":             "title",
					"message":           "message",
					"hide_open_button":  true,
					"hide_version_info": true,
				},
				credentials: map[string]string{"url": "url"},
			},
		},
		{
			name: "jira with a token", tfType: "jira", apiType: "jira",
			required: receiverTypeConfig{
				body: `
  api_url    = "https://example.atlassian.net/rest/api/3/"
  project    = "OPS"
  issue_type = "Incident"
  api_token  = "token"`,
				settings: map[string]any{
					"api_url":    "https://example.atlassian.net/rest/api/3/",
					"project":    "OPS",
					"issue_type": "Incident",
				},
				credentials: map[string]string{"api_token": "api_token"},
			},
			all: receiverTypeConfig{
				body: `
  api_url             = "https://example.atlassian.net/rest/api/3/"
  project             = "OPS"
  issue_type          = "Incident"
  api_token           = "token"
  summary             = "summary"
  description         = "description"
  labels              = ["one", "two"]
  priority            = "High"
  resolve_transition  = "Done"
  reopen_transition   = "Reopen"
  reopen_duration     = "1m"
  wont_fix_resolution = "Won't Fix"
  dedup_key_field     = "10001"
  fields              = { customfield_10002 = "value" }`,
				settings: map[string]any{
					"summary":             "summary",
					"description":         "description",
					"labels":              []string{"one", "two"},
					"priority":            "High",
					"resolve_transition":  "Done",
					"reopen_transition":   "Reopen",
					"reopen_duration":     "1m",
					"wont_fix_resolution": "Won't Fix",
					"dedup_key_field":     "10001",
					"fields":              map[string]string{"customfield_10002": "value"},
				},
				credentials: map[string]string{"api_token": "api_token"},
			},
		},
		{
			name: "jira with a user and password", tfType: "jira", apiType: "jira",
			required: receiverTypeConfig{
				body: `
  api_url    = "https://example.atlassian.net/rest/api/3/"
  project    = "OPS"
  issue_type = "Incident"
  user       = "user"
  password   = "password"`,
				credentials: map[string]string{"user": "user", "password": "password"},
			},
			all: receiverTypeConfig{
				body: `
  api_url    = "https://example.atlassian.net/rest/api/3/"
  project    = "OPS"
  issue_type = "Incident"
  user       = "user"
  password   = "password"`,
				credentials: map[string]string{"user": "user", "password": "password"},
			},
		},
		{
			name: "kafka", tfType: "kafka", apiType: "kafka",
			required: receiverTypeConfig{
				body: "rest_proxy_url = \"http://kafka-rest-proxy:8082\"\n  topic = \"alerts\"",
				settings: map[string]any{
					"kafkaRestProxy": "http://kafka-rest-proxy:8082",
					"kafkaTopic":     "alerts",
				},
			},
			all: receiverTypeConfig{
				body: `
  rest_proxy_url = "http://kafka-rest-proxy:8082"
  topic          = "alerts"
  username       = "user"
  password       = "password"
  api_version    = "v3"
  cluster_id     = "cluster"
  description    = "description"
  details        = "details"`,
				settings: map[string]any{
					"kafkaRestProxy": "http://kafka-rest-proxy:8082",
					"kafkaTopic":     "alerts",
					"username":       "user",
					"apiVersion":     "v3",
					"kafkaClusterId": "cluster",
					"description":    "description",
					"details":        "details",
				},
				credentials: map[string]string{"password": "password"},
			},
		},
		{
			name: "mqtt", tfType: "mqtt", apiType: "mqtt",
			required: receiverTypeConfig{
				body: "broker_url = \"tcp://mqtt:1883\"\n  topic = \"alerts\"",
				settings: map[string]any{
					"brokerUrl": "tcp://mqtt:1883",
					"topic":     "alerts",
				},
			},
			all: receiverTypeConfig{
				body: `
  broker_url     = "tcp://mqtt:1883"
  topic          = "alerts"
  message_format = "text"
  client_id      = "grafana"
  message        = "message"
  username       = "user"
  password       = "password"
  qos            = "1"
  retain         = true

  tls_config {
    insecure_skip_verify = true
  }`,
				settings: map[string]any{
					"brokerUrl":                    "tcp://mqtt:1883",
					"topic":                        "alerts",
					"messageFormat":                "text",
					"clientId":                     "grafana",
					"message":                      "message",
					"username":                     "user",
					"qos":                          "1",
					"retain":                       true,
					"tlsConfig.insecureSkipVerify": true,
				},
				credentials: map[string]string{"password": "password"},
			},
		},
		{
			name: "oncall", tfType: "oncall", apiType: "oncall",
			required: receiverTypeConfig{body: `url = "https://example.com/oncall"`},
			all: receiverTypeConfig{
				body: `
  url                 = "https://example.com/oncall"
  http_method         = "PUT"
  basic_auth_user     = "user"
  basic_auth_password = "password"
  max_alerts          = 5
  title               = "title"
  message             = "message"`,
				settings: map[string]any{
					"url":        "https://example.com/oncall",
					"httpMethod": "PUT",
					"username":   "user",
					"maxAlerts":  5,
					"title":      "title",
					"message":    "message",
				},
				credentials: map[string]string{"basic_auth_password": "password"},
			},
		},
		{
			name: "oncall with an authorization header", tfType: "oncall", apiType: "oncall",
			required: receiverTypeConfig{body: `url = "https://example.com/oncall"`},
			all: receiverTypeConfig{
				body: `
  url                       = "https://example.com/oncall"
  authorization_scheme      = "Bearer"
  authorization_credentials = "secret"`,
				settings:    map[string]any{"authorization_scheme": "Bearer"},
				credentials: map[string]string{"authorization_credentials": "authorization_credentials"},
			},
		},
		{
			name: "opsgenie", tfType: "opsgenie", apiType: "opsgenie",
			required: receiverTypeConfig{
				body:        `api_key = "key"`,
				credentials: map[string]string{"api_key": "apiKey"},
			},
			all: receiverTypeConfig{
				body: `
  api_key           = "key"
  url               = "https://api.opsgenie.com/v2/alerts"
  message           = "message"
  description       = "description"
  auto_close        = true
  override_priority = true
  send_tags_as      = "both"

  responders {
    type = "team"
    name = "operations"
  }

  responders {
    type     = "user"
    username = "user@example.com"
  }`,
				settings: map[string]any{
					"apiUrl":           "https://api.opsgenie.com/v2/alerts",
					"message":          "message",
					"description":      "description",
					"autoClose":        true,
					"overridePriority": true,
					"sendTagsAs":       "both",
				},
				credentials: map[string]string{"api_key": "apiKey"},
			},
		},
		{
			name: "pagerduty", tfType: "pagerduty", apiType: "pagerduty",
			required: receiverTypeConfig{
				body:        `integration_key = "key"`,
				credentials: map[string]string{"integration_key": "integrationKey"},
			},
			all: receiverTypeConfig{
				body: `
  integration_key = "key"
  severity        = "critical"
  class           = "ping failure"
  component       = "mysql"
  group           = "my service"
  summary         = "summary"
  source          = "source"
  client          = "client"
  client_url      = "https://example.com/client"
  details         = { one = "two" }
  url             = "https://events.pagerduty.com/v2/enqueue"`,
				settings: map[string]any{
					"severity":   "critical",
					"class":      "ping failure",
					"component":  "mysql",
					"group":      "my service",
					"summary":    "summary",
					"source":     "source",
					"client":     "client",
					"client_url": "https://example.com/client",
					"details":    map[string]string{"one": "two"},
					"url":        "https://events.pagerduty.com/v2/enqueue",
				},
				credentials: map[string]string{"integration_key": "integrationKey"},
			},
		},
		{
			name: "pushover", tfType: "pushover", apiType: "pushover",
			required: receiverTypeConfig{
				body:        "api_token = \"token\"\n  user_key = \"user\"",
				credentials: map[string]string{"api_token": "apiToken", "user_key": "userKey"},
			},
			all: receiverTypeConfig{
				body: `
  api_token    = "token"
  user_key     = "user"
  device       = "device"
  priority     = 2
  ok_priority  = 0
  retry        = 60
  expire       = 3600
  sound        = "bugle"
  ok_sound     = "cashregister"
  upload_image = true
  title        = "title"
  message      = "message"`,
				settings: map[string]any{
					"device":      "device",
					"priority":    2,
					"okPriority":  0,
					"retry":       60,
					"expire":      3600,
					"sound":       "bugle",
					"okSound":     "cashregister",
					"uploadImage": true,
					"title":       "title",
					"message":     "message",
				},
				credentials: map[string]string{"api_token": "apiToken", "user_key": "userKey"},
			},
		},
		{
			name: "sensugo", tfType: "sensugo", apiType: "sensugo",
			required: receiverTypeConfig{
				body:        "url = \"https://example.com/sensu\"\n  api_key = \"key\"",
				settings:    map[string]any{"url": "https://example.com/sensu"},
				credentials: map[string]string{"api_key": "apikey"},
			},
			all: receiverTypeConfig{
				body: `
  url       = "https://example.com/sensu"
  api_key   = "key"
  entity    = "entity"
  check     = "check"
  handler   = "handler"
  namespace = "namespace"
  message   = "message"`,
				settings: map[string]any{
					"url":       "https://example.com/sensu",
					"entity":    "entity",
					"check":     "check",
					"handler":   "handler",
					"namespace": "namespace",
					"message":   "message",
				},
				credentials: map[string]string{"api_key": "apikey"},
			},
		},
		{
			name: "slack with a webhook url", tfType: "slack", apiType: "slack",
			required: receiverTypeConfig{
				body:        `url = "https://example.com/slack"`,
				credentials: map[string]string{"url": "url"},
			},
			all: receiverTypeConfig{
				body: `
  url             = "https://example.com/slack"
  recipient       = "#alerts"
  username        = "grafana"
  icon_emoji      = ":bell:"
  icon_url        = "https://example.com/icon.png"
  mention_users   = "U1,U2"
  mention_groups  = "G1"
  mention_channel = "here"
  color           = "#ff0000"
  title           = "title"
  text            = "text"
  footer          = "footer"`,
				settings: map[string]any{
					"recipient":      "#alerts",
					"username":       "grafana",
					"icon_emoji":     ":bell:",
					"icon_url":       "https://example.com/icon.png",
					"mentionUsers":   "U1,U2",
					"mentionGroups":  "G1",
					"mentionChannel": "here",
					"color":          "#ff0000",
					"title":          "title",
					"text":           "text",
					"footer":         "footer",
				},
				credentials: map[string]string{"url": "url"},
			},
		},
		{
			name: "slack with the chat API", tfType: "slack", apiType: "slack",
			required: receiverTypeConfig{
				body:        "token = \"xoxb-token\"\n  recipient = \"#alerts\"",
				settings:    map[string]any{"recipient": "#alerts"},
				credentials: map[string]string{"token": "token"},
			},
			all: receiverTypeConfig{
				body: `
  token        = "xoxb-token"
  recipient    = "#alerts"
  endpoint_url = "https://slack.com/api/chat.postMessage"`,
				settings: map[string]any{
					"recipient":   "#alerts",
					"endpointUrl": "https://slack.com/api/chat.postMessage",
				},
				credentials: map[string]string{"token": "token"},
			},
		},
		{
			name: "sns", tfType: "sns", apiType: "sns",
			required: receiverTypeConfig{
				body:     `topic_arn = "arn:aws:sns:us-east-1:123456789012:topic"`,
				settings: map[string]any{"topic_arn": "arn:aws:sns:us-east-1:123456789012:topic"},
			},
			all: receiverTypeConfig{
				body: `
  api_url   = "https://sns.us-east-1.amazonaws.com"
  topic_arn = "arn:aws:sns:us-east-1:123456789012:topic"
  subject   = "subject"
  message   = "message"
  attributes = { key = "value" }

  sigv4 {
    region     = "us-east-1"
    access_key = "access"
    secret_key = "secret"
    profile    = "profile"
    role_arn   = "arn:aws:iam::123456789012:role/grafana"
  }`,
				settings: map[string]any{
					"api_url":        "https://sns.us-east-1.amazonaws.com",
					"subject":        "subject",
					"message":        "message",
					"attributes":     map[string]string{"key": "value"},
					"sigv4.region":   "us-east-1",
					"sigv4.profile":  "profile",
					"sigv4.role_arn": "arn:aws:iam::123456789012:role/grafana",
				},
				credentials: map[string]string{"sigv4.access_key": "sigv4.access_key", "sigv4.secret_key": "sigv4.secret_key"}, // #nosec G101 -- attribute names, not credentials
			},
		},
		{
			name: "teams", tfType: "teams", apiType: "teams",
			required: receiverTypeConfig{
				body:     `url = "https://example.com/teams"`,
				settings: map[string]any{"url": "https://example.com/teams"},
			},
			all: receiverTypeConfig{
				body: `
  url           = "https://example.com/teams"
  title         = "title"
  section_title = "section"
  message       = "message"`,
				settings: map[string]any{
					"url":          "https://example.com/teams",
					"title":        "title",
					"sectiontitle": "section",
					"message":      "message",
				},
			},
		},
		{
			name: "telegram", tfType: "telegram", apiType: "telegram",
			required: receiverTypeConfig{
				body:        "token = \"token\"\n  chat_id = \"123\"",
				settings:    map[string]any{"chatid": "123"},
				credentials: map[string]string{"token": "bottoken"},
			},
			all: receiverTypeConfig{
				body: `
  token                    = "token"
  chat_id                  = "123"
  message_thread_id        = "456"
  message                  = "message"
  parse_mode               = "HTML"
  disable_web_page_preview = true
  protect_content          = true
  disable_notifications    = true`,
				settings: map[string]any{
					"chatid":                   "123",
					"message_thread_id":        "456",
					"message":                  "message",
					"parse_mode":               "HTML",
					"disable_web_page_preview": true,
					"protect_content":          true,
					"disable_notifications":    true,
				},
				credentials: map[string]string{"token": "bottoken"},
			},
		},
		{
			name: "threema", tfType: "threema", apiType: "threema",
			required: receiverTypeConfig{
				body: "gateway_id = \"*MYGATE1\"\n  recipient_id = \"ECHOECHO\"\n  api_secret = \"secret\"",
				settings: map[string]any{
					"gateway_id":   "*MYGATE1",
					"recipient_id": "ECHOECHO",
				},
				credentials: map[string]string{"api_secret": "api_secret"},
			},
			all: receiverTypeConfig{
				body: `
  gateway_id   = "*MYGATE1"
  recipient_id = "ECHOECHO"
  api_secret   = "secret"
  title        = "title"
  description  = "description"`,
				settings: map[string]any{
					"gateway_id":   "*MYGATE1",
					"recipient_id": "ECHOECHO",
					"title":        "title",
					"description":  "description",
				},
				credentials: map[string]string{"api_secret": "api_secret"},
			},
		},
		{
			name: "victorops", tfType: "victorops", apiType: "victorops",
			required: receiverTypeConfig{
				body:        `url = "https://example.com/victorops"`,
				credentials: map[string]string{"url": "url"},
			},
			all: receiverTypeConfig{
				body: `
  url          = "https://example.com/victorops"
  message_type = "WARNING"
  title        = "title"
  description  = "description"`,
				settings: map[string]any{
					"messageType": "WARNING",
					"title":       "title",
					"description": "description",
				},
				credentials: map[string]string{"url": "url"},
			},
		},
		{
			name: "webex", tfType: "webex", apiType: "webex",
			required: receiverTypeConfig{
				body:        "room_id = \"room\"\n  token = \"token\"",
				settings:    map[string]any{"room_id": "room"},
				credentials: map[string]string{"token": "bot_token"},
			},
			all: receiverTypeConfig{
				body: `
  room_id = "room"
  token   = "token"
  api_url = "https://webexapis.com/v1/messages"
  message = "message"`,
				settings: map[string]any{
					"room_id": "room",
					"api_url": "https://webexapis.com/v1/messages",
					"message": "message",
				},
				credentials: map[string]string{"token": "bot_token"},
			},
		},
		{
			name: "webhook", tfType: "webhook", apiType: "webhook",
			required: receiverTypeConfig{
				body:     `url = "https://example.com/hook"`,
				settings: map[string]any{"url": "https://example.com/hook"},
			},
			all: receiverTypeConfig{
				body: `
  url                 = "https://example.com/hook"
  http_method         = "PUT"
  basic_auth_user     = "user"
  basic_auth_password = "password"
  headers             = { "X-Custom" = "value" }
  max_alerts          = 5
  title               = "title"
  message             = "message"

  payload {
    template = "{{ .Message }}"
    vars     = { key = "value" }
  }

  tls_config {
    insecure_skip_verify = true
  }

  hmac_config {
    secret           = "hmac-secret"
    header           = "X-Signature"
    timestamp_header = "X-Timestamp"
  }

  http_config {
    oauth2 {
      token_url       = "https://example.com/oauth/token"
      client_id       = "grafana"
      client_secret   = "client-secret"
      scopes          = ["one", "two"]
      endpoint_params = { audience = "grafana" }

      tls_config {
        insecure_skip_verify = true
      }

      proxy_config {
        proxy_url            = "https://proxy.example.com"
        no_proxy             = "localhost"
        proxy_connect_header = { "X-Proxy" = "value" }
      }
    }
  }`,
				settings: map[string]any{ // #nosec G101 -- attribute names, not credentials
					"httpMethod":                         "PUT",
					"username":                           "user",
					"headers":                            map[string]string{"X-Custom": "value"},
					"maxAlerts":                          5,
					"title":                              "title",
					"message":                            "message",
					"payload.template":                   "{{ .Message }}",
					"payload.vars":                       map[string]string{"key": "value"},
					"tlsConfig.insecureSkipVerify":       true,
					"hmacConfig.header":                  "X-Signature",
					"hmacConfig.timestampHeader":         "X-Timestamp",
					"http_config.oauth2.token_url":       "https://example.com/oauth/token",
					"http_config.oauth2.client_id":       "grafana",
					"http_config.oauth2.scopes":          []string{"one", "two"},
					"http_config.oauth2.endpoint_params": map[string]string{"audience": "grafana"},
					"http_config.oauth2.tls_config.insecureSkipVerify":     true,
					"http_config.oauth2.proxy_config.proxy_url":            "https://proxy.example.com",
					"http_config.oauth2.proxy_config.no_proxy":             "localhost",
					"http_config.oauth2.proxy_config.proxy_connect_header": map[string]string{"X-Proxy": "value"},
				},
				credentials: map[string]string{ // #nosec G101 -- attribute names, not credentials
					"basic_auth_password":              "password",
					"hmac_config.secret":               "hmacConfig.secret",
					"http_config.oauth2.client_secret": "http_config.oauth2.client_secret",
				},
			},
		},
		{
			name: "webhook with an authorization header", tfType: "webhook", apiType: "webhook",
			required: receiverTypeConfig{body: `url = "https://example.com/hook"`},
			all: receiverTypeConfig{
				body: `
  url                       = "https://example.com/hook"
  authorization_scheme      = "Bearer"
  authorization_credentials = "secret"`,
				settings:    map[string]any{"authorization_scheme": "Bearer"},
				credentials: map[string]string{"authorization_credentials": "authorization_credentials"},
			},
		},
		{
			name: "wecom with a webhook url", tfType: "wecom", apiType: "wecom",
			required: receiverTypeConfig{
				body:        `url = "https://example.com/wecom"`,
				credentials: map[string]string{"url": "url"},
			},
			all: receiverTypeConfig{
				body: `
  url      = "https://example.com/wecom"
  msg_type = "markdown"
  message  = "message"
  title    = "title"
  to_user  = "user1|user2"`,
				settings: map[string]any{
					"msgtype": "markdown",
					"message": "message",
					"title":   "title",
					"touser":  "user1|user2",
				},
				credentials: map[string]string{"url": "url"},
			},
		},
		{
			name: "wecom with a secret", tfType: "wecom", apiType: "wecom",
			required: receiverTypeConfig{
				body: `
  secret   = "secret"
  agent_id = "1000001"
  corp_id  = "corp"`,
				settings: map[string]any{
					"agent_id": "1000001",
					"corp_id":  "corp",
				},
				credentials: map[string]string{"secret": "secret"},
			},
			all: receiverTypeConfig{
				body: `
  secret       = "secret"
  agent_id     = "1000001"
  corp_id      = "corp"
  endpoint_url = "https://qyapi.weixin.qq.com/cgi-bin"`,
				settings: map[string]any{
					"agent_id":    "1000001",
					"corp_id":     "corp",
					"endpointUrl": "https://qyapi.weixin.qq.com/cgi-bin",
				},
				credentials: map[string]string{"secret": "secret"},
			},
		},
	}
}

// TestReceiverTypeCasesCoverEveryIntegrationType fails when the schema has an integration type that
// TestAccReceiver_integrationTypes has no case for.
func TestReceiverTypeCasesCoverEveryIntegrationType(t *testing.T) {
	types, err := integrationschemas.Load()
	if err != nil {
		t.Fatal(err)
	}
	covered := map[string]string{}
	for _, c := range receiverTypeCases() {
		covered[c.tfType] = c.apiType
	}
	for _, it := range types {
		apiType, ok := covered[it.TFName]
		if !ok {
			t.Errorf("integration type %q has no case in receiverTypeCases", it.TFName)
		} else if apiType != it.APIType {
			t.Errorf("the cases for %q use API type %q, want %q", it.TFName, apiType, it.APIType)
		}
	}
	for tfType := range covered {
		found := false
		for _, it := range types {
			found = found || it.TFName == tfType
		}
		if !found {
			t.Errorf("receiverTypeCases has a case for %q, which is not an integration type", tfType)
		}
	}
}

func testAccReceiverConfig(uid, spec string) string {
	return fmt.Sprintf(`
resource "grafana_apps_notifications_receiver_v1beta1" "test" {
  metadata {
    uid = %q
  }

  spec {
    %s
  }
}
`, uid, spec)
}

// testAccReceiverIntegration is what Grafana is expected to have stored for one integration.
type testAccReceiverIntegration struct {
	typ string
	// secureFields are the names of the credentials Grafana reports as set; empty means none.
	secureFields []string
	// settings is a subset of the stored (non-secret) settings.
	settings map[string]any
}

// testAccCheckReceiverBackend reads the receiver from Grafana and compares the title, the integration
// UIDs and types, the stored credentials and the given settings.
func testAccCheckReceiverBackend(uid, title string, want map[string]testAccReceiverIntegration) terraformresource.TestCheckFunc {
	return func(_ *terraform.State) error {
		recv, err := testAccGetReceiver(uid)
		if err != nil {
			return err
		}
		if recv.Spec.Title != title {
			return fmt.Errorf("receiver %s: title is %q, want %q", uid, recv.Spec.Title, title)
		}
		if len(recv.Spec.Integrations) != len(want) {
			return fmt.Errorf("receiver %s: has %d integrations, want %d", uid, len(recv.Spec.Integrations), len(want))
		}
		for _, in := range recv.Spec.Integrations {
			if in.Uid == nil {
				return fmt.Errorf("receiver %s: integration without uid", uid)
			}
			exp, ok := want[*in.Uid]
			if !ok {
				return fmt.Errorf("receiver %s: unexpected integration uid %q", uid, *in.Uid)
			}
			if in.Type != exp.typ || in.Version != "v1" {
				return fmt.Errorf("integration %s: is %s/%s, want %s/v1", *in.Uid, in.Type, in.Version, exp.typ)
			}
			var secure []string
			for k, set := range in.SecureFields {
				if set {
					secure = append(secure, k)
				}
			}
			sort.Strings(secure)
			wantSecure := append([]string(nil), exp.secureFields...)
			sort.Strings(wantSecure)
			if fmt.Sprint(secure) != fmt.Sprint(wantSecure) {
				return fmt.Errorf("integration %s: secure fields are %v, want %v", *in.Uid, secure, wantSecure)
			}
			for k, v := range exp.settings {
				if got := in.Settings[k]; fmt.Sprint(got) != fmt.Sprint(v) {
					return fmt.Errorf("integration %s: setting %s is %v, want %v", *in.Uid, k, got, v)
				}
			}
		}
		return nil
	}
}

func testAccReceiverClient() (*sdkresource.NamespacedClient[*v1beta1.Receiver, *v1beta1.ReceiverList], error) {
	client := testutils.Provider.Meta().(*common.Client)
	rcli, err := client.GrafanaAppPlatformAPI.ClientFor(v1beta1.ReceiverKind())
	if err != nil {
		return nil, fmt.Errorf("failed to create app platform client: %w", err)
	}
	return sdkresource.NewNamespaced(
		sdkresource.NewTypedClient[*v1beta1.Receiver, *v1beta1.ReceiverList](rcli, v1beta1.ReceiverKind()),
		claims.OrgNamespaceFormatter(client.GrafanaOrgID),
	), nil
}

func testAccGetReceiver(uid string) (*v1beta1.Receiver, error) {
	cli, err := testAccReceiverClient()
	if err != nil {
		return nil, err
	}
	return cli.Get(context.Background(), uid)
}

func testAccCheckReceiverDestroy(s *terraform.State) error {
	for _, r := range s.RootModule().Resources {
		if r.Type != receiverResourceType {
			continue
		}
		uid := r.Primary.Attributes["metadata.uid"]
		if _, err := testAccGetReceiver(uid); err == nil {
			return fmt.Errorf("Receiver %s still exists", uid)
		} else if !apierrors.IsNotFound(err) {
			return fmt.Errorf("error checking if Receiver %s exists: %w", uid, err)
		}
	}
	return nil
}
