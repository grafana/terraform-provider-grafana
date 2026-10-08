package incident

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	incident "github.com/grafana/incident-go"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestUnitIncidentRole_ArchiveSurvivesFailedUpdate pins that an archive the API
// applied is kept in state when the UpdateRole that follows it fails, for
// example on a rename to a name another role already has.
//
// It calls Update directly rather than going through resource.UnitTest: the
// test harness refreshes before every plan, and that refresh reads the archive
// back from the API and hides whatever state the failed apply left behind. A
// practitioner running with -refresh=false, or reading state before the next
// refresh, sees that state as is.
func TestUnitIncidentRole_ArchiveSurvivesFailedUpdate(t *testing.T) {
	var (
		mu    sync.Mutex
		calls []string
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
		mu.Lock()
		calls = append(calls, method)
		mu.Unlock()

		switch method {
		case "RolesService.ArchiveRole":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{})
		case "RolesService.UpdateRole":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": "a role with this name already exists"})
		default:
			t.Errorf("unexpected call to %s", method)
			w.WriteHeader(http.StatusNotImplemented)
		}
	}))
	t.Cleanup(server.Close)

	ctx := context.Background()
	r := &roleResource{client: incident.NewClient(server.URL+"/", "")}

	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	s := schemaResp.Schema

	prior := roleModel{
		ID:          types.StringValue("7"),
		Name:        types.StringValue("commander"),
		Description: types.StringNull(),
		Important:   types.BoolValue(false),
		Mandatory:   types.BoolValue(false),
		Archived:    types.BoolValue(false),
	}
	planned := prior
	planned.Name = types.StringValue("investigator")
	planned.Archived = types.BoolValue(true)

	req := resource.UpdateRequest{
		Plan:  tfsdk.Plan{Schema: s},
		State: tfsdk.State{Schema: s},
	}
	if diags := req.Plan.Set(ctx, &planned); diags.HasError() {
		t.Fatalf("building plan: %v", diags)
	}
	if diags := req.State.Set(ctx, &prior); diags.HasError() {
		t.Fatalf("building prior state: %v", diags)
	}
	// The framework hands Update a response state seeded with the prior state.
	resp := resource.UpdateResponse{State: tfsdk.State{Schema: s, Raw: req.State.Raw.Copy()}}

	r.Update(ctx, req, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected Update to report the UpdateRole failure")
	}
	if got := strings.Join(calls, ","); got != "RolesService.ArchiveRole,RolesService.UpdateRole" {
		t.Fatalf("unexpected calls: %s", got)
	}

	var got roleModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("reading state after Update: %v", diags)
	}
	if !got.Archived.ValueBool() {
		t.Error("state lost the archive the API applied before UpdateRole failed")
	}
	if got.Name.ValueString() != "commander" {
		t.Errorf("state took the rename UpdateRole never applied: name = %q", got.Name.ValueString())
	}
}
