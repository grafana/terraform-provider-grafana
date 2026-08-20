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

// The Assistant API omits empty optional fields from its responses, so "unset"
// and "set to an empty value" come back identically. Collapsing both to null
// loses the distinction: when the configuration said `""`, `[]` or `{}`,
// Terraform fails the apply with "Provider produced inconsistent result after
// apply ... was cty.StringVal(\"\"), but now null".
//
// The reconcile* helpers below take the corresponding plan (or, on refresh,
// prior state) value and fall back to it whenever the API returned nothing, so
// an explicitly empty value survives the round trip.

func reconcileList(ctx context.Context, prior types.List, values []string) (types.List, diag.Diagnostics) {
	if len(values) > 0 {
		return types.ListValueFrom(ctx, types.StringType, values)
	}
	if !prior.IsNull() && !prior.IsUnknown() {
		return prior, nil
	}
	return types.ListNull(types.StringType), nil
}

func reconcileMap(ctx context.Context, prior types.Map, values map[string]string) (types.Map, diag.Diagnostics) {
	if len(values) > 0 {
		return types.MapValueFrom(ctx, types.StringType, values)
	}
	if !prior.IsNull() && !prior.IsUnknown() {
		return prior, nil
	}
	return types.MapNull(types.StringType), nil
}

func reconcileString(prior types.String, value string) types.String {
	if value != "" {
		return types.StringValue(value)
	}
	if !prior.IsNull() && !prior.IsUnknown() {
		return prior
	}
	return types.StringNull()
}

func reconcileRawJSON(prior types.String, raw json.RawMessage) types.String {
	if len(raw) > 0 {
		return types.StringValue(string(raw))
	}
	if !prior.IsNull() && !prior.IsUnknown() {
		return prior
	}
	return types.StringNull()
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
