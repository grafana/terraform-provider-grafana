package incident

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"

	incident "github.com/grafana/incident-go"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

const existingFieldUUID = "5c1f2f0e-7a53-4c43-b3b5-0d6f4a1d9e21"

// storedFieldsAPI answers the field calls the resource's Create and ImportState
// make. It holds the fields in others, and stores the field AddField receives
// under the UUID the request proposes, as the API does.
type storedFieldsAPI struct {
	t *testing.T
	// others are the fields that exist before the test runs.
	others []incident.CustomMetadataField
	// addFieldError, when set, is the error body AddField answers with, with
	// addFieldStatus. When storeOnError is also set, the field is stored first,
	// as when the API stores the field but the response is lost.
	addFieldError  map[string]any
	addFieldStatus int
	storeOnError   bool

	mu           sync.Mutex
	calls        []string
	proposedUUID string
	stored       *incident.CustomMetadataField
}

func (f *storedFieldsAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	method := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
	f.calls = append(f.calls, method)

	switch method {
	case "FieldsService.AddField":
		var req incident.AddFieldRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			f.t.Errorf("decoding AddField request: %s", err)
		}
		f.proposedUUID = req.Field.UUID
		field := req.Field
		field.Version = 1
		if f.addFieldError == nil || f.storeOnError {
			f.stored = &field
		}
		if f.addFieldError != nil {
			writeJSON(w, f.addFieldStatus, f.addFieldError)
			return
		}
		writeJSON(w, http.StatusOK, incident.AddFieldResponse{Field: field})
	case "FieldsService.GetField":
		var req incident.GetFieldRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			f.t.Errorf("decoding GetField request: %s", err)
		}
		for _, field := range f.fields() {
			if field.UUID == req.FieldUUID {
				writeJSON(w, http.StatusOK, incident.GetFieldResponse{Field: field})
				return
			}
		}
		writeJSON(w, http.StatusNotFound, map[string]any{
			"status": http.StatusNotFound,
			"code":   string(incident.ErrFieldNotFound),
			"title":  "Field Not Found",
			"detail": "The field does not exist",
		})
	case "FieldsService.GetFields":
		writeJSON(w, http.StatusOK, incident.GetFieldsResponse{Fields: f.fields()})
	default:
		f.t.Errorf("unexpected call to %s", method)
		w.WriteHeader(http.StatusNotImplemented)
	}
}

// fields returns the fields the API holds.
func (f *storedFieldsAPI) fields() []incident.CustomMetadataField {
	fields := slices.Clone(f.others)
	if f.stored != nil {
		fields = append(fields, *f.stored)
	}
	return fields
}

// newCustomFieldResource returns a resource whose client talks to the API.
func newCustomFieldResource(t *testing.T, api http.Handler) (*customFieldResource, resource.SchemaResponse) {
	t.Helper()

	server := httptest.NewServer(api)
	t.Cleanup(server.Close)

	r := &customFieldResource{client: incident.NewClient(server.URL+"/", "")}
	var schemaResp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	return r, schemaResp
}

// createCustomField runs Create for an unarchived string field with the slug
// team against the API.
func createCustomField(t *testing.T, api *storedFieldsAPI) resource.CreateResponse {
	t.Helper()

	ctx := context.Background()
	r, schemaResp := newCustomFieldResource(t, api)
	s := schemaResp.Schema

	planned := customFieldModel{
		ID:              types.StringUnknown(),
		Name:            types.StringValue("Team"),
		Slug:            types.StringValue("team"),
		Type:            types.StringValue(customFieldTypeString),
		Description:     types.StringNull(),
		Color:           types.StringNull(),
		Icon:            types.StringNull(),
		Required:        types.BoolValue(false),
		Archived:        types.BoolValue(false),
		SelectOptions:   types.SetValueMust(customFieldOptionType, []attr.Value{}),
		SelectOptionIDs: types.MapUnknown(types.StringType),
	}
	req := resource.CreateRequest{Plan: tfsdk.Plan{Schema: s}}
	if diags := req.Plan.Set(ctx, &planned); diags.HasError() {
		t.Fatalf("building plan: %v", diags)
	}
	resp := resource.CreateResponse{State: tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}}

	r.Create(ctx, req, &resp)
	return resp
}

// createdID returns the ID Create recorded in the state.
func createdID(t *testing.T, resp resource.CreateResponse) string {
	t.Helper()

	var got customFieldModel
	if diags := resp.State.Get(context.Background(), &got); diags.HasError() {
		t.Fatalf("reading state after Create: %v", diags)
	}
	return got.ID.ValueString()
}

// serverError is the error body of a request that failed on the server.
var serverError = map[string]any{
	"status": http.StatusInternalServerError,
	"code":   "internal_error",
	"title":  "Internal Error",
	"detail": "The request could not be completed",
}

// TestUnitIncidentCustomField_CreateRecordsProposedUUID pins that Create
// records the UUID it proposed, which the API keeps.
func TestUnitIncidentCustomField_CreateRecordsProposedUUID(t *testing.T) {
	api := &storedFieldsAPI{t: t}

	resp := createCustomField(t, api)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Create failed: %v", resp.Diagnostics)
	}
	if api.proposedUUID == "" {
		t.Fatal("Create did not propose a UUID, which the recovery from a lost AddField response needs")
	}
	if got := createdID(t, resp); got != api.proposedUUID {
		t.Errorf("id = %q, want the proposed UUID %q", got, api.proposedUUID)
	}
}

// TestUnitIncidentCustomField_CreateRecoversStoredField pins that when AddField
// fails after the API stored the field, as when the response is lost, Create
// finds the field under the UUID it proposed and records it, instead of failing
// and leaving the field untracked.
func TestUnitIncidentCustomField_CreateRecoversStoredField(t *testing.T) {
	api := &storedFieldsAPI{t: t, addFieldError: serverError, addFieldStatus: http.StatusInternalServerError, storeOnError: true}

	resp := createCustomField(t, api)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Create failed although the API stored the field: %v", resp.Diagnostics)
	}
	if got := createdID(t, resp); got != api.proposedUUID {
		t.Errorf("id = %q, want the proposed UUID %q", got, api.proposedUUID)
	}
	if got, want := strings.Join(api.calls, ","), "FieldsService.AddField,FieldsService.GetField,FieldsService.GetField"; got != want {
		t.Errorf("calls = %s, want %s", got, want)
	}
}

// TestUnitIncidentCustomField_CreateFailsWhenNothingStored pins that when
// AddField fails and the API stored nothing, Create fails with the AddField
// error and records nothing.
func TestUnitIncidentCustomField_CreateFailsWhenNothingStored(t *testing.T) {
	api := &storedFieldsAPI{t: t, addFieldError: serverError, addFieldStatus: http.StatusInternalServerError}

	resp := createCustomField(t, api)

	if !resp.Diagnostics.HasError() {
		t.Fatal("Create succeeded although the API stored nothing")
	}
	errs := resp.Diagnostics.Errors()
	if got, want := errs[0].Summary(), "Failed to create incident custom field"; got != want {
		t.Errorf("error summary = %q, want %q", got, want)
	}
	if detail := errs[0].Detail(); !strings.Contains(detail, "The request could not be completed") || strings.Contains(detail, "is held by") {
		t.Errorf("error detail = %q, want the AddField error and no slug hint", detail)
	}
	if !resp.State.Raw.IsNull() {
		t.Errorf("Create recorded state although the API stored nothing: %v", resp.State.Raw)
	}
	if got, want := strings.Join(api.calls, ","), "FieldsService.AddField,FieldsService.GetField"; got != want {
		t.Errorf("calls = %s, want %s", got, want)
	}
}

// TestUnitIncidentCustomField_CreateSlugConflictNamesHolder pins that when
// AddField is refused because another field holds the slug, the error names
// that field. The holder is archived, since archived fields keep their slugs.
func TestUnitIncidentCustomField_CreateSlugConflictNamesHolder(t *testing.T) {
	api := &storedFieldsAPI{
		t: t,
		others: []incident.CustomMetadataField{
			{UUID: "0a9a3a43-5a8e-4d3c-9d1c-6d0f7c3f2b10", Name: "Region", Slug: "region", Type: customFieldTypeString, DomainName: customFieldDomain},
			{UUID: existingFieldUUID, Name: "Old team", Slug: "team", Type: customFieldTypeString, DomainName: customFieldDomain, Archived: true},
		},
		addFieldError: map[string]any{
			"status": http.StatusConflict,
			"code":   string(incident.ErrFieldSlugConflict),
			"title":  "Field Slug Conflict",
			"detail": "A field with this slug already exists",
		},
		addFieldStatus: http.StatusConflict,
	}

	resp := createCustomField(t, api)

	if !resp.Diagnostics.HasError() {
		t.Fatal("Create succeeded although the slug is taken")
	}
	detail := resp.Diagnostics.Errors()[0].Detail()
	for _, want := range []string{
		`Slug "team" is held by custom field "Old team" (` + existingFieldUUID + `)`,
		"archived ones included",
	} {
		if !strings.Contains(detail, want) {
			t.Errorf("error detail = %q, want it to contain %q", detail, want)
		}
	}
	if got, want := strings.Join(api.calls, ","), "FieldsService.AddField,FieldsService.GetField,FieldsService.GetFields"; got != want {
		t.Errorf("calls = %s, want %s", got, want)
	}
}

// TestUnitIncidentCustomField_ImportState pins which fields can be imported:
// only fields in the incident domain, of a type the resource supports, that
// the API lets users change.
func TestUnitIncidentCustomField_ImportState(t *testing.T) {
	incidentField := incident.CustomMetadataField{
		UUID:       existingFieldUUID,
		Name:       "Team",
		Slug:       "team",
		Type:       customFieldTypeString,
		DomainName: customFieldDomain,
	}

	for name, tc := range map[string]struct {
		field     *incident.CustomMetadataField
		wantError string
	}{
		"incident field": {
			field: &incidentField,
		},
		"missing field": {
			wantError: "No incident custom field with UUID",
		},
		"field synced with Grafana labels": {
			field: func() *incident.CustomMetadataField {
				f := incidentField
				f.DomainName = "labels"
				return &f
			}(),
			wantError: `is in the "labels" domain`,
		},
		"system-managed field": {
			field: func() *incident.CustomMetadataField {
				f := incidentField
				f.Immutable = true
				return &f
			}(),
			wantError: "is generated by Grafana",
		},
		"unsupported type": {
			field: func() *incident.CustomMetadataField {
				f := incidentField
				f.Type = "url"
				return &f
			}(),
			wantError: "is a url field",
		},
	} {
		t.Run(name, func(t *testing.T) {
			api := &storedFieldsAPI{t: t}
			if tc.field != nil {
				api.others = []incident.CustomMetadataField{*tc.field}
			}
			ctx := context.Background()
			r, schemaResp := newCustomFieldResource(t, api)
			s := schemaResp.Schema
			resp := resource.ImportStateResponse{State: tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}}

			r.ImportState(ctx, resource.ImportStateRequest{ID: existingFieldUUID}, &resp)

			if tc.wantError == "" {
				if resp.Diagnostics.HasError() {
					t.Fatalf("ImportState failed: %v", resp.Diagnostics)
				}
				var got customFieldModel
				if diags := resp.State.Get(ctx, &got); diags.HasError() {
					t.Fatalf("reading state after ImportState: %v", diags)
				}
				if got.ID.ValueString() != existingFieldUUID || got.Slug.ValueString() != "team" {
					t.Errorf("imported id %q and slug %q, want %q and team", got.ID.ValueString(), got.Slug.ValueString(), existingFieldUUID)
				}
				return
			}

			if !resp.Diagnostics.HasError() {
				t.Fatal("ImportState succeeded, want it refused")
			}
			if detail := resp.Diagnostics.Errors()[0].Detail(); !strings.Contains(detail, tc.wantError) {
				t.Errorf("error detail = %q, want it to contain %q", detail, tc.wantError)
			}
			if !resp.State.Raw.IsNull() {
				t.Errorf("ImportState recorded state for a refused field: %v", resp.State.Raw)
			}
		})
	}
}
