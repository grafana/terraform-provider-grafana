package agento11y

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common/agento11yapi"
)

// validateCollectionAttribute runs the schema validators wired to one collection
// attribute and reports whether they rejected the value.
func validateCollectionAttribute(t *testing.T, name, value string) bool {
	t.Helper()

	schemaResp := &resource.SchemaResponse{}
	(&collectionResource{}).Schema(context.Background(), resource.SchemaRequest{}, schemaResp)

	attribute, ok := schemaResp.Schema.Attributes[name].(schema.StringAttribute)
	if !ok {
		t.Fatalf("attribute %q is not a string attribute", name)
	}

	req := validator.StringRequest{
		Path:        path.Root(name),
		ConfigValue: types.StringValue(value),
	}
	resp := &validator.StringResponse{}
	for _, v := range attribute.Validators {
		v.ValidateString(context.Background(), req, resp)
	}
	return resp.Diagnostics.HasError()
}

func TestUnitCollectionSchemaValidators(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		attribute  string
		value      string
		wantReject bool
	}{
		{"name accepts a plain value", "name", "Failed evaluations", false},
		{"name accepts inner whitespace", "name", "Failed\nevaluations", false},
		{"name rejects leading whitespace", "name", " padded", true},
		{"name rejects trailing whitespace", "name", "padded ", true},
		{"name rejects a leading non-breaking space", "name", "\u00a0padded", true},
		{"name rejects a trailing vertical tab", "name", "padded\v", true},
		{"name rejects an empty value", "name", "", true},
		{"name accepts the column limit", "name", strings.Repeat("a", collectionNameMaxLength), false},
		{"name accepts a multi-byte value at the column limit", "name", strings.Repeat("\u044f", collectionNameMaxLength), false},
		{"name rejects one character over the column limit", "name", strings.Repeat("a", collectionNameMaxLength+1), true},
		{"description accepts a plain value", "description", "Conversations where every evaluator failed.", false},
		{"description rejects an empty value", "description", "", true},
		{"description rejects leading whitespace", "description", " padded", true},
		{"description rejects trailing whitespace", "description", "padded ", true},
		{"description rejects a lone vertical tab", "description", "\v", true},
		{"description accepts the column limit", "description", strings.Repeat("a", collectionDescriptionMaxLength), false},
		{"description rejects one character over the column limit", "description", strings.Repeat("a", collectionDescriptionMaxLength+1), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := validateCollectionAttribute(t, tt.attribute, tt.value); got != tt.wantReject {
				t.Fatalf("%s = %q: rejected = %v, want %v", tt.attribute, tt.value, got, tt.wantReject)
			}
		})
	}
}

func TestUnitCollectionToModel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		collection      agento11yapi.Collection
		wantDescription types.String
	}{
		{
			name:            "description present",
			collection:      agento11yapi.Collection{CollectionID: "uuid-1", Name: "Failed evaluations", Description: "Every evaluator failed."},
			wantDescription: types.StringValue("Every evaluator failed."),
		},
		{
			// A cleared description must read back as null, so that omitting the
			// attribute does not produce a perpetual diff.
			name:            "description cleared server-side",
			collection:      agento11yapi.Collection{CollectionID: "uuid-1", Name: "Failed evaluations"},
			wantDescription: types.StringNull(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := collectionToModel(tt.collection)
			if got.ID != types.StringValue(tt.collection.CollectionID) {
				t.Fatalf("id = %v, want %v", got.ID, tt.collection.CollectionID)
			}
			if got.Name != types.StringValue(tt.collection.Name) {
				t.Fatalf("name = %v, want %v", got.Name, tt.collection.Name)
			}
			if got.Description != tt.wantDescription {
				t.Fatalf("description = %v, want %v", got.Description, tt.wantDescription)
			}
		})
	}
}
