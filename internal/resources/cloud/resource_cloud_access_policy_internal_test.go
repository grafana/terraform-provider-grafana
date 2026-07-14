package cloud

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const accessPolicyBody = `{"id":"policy-1","name":"test","realms":[],"createdAt":"2024-01-01T00:00:00Z"}`

func TestUnitReadCloudAccessPolicy_StatusCodes(t *testing.T) {
	tests := []struct {
		name         string
		script       []stubResponse
		wantErr      string // substring the error diagnostic must contain; empty means no error
		wantMissing  bool
		wantAttempts int
	}{
		{name: "200 ok", script: []stubResponse{{status: 200, body: accessPolicyBody}}, wantAttempts: 1},
		{name: "429 then 200 (retried)", script: []stubResponse{retryAfterZero(), {status: 200, body: accessPolicyBody}}, wantAttempts: 2},
		{name: "500 then 200 (retried)", script: []stubResponse{{status: 500}, {status: 200, body: accessPolicyBody}}, wantAttempts: 2},
		{name: "503 then 200 (retried)", script: []stubResponse{{status: 503}, {status: 200, body: accessPolicyBody}}, wantAttempts: 2},
		{name: "504 then 200 (retried)", script: []stubResponse{{status: 504}, {status: 200, body: accessPolicyBody}}, wantAttempts: 2},
		{name: "404 removed from state", script: codes(http.StatusNotFound), wantMissing: true, wantAttempts: 1},
		{name: "403 terminal error", script: codes(http.StatusForbidden), wantErr: "403 Forbidden", wantAttempts: 1},
		{name: "400 terminal error", script: codes(http.StatusBadRequest), wantErr: "400 Bad Request", wantAttempts: 1},
		{name: "409 terminal error (not retried)", script: codes(http.StatusConflict), wantErr: "409 Conflict", wantAttempts: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route := &stubRoute{match: methodContains(http.MethodGet, "/accesspolicies/"), script: tt.script}
			stub := newStubbedGcomClient(t, route)
			d := schema.TestResourceDataRaw(t, resourceAccessPolicy().Schema.Schema, map[string]any{})
			d.SetId("us:policy-1")

			diags := readCloudAccessPolicy(context.Background(), d, stub.client)

			assertWantErr(t, diags, tt.wantErr)
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
		wantErr      string
		wantAttempts int
	}{
		{name: "200 ok", script: codes(http.StatusOK), wantAttempts: 1},
		{name: "404 idempotent success", script: codes(http.StatusNotFound), wantAttempts: 1},
		{name: "429 then 200 (retried)", script: []stubResponse{retryAfterZero(), {status: 200}}, wantAttempts: 2},
		{name: "500 then 200 (retried)", script: []stubResponse{{status: 500}, {status: 200}}, wantAttempts: 2},
		{name: "503 then 200 (retried)", script: []stubResponse{{status: 503}, {status: 200}}, wantAttempts: 2},
		{name: "400 terminal error", script: codes(http.StatusBadRequest), wantErr: "400 Bad Request", wantAttempts: 1},
		{name: "403 terminal error", script: codes(http.StatusForbidden), wantErr: "403 Forbidden", wantAttempts: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route := &stubRoute{match: methodContains(http.MethodDelete, "/accesspolicies/"), script: tt.script}
			stub := newStubbedGcomClient(t, route)
			d := schema.TestResourceDataRaw(t, resourceAccessPolicy().Schema.Schema, map[string]any{})
			d.SetId("us:policy-1")

			diags := deleteCloudAccessPolicy(context.Background(), d, stub.client)

			assertWantErr(t, diags, tt.wantErr)
			if route.count != tt.wantAttempts {
				t.Fatalf("attempts = %d, want %d", route.count, tt.wantAttempts)
			}
		})
	}
}
