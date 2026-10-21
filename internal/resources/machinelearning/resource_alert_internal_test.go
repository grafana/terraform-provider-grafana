package machinelearning

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grafana/machine-learning-go-client/mlapi"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	fwschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func alertTestSchema(t *testing.T) fwschema.Schema {
	t.Helper()
	r := &alertResource{}
	resp := &fwresource.SchemaResponse{}
	r.Schema(context.Background(), fwresource.SchemaRequest{}, resp)
	return resp.Schema
}

func alertTestState(t *testing.T, schema fwschema.Schema) tfsdk.State {
	t.Helper()
	state := tfsdk.State{Schema: schema}
	diags := state.Set(context.Background(), resourceAlertModel{
		ID:               types.StringValue("alert-id"),
		JobID:            types.StringValue("job-id"),
		Title:            types.StringValue("Test alert"),
		AnomalyCondition: types.StringValue("any"),
		For:              types.StringValue("0s"),
		Window:           types.StringValue("0s"),
		Labels:           types.MapNull(types.StringType),
		Annotations:      types.MapNull(types.StringType),
	})
	if diags.HasError() {
		t.Fatalf("setting alert state: %v", diags)
	}
	return state
}

func alertTestResource(t *testing.T, method string, status int) *alertResource {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != method {
			t.Errorf("method = %q, want %q", req.Method, method)
		}
		wantPath := "/manage/api/v1/jobs/job-id/alerts/alert-id"
		if req.URL.Path != wantPath {
			t.Errorf("path = %q, want %q", req.URL.Path, wantPath)
		}
		w.WriteHeader(status)
		if status >= http.StatusBadRequest {
			_, _ = fmt.Fprint(w, `{"status":"error","error":"test error"}`)
		}
	}))
	t.Cleanup(server.Close)

	client, err := mlapi.New(server.URL, mlapi.Config{})
	if err != nil {
		t.Fatalf("creating ML API client: %v", err)
	}
	return &alertResource{mlapi: client}
}

func TestUnitAlertReadStatusCodes(t *testing.T) {
	schema := alertTestSchema(t)
	tests := []struct {
		name        string
		status      int
		wantError   bool
		wantRemoved bool
	}{
		{name: "404 removes resource from state", status: http.StatusNotFound, wantRemoved: true},
		{name: "500 returns an error", status: http.StatusInternalServerError, wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := alertTestState(t, schema)
			r := alertTestResource(t, http.MethodGet, tt.status)
			resp := &fwresource.ReadResponse{State: state}

			r.Read(context.Background(), fwresource.ReadRequest{State: state}, resp)

			if got := resp.Diagnostics.HasError(); got != tt.wantError {
				t.Fatalf("diagnostics contain error = %v, want %v: %v", got, tt.wantError, resp.Diagnostics)
			}
			if got := resp.State.Raw.IsNull(); got != tt.wantRemoved {
				t.Fatalf("resource removed from state = %v, want %v", got, tt.wantRemoved)
			}
		})
	}
}

func TestUnitAlertDeleteStatusCodes(t *testing.T) {
	schema := alertTestSchema(t)
	tests := []struct {
		name      string
		status    int
		wantError bool
	}{
		{name: "404 is an idempotent success", status: http.StatusNotFound},
		{name: "500 returns an error", status: http.StatusInternalServerError, wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := alertTestState(t, schema)
			r := alertTestResource(t, http.MethodDelete, tt.status)
			resp := &fwresource.DeleteResponse{State: state}

			r.Delete(context.Background(), fwresource.DeleteRequest{State: state}, resp)

			if got := resp.Diagnostics.HasError(); got != tt.wantError {
				t.Fatalf("diagnostics contain error = %v, want %v: %v", got, tt.wantError, resp.Diagnostics)
			}
		})
	}
}
