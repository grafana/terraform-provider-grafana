package cloud

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestUnitDeleteStack_StatusCodes(t *testing.T) {
	// The stack delete is deliberately NOT idempotent: a 404 is a terminal error, in
	// contrast to the token / org-member deletes.
	tests := []struct {
		name         string
		script       []stubResponse
		wantErr      string
		wantAttempts int
	}{
		{name: "200 ok", script: codes(http.StatusOK), wantAttempts: 1},
		{name: "404 terminal error (not idempotent)", script: codes(http.StatusNotFound), wantErr: "404 Not Found", wantAttempts: 1},
		{name: "409 terminal error (not retried)", script: codes(http.StatusConflict), wantErr: "409 Conflict", wantAttempts: 1},
		{name: "429 then 200 (retried)", script: []stubResponse{retryAfterZero(), {status: 200}}, wantAttempts: 2},
		{name: "500 then 200 (retried)", script: []stubResponse{{status: 500}, {status: 200}}, wantAttempts: 2},
		{name: "503 then 200 (retried)", script: []stubResponse{{status: 503}, {status: 200}}, wantAttempts: 2},
		{name: "403 terminal error", script: codes(http.StatusForbidden), wantErr: "403 Forbidden", wantAttempts: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route := &stubRoute{match: methodContains(http.MethodDelete, "/stacks/"), script: tt.script}
			stub := newStubbedGcomClient(t, route)
			d := schema.TestResourceDataRaw(t, resourceStack().Schema.Schema, map[string]any{})
			d.SetId("my-stack")

			diags := deleteStack(context.Background(), d, stub.client)

			assertWantErr(t, diags, tt.wantErr)
			if route.count != tt.wantAttempts {
				t.Fatalf("attempts = %d, want %d", route.count, tt.wantAttempts)
			}
		})
	}
}
