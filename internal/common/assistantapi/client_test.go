package assistantapi

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
	"time"

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

func TestClient_CreateAndGetRule(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == pathPrefix+"/rules":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(apiResponseWrapper[Rule]{
				Status: "success",
				Data: Rule{
					ID:          "11111111-1111-1111-1111-111111111111",
					Name:        "test",
					RuleContent: "content",
					Scope:       "tenant",
					Enabled:     util.Ptr(true),
				},
			})
		case r.Method == http.MethodGet && r.URL.Path == pathPrefix+"/rules/11111111-1111-1111-1111-111111111111":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(apiResponseWrapper[Rule]{
				Status: "success",
				Data: Rule{
					ID:          "11111111-1111-1111-1111-111111111111",
					Name:        "test",
					RuleContent: "content",
					Scope:       "tenant",
					Enabled:     util.Ptr(true),
				},
			})
		case r.Method == http.MethodDelete && r.URL.Path == pathPrefix+"/rules/11111111-1111-1111-1111-111111111111":
			if r.Header.Get("X-Resource-Scope") != "tenant" {
				http.Error(w, "missing scope header", http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClient(server.URL, nil, "test-token", server.Client(), "test-agent", nil)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	created, err := client.CreateRule(context.Background(), RuleCreate{
		Scope:       "tenant",
		Name:        "test",
		RuleContent: "content",
		Enabled:     util.Ptr(true),
	})
	if err != nil {
		t.Fatalf("CreateRule: %v", err)
	}
	if created.ID != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("unexpected id: %s", created.ID)
	}

	got, err := client.GetRule(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetRule: %v", err)
	}
	if got.Name != "test" {
		t.Fatalf("unexpected name: %s", got.Name)
	}

	if err := client.DeleteRule(context.Background(), created.ID, "tenant"); err != nil {
		t.Fatalf("DeleteRule: %v", err)
	}
}

func TestClient_SkillCRUD(t *testing.T) {
	t.Parallel()

	const id = "22222222-2222-2222-2222-222222222222"
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == pathPrefix+"/skills":
			writeJSON(t, w, apiResponseWrapper[Skill]{
				Status: "success",
				Data:   Skill{ID: id, Name: "deploy", Body: "body", Scope: "tenant"},
			})
		case r.Method == http.MethodGet && r.URL.Path == pathPrefix+"/skills/"+id:
			writeJSON(t, w, apiResponseWrapper[Skill]{
				Status: "success",
				Data:   Skill{ID: id, Name: "deploy", Body: "body", Scope: "tenant"},
			})
		case r.Method == http.MethodPut && r.URL.Path == pathPrefix+"/skills/"+id:
			if r.Header.Get("X-Resource-Scope") != "tenant" {
				http.Error(w, "missing scope header", http.StatusBadRequest)
				return
			}
			writeJSON(t, w, apiResponseWrapper[Skill]{
				Status: "success",
				Data:   Skill{ID: id, Name: "deploy v2", Body: "body", Scope: "tenant"},
			})
		case r.Method == http.MethodDelete && r.URL.Path == pathPrefix+"/skills/"+id:
			if r.Header.Get("X-Resource-Scope") != "tenant" {
				http.Error(w, "missing scope header", http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	})

	created, err := client.CreateSkill(context.Background(), SkillCreate{
		Name:  "deploy",
		Body:  "body",
		Scope: util.Ptr("tenant"),
	})
	if err != nil {
		t.Fatalf("CreateSkill: %v", err)
	}
	if created.ID != id {
		t.Fatalf("unexpected id: %s", created.ID)
	}

	got, err := client.GetSkill(context.Background(), id)
	if err != nil {
		t.Fatalf("GetSkill: %v", err)
	}
	if got.Name != "deploy" {
		t.Fatalf("unexpected name: %s", got.Name)
	}

	updated, err := client.UpdateSkill(context.Background(), id, "tenant", SkillUpdate{
		Name: util.Ptr("deploy v2"),
	})
	if err != nil {
		t.Fatalf("UpdateSkill: %v", err)
	}
	if updated.Name != "deploy v2" {
		t.Fatalf("unexpected updated name: %s", updated.Name)
	}

	if err := client.DeleteSkill(context.Background(), id, "tenant"); err != nil {
		t.Fatalf("DeleteSkill: %v", err)
	}
}

func TestClient_SetSkillCommand(t *testing.T) {
	t.Parallel()

	const id = "22222222-2222-2222-2222-222222222222"
	commandEnabledAt := time.Now()
	commandRequests := 0
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != pathPrefix+"/skills/"+id+"/command" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("X-Resource-Scope") != "tenant" {
			http.Error(w, "missing scope header", http.StatusBadRequest)
			return
		}

		var body map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode skill command request: %v", err)
		}
		commandRequests++
		enabledAt := &commandEnabledAt
		if commandRequests == 1 {
			if got := string(body["commandName"]); got != `"deploy"` {
				t.Fatalf("unexpected enable commandName: %s", got)
			}
		} else {
			if got := string(body["commandName"]); got != "null" {
				t.Fatalf("unexpected disable commandName: %s", got)
			}
			enabledAt = nil
		}
		writeJSON(t, w, apiResponseWrapper[Skill]{
			Status: "success",
			Data:   Skill{ID: id, CommandName: util.Ptr("deploy"), CommandEnabledAt: enabledAt, Scope: "tenant"},
		})
	})

	enabled, err := client.SetSkillCommand(context.Background(), id, "tenant", util.Ptr("deploy"))
	if err != nil {
		t.Fatalf("SetSkillCommand: %v", err)
	}
	if enabled.CommandName == nil || *enabled.CommandName != "deploy" || enabled.CommandEnabledAt == nil {
		t.Fatalf("unexpected enabled command: %+v", enabled)
	}

	disabled, err := client.SetSkillCommand(context.Background(), id, "tenant", nil)
	if err != nil {
		t.Fatalf("SetSkillCommand disable: %v", err)
	}
	if disabled.CommandName == nil || *disabled.CommandName != "deploy" || disabled.CommandEnabledAt != nil {
		t.Fatalf("unexpected disabled command: %+v", disabled)
	}
}

func TestClient_QuickstartCRUD(t *testing.T) {
	t.Parallel()

	const id = "33333333-3333-3333-3333-333333333333"
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == pathPrefix+"/quickstarts":
			writeJSON(t, w, apiResponseWrapper[Quickstart]{
				Status: "success",
				Data:   Quickstart{ID: id, Title: util.Ptr("show errors"), Prompt: "prompt", Scope: "tenant", Enabled: util.Ptr(true)},
			})
		case r.Method == http.MethodGet && r.URL.Path == pathPrefix+"/quickstarts/"+id:
			writeJSON(t, w, apiResponseWrapper[Quickstart]{
				Status: "success",
				Data:   Quickstart{ID: id, Title: util.Ptr("show errors"), Prompt: "prompt", Scope: "tenant", Enabled: util.Ptr(true)},
			})
		case r.Method == http.MethodPut && r.URL.Path == pathPrefix+"/quickstarts/"+id:
			if r.Header.Get("X-Resource-Scope") != "tenant" {
				http.Error(w, "missing scope header", http.StatusBadRequest)
				return
			}
			writeJSON(t, w, apiResponseWrapper[Quickstart]{
				Status: "success",
				Data:   Quickstart{ID: id, Title: util.Ptr("show errors"), Prompt: "new prompt", Scope: "tenant", Enabled: util.Ptr(true)},
			})
		case r.Method == http.MethodDelete && r.URL.Path == pathPrefix+"/quickstarts/"+id:
			if r.Header.Get("X-Resource-Scope") != "tenant" {
				http.Error(w, "missing scope header", http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	})

	created, err := client.CreateQuickstart(context.Background(), QuickstartCreate{
		Scope:  "tenant",
		Title:  util.Ptr("show errors"),
		Prompt: "prompt",
	})
	if err != nil {
		t.Fatalf("CreateQuickstart: %v", err)
	}
	if created.ID != id {
		t.Fatalf("unexpected id: %s", created.ID)
	}

	got, err := client.GetQuickstart(context.Background(), id)
	if err != nil {
		t.Fatalf("GetQuickstart: %v", err)
	}
	if got.Prompt != "prompt" {
		t.Fatalf("unexpected prompt: %s", got.Prompt)
	}

	updated, err := client.UpdateQuickstart(context.Background(), id, "tenant", QuickstartUpdate{
		Scope:  "tenant",
		Prompt: util.Ptr("new prompt"),
	})
	if err != nil {
		t.Fatalf("UpdateQuickstart: %v", err)
	}
	if updated.Prompt != "new prompt" {
		t.Fatalf("unexpected updated prompt: %s", updated.Prompt)
	}

	if err := client.DeleteQuickstart(context.Background(), id, "tenant"); err != nil {
		t.Fatalf("DeleteQuickstart: %v", err)
	}
}

func TestClient_IntegrationCRUD(t *testing.T) {
	t.Parallel()

	const id = "44444444-4444-4444-4444-444444444444"
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == pathPrefix+"/integrations":
			// Verify the request body uses the camelCase customHeaders field.
			var body IntegrationCreate
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "bad body", http.StatusBadRequest)
				return
			}
			if len(body.CustomHeaders) != 1 || body.CustomHeaders[0].Key != "Authorization" {
				http.Error(w, "missing custom headers", http.StatusBadRequest)
				return
			}
			writeJSON(t, w, apiResponseWrapper[Integration]{
				Status: "success",
				Data:   Integration{ID: id, Name: "github", Type: "mcp", Scope: "tenant", CustomHeaders: body.CustomHeaders},
			})
		case r.Method == http.MethodGet && r.URL.Path == pathPrefix+"/integrations/"+id:
			writeJSON(t, w, apiResponseWrapper[Integration]{
				Status: "success",
				Data:   Integration{ID: id, Name: "github", Type: "mcp", Scope: "tenant"},
			})
		case r.Method == http.MethodPut && r.URL.Path == pathPrefix+"/integrations/"+id:
			if r.Header.Get("X-Resource-Scope") != "tenant" {
				http.Error(w, "missing scope header", http.StatusBadRequest)
				return
			}
			writeJSON(t, w, apiResponseWrapper[Integration]{
				Status: "success",
				Data:   Integration{ID: id, Name: "github v2", Type: "mcp", Scope: "tenant"},
			})
		case r.Method == http.MethodDelete && r.URL.Path == pathPrefix+"/integrations/"+id:
			if r.Header.Get("X-Resource-Scope") != "tenant" {
				http.Error(w, "missing scope header", http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	})

	created, err := client.CreateIntegration(context.Background(), IntegrationCreate{
		Scope:         "tenant",
		Name:          "github",
		Type:          "mcp",
		CustomHeaders: []Header{{Key: "Authorization", Value: "Bearer x"}},
	})
	if err != nil {
		t.Fatalf("CreateIntegration: %v", err)
	}
	if created.ID != id {
		t.Fatalf("unexpected id: %s", created.ID)
	}

	got, err := client.GetIntegration(context.Background(), id)
	if err != nil {
		t.Fatalf("GetIntegration: %v", err)
	}
	if got.Name != "github" {
		t.Fatalf("unexpected name: %s", got.Name)
	}

	updated, err := client.UpdateIntegration(context.Background(), id, "tenant", IntegrationUpdate{
		Scope: "tenant",
		Name:  util.Ptr("github v2"),
	})
	if err != nil {
		t.Fatalf("UpdateIntegration: %v", err)
	}
	if updated.Name != "github v2" {
		t.Fatalf("unexpected updated name: %s", updated.Name)
	}

	if err := client.DeleteIntegration(context.Background(), id, "tenant"); err != nil {
		t.Fatalf("DeleteIntegration: %v", err)
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
		_, err := client.GetSkill(context.Background(), "x")
		if err == nil {
			t.Fatal("expected error")
		}
		if errors.Is(err, ErrNotFound) || errors.Is(err, ErrUnauthorized) {
			t.Fatalf("unexpected sentinel error: %v", err)
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
		writeJSON(t, w, apiResponseWrapper[Rule]{Status: "success", Data: Rule{ID: maliciousID}})
	})

	if _, err := client.GetRule(context.Background(), maliciousID); err != nil {
		t.Fatalf("GetRule: %v", err)
	}

	want := pathPrefix + "/rules/" + url.PathEscape(maliciousID)
	if gotPath != want {
		t.Fatalf("id not escaped in path:\n got  %q\n want %q", gotPath, want)
	}
}

func TestClient_ListSkills_Paginates(t *testing.T) {
	t.Parallel()

	var requests int
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		offset := r.URL.Query().Get("offset")
		if got := r.URL.Query().Get("limit"); got != fmt.Sprintf("%d", listPageSize) {
			http.Error(w, "unexpected limit", http.StatusBadRequest)
			return
		}
		var skills []Skill
		switch offset {
		case "0":
			skills = make([]Skill, listPageSize)
			for i := range skills {
				skills[i] = Skill{ID: fmt.Sprintf("p0-%d", i)}
			}
		case fmt.Sprintf("%d", listPageSize):
			skills = []Skill{{ID: "p1-0"}}
		}
		writeJSON(t, w, apiResponseWrapper[skillListData]{
			Status: "success",
			Data:   skillListData{Skills: skills, Pagination: pagination{Total: int64(listPageSize + 1)}},
		})
	})

	got, err := client.ListSkills(context.Background())
	if err != nil {
		t.Fatalf("ListSkills: %v", err)
	}
	if len(got) != listPageSize+1 {
		t.Fatalf("want %d skills, got %d", listPageSize+1, len(got))
	}
	if requests != 2 {
		t.Fatalf("want 2 paginated requests, got %d", requests)
	}
}

func TestClient_ListEndpoints_SinglePage(t *testing.T) {
	t.Parallel()

	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case pathPrefix + "/rules":
			writeJSON(t, w, apiResponseWrapper[ruleListData]{Data: ruleListData{Rules: []Rule{{ID: "r1"}, {ID: "r2"}}}})
		case pathPrefix + "/quickstarts":
			writeJSON(t, w, apiResponseWrapper[quickstartListData]{Data: quickstartListData{Quickstarts: []Quickstart{{ID: "q1"}}}})
		case pathPrefix + "/integrations":
			writeJSON(t, w, apiResponseWrapper[integrationListData]{Data: integrationListData{Integrations: []Integration{{ID: "i1"}, {ID: "i2"}, {ID: "i3"}}}})
		default:
			http.NotFound(w, r)
		}
	})

	rules, err := client.ListRules(context.Background())
	if err != nil {
		t.Fatalf("ListRules: %v", err)
	}
	if len(rules) != 2 {
		t.Fatalf("want 2 rules, got %d", len(rules))
	}

	quickstarts, err := client.ListQuickstarts(context.Background())
	if err != nil {
		t.Fatalf("ListQuickstarts: %v", err)
	}
	if len(quickstarts) != 1 {
		t.Fatalf("want 1 quickstart, got %d", len(quickstarts))
	}

	integrations, err := client.ListIntegrations(context.Background())
	if err != nil {
		t.Fatalf("ListIntegrations: %v", err)
	}
	if len(integrations) != 3 {
		t.Fatalf("want 3 integrations, got %d", len(integrations))
	}
}

func TestClient_List_PropagatesError(t *testing.T) {
	t.Parallel()

	client := testClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	if _, err := client.ListRules(context.Background()); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("want ErrUnauthorized, got %v", err)
	}
}
