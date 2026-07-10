package cloud

import (
	"context"
	"net/http"
	"strings"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	fwschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

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
		wantErr      string
		wantNil      bool
		wantAttempts int
	}{
		{name: "200 ok", script: []stubResponse{{status: 200, body: orgMemberBody}}, wantAttempts: 1},
		{name: "404 not found (no error)", script: codes(http.StatusNotFound), wantNil: true, wantAttempts: 1},
		{name: "429 then 200 (retried)", script: []stubResponse{retryAfterZero(), {status: 200, body: orgMemberBody}}, wantAttempts: 2},
		{name: "403 terminal error", script: codes(http.StatusForbidden), wantErr: "403 Forbidden", wantNil: true, wantAttempts: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route := &stubRoute{match: methodContains(http.MethodGet, "/members/"), script: tt.script}
			stub := newStubbedGcomClient(t, route)
			r := &orgMemberResource{}
			r.client = stub.client

			data, diags := r.readFromID(context.Background(), "my-org:my-user")

			assertWantErrFw(t, diags, tt.wantErr)
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
		wantErr    string
	}{
		{name: "200 created", postScript: codes(http.StatusOK), getScript: []stubResponse{{status: 200, body: orgMemberBody}}},
		{name: "409 adopts existing member", postScript: codes(http.StatusConflict), getScript: []stubResponse{{status: 200, body: orgMemberBody}}},
		{name: "409 but member absent -> error", postScript: codes(http.StatusConflict), getScript: codes(http.StatusNotFound), wantErr: "409 Conflict"},
		{name: "503 then 200 (retried)", postScript: []stubResponse{{status: 503}, {status: 200}}, getScript: []stubResponse{{status: 200, body: orgMemberBody}}},
		{name: "400 terminal error", postScript: codes(http.StatusBadRequest), getScript: []stubResponse{{status: 200, body: orgMemberBody}}, wantErr: "400 Bad Request"},
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

			assertWantErrFw(t, resp.Diagnostics, tt.wantErr)
		})
	}
}

func TestUnitOrgMemberUpdate_StatusCodes(t *testing.T) {
	sch := orgMemberTestSchema(t)
	tests := []struct {
		name           string
		updateScript   []stubResponse // POST .../members/{user}
		recreateScript []stubResponse // POST .../members
		wantErr        string
	}{
		{name: "200 updated", updateScript: codes(http.StatusOK)},
		{name: "404 recovers by re-adding member", updateScript: codes(http.StatusNotFound), recreateScript: codes(http.StatusOK)},
		{name: "404 then recreate fails", updateScript: codes(http.StatusNotFound), recreateScript: codes(http.StatusForbidden), wantErr: "403 Forbidden"},
		{name: "429 then 200 (retried)", updateScript: []stubResponse{retryAfterZero(), {status: 200}}},
		{name: "403 terminal error", updateScript: codes(http.StatusForbidden), wantErr: "403 Forbidden"},
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

			assertWantErrFw(t, resp.Diagnostics, tt.wantErr)
		})
	}
}

func TestUnitOrgMemberDelete_StatusCodes(t *testing.T) {
	sch := orgMemberTestSchema(t)
	// Org member delete was made idempotent in review: a 404 counts as success.
	tests := []struct {
		name    string
		script  []stubResponse
		wantErr string
	}{
		{name: "200 ok", script: codes(http.StatusOK)},
		{name: "404 idempotent success", script: codes(http.StatusNotFound)},
		{name: "429 then 200 (retried)", script: []stubResponse{retryAfterZero(), {status: 200}}},
		{name: "403 terminal error", script: codes(http.StatusForbidden), wantErr: "403 Forbidden"},
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

			assertWantErrFw(t, resp.Diagnostics, tt.wantErr)
		})
	}
}
