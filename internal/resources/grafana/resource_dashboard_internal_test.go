package grafana

import (
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
)

func TestIsKubernetesStyleDashboard(t *testing.T) {
	t.Run("legacy dashboard", func(t *testing.T) {
		if isKubernetesStyleDashboard(map[string]any{"title": "legacy"}) {
			t.Fatal("expected legacy dashboard shape to be treated as non-kubernetes")
		}
	})

	t.Run("kubernetes style dashboard", func(t *testing.T) {
		if !isKubernetesStyleDashboard(map[string]any{
			"apiVersion": "dashboard.grafana.app/v2beta1",
			"kind":       "Dashboard",
			"metadata": map[string]any{
				"name": "test-dashboard",
			},
			"spec": map[string]any{
				"title": "test dashboard",
			},
		}) {
			t.Fatal("expected kubernetes dashboard shape to be detected")
		}
	})
}

func TestNormalizeDashboardConfigJSONForState(t *testing.T) {
	t.Run("preserves kubernetes dashboard shape when remote body matches local spec", func(t *testing.T) {
		configJSON := `{"apiVersion":"dashboard.grafana.app/v2beta1","kind":"Dashboard","metadata":{"name":"test-dashboard"},"spec":{"title":"test dashboard"}}`
		remoteDashJSON := map[string]any{
			"title":   "test dashboard",
			"id":      7,
			"uid":     "test-dashboard",
			"version": 3,
		}

		got, err := normalizeDashboardConfigJSONForState(configJSON, remoteDashJSON)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != NormalizeDashboardConfigJSON(configJSON) {
			t.Fatalf("expected kubernetes-shaped config to be preserved, got %s", got)
		}
	})

	t.Run("stores remote dashboard body under spec when kubernetes dashboard drifts", func(t *testing.T) {
		configJSON := `{"apiVersion":"dashboard.grafana.app/v2beta1","kind":"Dashboard","metadata":{"name":"test-dashboard"},"spec":{"title":"local dashboard"}}`
		remoteDashJSON := map[string]any{
			"title":   "remote dashboard",
			"id":      11,
			"uid":     "test-dashboard",
			"version": 5,
		}

		got, err := normalizeDashboardConfigJSONForState(configJSON, remoteDashJSON)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := `{"apiVersion":"dashboard.grafana.app/v2beta1","kind":"Dashboard","metadata":{"name":"test-dashboard"},"spec":{"title":"remote dashboard"}}`
		if got != want {
			t.Fatalf("expected remote dashboard body to be stored under spec, got %s", got)
		}
	})

	t.Run("drops generated uid for legacy dashboard config without uid", func(t *testing.T) {
		configJSON := `{"title":"legacy dashboard"}`
		remoteDashJSON := map[string]any{
			"title": "legacy dashboard",
			"uid":   "generated-uid",
		}

		got, err := normalizeDashboardConfigJSONForState(configJSON, remoteDashJSON)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != `{"title":"legacy dashboard"}` {
			t.Fatalf("expected generated uid to be removed, got %s", got)
		}
	})
}

func TestPreferredDashboardAPIVersion(t *testing.T) {
	t.Run("extracts version from kubernetes dashboard config", func(t *testing.T) {
		configJSON := `{"apiVersion":"dashboard.grafana.app/v2beta1","kind":"Dashboard","metadata":{"name":"test-dashboard"},"spec":{"title":"test dashboard"}}`

		got := preferredDashboardAPIVersion(configJSON)
		if got != "v2beta1" {
			t.Fatalf("expected v2beta1, got %q", got)
		}
	})

	t.Run("returns empty for legacy dashboard config", func(t *testing.T) {
		if got := preferredDashboardAPIVersion(`{"title":"legacy dashboard"}`); got != "" {
			t.Fatalf("expected empty version for legacy dashboard, got %q", got)
		}
	})

	t.Run("returns empty for sha256 config", func(t *testing.T) {
		if got := preferredDashboardAPIVersion("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"); got != "" {
			t.Fatalf("expected empty version for sha256 state, got %q", got)
		}
	})
}

func TestReadDashboardByUIDParamsWriteToRequest(t *testing.T) {
	req := &testClientRequest{
		queryParams: make(url.Values),
	}

	err := newReadDashboardByUIDParams(context.Background(), "test-dashboard", "v2beta1").WriteToRequest(req, strfmt.Default)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.pathParams["uid"] != "test-dashboard" {
		t.Fatalf("expected uid path param to be set, got %q", req.pathParams["uid"])
	}
	if got := req.queryParams.Get("apiVersion"); got != "v2beta1" {
		t.Fatalf("expected apiVersion query param to be set, got %q", got)
	}
	if req.timeout != 30*time.Second {
		t.Fatalf("expected default timeout to be set, got %s", req.timeout)
	}
}

type testClientRequest struct {
	headers     http.Header
	queryParams url.Values
	pathParams  map[string]string
	timeout     time.Duration
}

func (r *testClientRequest) SetHeaderParam(name string, values ...string) error {
	if r.headers == nil {
		r.headers = make(http.Header)
	}
	r.headers.Set(name, values[0])
	return nil
}

func (r *testClientRequest) GetHeaderParams() http.Header {
	if r.headers == nil {
		r.headers = make(http.Header)
	}
	return r.headers
}

func (r *testClientRequest) SetQueryParam(name string, values ...string) error {
	if r.queryParams == nil {
		r.queryParams = make(url.Values)
	}
	r.queryParams.Set(name, values[0])
	return nil
}

func (r *testClientRequest) SetFormParam(string, ...string) error { return nil }

func (r *testClientRequest) SetPathParam(name, value string) error {
	if r.pathParams == nil {
		r.pathParams = make(map[string]string)
	}
	r.pathParams[name] = value
	return nil
}

func (r *testClientRequest) GetQueryParams() url.Values { return r.queryParams }

func (r *testClientRequest) SetFileParam(string, ...runtime.NamedReadCloser) error { return nil }

func (r *testClientRequest) SetBodyParam(interface{}) error { return nil }

func (r *testClientRequest) SetTimeout(timeout time.Duration) error {
	r.timeout = timeout
	return nil
}

func (r *testClientRequest) GetMethod() string { return "" }

func (r *testClientRequest) GetPath() string { return "" }

func (r *testClientRequest) GetBody() []byte { return nil }

func (r *testClientRequest) GetBodyParam() interface{} { return nil }

func (r *testClientRequest) GetFileParam() map[string][]runtime.NamedReadCloser { return nil }
