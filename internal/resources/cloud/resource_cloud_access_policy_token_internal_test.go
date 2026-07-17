package cloud

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestUnitDeleteCloudAccessPolicyToken_StatusCodes(t *testing.T) {
	// Token delete was made idempotent in review: a 404 now counts as success. The status-code
	// matrix mirrors the other idempotent deletes (access policy / org member).
	tests := []struct {
		name         string
		script       []stubResponse
		wantErr      string
		wantAttempts int
	}{
		{name: "200 ok", script: codes(http.StatusOK), wantAttempts: 1},
		{name: "404 idempotent success", script: codes(http.StatusNotFound), wantAttempts: 1},
		{name: "400 terminal error", script: codes(http.StatusBadRequest), wantErr: "400 Bad Request", wantAttempts: 1},
		{name: "403 terminal error", script: codes(http.StatusForbidden), wantErr: "403 Forbidden", wantAttempts: 1},
		{name: "409 terminal error (not retried)", script: codes(http.StatusConflict), wantErr: "409 Conflict", wantAttempts: 1},
		{name: "429 then 200 (retried)", script: []stubResponse{retryAfterZero(), {status: 200}}, wantAttempts: 2},
		{name: "500 then 200 (retried)", script: []stubResponse{{status: 500}, {status: 200}}, wantAttempts: 2},
		{name: "503 then 200 (retried)", script: []stubResponse{{status: 503}, {status: 200}}, wantAttempts: 2},
		{name: "504 then 200 (retried)", script: []stubResponse{{status: 504}, {status: 200}}, wantAttempts: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route := &stubRoute{match: methodContains(http.MethodDelete, "/tokens/"), script: tt.script}
			stub := newStubbedGcomClient(t, route)
			d := schema.TestResourceDataRaw(t, resourceAccessPolicyToken().Schema.Schema, map[string]any{})
			d.SetId("us:token-1")

			diags := deleteCloudAccessPolicyToken(context.Background(), d, stub.client)

			assertWantErr(t, diags, tt.wantErr)
			if route.count != tt.wantAttempts {
				t.Fatalf("attempts = %d, want %d", route.count, tt.wantAttempts)
			}
		})
	}
}
