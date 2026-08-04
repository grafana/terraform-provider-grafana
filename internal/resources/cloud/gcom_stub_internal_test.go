package cloud

// Shared test scaffolding for the grafana.com retry/idempotency status-code suites. Each
// resource has its own *_internal_test.go file (matching the source file the behaviour is
// defined in) that drives the real CRUD functions against the scripted gcomStub below and
// asserts the number of attempts and the resulting diagnostics. The stub, script helpers and
// diagnostic matchers live here so every suite shares one implementation.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	fwdiag "github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

// stubResponse is one scripted HTTP response.
type stubResponse struct {
	status int
	body   string
	header map[string]string
}

// stubRoute matches requests and replies with a scripted sequence of responses. The Nth
// matching request returns the Nth entry; once the script is exhausted the final entry
// repeats. count records how many requests the route served.
type stubRoute struct {
	match  func(*http.Request) bool
	script []stubResponse
	count  int
}

// gcomStub is an httptest handler that routes grafana.com requests to scripted responses.
type gcomStub struct {
	mu     sync.Mutex
	routes []*stubRoute
	client *gcom.APIClient
}

func (s *gcomStub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, route := range s.routes {
		if route.match == nil || !route.match(r) {
			continue
		}
		idx := route.count
		if idx >= len(route.script) {
			idx = len(route.script) - 1
		}
		route.count++
		resp := route.script[idx]
		for k, v := range resp.header {
			w.Header().Set(k, v)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.status)
		body := resp.body
		if body == "" {
			body = "{}"
		}
		_, _ = w.Write([]byte(body))
		return
	}
	// Unmatched requests succeed with an empty object so incidental calls (follow-up reads
	// that a scenario does not care about) don't fail the operation under test.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("{}"))
}

func newStubbedGcomClient(t *testing.T, routes ...*stubRoute) *gcomStub {
	t.Helper()
	stub := &gcomStub{routes: routes}
	srv := httptest.NewServer(stub)
	t.Cleanup(srv.Close)
	stub.client = newTestGcomAPIClient(t, srv)
	return stub
}

// codes builds a script from a list of status codes with empty bodies.
func codes(cs ...int) []stubResponse {
	out := make([]stubResponse, len(cs))
	for i, c := range cs {
		out[i] = stubResponse{status: c}
	}
	return out
}

// retryAfterZero is a 429 response with Retry-After: 0, so the retry proceeds without an
// extra sleep (keeps the tests fast while still exercising the 429 path).
func retryAfterZero() stubResponse {
	return stubResponse{status: http.StatusTooManyRequests, header: map[string]string{"Retry-After": "0"}}
}

func methodContains(method, substr string) func(*http.Request) bool {
	return func(r *http.Request) bool {
		return r.Method == method && strings.Contains(r.URL.Path, substr)
	}
}

// diagsContainErr reports whether any SDKv2 error diagnostic's summary or detail contains want.
func diagsContainErr(diags diag.Diagnostics, want string) bool {
	for _, d := range diags {
		if d.Severity == diag.Error && (strings.Contains(d.Summary, want) || strings.Contains(d.Detail, want)) {
			return true
		}
	}
	return false
}

// fwDiagsContainErr reports whether any framework error diagnostic's summary or detail contains want.
func fwDiagsContainErr(diags fwdiag.Diagnostics, want string) bool {
	for _, d := range diags.Errors() {
		if strings.Contains(d.Summary(), want) || strings.Contains(d.Detail(), want) {
			return true
		}
	}
	return false
}

// assertWantErr asserts on SDKv2 diagnostics: when want is empty no error is expected,
// otherwise an error diagnostic whose summary or detail contains want must be present.
func assertWantErr(t *testing.T, diags diag.Diagnostics, want string) {
	t.Helper()
	if want == "" {
		if diags.HasError() {
			t.Fatalf("unexpected error: %v", diags)
		}
		return
	}
	if !diagsContainErr(diags, want) {
		t.Fatalf("want error containing %q, got %v", want, diags)
	}
}

// assertWantErrFw is the framework-diagnostics counterpart of assertWantErr.
func assertWantErrFw(t *testing.T, diags fwdiag.Diagnostics, want string) {
	t.Helper()
	if want == "" {
		if diags.HasError() {
			t.Fatalf("unexpected error: %v", diags)
		}
		return
	}
	if !fwDiagsContainErr(diags, want) {
		t.Fatalf("want error containing %q, got %v", want, diags)
	}
}
