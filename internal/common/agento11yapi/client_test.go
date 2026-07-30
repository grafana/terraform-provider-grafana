package agento11yapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/util"
)

// testClient spins up an httptest server with the given handler and returns a
// Client pointed at it. The server is closed via t.Cleanup.
func testClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := NewClient(server.URL, nil, "test-token", server.Client(), "test-agent", nil)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return client
}

func writeJSON(t *testing.T, w http.ResponseWriter, v any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}

func TestClient_EvaluatorCRUD(t *testing.T) {
	t.Parallel()

	const id = "toxicity"
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == pathPrefix+"/evaluators":
			var body EvaluatorWrite
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "bad body", http.StatusBadRequest)
				return
			}
			if body.EvaluatorID != id || body.Kind != "regex" {
				http.Error(w, "unexpected body", http.StatusBadRequest)
				return
			}
			writeJSON(t, w, Evaluator{EvaluatorID: id, Version: "1", Kind: "regex", Config: json.RawMessage(`{"pattern":"foo"}`)})
		case r.Method == http.MethodGet && r.URL.Path == pathPrefix+"/evaluators/"+id:
			writeJSON(t, w, Evaluator{EvaluatorID: id, Version: "1", Kind: "regex"})
		case r.Method == http.MethodDelete && r.URL.Path == pathPrefix+"/evaluators/"+id:
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	})

	created, err := client.CreateEvaluator(context.Background(), EvaluatorWrite{
		EvaluatorID: id,
		Version:     "1",
		Kind:        "regex",
		Config:      json.RawMessage(`{"pattern":"foo"}`),
		OutputKeys:  []OutputKey{{Key: "match", Type: "bool"}},
	})
	if err != nil {
		t.Fatalf("CreateEvaluator: %v", err)
	}
	if created.EvaluatorID != id {
		t.Fatalf("unexpected id: %s", created.EvaluatorID)
	}

	got, err := client.GetEvaluator(context.Background(), id)
	if err != nil {
		t.Fatalf("GetEvaluator: %v", err)
	}
	if got.Kind != "regex" {
		t.Fatalf("unexpected kind: %s", got.Kind)
	}

	if err := client.DeleteEvaluator(context.Background(), id); err != nil {
		t.Fatalf("DeleteEvaluator: %v", err)
	}
}

func TestClient_RuleCRUD(t *testing.T) {
	t.Parallel()

	const id = "toxicity-rule"
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == pathPrefix+"/rules":
			writeJSON(t, w, Rule{RuleID: id, Enabled: true, Selector: "user_visible_turn", SampleRate: 0.5, EvaluatorIDs: []string{"toxicity"}})
		case r.Method == http.MethodGet && r.URL.Path == pathPrefix+"/rules/"+id:
			writeJSON(t, w, Rule{RuleID: id, Enabled: true, Selector: "user_visible_turn", SampleRate: 0.5, EvaluatorIDs: []string{"toxicity"}})
		case r.Method == http.MethodPatch && r.URL.Path == pathPrefix+"/rules/"+id:
			var body RulePatch
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "bad body", http.StatusBadRequest)
				return
			}
			enabled := body.Enabled != nil && *body.Enabled
			writeJSON(t, w, Rule{RuleID: id, Enabled: enabled, Selector: "user_visible_turn", SampleRate: 0.5, EvaluatorIDs: []string{"toxicity"}})
		case r.Method == http.MethodDelete && r.URL.Path == pathPrefix+"/rules/"+id:
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	})

	created, err := client.CreateRule(context.Background(), RuleWrite{
		RuleID:       id,
		SampleRate:   util.Ptr(0.5),
		EvaluatorIDs: []string{"toxicity"},
	})
	if err != nil {
		t.Fatalf("CreateRule: %v", err)
	}
	if created.RuleID != id {
		t.Fatalf("unexpected id: %s", created.RuleID)
	}

	if _, err := client.GetRule(context.Background(), id); err != nil {
		t.Fatalf("GetRule: %v", err)
	}

	updated, err := client.UpdateRule(context.Background(), id, RulePatch{Enabled: util.Ptr(false)})
	if err != nil {
		t.Fatalf("UpdateRule: %v", err)
	}
	if updated.Enabled {
		t.Fatalf("expected rule disabled after update")
	}

	if err := client.DeleteRule(context.Background(), id); err != nil {
		t.Fatalf("DeleteRule: %v", err)
	}
}

func TestClient_RuleActionCRUD(t *testing.T) {
	t.Parallel()

	const ruleID = "toxicity-rule"
	const actionID = "ra_abc"
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == pathPrefix+"/rules/"+ruleID+"/actions":
			var body RuleActionCreate
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "bad body", http.StatusBadRequest)
				return
			}
			if body.ActionConfig.Kind != "add_to_collection" || len(body.ActionConfig.CollectionIDs) != 1 {
				http.Error(w, "unexpected action config", http.StatusBadRequest)
				return
			}
			writeJSON(t, w, RuleAction{ActionID: actionID, RuleID: ruleID, Condition: body.Condition, ActionConfig: body.ActionConfig, Enabled: true})
		case r.Method == http.MethodGet && r.URL.Path == pathPrefix+"/rules/"+ruleID+"/actions/"+actionID:
			writeJSON(t, w, RuleAction{ActionID: actionID, RuleID: ruleID, Condition: RuleActionCondition{Kind: "all_evaluators_fail"}, ActionConfig: RuleActionConfig{Kind: "add_to_collection", CollectionIDs: []string{"col1"}}, Enabled: true})
		case r.Method == http.MethodPatch && r.URL.Path == pathPrefix+"/rules/"+ruleID+"/actions/"+actionID:
			writeJSON(t, w, RuleAction{ActionID: actionID, RuleID: ruleID, Condition: RuleActionCondition{Kind: "all_evaluators_fail"}, ActionConfig: RuleActionConfig{Kind: "add_to_collection", CollectionIDs: []string{"col1"}}, Enabled: false})
		case r.Method == http.MethodDelete && r.URL.Path == pathPrefix+"/rules/"+ruleID+"/actions/"+actionID:
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	})

	created, err := client.CreateRuleAction(context.Background(), ruleID, RuleActionCreate{
		Condition:    RuleActionCondition{Kind: "all_evaluators_fail"},
		ActionConfig: RuleActionConfig{Kind: "add_to_collection", CollectionIDs: []string{"col1"}},
	})
	if err != nil {
		t.Fatalf("CreateRuleAction: %v", err)
	}
	if created.ActionID != actionID {
		t.Fatalf("unexpected action id: %s", created.ActionID)
	}

	if _, err := client.GetRuleAction(context.Background(), ruleID, actionID); err != nil {
		t.Fatalf("GetRuleAction: %v", err)
	}

	updated, err := client.UpdateRuleAction(context.Background(), ruleID, actionID, RuleActionUpdate{Enabled: util.Ptr(false)})
	if err != nil {
		t.Fatalf("UpdateRuleAction: %v", err)
	}
	if updated.Enabled {
		t.Fatalf("expected action disabled after update")
	}

	if err := client.DeleteRuleAction(context.Background(), ruleID, actionID); err != nil {
		t.Fatalf("DeleteRuleAction: %v", err)
	}
}

func TestClient_HookRuleCRUD(t *testing.T) {
	t.Parallel()

	const id = "block-secrets"
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == pathPrefix+"/hook-rules":
			writeJSON(t, w, HookRule{RuleID: id, Enabled: true, Phase: "preflight", Selector: "all", ActionOnFail: "deny", ShortCircuit: true, ToolFilter: &HookToolFilter{BlockedNames: []string{"delete_*"}}})
		case r.Method == http.MethodGet && r.URL.Path == pathPrefix+"/hook-rules/"+id:
			writeJSON(t, w, HookRule{RuleID: id, Enabled: true, Phase: "preflight", Selector: "all", ActionOnFail: "deny", ShortCircuit: true, ToolFilter: &HookToolFilter{BlockedNames: []string{"delete_*"}}})
		case r.Method == http.MethodPut && r.URL.Path == pathPrefix+"/hook-rules/"+id:
			writeJSON(t, w, HookRule{RuleID: id, Enabled: false, Phase: "preflight", Selector: "all", ActionOnFail: "deny", ShortCircuit: true, ToolFilter: &HookToolFilter{BlockedNames: []string{"delete_*"}}})
		case r.Method == http.MethodDelete && r.URL.Path == pathPrefix+"/hook-rules/"+id:
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	})

	created, err := client.CreateHookRule(context.Background(), HookRuleWrite{
		RuleID:     id,
		ToolFilter: &HookToolFilter{BlockedNames: []string{"delete_*"}},
	})
	if err != nil {
		t.Fatalf("CreateHookRule: %v", err)
	}
	if created.RuleID != id {
		t.Fatalf("unexpected id: %s", created.RuleID)
	}

	if _, err := client.GetHookRule(context.Background(), id); err != nil {
		t.Fatalf("GetHookRule: %v", err)
	}

	updated, err := client.UpsertHookRule(context.Background(), id, HookRuleWrite{
		RuleID:     id,
		Enabled:    util.Ptr(false),
		ToolFilter: &HookToolFilter{BlockedNames: []string{"delete_*"}},
	})
	if err != nil {
		t.Fatalf("UpsertHookRule: %v", err)
	}
	if updated.Enabled {
		t.Fatalf("expected hook rule disabled after upsert")
	}

	if err := client.DeleteHookRule(context.Background(), id); err != nil {
		t.Fatalf("DeleteHookRule: %v", err)
	}
}

func TestClient_ErrorMapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		status  int
		wantErr error
	}{
		{"not found maps to ErrNotFound", http.StatusNotFound, ErrNotFound},
		{"unauthorized maps to ErrUnauthorized", http.StatusUnauthorized, ErrUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client := testClient(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
			})
			_, err := client.GetRule(context.Background(), "missing")
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("want %v, got %v", tt.wantErr, err)
			}
		})
	}

	t.Run("server error surfaces status and body", func(t *testing.T) {
		t.Parallel()
		client := testClient(t, func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "boom", http.StatusInternalServerError)
		})
		_, err := client.GetEvaluator(context.Background(), "x")
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "500") || !strings.Contains(err.Error(), "boom") {
			t.Fatalf("expected status and body in error, got %v", err)
		}
	})
}

func TestClient_PathEscapesID(t *testing.T) {
	t.Parallel()

	const maliciousID = "../../admin secret"
	var gotPath string
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		writeJSON(t, w, Rule{RuleID: maliciousID})
	})

	if _, err := client.GetRule(context.Background(), maliciousID); err != nil {
		t.Fatalf("GetRule: %v", err)
	}

	want := pathPrefix + "/rules/" + url.PathEscape(maliciousID)
	if gotPath != want {
		t.Fatalf("id not escaped in path:\n got  %q\n want %q", gotPath, want)
	}
}

func TestClient_ListEvaluators_Paginates(t *testing.T) {
	t.Parallel()

	var requests int
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		cursor := r.URL.Query().Get("cursor")
		var items []Evaluator
		next := ""
		switch cursor {
		case "":
			items = make([]Evaluator, listPageSize)
			for i := range items {
				items[i] = Evaluator{EvaluatorID: fmt.Sprintf("p0-%d", i)}
			}
			next = "42"
		case "42":
			items = []Evaluator{{EvaluatorID: "p1-0"}}
		}
		writeJSON(t, w, listResponse[Evaluator]{Items: items, NextCursor: next})
	})

	got, err := client.ListEvaluators(context.Background())
	if err != nil {
		t.Fatalf("ListEvaluators: %v", err)
	}
	if len(got) != listPageSize+1 {
		t.Fatalf("want %d evaluators, got %d", listPageSize+1, len(got))
	}
	if requests != 2 {
		t.Fatalf("want 2 paginated requests, got %d", requests)
	}
}

func TestClient_ListRuleActions(t *testing.T) {
	t.Parallel()

	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != pathPrefix+"/rules/r1/actions" {
			http.NotFound(w, r)
			return
		}
		writeJSON(t, w, itemsResponse[RuleAction]{Items: []RuleAction{{ActionID: "ra_1", RuleID: "r1"}, {ActionID: "ra_2", RuleID: "r1"}}})
	})

	got, err := client.ListRuleActions(context.Background(), "r1")
	if err != nil {
		t.Fatalf("ListRuleActions: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 actions, got %d", len(got))
	}
}
