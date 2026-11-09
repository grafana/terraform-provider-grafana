package assistant_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common/assistantapi"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
)

func TestAccAssistantAutomation_basic(t *testing.T) {
	testutils.CheckAssistantTestsEnabled(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "resources/grafana_assistant_automation/_acc_basic.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_assistant_automation.test", "id"),
					resource.TestCheckResourceAttr("grafana_assistant_automation.test", "name", "tf-acc-test-automation"),
					resource.TestCheckResourceAttr("grafana_assistant_automation.test", "scope", "tenant"),
					resource.TestCheckResourceAttr("grafana_assistant_automation.test", "enabled", "false"),
					resource.TestCheckResourceAttr("grafana_assistant_automation.test", "schedule_cron", "0 9 * * *"),
					resource.TestCheckResourceAttr("grafana_assistant_automation.test", "schedule_timezone", "UTC"),
					testutils.CheckLister("grafana_assistant_automation.test"),
				),
			},
			{
				ResourceName:      "grafana_assistant_automation.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_assistant_automation/_acc_basic.tf", map[string]string{
					"tf-acc-test-automation": "tf-acc-test-automation-updated",
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_assistant_automation.test", "name", "tf-acc-test-automation-updated"),
				),
			},
		},
	})
}

// Requires an organization Admin caller and bounded schedules enabled on the
// backend. Automations and email notifications remain disabled throughout.
func TestAccAssistantAutomation_configuration(t *testing.T) {
	testutils.CheckAssistantTestsEnabled(t)
	name := "tf-acc-automation-" + uuid.NewString()
	base := fmt.Sprintf(`resource "grafana_assistant_automation" "test" {
name = %q
prompt = "Terraform configuration test only."
scope = "tenant"
enabled = false
%%s
}`, name)
	manual := fmt.Sprintf(base, "")
	scheduled := fmt.Sprintf(base, fmt.Sprintf(`
execution_mode = "service_account"
labels = { env = "terraform-test" }
schedule_cron = "0 9 * * 1-5;0 10 * * 0"
schedule_ends_at = %q
notifications {
  email {
    enabled = false
    addresses = ["terraform-test@example.invalid"]
    notify_on = ["failed", "completed"]
  }
}
`, time.Now().UTC().Add(72*time.Hour).Truncate(time.Second).Format(time.RFC3339)))
	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             checkAssistantAutomationsDestroyed,
		Steps: []resource.TestStep{
			{Config: manual, Check: resource.TestCheckNoResourceAttr("grafana_assistant_automation.test", "schedule_timezone")},
			{ResourceName: "grafana_assistant_automation.test", ImportState: true, ImportStateVerify: true},
			{Config: scheduled, Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("grafana_assistant_automation.test", "execution_mode", "service_account"),
				resource.TestCheckResourceAttr("grafana_assistant_automation.test", "labels.env", "terraform-test"),
				resource.TestCheckResourceAttr("grafana_assistant_automation.test", "schedule_timezone", "UTC"),
			)},
			{Config: scheduled, PlanOnly: true},
			// Import has no configured spelling to preserve. Assert the API's
			// canonical values separately rather than ignoring their contents.
			{ResourceName: "grafana_assistant_automation.test", ImportState: true, ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{"schedule_cron", "notifications.email.notify_on"},
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if len(states) != 1 {
						return fmt.Errorf("expected one imported Automation, got %d", len(states))
					}
					attrs := states[0].Attributes
					if attrs["schedule_cron"] != "0 9 * * 1-5; 0 10 * * 0" || attrs["notifications.email.notify_on.0"] != "completed" || attrs["notifications.email.notify_on.1"] != "failed" {
						return fmt.Errorf("import did not retain canonical schedule and notification outcomes")
					}
					return nil
				}},
			{Config: strings.Replace(scheduled, name, name+"-renamed", 1)},
			{Config: manual, Check: resource.ComposeTestCheckFunc(
				resource.TestCheckNoResourceAttr("grafana_assistant_automation.test", "schedule_timezone"),
				resource.TestCheckNoResourceAttr("grafana_assistant_automation.test", "schedule_ends_at"),
				resource.TestCheckNoResourceAttr("grafana_assistant_automation.test", "labels.%"),
				resource.TestCheckNoResourceAttr("grafana_assistant_automation.test", "notifications.%"),
			)},
			{Config: manual, PlanOnly: true},
		},
	})
}

// Run separately with GRAFANA_AUTH set to a non-Admin caller. Never downgrade an
// existing account to set up this test; provision a disposable test identity.
func TestAccAssistantAutomation_serviceAccountDenied(t *testing.T) {
	testutils.CheckAssistantTestsEnabled(t)
	if !testutils.AccTestsEnabled("TF_ACC_ASSISTANT_NON_ADMIN") {
		t.Skip("TF_ACC_ASSISTANT_NON_ADMIN requires a non-Admin caller")
	}
	base := fmt.Sprintf(`resource "grafana_assistant_automation" "test" {
name = %q
prompt = "Terraform permission test only."
scope = "tenant"
enabled = false
execution_mode = %%q
}`, "tf-acc-denied-"+uuid.NewString())
	denied := regexp.MustCompile(`organization\s+Admin`)
	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             checkAssistantAutomationsDestroyed,
		Steps: []resource.TestStep{
			{Config: fmt.Sprintf(base, "service_account"), ExpectError: denied},
			{Config: fmt.Sprintf(base, "user"), Check: resource.TestCheckResourceAttr("grafana_assistant_automation.test", "execution_mode", "user")},
			{Config: fmt.Sprintf(base, "service_account"), ExpectError: denied},
			{Config: fmt.Sprintf(base, "user"), PlanOnly: true},
		},
	})
}

// Requires Watcher/Automation event triggers enabled. The source Watcher stays
// a draft; the Automation stays disabled. This test never emits an event.
func TestAccAssistantAutomation_watcherTrigger(t *testing.T) {
	testutils.CheckAssistantTestsEnabled(t)
	watcherID := createAutomationTriggerSource(t)
	id := uuid.NewString()
	base := fmt.Sprintf(`resource "grafana_assistant_automation" "test" {
name = %q
prompt = "Terraform trigger configuration test only."
scope = "tenant"
execution_mode = "service_account"
enabled = false
%%s
}`, "tf-acc-trigger-"+uuid.NewString())
	withTrigger := fmt.Sprintf(base, fmt.Sprintf(`triggers = {%q = {type = "watcher_finding", config = jsonencode({watcherId = %q, severity = "critical"})}}`, id, watcherID))
	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             checkAssistantAutomationsDestroyed,
		Steps: []resource.TestStep{
			{Config: withTrigger, Check: resource.TestCheckResourceAttr("grafana_assistant_automation.test", "triggers."+id+".type", "watcher_finding")},
			{Config: withTrigger, PlanOnly: true},
			{ResourceName: "grafana_assistant_automation.test", ImportState: true, ImportStateVerify: true, ImportStateVerifyIgnore: []string{"triggers." + id + ".config"}},
			{Config: fmt.Sprintf(base, "triggers = {}"), Check: resource.TestCheckResourceAttr("grafana_assistant_automation.test", "triggers.%", "0")},
		},
	})
}

func checkAssistantAutomationsDestroyed(state *terraform.State) error {
	client := testutils.Provider.Meta().(*common.Client).AssistantAPIClient
	for _, item := range state.RootModule().Resources {
		if item.Type != "grafana_assistant_automation" {
			continue
		}
		_, err := client.GetAutomation(context.Background(), item.Primary.ID)
		if !errors.Is(err, assistantapi.ErrNotFound) {
			return fmt.Errorf("automation %s still exists or cleanup could not be verified: %v", item.Primary.ID, err)
		}
	}
	return nil
}

// This source is a backend fixture, not a Terraform Watcher resource. Keep its
// minimal create/delete contract in test code so Automations can ship alone.
func createAutomationTriggerSource(t *testing.T) string {
	t.Helper()
	const endpoint = "/api/plugins/grafana-assistant-app/resources/api/v1/watcher-agents"
	body := fmt.Sprintf(`{"name":%q,"prompt":"Terraform trigger configuration test only.","triggerIntervalSeconds":600}`, "tf-acc-trigger-source-"+uuid.NewString())
	status, raw := automationTriggerFixtureRequest(t, http.MethodPost, endpoint, body)
	if status < 200 || status >= 300 {
		t.Fatalf("create draft trigger source: HTTP %d: %s", status, raw)
	}
	var created struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &created); err != nil || created.Data.ID == "" {
		t.Fatalf("create draft trigger source returned no usable ID: %s", raw)
	}
	path := endpoint + "/" + url.PathEscape(created.Data.ID)
	t.Cleanup(func() {
		status, raw := automationTriggerFixtureRequest(t, http.MethodDelete, path, "")
		if (status < 200 || status >= 300) && status != http.StatusNotFound {
			t.Errorf("delete draft trigger source: HTTP %d: %s", status, raw)
		}
		status, _ = automationTriggerFixtureRequest(t, http.MethodGet, path, "")
		if status != http.StatusNotFound {
			t.Errorf("draft trigger source cleanup not verified: HTTP %d", status)
		}
	})
	return created.Data.ID
}

func automationTriggerFixtureRequest(t *testing.T, method, path, body string) (int, []byte) {
	t.Helper()
	client := testutils.Provider.Meta().(*common.Client)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(client.GrafanaAPIURL, "/")+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	// The configured transport supplies authentication, TLS and provider headers.
	resp, err := client.GrafanaHTTPClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, raw
}
