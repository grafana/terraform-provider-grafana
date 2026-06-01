package assistant

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common/assistantapi"
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

func boolPtr(b bool) *bool {
	return &b
}

func stringPtr(s string) *string {
	return &s
}

func headersFromMap(headers types.Map) ([]assistantapi.Header, diag.Diagnostics) {
	if headers.IsNull() || headers.IsUnknown() {
		return nil, nil
	}
	var diags diag.Diagnostics
	elements := headers.Elements()
	result := make([]assistantapi.Header, 0, len(elements))
	for key, value := range elements {
		val, ok := value.(types.String)
		if !ok {
			diags.AddError("Invalid custom header value", "custom header values must be strings")
			return nil, diags
		}
		result = append(result, assistantapi.Header{
			Key:   key,
			Value: val.ValueString(),
		})
	}
	return result, diags
}

func rawJSONFromString(ctx context.Context, s types.String) (json.RawMessage, diag.Diagnostics) {
	if s.IsNull() || s.IsUnknown() || s.ValueString() == "" {
		return nil, nil
	}
	if !json.Valid([]byte(s.ValueString())) {
		return nil, diag.Diagnostics{diag.NewErrorDiagnostic("Invalid JSON", "context_items must be valid JSON")}
	}
	return json.RawMessage(s.ValueString()), nil
}

func stringFromRawJSON(raw json.RawMessage) types.String {
	if len(raw) == 0 {
		return types.StringNull()
	}
	return types.StringValue(string(raw))
}

func stringValueOrNull(value string) types.String {
	if value == "" {
		return types.StringNull()
	}
	return types.StringValue(value)
}
