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
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

const serverAssignedFieldUUID = "5c1f2f0e-7a53-4c43-b3b5-0d6f4a1d9e21"

// createFieldsAPI answers the field calls the resource's Create makes. It
// stores the field AddField receives, under the UUID the request proposes or,
// when keepsProposedUUID is false, under one of its own, and answers GetField
// only for the UUID it stored.
type createFieldsAPI struct {
	t                 *testing.T
	keepsProposedUUID bool

	mu           sync.Mutex
	calls        []string
	proposedUUID string
	stored       *incident.CustomMetadataField
}

func (f *createFieldsAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
		if !f.keepsProposedUUID {
			field.UUID = serverAssignedFieldUUID
		}
		field.Version = 1
		f.stored = &field
		writeJSON(w, http.StatusOK, incident.AddFieldResponse{Field: field})
	case "FieldsService.GetField":
		var req incident.GetFieldRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			f.t.Errorf("decoding GetField request: %s", err)
		}
		if f.stored == nil || f.stored.UUID != req.FieldUUID {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"status": http.StatusNotFound,
				"code":   string(incident.ErrFieldNotFound),
				"title":  "Field Not Found",
				"detail": "The field does not exist",
			})
			return
		}
		writeJSON(w, http.StatusOK, incident.GetFieldResponse{Field: *f.stored})
	default:
		f.t.Errorf("unexpected call to %s", method)
		w.WriteHeader(http.StatusNotImplemented)
	}
}

// createCustomField runs Create for an unarchived string field against the API.
func createCustomField(t *testing.T, api *createFieldsAPI) resource.CreateResponse {
	t.Helper()

	server := httptest.NewServer(api)
	t.Cleanup(server.Close)

	ctx := context.Background()
	r := &customFieldResource{client: incident.NewClient(server.URL+"/", "")}

	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
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

// TestUnitIncidentCustomField_CreateUsesStoredUUID pins that the ID Create
// records is the UUID the API stored the field under, whether or not the API
// kept the UUID Create proposed. If it did not, the proposed UUID names no
// field, and using it would fail the create and orphan the field the API made.
func TestUnitIncidentCustomField_CreateUsesStoredUUID(t *testing.T) {
	for name, keepsProposedUUID := range map[string]bool{
		"api keeps the proposed uuid": true,
		"api assigns its own uuid":    false,
	} {
		t.Run(name, func(t *testing.T) {
			api := &createFieldsAPI{t: t, keepsProposedUUID: keepsProposedUUID}

			resp := createCustomField(t, api)

			if resp.Diagnostics.HasError() {
				t.Fatalf("Create failed: %v", resp.Diagnostics)
			}
			if api.proposedUUID == "" {
				t.Fatal("Create did not propose a UUID, which the recovery from a lost AddField response needs")
			}
			if keepsProposedUUID == (api.stored.UUID != api.proposedUUID) {
				t.Fatalf("fake API stored UUID %q for proposed UUID %q, which is not the case under test", api.stored.UUID, api.proposedUUID)
			}

			var got customFieldModel
			if diags := resp.State.Get(context.Background(), &got); diags.HasError() {
				t.Fatalf("reading state after Create: %v", diags)
			}
			if got.ID.ValueString() != api.stored.UUID {
				t.Errorf("id = %q, want the stored UUID %q", got.ID.ValueString(), api.stored.UUID)
			}
		})
	}
}
