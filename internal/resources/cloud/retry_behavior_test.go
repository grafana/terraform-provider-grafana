package cloud

// These tests exercise how the grafana.com resource calls behave for various HTTP status
// codes returned by the API: which codes are retried, which are terminal, and which are
// treated as success (idempotent deletes / adopt-on-conflict). They drive the real CRUD
// functions against a scripted mock of the grafana.com API and assert the number of
// attempts and the resulting diagnostics.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	fwschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
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

const accessPolicyBody = `{"id":"policy-1","name":"test","realms":[],"createdAt":"2024-01-01T00:00:00Z"}`

func TestUnitReadCloudAccessPolicy_StatusCodes(t *testing.T) {
	tests := []struct {
		name         string
		script       []stubResponse
		wantErr      bool
		wantMissing  bool
		wantAttempts int
	}{
		{"200 ok", []stubResponse{{status: 200, body: accessPolicyBody}}, false, false, 1},
		{"429 then 200 (retried)", []stubResponse{retryAfterZero(), {status: 200, body: accessPolicyBody}}, false, false, 2},
		{"503 then 200 (retried)", []stubResponse{{status: 503}, {status: 200, body: accessPolicyBody}}, false, false, 2},
		{"404 removed from state", codes(http.StatusNotFound), false, true, 1},
		{"403 terminal error", codes(http.StatusForbidden), true, false, 1},
		{"400 terminal error", codes(http.StatusBadRequest), true, false, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route := &stubRoute{match: methodContains(http.MethodGet, "/accesspolicies/"), script: tt.script}
			stub := newStubbedGcomClient(t, route)
			d := schema.TestResourceDataRaw(t, resourceAccessPolicy().Schema.Schema, map[string]any{})
			d.SetId("us:policy-1")

			diags := readCloudAccessPolicy(context.Background(), d, stub.client)

			if diags.HasError() != tt.wantErr {
				t.Fatalf("HasError = %v, want %v (diags: %v)", diags.HasError(), tt.wantErr, diags)
			}
			if gotMissing := d.Id() == ""; gotMissing != tt.wantMissing {
				t.Fatalf("resource removed = %v, want %v", gotMissing, tt.wantMissing)
			}
			if route.count != tt.wantAttempts {
				t.Fatalf("attempts = %d, want %d", route.count, tt.wantAttempts)
			}
		})
	}
}

func TestUnitDeleteCloudAccessPolicy_StatusCodes(t *testing.T) {
	// The access policy delete is idempotent: a 404 is treated as success.
	tests := []struct {
		name         string
		script       []stubResponse
		wantErr      bool
		wantAttempts int
	}{
		{"200 ok", codes(http.StatusOK), false, 1},
		{"404 idempotent success", codes(http.StatusNotFound), false, 1},
		{"429 then 200 (retried)", []stubResponse{retryAfterZero(), {status: 200}}, false, 2},
		{"403 terminal error", codes(http.StatusForbidden), true, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route := &stubRoute{match: methodContains(http.MethodDelete, "/accesspolicies/"), script: tt.script}
			stub := newStubbedGcomClient(t, route)
			d := schema.TestResourceDataRaw(t, resourceAccessPolicy().Schema.Schema, map[string]any{})
			d.SetId("us:policy-1")

			diags := deleteCloudAccessPolicy(context.Background(), d, stub.client)

			if diags.HasError() != tt.wantErr {
				t.Fatalf("HasError = %v, want %v (diags: %v)", diags.HasError(), tt.wantErr, diags)
			}
			if route.count != tt.wantAttempts {
				t.Fatalf("attempts = %d, want %d", route.count, tt.wantAttempts)
			}
		})
	}
}

func TestUnitDeleteCloudAccessPolicyToken_StatusCodes(t *testing.T) {
	// Token delete was made idempotent in review: a 404 now counts as success.
	tests := []struct {
		name    string
		script  []stubResponse
		wantErr bool
	}{
		{"200 ok", codes(http.StatusOK), false},
		{"404 idempotent success", codes(http.StatusNotFound), false},
		{"429 then 200 (retried)", []stubResponse{retryAfterZero(), {status: 200}}, false},
		{"500 then 200 (retried)", []stubResponse{{status: 500}, {status: 200}}, false},
		{"403 terminal error", codes(http.StatusForbidden), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route := &stubRoute{match: methodContains(http.MethodDelete, "/tokens/"), script: tt.script}
			stub := newStubbedGcomClient(t, route)
			d := schema.TestResourceDataRaw(t, resourceAccessPolicyToken().Schema.Schema, map[string]any{})
			d.SetId("us:token-1")

			diags := deleteCloudAccessPolicyToken(context.Background(), d, stub.client)

			if diags.HasError() != tt.wantErr {
				t.Fatalf("HasError = %v, want %v (diags: %v)", diags.HasError(), tt.wantErr, diags)
			}
		})
	}
}

func TestUnitDeleteStack_StatusCodes(t *testing.T) {
	// The stack delete is deliberately NOT idempotent: a 404 is a terminal error, in
	// contrast to the token / org-member deletes above.
	tests := []struct {
		name    string
		script  []stubResponse
		wantErr bool
	}{
		{"200 ok", codes(http.StatusOK), false},
		{"404 terminal error (not idempotent)", codes(http.StatusNotFound), true},
		{"503 then 200 (retried)", []stubResponse{{status: 503}, {status: 200}}, false},
		{"403 terminal error", codes(http.StatusForbidden), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route := &stubRoute{match: methodContains(http.MethodDelete, "/stacks/"), script: tt.script}
			stub := newStubbedGcomClient(t, route)
			d := schema.TestResourceDataRaw(t, resourceStack().Schema.Schema, map[string]any{})
			d.SetId("my-stack")

			diags := deleteStack(context.Background(), d, stub.client)

			if diags.HasError() != tt.wantErr {
				t.Fatalf("HasError = %v, want %v (diags: %v)", diags.HasError(), tt.wantErr, diags)
			}
		})
	}
}

// --- Framework org member (idempotency added during review) ---

const orgMemberBody = `{"role":"Editor","billing":0}`

func orgMemberTestSchema(t *testing.T) fwschema.Schema {
	t.Helper()
	r := &orgMemberResource{}
	resp := &fwresource.SchemaResponse{}
	r.Schema(context.Background(), fwresource.SchemaRequest{}, resp)
	return resp.Schema
}

func orgMemberObjectValue(t *testing.T, sch fwschema.Schema, id, org, user, role string, billing bool) tftypes.Value {
	t.Helper()
	objType, ok := sch.Type().TerraformType(context.Background()).(tftypes.Object)
	if !ok {
		t.Fatalf("org member schema is not an object type")
	}
	idVal := tftypes.NewValue(tftypes.String, nil)
	if id != "" {
		idVal = tftypes.NewValue(tftypes.String, id)
	}
	return tftypes.NewValue(objType, map[string]tftypes.Value{
		"id":                     idVal,
		"org":                    tftypes.NewValue(tftypes.String, org),
		"user":                   tftypes.NewValue(tftypes.String, user),
		"role":                   tftypes.NewValue(tftypes.String, role),
		"receive_billing_emails": tftypes.NewValue(tftypes.Bool, billing),
	})
}

func TestUnitOrgMemberReadFromID_StatusCodes(t *testing.T) {
	tests := []struct {
		name         string
		script       []stubResponse
		wantErr      bool
		wantNil      bool
		wantAttempts int
	}{
		{"200 ok", []stubResponse{{status: 200, body: orgMemberBody}}, false, false, 1},
		{"404 not found (no error)", codes(http.StatusNotFound), false, true, 1},
		{"429 then 200 (retried)", []stubResponse{retryAfterZero(), {status: 200, body: orgMemberBody}}, false, false, 2},
		{"403 terminal error", codes(http.StatusForbidden), true, true, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route := &stubRoute{match: methodContains(http.MethodGet, "/members/"), script: tt.script}
			stub := newStubbedGcomClient(t, route)
			r := &orgMemberResource{}
			r.client = stub.client

			data, diags := r.readFromID(context.Background(), "my-org:my-user")

			if diags.HasError() != tt.wantErr {
				t.Fatalf("HasError = %v, want %v (diags: %v)", diags.HasError(), tt.wantErr, diags)
			}
			if gotNil := data == nil; gotNil != tt.wantNil {
				t.Fatalf("data == nil = %v, want %v", gotNil, tt.wantNil)
			}
			if route.count != tt.wantAttempts {
				t.Fatalf("attempts = %d, want %d", route.count, tt.wantAttempts)
			}
		})
	}
}

func TestUnitOrgMemberCreate_StatusCodes(t *testing.T) {
	sch := orgMemberTestSchema(t)
	tests := []struct {
		name       string
		postScript []stubResponse // POST .../members (create)
		getScript  []stubResponse // GET .../members/{user} (existence check + read)
		wantErr    bool
	}{
		{"200 created", codes(http.StatusOK), []stubResponse{{status: 200, body: orgMemberBody}}, false},
		{"409 adopts existing member", codes(http.StatusConflict), []stubResponse{{status: 200, body: orgMemberBody}}, false},
		{"409 but member absent -> error", codes(http.StatusConflict), codes(http.StatusNotFound), true},
		{"503 then 200 (retried)", []stubResponse{{status: 503}, {status: 200}}, []stubResponse{{status: 200, body: orgMemberBody}}, false},
		{"400 terminal error", codes(http.StatusBadRequest), []stubResponse{{status: 200, body: orgMemberBody}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			createRoute := &stubRoute{
				match: func(r *http.Request) bool {
					return r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/members")
				},
				script: tt.postScript,
			}
			getRoute := &stubRoute{match: methodContains(http.MethodGet, "/members/"), script: tt.getScript}
			stub := newStubbedGcomClient(t, createRoute, getRoute)
			r := &orgMemberResource{}
			r.client = stub.client

			req := fwresource.CreateRequest{Plan: tfsdk.Plan{Schema: sch, Raw: orgMemberObjectValue(t, sch, "", "my-org", "my-user", "Editor", false)}}
			resp := &fwresource.CreateResponse{State: tfsdk.State{Schema: sch}}
			r.Create(context.Background(), req, resp)

			if resp.Diagnostics.HasError() != tt.wantErr {
				t.Fatalf("HasError = %v, want %v (diags: %v)", resp.Diagnostics.HasError(), tt.wantErr, resp.Diagnostics)
			}
		})
	}
}

func TestUnitOrgMemberUpdate_StatusCodes(t *testing.T) {
	sch := orgMemberTestSchema(t)
	tests := []struct {
		name           string
		updateScript   []stubResponse // POST .../members/{user}
		recreateScript []stubResponse // POST .../members
		wantErr        bool
	}{
		{"200 updated", codes(http.StatusOK), nil, false},
		{"404 recovers by re-adding member", codes(http.StatusNotFound), codes(http.StatusOK), false},
		{"404 then recreate fails", codes(http.StatusNotFound), codes(http.StatusForbidden), true},
		{"429 then 200 (retried)", []stubResponse{retryAfterZero(), {status: 200}}, nil, false},
		{"403 terminal error", codes(http.StatusForbidden), nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateRoute := &stubRoute{match: methodContains(http.MethodPost, "/members/"), script: tt.updateScript}
			recreateRoute := &stubRoute{
				match: func(r *http.Request) bool {
					return r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/members")
				},
				script: tt.recreateScript,
			}
			getRoute := &stubRoute{match: methodContains(http.MethodGet, "/members/"), script: []stubResponse{{status: 200, body: orgMemberBody}}}
			stub := newStubbedGcomClient(t, updateRoute, recreateRoute, getRoute)
			r := &orgMemberResource{}
			r.client = stub.client

			plan := tfsdk.Plan{Schema: sch, Raw: orgMemberObjectValue(t, sch, "my-org:my-user", "my-org", "my-user", "Editor", false)}
			resp := &fwresource.UpdateResponse{State: tfsdk.State{Schema: sch}}
			r.Update(context.Background(), fwresource.UpdateRequest{Plan: plan}, resp)

			if resp.Diagnostics.HasError() != tt.wantErr {
				t.Fatalf("HasError = %v, want %v (diags: %v)", resp.Diagnostics.HasError(), tt.wantErr, resp.Diagnostics)
			}
		})
	}
}

func TestUnitOrgMemberDelete_StatusCodes(t *testing.T) {
	sch := orgMemberTestSchema(t)
	// Org member delete was made idempotent in review: a 404 counts as success.
	tests := []struct {
		name    string
		script  []stubResponse
		wantErr bool
	}{
		{"200 ok", codes(http.StatusOK), false},
		{"404 idempotent success", codes(http.StatusNotFound), false},
		{"429 then 200 (retried)", []stubResponse{retryAfterZero(), {status: 200}}, false},
		{"403 terminal error", codes(http.StatusForbidden), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route := &stubRoute{match: methodContains(http.MethodDelete, "/members/"), script: tt.script}
			stub := newStubbedGcomClient(t, route)
			r := &orgMemberResource{}
			r.client = stub.client

			state := tfsdk.State{Schema: sch, Raw: orgMemberObjectValue(t, sch, "my-org:my-user", "my-org", "my-user", "Editor", false)}
			resp := &fwresource.DeleteResponse{State: state}
			r.Delete(context.Background(), fwresource.DeleteRequest{State: state}, resp)

			if resp.Diagnostics.HasError() != tt.wantErr {
				t.Fatalf("HasError = %v, want %v (diags: %v)", resp.Diagnostics.HasError(), tt.wantErr, resp.Diagnostics)
			}
		})
	}
}
