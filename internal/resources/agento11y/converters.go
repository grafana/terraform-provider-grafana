package agento11y

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func listValueToStrings(ctx context.Context, list types.List) ([]string, diag.Diagnostics) {
	if list.IsNull() || list.IsUnknown() {
		return nil, nil
	}
	var diags diag.Diagnostics
	var values []string
	diags.Append(list.ElementsAs(ctx, &values, false)...)
	return values, diags
}

func stringsToListValue(ctx context.Context, values []string) (types.List, diag.Diagnostics) {
	if len(values) == 0 {
		return types.ListNull(types.StringType), nil
	}
	return types.ListValueFrom(ctx, types.StringType, values)
}

func stringValueOrNull(value string) types.String {
	if value == "" {
		return types.StringNull()
	}
	return types.StringValue(value)
}

// rawJSONFromNormalized returns the JSON bytes of a normalized JSON attribute,
// or nil when the attribute is null/unknown/empty.
func rawJSONFromNormalized(v jsontypes.Normalized) json.RawMessage {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	s := v.ValueString()
	if s == "" {
		return nil
	}
	return json.RawMessage(s)
}

// normalizedFromRawJSON converts JSON bytes from the API into a normalized JSON
// attribute value, returning null when empty.
func normalizedFromRawJSON(raw json.RawMessage) jsontypes.Normalized {
	if len(raw) == 0 {
		return jsontypes.NewNormalizedNull()
	}
	return jsontypes.NewNormalizedValue(string(raw))
}

// normalizedMatchOrNull converts a `match` JSON payload from the API into a
// normalized attribute value. An empty, null, or empty-object payload is
// treated as null so that omitting `match` (match everything) does not produce
// a perpetual diff.
func normalizedMatchOrNull(raw json.RawMessage) jsontypes.Normalized {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" || trimmed == "{}" {
		return jsontypes.NewNormalizedNull()
	}
	return jsontypes.NewNormalizedValue(trimmed)
}

// int64PtrValue converts an optional int pointer to a framework value.
func int64PtrValue(v *int) types.Int64 {
	if v == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*v))
}
