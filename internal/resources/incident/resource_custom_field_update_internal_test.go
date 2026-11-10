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
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// concurrentFieldsAPI answers the field calls the resource's Update makes for
// one field, and has an edit land between the read and the write of the first
// conflicts UpdateFields: each of those bumps the version, so the request
// carries a stale one and is refused with fields.version_conflict.
type concurrentFieldsAPI struct {
	t         *testing.T
	mu        sync.Mutex
	field     incident.CustomMetadataField
	conflicts int
	calls     []string
	versions  []int
	options   []incident.CustomMetadataFieldSelectOption
	// refusal, when set, is the error body UpdateField and DeleteField
	// answer with, with a 409.
	refusal map[string]any
}

func (f *concurrentFieldsAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	method := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
	f.calls = append(f.calls, method)

	switch method {
	case "FieldsService.GetField":
		writeJSON(w, http.StatusOK, incident.GetFieldResponse{Field: f.field})
	case "FieldsService.UpdateField":
		var req incident.UpdateFieldRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			f.t.Errorf("decoding UpdateField request: %s", err)
		}
		f.versions = append(f.versions, req.Version)
		if f.refusal != nil {
			writeJSON(w, http.StatusConflict, f.refusal)
			return
		}

		if f.conflicts > 0 {
			f.conflicts--
			f.field.Version++
		}
		if req.Version != f.field.Version {
			writeJSON(w, http.StatusConflict, map[string]any{
				"status": http.StatusConflict,
				"code":   string(incident.ErrFieldVersionConflict),
				"title":  "Field Version Conflict",
				"detail": "The field was changed after it was read",
			})
			return
		}

		f.options = req.Selectoptions
		f.field.Name = req.Name
		f.field.Version++
		writeJSON(w, http.StatusOK, incident.UpdateFieldResponse{Field: f.field})
	case "FieldsService.DeleteField":
		if f.refusal != nil {
			writeJSON(w, http.StatusConflict, f.refusal)
			return
		}
		writeJSON(w, http.StatusOK, incident.DeleteFieldResponse{})
	case "FieldsService.ArchiveField":
		f.field.Archived = true
		writeJSON(w, http.StatusOK, incident.ArchiveFieldResponse{})
	default:
		f.t.Errorf("unexpected call to %s", method)
		w.WriteHeader(http.StatusNotImplemented)
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// renameCustomField runs Update for a rename of the field the API holds.
func renameCustomField(t *testing.T, api *concurrentFieldsAPI) resource.UpdateResponse {
	t.Helper()

	prior := customFieldModel{
		ID:              types.StringValue(api.field.UUID),
		Name:            types.StringValue(api.field.Name),
		Slug:            types.StringValue(api.field.Slug),
		Type:            types.StringValue(api.field.Type),
		Description:     types.StringNull(),
		Color:           types.StringNull(),
		Icon:            types.StringNull(),
		Required:        types.BoolValue(false),
		Archived:        types.BoolValue(false),
		SelectOptions:   types.SetValueMust(customFieldOptionType, []attr.Value{}),
		SelectOptionIDs: types.MapValueMust(types.StringType, map[string]attr.Value{}),
	}
	planned := prior
	planned.Name = types.StringValue("Affected team")
	return updateCustomField(t, api, prior, planned)
}

// updateCustomField runs Update from prior to planned against the API.
func updateCustomField(t *testing.T, api *concurrentFieldsAPI, prior, planned customFieldModel) resource.UpdateResponse {
	t.Helper()

	server := httptest.NewServer(api)
	t.Cleanup(server.Close)

	ctx := context.Background()
	r := &customFieldResource{client: incident.NewClient(server.URL+"/", "")}

	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	s := schemaResp.Schema

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
	resp := resource.UpdateResponse{State: tfsdk.State{Schema: s, Raw: req.State.Raw.Copy()}}

	r.Update(ctx, req, &resp)
	return resp
}

func newConcurrentFieldsAPI(t *testing.T, conflicts int) *concurrentFieldsAPI {
	return &concurrentFieldsAPI{
		t: t,
		field: incident.CustomMetadataField{
			UUID:          "8a0d3c2e-5f4b-4d8e-9c1a-2b3c4d5e6f70",
			Name:          "Team",
			Slug:          "team",
			Type:          customFieldTypeString,
			DomainName:    customFieldDomain,
			Selectoptions: []incident.CustomMetadataFieldSelectOption{},
			Version:       1,
		},
		conflicts: conflicts,
	}
}

// TestUnitIncidentCustomField_UpdateRetriesVersionConflict pins that an edit
// landing between the read and the write does not fail the apply: the field is
// read again and the update sent with the new version.
func TestUnitIncidentCustomField_UpdateRetriesVersionConflict(t *testing.T) {
	api := newConcurrentFieldsAPI(t, 2)

	resp := renameCustomField(t, api)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Update failed: %v", resp.Diagnostics)
	}
	// The field is read back after the update, since the UpdateField response
	// leaves archived options out.
	want := strings.Repeat("FieldsService.GetField,FieldsService.UpdateField,", 3) + "FieldsService.GetField,"
	if got := strings.Join(api.calls, ",") + ","; got != want {
		t.Fatalf("calls = %s, want %s", got, want)
	}
	// Each attempt sends the version its own read returned.
	if got := api.versions; len(got) != 3 || got[0] != 1 || got[1] != 2 || got[2] != 3 {
		t.Errorf("UpdateField versions = %v, want [1 2 3]", got)
	}

	var got customFieldModel
	if diags := resp.State.Get(context.Background(), &got); diags.HasError() {
		t.Fatalf("reading state after Update: %v", diags)
	}
	if got.Name.ValueString() != "Affected team" {
		t.Errorf("name = %q, want %q", got.Name.ValueString(), "Affected team")
	}
}

// TestUnitIncidentCustomField_UpdateGivesUpOnVersionConflicts pins that the
// retry is bounded, and that the field is left as the API holds it.
func TestUnitIncidentCustomField_UpdateGivesUpOnVersionConflicts(t *testing.T) {
	api := newConcurrentFieldsAPI(t, customFieldUpdateAttempts)

	resp := renameCustomField(t, api)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected Update to fail after every attempt conflicted")
	}
	if summary := resp.Diagnostics.Errors()[0].Summary(); summary != "Incident custom field changed during the update" {
		t.Errorf("error summary = %q", summary)
	}
	if got := len(api.calls); got != 2*customFieldUpdateAttempts {
		t.Errorf("made %d calls, want %d: %v", got, 2*customFieldUpdateAttempts, api.calls)
	}
	if api.field.Name != "Team" {
		t.Errorf("the API field was renamed to %q", api.field.Name)
	}
}

// TestUnitIncidentCustomField_UpdateArchives pins the order of an update that
// removes an archived option and archives an option and the field: the removed
// option is left out of the update, which deletes it, the update carries each
// option's flag, and the field is archived and read back last.
func TestUnitIncidentCustomField_UpdateArchives(t *testing.T) {
	api := newConcurrentFieldsAPI(t, 0)
	api.field.Type = customFieldTypeSingleSelect
	api.field.Selectoptions = []incident.CustomMetadataFieldSelectOption{
		{UUID: "u1", Value: "old", Label: "old", Archived: true},
		{UUID: "u2", Value: "high", Label: "high"},
	}

	prior := customFieldModel{
		ID:          types.StringValue(api.field.UUID),
		Name:        types.StringValue(api.field.Name),
		Slug:        types.StringValue(api.field.Slug),
		Type:        types.StringValue(api.field.Type),
		Description: types.StringNull(),
		Color:       types.StringNull(),
		Icon:        types.StringNull(),
		Required:    types.BoolValue(false),
		Archived:    types.BoolValue(false),
		SelectOptions: types.SetValueMust(customFieldOptionType, []attr.Value{
			customFieldArchivedOption("old", types.BoolValue(true)),
			customFieldArchivedOption("high", types.BoolValue(false)),
		}),
		SelectOptionIDs: types.MapValueMust(types.StringType, map[string]attr.Value{
			"old": types.StringValue("u1"), "high": types.StringValue("u2"),
		}),
	}
	planned := prior
	planned.Archived = types.BoolValue(true)
	planned.SelectOptions = types.SetValueMust(customFieldOptionType, []attr.Value{
		customFieldArchivedOption("high", types.BoolValue(true)),
	})

	resp := updateCustomField(t, api, prior, planned)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Update failed: %v", resp.Diagnostics)
	}
	want := "FieldsService.GetField,FieldsService.UpdateField," +
		"FieldsService.GetField,FieldsService.ArchiveField,FieldsService.GetField"
	if got := strings.Join(api.calls, ","); got != want {
		t.Fatalf("calls = %s, want %s", got, want)
	}
	if len(api.options) != 1 || api.options[0].Value != "high" || !api.options[0].Archived {
		t.Errorf("UpdateField options = %+v, want high archived", api.options)
	}
}

// TestUnitIncidentCustomField_UpdateOptionInUse pins that an update refused
// because it removes an option that incidents use names the option and points
// at archiving it.
func TestUnitIncidentCustomField_UpdateOptionInUse(t *testing.T) {
	api := newConcurrentFieldsAPI(t, 0)
	api.field.Type = customFieldTypeSingleSelect
	api.field.Selectoptions = []incident.CustomMetadataFieldSelectOption{{UUID: "u1", Value: "old", Label: "old"}}
	api.refusal = map[string]any{
		"status":  http.StatusConflict,
		"code":    string(incident.ErrFieldOptionInUse),
		"title":   "Field Option In Use",
		"context": map[string]any{"select_option_uuid": "u1", "incidents_count": 3},
	}

	prior := customFieldModel{
		ID:            types.StringValue(api.field.UUID),
		Name:          types.StringValue(api.field.Name),
		Slug:          types.StringValue(api.field.Slug),
		Type:          types.StringValue(api.field.Type),
		Description:   types.StringNull(),
		Color:         types.StringNull(),
		Icon:          types.StringNull(),
		Required:      types.BoolValue(false),
		Archived:      types.BoolValue(false),
		SelectOptions: types.SetValueMust(customFieldOptionType, []attr.Value{customFieldArchivedOption("old", types.BoolValue(false))}),
		SelectOptionIDs: types.MapValueMust(types.StringType, map[string]attr.Value{
			"old": types.StringValue("u1"),
		}),
	}
	planned := prior
	planned.SelectOptions = types.SetValueMust(customFieldOptionType, []attr.Value{})

	resp := updateCustomField(t, api, prior, planned)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected Update to fail")
	}
	got := resp.Diagnostics.Errors()[0]
	if got.Summary() != "Incident custom field select option is in use and cannot be deleted" {
		t.Errorf("error summary = %q", got.Summary())
	}
	for _, want := range []string{`option "old"`, "3 incidents", "archived = true"} {
		if !strings.Contains(got.Detail(), want) {
			t.Errorf("error detail %q does not mention %q", got.Detail(), want)
		}
	}
}

// TestUnitIncidentCustomField_DeleteInUse pins that a delete refused because
// incidents use the field points at archiving it or a removed block.
func TestUnitIncidentCustomField_DeleteInUse(t *testing.T) {
	api := newConcurrentFieldsAPI(t, 0)
	api.refusal = map[string]any{
		"status":  http.StatusConflict,
		"code":    string(incident.ErrFieldInUse),
		"title":   "Field In Use",
		"context": map[string]any{"incidents_count": 1},
	}
	server := httptest.NewServer(api)
	t.Cleanup(server.Close)

	ctx := context.Background()
	r := &customFieldResource{client: incident.NewClient(server.URL+"/", "")}
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)

	state := customFieldModel{
		ID:              types.StringValue(api.field.UUID),
		Name:            types.StringValue(api.field.Name),
		Slug:            types.StringValue(api.field.Slug),
		Type:            types.StringValue(api.field.Type),
		Description:     types.StringNull(),
		Color:           types.StringNull(),
		Icon:            types.StringNull(),
		Required:        types.BoolValue(false),
		Archived:        types.BoolValue(false),
		SelectOptions:   types.SetValueMust(customFieldOptionType, []attr.Value{}),
		SelectOptionIDs: types.MapValueMust(types.StringType, map[string]attr.Value{}),
	}
	req := resource.DeleteRequest{State: tfsdk.State{Schema: schemaResp.Schema}}
	if diags := req.State.Set(ctx, &state); diags.HasError() {
		t.Fatalf("building state: %v", diags)
	}
	var resp resource.DeleteResponse
	r.Delete(ctx, req, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected Delete to fail")
	}
	got := resp.Diagnostics.Errors()[0]
	if got.Summary() != "Incident custom field is in use and cannot be deleted" {
		t.Errorf("error summary = %q", got.Summary())
	}
	for _, want := range []string{"1 incident,", "archived = true", "removed block"} {
		if !strings.Contains(got.Detail(), want) {
			t.Errorf("error detail %q does not mention %q", got.Detail(), want)
		}
	}
}
