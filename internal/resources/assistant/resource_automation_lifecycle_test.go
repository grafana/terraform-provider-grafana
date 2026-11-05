package assistant_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
)

// These tests run the real muxed provider through Terraform against a local
// contract fixture. They exercise planning and state consistency, not backend
// scheduling, permissions or model execution. No run endpoint is implemented.
func TestUnitAutomationTerraformScheduleLifecycle(t *testing.T) {
	f := newAutomationFixture(t)
	config := func(schedule string) string { return f.config(schedule) }
	manual := config("")
	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             f.checkDestroyed,
		Steps: []resource.TestStep{
			{Config: manual, Check: resource.ComposeTestCheckFunc(
				resource.TestCheckNoResourceAttr("grafana_assistant_automation.test", "schedule_timezone"),
				f.checkWrites(1),
			)},
			{Config: manual, PlanOnly: true, Check: f.checkWrites(1)},
			{ResourceName: "grafana_assistant_automation.test", ImportState: true, ImportStateVerify: true},
			{Config: config(`schedule_cron = "0 9 * * *"`), Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("grafana_assistant_automation.test", "schedule_timezone", "UTC"),
				f.checkWrites(2),
			)},
			{Config: config("schedule_cron = \"0 9 * * *\"\nschedule_timezone = \"Europe/Paris\""), Check: resource.TestCheckResourceAttr("grafana_assistant_automation.test", "schedule_timezone", "Europe/Paris")},
			{Config: config(`schedule_cron = "0 9 * * 1-5; 0 10 * * 0"`), Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("grafana_assistant_automation.test", "schedule_cron", "0 9 * * 1-5; 0 10 * * 0"),
				resource.TestCheckResourceAttr("grafana_assistant_automation.test", "schedule_timezone", "UTC"),
			)},
			{ResourceName: "grafana_assistant_automation.test", ImportState: true, ImportStateVerify: true},
			{Config: manual, Check: resource.ComposeTestCheckFunc(
				resource.TestCheckNoResourceAttr("grafana_assistant_automation.test", "schedule_cron"),
				resource.TestCheckNoResourceAttr("grafana_assistant_automation.test", "schedule_timezone"),
				f.checkWrites(5),
			)},
			{Config: manual, PlanOnly: true, Check: f.checkWrites(5)},
		},
	})
}

func TestUnitAutomationTerraformNotificationDefaults(t *testing.T) {
	for _, channel := range []string{"email", "slack"} {
		t.Run(channel, func(t *testing.T) {
			f := newAutomationFixture(t)
			destination := `addresses = ["terraform-test@example.invalid"]`
			if channel == "slack" {
				destination = `target {
type = "channel"
channel_id = "C_TEST_FIXTURE"
}`
			}
			config := func(outcomes string) string {
				return f.config(fmt.Sprintf(`notifications {
  %s {
    enabled = true
    %s
    %s
  }
}`, channel, destination, outcomes))
			}
			defaults := config("")
			attribute := "notifications." + channel + ".notify_on"
			checkDefaults := resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("grafana_assistant_automation.test", attribute+".#", "2"),
				resource.TestCheckResourceAttr("grafana_assistant_automation.test", attribute+".0", "completed"),
				resource.TestCheckResourceAttr("grafana_assistant_automation.test", attribute+".1", "failed"),
			)
			resource.UnitTest(t, resource.TestCase{
				ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
				CheckDestroy:             f.checkDestroyed,
				Steps: []resource.TestStep{
					{Config: defaults, Check: checkDefaults},
					{Config: defaults, PlanOnly: true, Check: f.checkWrites(1)},
					{ResourceName: "grafana_assistant_automation.test", ImportState: true, ImportStateVerify: true},
					{Config: config(`notify_on = ["needs_approval"]`), Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr("grafana_assistant_automation.test", attribute+".#", "1"),
						resource.TestCheckResourceAttr("grafana_assistant_automation.test", attribute+".0", "needs_approval"),
					)},
					{Config: defaults, Check: checkDefaults},
					{Config: defaults, PlanOnly: true, Check: f.checkWrites(3)},
					{Config: f.config(""), Check: resource.TestCheckNoResourceAttr("grafana_assistant_automation.test", attribute+".#")},
					{Config: f.config(""), PlanOnly: true, Check: f.checkWrites(4)},
				},
			})
		})
	}
}

func TestUnitAutomationTerraformTimezoneRequiresSchedule(t *testing.T) {
	f := newAutomationFixture(t)
	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{{
			Config:      f.config(`schedule_timezone = "UTC"`),
			PlanOnly:    true,
			ExpectError: regexp.MustCompile("schedule_timezone requires schedule_cron"),
		}},
	})
	if err := f.checkWrites(0)(nil); err != nil {
		t.Fatal(err)
	}
}

func TestUnitAutomationTerraformConfigurationLifecycle(t *testing.T) {
	f := newAutomationFixture(t)
	full := `
scope = "tenant"
execution_mode = "service_account"
labels = { env = "test", owner = "platform" }
schedule_cron = "0 9 * * 1-5;0 10 * * 0"
schedule_ends_at = "2099-01-01T01:00:00+01:00"
context_items = "[ {\"type\": \"test-fixture\"} ]"
notifications {
  email {
    enabled = false
    addresses = ["terraform-test@example.invalid"]
  }
}
`
	cleared := f.config(`scope = "tenant"`)
	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             f.checkDestroyed,
		Steps: []resource.TestStep{
			{Config: f.config(full), Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("grafana_assistant_automation.test", "execution_mode", "service_account"),
				resource.TestCheckResourceAttr("grafana_assistant_automation.test", "labels.env", "test"),
				resource.TestCheckResourceAttr("grafana_assistant_automation.test", "schedule_ends_at", "2099-01-01T01:00:00+01:00"),
			)},
			{Config: f.config(full), PlanOnly: true, Check: f.checkWrites(1)},
			// Import uses the API's canonical UTC/JSON/cron spelling, while apply
			// retains semantically equivalent configured spelling.
			{ResourceName: "grafana_assistant_automation.test", ImportState: true, ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{"schedule_ends_at", "schedule_cron", "context_items"}},
			{PreConfig: func() {
				f.mu.Lock()
				defer f.mu.Unlock()
				f.data["notifications"].(map[string]any)["webhook"] = map[string]any{"enabled": true, "urlDisplay": "https://example.invalid/hook"}
				f.data["usageEstimate"] = map[string]any{"estimatedTokensPerRun": 100}
			}, Config: strings.Replace(f.config(full), "terraform-contract-test", "renamed", 1), Check: f.checkPatchKeys("name")},
			{Config: cleared, Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("grafana_assistant_automation.test", "execution_mode", "user"),
				resource.TestCheckNoResourceAttr("grafana_assistant_automation.test", "schedule_ends_at"),
				resource.TestCheckNoResourceAttr("grafana_assistant_automation.test", "labels.%"),
				resource.TestCheckNoResourceAttr("grafana_assistant_automation.test", "context_items"),
				func(_ *terraform.State) error {
					f.mu.Lock()
					defer f.mu.Unlock()
					providers := f.data["notifications"].(map[string]any)
					if providers["webhook"] == nil || providers["email"] != nil {
						return fmt.Errorf("notification removal did not preserve the unmodeled webhook: %v", providers)
					}
					return nil
				},
			)},
			{Config: cleared, PlanOnly: true},
			{Config: f.config(`scope = "tenant"
labels = {}`), Check: resource.TestCheckResourceAttr("grafana_assistant_automation.test", "labels.%", "0")},
		},
	})
}

func TestUnitAutomationTerraformInvalidConfiguration(t *testing.T) {
	for _, tc := range []struct{ name, config, message string }{
		{"service_account_private", `execution_mode = "service_account"`, "requires scope = tenant"},
		{"deadline_without_schedule", `schedule_ends_at = "2099-01-01T00:00:00Z"`, "schedule_ends_at requires schedule_cron"},
		{"invalid_deadline", "schedule_cron = \"0 9 * * *\"\nschedule_ends_at = \"tomorrow\"", "RFC3339 timestamp"},
		{"fractional_deadline", "schedule_cron = \"0 9 * * *\"\nschedule_ends_at = \"2099-01-01T00:00:00.123Z\"", "second precision"},
		{"empty_schedule", `schedule_cron = ""`, "Empty schedule"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newAutomationFixture(t)
			resource.UnitTest(t, resource.TestCase{
				ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
				Steps:                    []resource.TestStep{{Config: f.config(tc.config), PlanOnly: true, ExpectError: regexp.MustCompile(tc.message)}},
			})
			if err := f.checkWrites(0)(nil); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestUnitAutomationTerraformPermissionDenied(t *testing.T) {
	f := newAutomationFixture(t)
	f.denyServiceAccount = true
	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{{
			Config:      f.config("scope = \"tenant\"\nexecution_mode = \"service_account\""),
			ExpectError: regexp.MustCompile(`Grafana organization Admin role is\s+required`),
		}},
	})
	if err := f.checkWrites(0)(nil); err != nil {
		t.Fatal(err)
	}
}

func TestUnitAutomationTerraformTriggerLifecycle(t *testing.T) {
	f := newAutomationFixture(t)
	const firstID = "11111111-1111-4111-8111-111111111111"
	const secondID = "22222222-2222-4222-8222-222222222222"
	first := fmt.Sprintf(`
scope = "tenant"
execution_mode = "service_account"
triggers = {
  %q = {
    type = "github_event"
    config = jsonencode({repository = "grafana/test", event = "push"})
  }
}`, firstID)
	both := strings.Replace(first, "triggers = {", fmt.Sprintf(`triggers = {
  %q = {
    type = "watcher_finding"
    config = jsonencode({watcherId = "33333333-3333-4333-8333-333333333333"})
  }`, secondID), 1)
	updated := strings.Replace(both, "grafana/test", "grafana/updated", 1)
	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             f.checkDestroyed,
		Steps: []resource.TestStep{
			{Config: f.config(first), Check: resource.TestCheckResourceAttr("grafana_assistant_automation.test", "triggers."+firstID+".type", "github_event")},
			{Config: f.config(first), PlanOnly: true, Check: f.checkWrites(1)},
			{Config: f.config(both), Check: resource.ComposeTestCheckFunc(f.checkPatchKeys("triggers"), resource.TestCheckResourceAttr("grafana_assistant_automation.test", "triggers.%", "2"))},
			{Config: f.config(updated), Check: f.checkPatchKeys("triggers")},
			{ResourceName: "grafana_assistant_automation.test", ImportState: true, ImportStateVerify: true, ImportStateVerifyIgnore: []string{"triggers." + firstID + ".config", "triggers." + secondID + ".config"},
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if len(states) != 1 || states[0].Attributes["triggers."+firstID+".type"] != "github_event" || states[0].Attributes["triggers."+secondID+".type"] != "watcher_finding" {
						return fmt.Errorf("import did not retain both trigger UUIDs: %v", states)
					}
					return nil
				}},
			{Config: strings.Replace(f.config(updated), "terraform-contract-test", "renamed", 1), Check: f.checkPatchKeys("name")},
			{Config: f.config(first), Check: resource.TestCheckResourceAttr("grafana_assistant_automation.test", "triggers.%", "1")},
			{Config: f.config("scope = \"tenant\"\nexecution_mode = \"service_account\"\ntriggers = {}"), Check: resource.TestCheckResourceAttr("grafana_assistant_automation.test", "triggers.%", "0")},
			{Config: f.config("scope = \"tenant\"\nexecution_mode = \"service_account\""), Check: resource.TestCheckNoResourceAttr("grafana_assistant_automation.test", "triggers.%")},
		},
	})
}

func TestUnitAutomationTerraformInvalidTriggers(t *testing.T) {
	for _, tc := range []struct{ name, config, message string }{
		{"identity", `triggers = { invalid = {type = "slack_message", config = "{}"} }`, "Invalid trigger identity"},
		{"json", `triggers = { "11111111-1111-4111-8111-111111111111" = {type = "slack_message", config = "[]"} }`, "config must be a JSON object"},
		{"mode", `triggers = { "11111111-1111-4111-8111-111111111111" = {type = "github_event", config = "{}"} }`, "Service-account execution required"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newAutomationFixture(t)
			resource.UnitTest(t, resource.TestCase{
				ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
				Steps:                    []resource.TestStep{{Config: f.config(tc.config), PlanOnly: true, ExpectError: regexp.MustCompile(tc.message)}},
			})
		})
	}
}

func TestUnitAutomationTerraformUnknownTriggerConfiguration(t *testing.T) {
	f := newAutomationFixture(t)
	config := f.config(`
scope = "tenant"
execution_mode = "service_account"
triggers = {"11111111-1111-4111-8111-111111111111" = terraform_data.trigger.output}
`) + `
resource "terraform_data" "trigger" {
  input = {type = "watcher_finding", config = jsonencode({watcherId = "33333333-3333-4333-8333-333333333333"})}
}
`
	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             f.checkDestroyed,
		Steps: []resource.TestStep{
			{Config: config, Check: resource.TestCheckResourceAttr("grafana_assistant_automation.test", "triggers.11111111-1111-4111-8111-111111111111.type", "watcher_finding")},
			{Config: config, PlanOnly: true, Check: f.checkWrites(1)},
		},
	})
}

func (f *automationFixture) checkPatchKeys(keys ...string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		f.mu.Lock()
		defer f.mu.Unlock()
		if len(f.patches) == 0 {
			return fmt.Errorf("no patch received")
		}
		last := f.patches[len(f.patches)-1]
		if len(last) != len(keys) {
			return fmt.Errorf("expected only %v, got patch %v", keys, last)
		}
		for _, key := range keys {
			if _, ok := last[key]; !ok {
				return fmt.Errorf("patch missing %s", key)
			}
		}
		return nil
	}
}

type automationFixture struct {
	mu                 sync.Mutex
	server             *httptest.Server
	data               map[string]any
	writes             int
	patches            []map[string]any
	denyServiceAccount bool
}

func newAutomationFixture(t *testing.T) *automationFixture {
	t.Helper()
	f := &automationFixture{}
	f.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		const prefix = "/api/plugins/grafana-assistant-app/resources/api/v1/automations"
		if r.URL.Path != prefix && r.URL.Path != prefix+"/automation-id" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodPost, http.MethodPut:
			var patch map[string]any
			if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
				t.Error(err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if f.denyServiceAccount && patch["executionMode"] == "service_account" {
				http.Error(w, "Grafana organization Admin role is required", http.StatusForbidden)
				return
			}
			if f.data == nil {
				f.data = map[string]any{"id": "automation-id"}
			}
			for key, value := range patch {
				// A JSON null pointer field is absent after the backend decodes
				// AutomationUpdate; omitted context is not returned as literal null.
				if value == nil {
					continue
				}
				if key == "triggers" {
					applyFixtureTriggerDefaults(value.([]any))
				}
				if key == "notifications" {
					providers, _ := f.data[key].(map[string]any)
					if providers == nil {
						providers = map[string]any{}
					}
					for name, notification := range value.(map[string]any) {
						if notification == nil {
							delete(providers, name)
						} else {
							providers[name] = notification
						}
					}
					f.data[key] = providers
					continue
				}
				f.data[key] = value
			}
			if f.data["enabled"] != false {
				t.Error("configuration tests must keep automations disabled")
			}
			if cron, _ := f.data["scheduleCron"].(string); cron != "" {
				tz, _ := f.data["scheduleTimezone"].(string)
				if tz == "" {
					tz = "UTC"
				}
				parts := strings.Split(cron, ";")
				for i := range parts {
					parts[i] = strings.TrimSpace(parts[i])
				}
				schedule := map[string]any{"cron": strings.Join(parts, "; "), "timezone": tz}
				if deadline, _ := f.data["scheduleEndsAt"].(string); deadline != "" {
					parsed, err := time.Parse(time.RFC3339, deadline)
					if err != nil {
						t.Error(err)
					} else {
						schedule["endsAt"] = parsed.UTC().Format(time.RFC3339)
					}
				}
				f.data["schedule"] = schedule
			} else {
				delete(f.data, "schedule")
				delete(f.data, "scheduleEndsAt")
			}
			f.writes++
			f.patches = append(f.patches, patch)
		case http.MethodDelete:
			f.data = nil
			w.WriteHeader(http.StatusNoContent)
			return
		case http.MethodGet:
			if f.data == nil {
				w.WriteHeader(http.StatusNotFound)
				return
			}
		default:
			t.Errorf("unexpected method %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if err := json.NewEncoder(w).Encode(map[string]any{"data": f.data}); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(f.server.Close)
	return f
}

func applyFixtureTriggerDefaults(triggers []any) {
	for _, item := range triggers {
		trigger := item.(map[string]any)
		config := trigger["config"].(map[string]any)
		switch trigger["type"] {
		case "github_event":
			if config["event"] == "push" && config["actions"] == nil {
				config["actions"] = []any{"pushed"}
			}
		case "watcher_finding":
			if config["severity"] == nil {
				config["severity"] = "critical"
			}
		}
	}
}

func (f *automationFixture) config(extra string) string {
	return fmt.Sprintf(`
provider "grafana" {
  url = %q
  auth = "fixture-token"
}
resource "grafana_assistant_automation" "test" {
  name = "terraform-contract-test"
  prompt = "Configuration test only."
  enabled = false
  %s
}
`, f.server.URL, strings.TrimSpace(extra))
}

func (f *automationFixture) checkWrites(want int) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		f.mu.Lock()
		defer f.mu.Unlock()
		if f.writes != want {
			return fmt.Errorf("got %d configuration writes, want %d", f.writes, want)
		}
		return nil
	}
}

func (f *automationFixture) checkDestroyed(_ *terraform.State) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.data != nil {
		return fmt.Errorf("automation still exists after destroy")
	}
	return nil
}
