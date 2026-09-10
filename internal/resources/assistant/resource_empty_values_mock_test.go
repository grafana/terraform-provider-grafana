package assistant_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
)

// The Assistant API omits empty optional fields from its responses, so a field
// that was configured as `""`, `[]` or `{}` comes back indistinguishable from
// one that was never set. The provider used to write null into state for those,
// which makes Terraform abort the apply with "Provider produced inconsistent
// result after apply ... was cty.StringVal(\"\"), but now null".
//
// These tests pin that round trip using a mock Assistant API that echoes back
// exactly what it was sent — the behaviour of a well-behaved server. Each case
// also implicitly asserts plan stability: the test harness fails a step whose
// post-apply plan is non-empty.

// mockAssistantAPI serves the Assistant plugin endpoints for create/read/delete.
func mockAssistantAPI(t *testing.T) *httptest.Server {
	t.Helper()
	store := map[string]map[string]any{}

	writeJSON := func(w http.ResponseWriter, data any) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{"status": "ok", "data": data}); err != nil {
			t.Errorf("failed to encode response: %v", err)
		}
	}

	decodeBody := func(w http.ResponseWriter, r *http.Request) (map[string]any, bool) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read request body: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return nil, false
		}
		obj := map[string]any{}
		if err := json.Unmarshal(body, &obj); err != nil {
			t.Errorf("failed to decode request body: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return nil, false
		}
		return obj, true
	}

	const base = "/api/plugins/grafana-assistant-app/resources/api/v1"
	mux := http.NewServeMux()
	for _, kind := range []string{"rules", "skills", "integrations", "quickstarts"} {
		id := kind + "-1"
		mux.HandleFunc(base+"/"+kind, func(w http.ResponseWriter, r *http.Request) {
			obj, ok := decodeBody(w, r)
			if !ok {
				return
			}
			obj["id"] = id
			store[id] = obj
			writeJSON(w, obj)
		})
		mux.HandleFunc(base+"/"+kind+"/"+id, func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodDelete:
				delete(store, id)
				writeJSON(w, map[string]any{})
			case http.MethodPut:
				patch, ok := decodeBody(w, r)
				if !ok {
					return
				}
				// A PUT carries only the fields the provider chose to send;
				// merge so unsent fields keep their stored value.
				for k, v := range patch {
					store[id][k] = v
				}
				writeJSON(w, store[id])
			default:
				writeJSON(w, store[id])
			}
		})
	}

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestUnitAssistantEmptyOptionalValues_Mock(t *testing.T) {
	for _, tc := range []struct {
		name   string
		config string
		checks []resource.TestCheckFunc
	}{
		{
			name: "mcp server with empty tool maps and empty description",
			config: `
resource "grafana_assistant_mcp_server" "test" {
  name        = "test-mcp"
  scope       = "tenant"
  description = ""
  configuration {
    url                    = "https://mcp.example.com"
    tool_preferences       = {}
    tool_approval_policies = {}
  }
  custom_headers = {}
}`,
			checks: []resource.TestCheckFunc{
				resource.TestCheckResourceAttr("grafana_assistant_mcp_server.test", "description", ""),
				resource.TestCheckResourceAttr("grafana_assistant_mcp_server.test", "configuration.tool_preferences.%", "0"),
				resource.TestCheckResourceAttr("grafana_assistant_mcp_server.test", "configuration.tool_approval_policies.%", "0"),
				resource.TestCheckResourceAttr("grafana_assistant_mcp_server.test", "custom_headers.%", "0"),
			},
		},
		{
			name: "mcp server with empty applications",
			config: `
resource "grafana_assistant_mcp_server" "test" {
  name         = "test-mcp"
  scope        = "tenant"
  applications = []
  configuration {
    url = "https://mcp.example.com"
  }
}`,
			checks: []resource.TestCheckFunc{
				resource.TestCheckResourceAttr("grafana_assistant_mcp_server.test", "applications.#", "0"),
			},
		},
		{
			// `configuration` is an optional block, so omitting it must not fail.
			name: "mcp server without a configuration block",
			config: `
resource "grafana_assistant_mcp_server" "test" {
  name  = "test-mcp"
  scope = "tenant"
}`,
			checks: []resource.TestCheckFunc{
				resource.TestCheckResourceAttrSet("grafana_assistant_mcp_server.test", "id"),
				resource.TestCheckNoResourceAttr("grafana_assistant_mcp_server.test", "configuration.url"),
			},
		},
		{
			name: "rule with empty description and applications",
			config: `
resource "grafana_assistant_rule" "test" {
  name         = "test-rule"
  scope        = "tenant"
  description  = ""
  rule_content = "Be concise."
  applications = []
}`,
			checks: []resource.TestCheckFunc{
				resource.TestCheckResourceAttr("grafana_assistant_rule.test", "description", ""),
				resource.TestCheckResourceAttr("grafana_assistant_rule.test", "applications.#", "0"),
			},
		},
		{
			name: "skill with empty context items",
			config: `
resource "grafana_assistant_skill" "test" {
  name          = "test-skill"
  scope         = "tenant"
  body          = "# skill"
  context_items = ""
}`,
			checks: []resource.TestCheckFunc{
				resource.TestCheckResourceAttr("grafana_assistant_skill.test", "context_items", ""),
			},
		},
		{
			name: "quickstart with empty context items and title",
			config: `
resource "grafana_assistant_quickstart" "test" {
  scope         = "tenant"
  title         = ""
  prompt        = "How do I start?"
  context_items = ""
}`,
			checks: []resource.TestCheckFunc{
				resource.TestCheckResourceAttr("grafana_assistant_quickstart.test", "title", ""),
				resource.TestCheckResourceAttr("grafana_assistant_quickstart.test", "context_items", ""),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := mockAssistantAPI(t)
			t.Setenv("GRAFANA_URL", srv.URL)
			t.Setenv("GRAFANA_AUTH", "mock-token")

			resource.UnitTest(t, resource.TestCase{
				ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
				Steps: []resource.TestStep{{
					Config: tc.config,
					Check:  resource.ComposeTestCheckFunc(tc.checks...),
				}},
			})
		})
	}
}

// Update takes the plan rather than the prior state as its reconciliation
// source, so it needs its own coverage: an in-place change must not resurrect
// the null that Create avoided.
func TestUnitAssistantEmptyOptionalValuesOnUpdate_Mock(t *testing.T) {
	srv := mockAssistantAPI(t)
	t.Setenv("GRAFANA_URL", srv.URL)
	t.Setenv("GRAFANA_AUTH", "mock-token")

	config := func(name string) string {
		return `
resource "grafana_assistant_mcp_server" "test" {
  name        = "` + name + `"
  scope       = "tenant"
  description = ""
  configuration {
    url                    = "https://mcp.example.com"
    tool_approval_policies = {}
  }
}`
	}

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config("test-mcp"),
				Check:  resource.TestCheckResourceAttr("grafana_assistant_mcp_server.test", "description", ""),
			},
			{
				Config: config("test-mcp-renamed"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_assistant_mcp_server.test", "name", "test-mcp-renamed"),
					resource.TestCheckResourceAttr("grafana_assistant_mcp_server.test", "description", ""),
					resource.TestCheckResourceAttr("grafana_assistant_mcp_server.test", "configuration.tool_approval_policies.%", "0"),
				),
			},
			{
				ResourceName:            "grafana_assistant_mcp_server.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"custom_headers", "description", "configuration.tool_approval_policies"},
			},
		},
	})
}
