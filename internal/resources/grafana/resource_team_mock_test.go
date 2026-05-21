package grafana_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// TestUnitTeam_IgnoreExternallySyncedTransition_Mock tests the transition of
// ignore_externally_synced_members from false to true when members are not in
// config. It uses a mock Grafana API that returns members with external sync
// labels (simulating LDAP/SAML/team sync).
//
// The bug (support escalation #21901): With the Default([]) on the members
// attribute, removing members from config causes plan=[] which diffs against
// state (has members) and triggers mass removal. The correct behavior is to
// preserve members and silently clear them from state via Read() filtering.
func TestUnitTeam_IgnoreExternallySyncedTransition_Mock(t *testing.T) {
	// memberRemovals tracks how many times the mock received a DELETE request
	// to remove a team member. The fix should result in zero removals.
	var memberRemovals atomic.Int32

	mux := http.NewServeMux()

	// POST /api/teams — create team
	mux.HandleFunc("/api/teams", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"teamId": 1})
			return
		}
		http.NotFound(w, r)
	})

	// GET/PUT/DELETE /api/teams/1 — team CRUD
	mux.HandleFunc("/api/teams/1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			json.NewEncoder(w).Encode(map[string]any{
				"id": 1, "orgId": 1, "name": "test-team",
				"email": "test@example.com", "uid": "abc123",
			})
		case http.MethodPut:
			json.NewEncoder(w).Encode(map[string]any{"message": "Team updated"})
		case http.MethodDelete:
			json.NewEncoder(w).Encode(map[string]any{"message": "Team deleted"})
		default:
			http.NotFound(w, r)
		}
	})

	// GET/POST /api/teams/1/members — list/add members
	// Always returns members WITH labels (externally synced).
	mux.HandleFunc("/api/teams/1/members", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			json.NewEncoder(w).Encode([]map[string]any{
				{"userId": 10, "email": "user1@example.com", "login": "user1", "labels": []string{"ldap"}},
				{"userId": 11, "email": "user2@example.com", "login": "user2", "labels": []string{"ldap"}},
			})
		case http.MethodPost:
			json.NewEncoder(w).Encode(map[string]any{"message": "Member added"})
		default:
			http.NotFound(w, r)
		}
	})

	// DELETE /api/teams/1/members/{userId} — remove member
	mux.HandleFunc("/api/teams/1/members/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			memberRemovals.Add(1)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"message": "Member removed"})
			return
		}
		http.NotFound(w, r)
	})

	// GET/PUT /api/teams/1/preferences
	mux.HandleFunc("/api/teams/1/preferences", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			json.NewEncoder(w).Encode(map[string]any{})
		case http.MethodPut:
			json.NewEncoder(w).Encode(map[string]any{"message": "Preferences updated"})
		default:
			http.NotFound(w, r)
		}
	})

	// GET /api/org/users — list org users (needed for member ID resolution)
	mux.HandleFunc("/api/org/users", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{
			{"userId": 10, "email": "user1@example.com", "login": "user1", "role": "Viewer"},
			{"userId": 11, "email": "user2@example.com", "login": "user2", "role": "Viewer"},
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	t.Setenv("GRAFANA_URL", server.URL)
	t.Setenv("GRAFANA_AUTH", "admin:admin")

	// Step 1 config: ignore=false, explicit members.
	// With ignore=false, labeled members are included in state.
	configStep1 := `
resource "grafana_team" "test" {
	name                             = "test-team"
	email                            = "test@example.com"
	ignore_externally_synced_members = false
	members                          = ["user1@example.com", "user2@example.com"]
}`

	// Step 2 config: ignore=true, NO members in config.
	// This simulates the customer transitioning to "let external sync manage members".
	configStep2 := `
resource "grafana_team" "test" {
	name                             = "test-team"
	email                            = "test@example.com"
	ignore_externally_synced_members = true
}`

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: configStep1,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_team.test", "members.#", "2"),
					resource.TestCheckResourceAttr("grafana_team.test", "ignore_externally_synced_members", "false"),
				),
			},
			{
				PreConfig: func() {
					// Reset removal counter before the transition step.
					memberRemovals.Store(0)
				},
				Config: configStep2,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_team.test", "ignore_externally_synced_members", "true"),
					// Members are still in state after this apply because the
					// Update read used the old ignore=false value to match the
					// plan. They will be cleared on the next refresh cycle.
					resource.TestCheckResourceAttr("grafana_team.test", "members.#", "2"),
					func(_ *terraform.State) error {
						if n := memberRemovals.Load(); n > 0 {
							return fmt.Errorf("expected 0 member removal API calls, got %d", n)
						}
						return nil
					},
				),
			},
			// Step 3: Re-apply same config. The refresh (Read) now runs with
			// ignore=true, filtering externally synced members from state.
			// This is the "silent state clearing" — no diff, no API calls.
			{
				Config: configStep2,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_team.test", "ignore_externally_synced_members", "true"),
					// Members are now cleared from state (filtered by ignore=true).
					resource.TestCheckResourceAttr("grafana_team.test", "members.#", "0"),
					func(_ *terraform.State) error {
						if n := memberRemovals.Load(); n > 0 {
							return fmt.Errorf("expected 0 member removal API calls, got %d", n)
						}
						return nil
					},
				),
			},
		},
	})
}
