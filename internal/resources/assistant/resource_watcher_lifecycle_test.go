package assistant_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

type watcherFixture struct {
	holdCalibration                                 bool
	loseUpdateResponse                              bool
	lastUpdate                                      map[string]any
	updateAttempts, updateReplays                   int
	mu                                              sync.Mutex
	server                                          *httptest.Server
	agent                                           map[string]any
	calibration                                     map[string]any
	creates, patches, calibrations, starts, deletes int
	t                                               *testing.T
}

func newWatcherFixture(t *testing.T) *watcherFixture {
	f := &watcherFixture{t: t}
	f.server = httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.server.Close)
	return f
}
func (f *watcherFixture) serve(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	path := strings.TrimPrefix(r.URL.Path, "/api/plugins/grafana-assistant-app/resources/api/v1/watcher-agents")
	var body map[string]any
	if r.Body != nil && (r.Method == "POST" || r.Method == "PUT") {
		_ = json.NewDecoder(r.Body).Decode(&body)
	}
	var result any
	switch {
	case r.Method == "POST" && path == "":
		f.creates++
		if _, ok := body["teamUid"]; ok {
			f.t.Error("create sent team assignment")
		}
		if _, ok := body["requestId"]; ok {
			f.t.Error("create sent unsupported request ID")
		}
		f.agent = body
		f.agent["id"] = "watcher-fixture"
		f.agent["status"] = "draft"
		f.agent["definitionVersion"] = float64(1)
		f.calibration = map[string]any{"status": "idle", "startWhenReady": false}
		result = f.agent
	case f.agent == nil:
		w.WriteHeader(404)
		return
	case r.Method == "GET" && path == "/watcher-fixture":
		result = f.agent
	case r.Method == "PUT" && path == "/watcher-fixture":
		var ok bool
		result, ok = f.update(w, body)
		if !ok {
			return
		}

	case r.Method == "GET" && path == "/watcher-fixture/initial-calibration":
		result = f.calibration
	case r.Method == "POST" && path == "/watcher-fixture/initial-calibration":
		f.calibrations++
		if body["unattended"] != true {
			f.t.Error("calibration was not unattended")
		}
		f.calibration = map[string]any{"status": "completed", "unattended": true, "startWhenReady": body["startWhenReady"], "startedAt": time.Now().UTC().Format(time.RFC3339Nano)}
		f.agent["status"] = "ready"
		if body["startWhenReady"] == true {
			f.agent["status"] = "running"
		}
		f.agent["calibratedAt"] = "2026-11-05T12:00:00Z"
		f.agent["calibrationContext"] = "server-generated guidance"
		f.agent["queries"] = []any{map[string]any{"id": "learned-query", "type": "alerts", "expr": "alertname=\"Example\""}}
		f.agent["learningEpoch"] = float64(4)
		f.agent["definitionVersion"] = float64(2)
		if f.holdCalibration {
			f.calibration["status"] = "running"
			f.agent["status"] = "calibrating"
			delete(f.agent, "calibratedAt")
		}
		result = f.calibration
	case r.Method == "POST" && path == "/watcher-fixture/start":
		f.starts++
		f.agent["status"] = "running"
		result = f.agent
	case r.Method == "POST" && path == "/watcher-fixture/pause":
		f.agent["status"] = "paused"
		result = f.agent
	case r.Method == "DELETE" && path == "/watcher-fixture":
		f.deletes++
		f.agent = nil
		result = map[string]any{}
	default:
		f.t.Errorf("unexpected request %s %s", r.Method, path)
		w.WriteHeader(400)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": result})
}
func (f *watcherFixture) update(w http.ResponseWriter, body map[string]any) (any, bool) {
	f.updateAttempts++
	if _, ok := body["teamUid"]; ok {
		f.t.Error("update changed team assignment")
	}
	if f.loseUpdateResponse && reflect.DeepEqual(body, f.lastUpdate) {
		f.updateReplays++
		return f.agent, true
	}
	if _, ok := body["queries"]; ok {
		f.t.Error("provider rewrote generated queries")
	}
	if _, ok := body["calibrationContext"]; ok {
		f.t.Error("provider rewrote calibration guidance")
	}
	if body["expectedDefinitionVersion"] != f.agent["definitionVersion"] {
		w.WriteHeader(409)
		return nil, false
	}
	f.patches++
	for k, v := range body {
		if k != "expectedDefinitionVersion" {
			f.agent[k] = v
		}
	}
	f.agent["definitionVersion"] = f.agent["definitionVersion"].(float64) + 1
	if f.loseUpdateResponse {
		f.lastUpdate = body
		f.agent["status"] = "calibrating"
		w.WriteHeader(http.StatusBadGateway)
		return nil, false
	}
	return f.agent, true
}

func (f *watcherFixture) config(name, extra, setup string) string {
	return fmt.Sprintf(`provider "grafana" {
 url = %q
 auth = "fixture-token"
 retries = 2
 retry_wait = 1
}
resource "grafana_assistant_watcher" "test" {
 name = %q
 prompt = "Monitor managed alerts"
 %s
}
%s`, f.server.URL, name, extra, setup)
}
func (f *watcherFixture) preserved(s *terraform.State) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.creates != 1 || f.calibrations != 1 {
		return fmt.Errorf("creates=%d calibrations=%d", f.creates, f.calibrations)
	}
	if f.agent["learningEpoch"] != float64(4) {
		return fmt.Errorf("learning epoch was lost")
	}
	queries, ok := f.agent["queries"].([]any)
	if !ok || queries[0].(map[string]any)["id"] != "learned-query" {
		return fmt.Errorf("query identity lost")
	}
	return nil
}
func TestUnitWatcherTerraformLifecycle(t *testing.T) {
	f := newWatcherFixture(t)
	setup := `resource "grafana_assistant_watcher_calibration" "test" {
 watcher_id = grafana_assistant_watcher.test.id
 started = true
}`
	actions := `actions {
 investigation {
 enabled = true
 team_names = ["Platform"]
 }
}`
	initial := f.config("original", actions, setup)
	renamed := f.config("renamed", actions, setup)
	removed := f.config("renamed", "", setup)
	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy: func(*terraform.State) error {
			f.mu.Lock()
			defer f.mu.Unlock()
			if f.agent != nil {
				return fmt.Errorf("watcher leaked")
			}
			return nil
		},
		Steps: []resource.TestStep{
			{Config: initial, Check: resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr("grafana_assistant_watcher_calibration.test", "status", "completed"), f.preserved)},
			{Config: initial, PlanOnly: true},
			{ResourceName: "grafana_assistant_watcher.test", ImportState: true, ImportStateVerify: true},
			{ResourceName: "grafana_assistant_watcher_calibration.test", ImportState: true, ImportStateVerify: true},
			{Config: renamed, Check: f.preserved},
			{Config: removed, Check: f.preserved},
			{Config: removed, PlanOnly: true, PreConfig: func() { f.mu.Lock(); defer f.mu.Unlock(); f.agent["status"] = "paused" }},
			{Config: f.config("metadata-after-pause", "", setup), Check: func(s *terraform.State) error {
				f.mu.Lock()
				defer f.mu.Unlock()
				if f.starts != 0 || f.agent["status"] != "paused" {
					return fmt.Errorf("metadata edit resumed paused watcher")
				}
				return nil
			}},
		},
	})
}

func TestUnitWatcherTerraformTimeoutRetainsParent(t *testing.T) {
	f := newWatcherFixture(t)
	f.holdCalibration = true
	config := f.config("timeout", "", `resource "grafana_assistant_watcher_calibration" "test" {
 watcher_id = grafana_assistant_watcher.test.id
 timeout_seconds = 1
}`)
	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config, ExpectError: regexp.MustCompile("Watcher setup did not complete")},
			{Config: config, PreConfig: func() {
				f.mu.Lock()
				defer f.mu.Unlock()
				if f.agent == nil {
					t.Fatal("parent disappeared after timeout")
				}
				f.calibration["status"] = "completed"
				f.agent["status"] = "ready"
				f.agent["calibratedAt"] = "2026-11-05T12:00:00Z"
			}, Check: resource.ComposeTestCheckFunc(f.preserved, func(*terraform.State) error {
				f.mu.Lock()
				defer f.mu.Unlock()
				if f.deletes != 0 {
					return fmt.Errorf("setup retry deleted parent")
				}
				return nil
			})},
			{Config: config, PlanOnly: true},
		},
	})
}

func TestUnitWatcherTerraformScopeReplacement(t *testing.T) {
	f := newWatcherFixture(t)
	initial := f.config("scope", "", "")
	changed := strings.Replace(initial, "Monitor managed alerts", "Monitor different alerts", 1)
	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{Config: initial},
			{Config: changed, Check: func(*terraform.State) error {
				f.mu.Lock()
				defer f.mu.Unlock()
				if f.creates != 2 || f.deletes != 1 || f.agent["prompt"] != "Monitor different alerts" {
					return fmt.Errorf("scope edit did not replace the parent")
				}
				return nil
			}},
			{Config: changed, PlanOnly: true},
		},
	})
}

func TestUnitWatcherTerraformUnknownActionValues(t *testing.T) {
	f := newWatcherFixture(t)
	config := f.config("unknown", `actions {
 investigation {
 enabled = true
 team_names = [terraform_data.team.output]
 }
}`, `resource "terraform_data" "team" {
 input = "test-team"
}`)
	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config, Check: resource.TestCheckResourceAttr("grafana_assistant_watcher.test", "actions.investigation.team_names.0", "test-team")},
			{Config: config, PlanOnly: true},
		},
	})
}

func TestUnitWatcherTerraformEmptyOptionalConfiguration(t *testing.T) {
	f := newWatcherFixture(t)
	config := f.config("empty", `description = ""
 datasource_uids = []
 labels = {}
 actions {}`, "")
	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps:                    []resource.TestStep{{Config: config}, {Config: config, PlanOnly: true}, {Config: f.config("removed", "", "")}, {Config: f.config("removed", "", ""), PlanOnly: true}},
	})
}

func TestUnitWatcherTerraformInvestigationTeams(t *testing.T) {
	for _, tc := range []struct{ name, settings, want string }{
		{"omitted", "enabled = true", "Missing investigation teams"},
		{"empty", "enabled = true\nteam_names = []", "Missing investigation teams"},
		{"blank", "enabled = true\nteam_names = [\" \"]", "Invalid investigation team"},
		{"mixed", "enabled = true\nteam_names = [\"Platform\", \"\"]", "Invalid investigation team"},
		{"null", "enabled = true\nteam_names = [null]", "Invalid investigation team"},
		{"disabled", "enabled = false", ""},
		{"default disabled", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newWatcherFixture(t)
			step := resource.TestStep{Config: f.config("validation", "actions {\n investigation {\n"+tc.settings+"\n }\n}", "")}
			if tc.want != "" {
				step.ExpectError = regexp.MustCompile(tc.want)
			}
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories, Steps: []resource.TestStep{step}})
			if tc.want != "" && f.creates != 0 {
				t.Fatal("invalid investigation reached the API")
			}
		})
	}
}

func TestUnitWatcherTerraformCombinedUpdateRetryPreservesTeam(t *testing.T) {
	f := newWatcherFixture(t)
	initial := f.config("retry", "", "")
	changed := f.config("retry", `trigger_interval_seconds = 1800
 labels = {service = "checkout"}
 pause_at = "2099-01-01T00:00:00Z"
 actions {
  investigation {
   enabled = true
   team_names = ["Platform"]
  }
 }`, "")
	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{Config: initial},
			{ResourceName: "grafana_assistant_watcher.test", ImportState: true, ImportStateVerify: true, PreConfig: func() {
				f.mu.Lock()
				defer f.mu.Unlock()
				f.agent["team"] = map[string]any{"uid": "externally-assigned"}
			}},
			{Config: changed, PreConfig: func() { f.mu.Lock(); defer f.mu.Unlock(); f.loseUpdateResponse = true }, Check: func(*terraform.State) error {
				f.mu.Lock()
				defer f.mu.Unlock()
				if f.patches != 1 || f.updateAttempts != 2 || f.updateReplays != 1 {
					return fmt.Errorf("patches=%d attempts=%d replays=%d", f.patches, f.updateAttempts, f.updateReplays)
				}
				if f.agent["team"].(map[string]any)["uid"] != "externally-assigned" {
					return fmt.Errorf("import/update cleared external team")
				}
				return nil
			}},
			{Config: changed, PlanOnly: true},
		},
	})
}

func TestUnitWatcherTerraformResolvedInvalidTeams(t *testing.T) {
	for _, tc := range []struct{ name, value, want string }{
		{"empty", `[]`, "Missing investigation teams"},
		{"blank", `[" "]`, "Invalid investigation team"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newWatcherFixture(t)
			config := f.config("resolved-invalid", `actions {
 investigation {
  enabled = true
  team_names = terraform_data.teams.output
 }
}`, fmt.Sprintf(`resource "terraform_data" "teams" { input = %s }`, tc.value))
			resource.UnitTest(t, resource.TestCase{
				ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
				Steps:                    []resource.TestStep{{Config: config, ExpectError: regexp.MustCompile(tc.want)}},
			})
			if f.creates != 0 {
				t.Fatal("resolved invalid investigation reached the API")
			}
		})
	}
}
