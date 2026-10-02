package cloud

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestUnitEnsureStackExistenceAndReadiness_Success(t *testing.T) {
	t.Parallel()

	stackMux := http.NewServeMux()
	stackMux.HandleFunc("/login", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	stackMux.HandleFunc("/api/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})
	stackSrv := httptest.NewServer(stackMux)
	t.Cleanup(stackSrv.Close)

	cloudSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !strings.HasPrefix(r.URL.Path, "/api/instances/") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(map[string]string{"url": stackSrv.URL}); err != nil {
			t.Fatalf("encode instance body: %v", err)
		}
	}))
	t.Cleanup(cloudSrv.Close)

	client := newTestGcomAPIClient(t, cloudSrv)
	d := schema.TestResourceDataRaw(t, map[string]*schema.Schema{}, map[string]any{})
	const stateID = "org:resource-id"
	d.SetId(stateID)

	ctx := context.Background()
	diags := ensureStackExistenceAndReadiness(ctx, time.Second, "stack service account", "my-stack-slug", client, d)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if d.Id() != stateID {
		t.Fatalf("resource ID: got %q, want unchanged %q", d.Id(), stateID)
	}
}

func TestUnitEnsureStackExistenceAndReadiness_InstanceNotFound(t *testing.T) {
	t.Parallel()

	cloudSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/instances/") {
			http.NotFound(w, r)
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(cloudSrv.Close)

	client := newTestGcomAPIClient(t, cloudSrv)
	d := schema.TestResourceDataRaw(t, map[string]*schema.Schema{}, map[string]any{})
	d.SetId("1:existing-state-id")

	ctx := context.Background()
	diags := ensureStackExistenceAndReadiness(ctx, time.Second, "stack service account", "missing-stack", client, d)

	if len(diags) != 1 || diags[0].Severity != diag.Warning {
		t.Fatalf("want exactly one warning diagnostic, got %#v", diags)
	}
	if d.Id() != "" {
		t.Fatalf("resource ID should be cleared on missing stack, got %q", d.Id())
	}
}

func TestUnitEnsureStackExistenceAndReadiness_InstanceAPIError(t *testing.T) {
	t.Parallel()

	cloudSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/instances/") {
			http.Error(w, `{"message":"forbidden"}`, http.StatusForbidden)
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(cloudSrv.Close)

	client := newTestGcomAPIClient(t, cloudSrv)
	d := schema.TestResourceDataRaw(t, map[string]*schema.Schema{}, map[string]any{})
	d.SetId("1:state-id")

	ctx := context.Background()
	diags := ensureStackExistenceAndReadiness(ctx, time.Second, "stack service account", "my-stack", client, d)

	if !diags.HasError() || len(diags) != 1 {
		t.Fatalf("want one error diagnostic, got %#v", diags)
	}
	if diags[0].Severity != diag.Error {
		t.Fatalf("severity: got %v, want Error", diags[0].Severity)
	}
	if d.Id() != "1:state-id" {
		t.Fatalf("resource ID should be unchanged on API error, got %q", d.Id())
	}
}

func newTestGcomAPIClient(t *testing.T, cloudSrv *httptest.Server) *gcom.APIClient {
	t.Helper()
	cfg := gcom.NewConfiguration()
	u, err := url.Parse(cloudSrv.URL)
	if err != nil {
		t.Fatalf("parse httptest URL: %v", err)
	}
	cfg.Scheme = u.Scheme
	cfg.Host = u.Host
	cfg.HTTPClient = cloudSrv.Client()
	return gcom.NewAPIClient(cfg)
}
