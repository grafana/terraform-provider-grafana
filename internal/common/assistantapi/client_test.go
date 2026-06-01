package assistantapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

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
					Enabled:     boolPtr(true),
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
					Enabled:     boolPtr(true),
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
		Enabled:     boolPtr(true),
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

func boolPtr(b bool) *bool {
	return &b
}
