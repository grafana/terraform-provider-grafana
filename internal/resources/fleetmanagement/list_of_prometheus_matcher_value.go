package fleetmanagement

import (
	"context"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/prometheus/alertmanager/matchers/parse"
)

var (
	_ basetypes.ListValuable                   = ListOfPrometheusMatcherValue{}
	_ basetypes.ListValuableWithSemanticEquals = ListOfPrometheusMatcherValue{}
)

type ListOfPrometheusMatcherValue struct {
	basetypes.ListValue
}

func NewListOfPrometheusMatcherValueNull() ListOfPrometheusMatcherValue {
	return ListOfPrometheusMatcherValue{
		ListValue: basetypes.NewListNull(types.StringType),
	}
}

func NewListOfPrometheusMatcherValueUnknown() ListOfPrometheusMatcherValue {
	return ListOfPrometheusMatcherValue{
		ListValue: basetypes.NewListUnknown(types.StringType),
	}
}

func NewListOfPrometheusMatcherValue(elements []attr.Value) (ListOfPrometheusMatcherValue, diag.Diagnostics) {
	value, diags := basetypes.NewListValue(types.StringType, elements)
	if diags.HasError() {
		return NewListOfPrometheusMatcherValueUnknown(), diags
	}

	return ListOfPrometheusMatcherValue{
		ListValue: value,
	}, nil
}

func NewListOfPrometheusMatcherValueFrom(ctx context.Context, elements []string) (ListOfPrometheusMatcherValue, diag.Diagnostics) {
	value, diags := basetypes.NewListValueFrom(ctx, types.StringType, elements)
	if diags.HasError() {
		return NewListOfPrometheusMatcherValueUnknown(), diags
	}

	return ListOfPrometheusMatcherValue{
		ListValue: value,
	}, nil
}

func NewListOfPrometheusMatcherValueMust(elements []attr.Value) ListOfPrometheusMatcherValue {
	return ListOfPrometheusMatcherValue{
		ListValue: basetypes.NewListValueMust(types.StringType, elements),
	}
}

func (v ListOfPrometheusMatcherValue) Equal(o attr.Value) bool {
	other, ok := o.(ListOfPrometheusMatcherValue)
	if !ok {
		return false
	}

	return v.ListValue.Equal(other.ListValue)
}

func (v ListOfPrometheusMatcherValue) Type(ctx context.Context) attr.Type {
	return ListOfPrometheusMatcherType
}

func (v ListOfPrometheusMatcherValue) ListSemanticEquals(ctx context.Context, newValuable basetypes.ListValuable) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	newValue, ok := newValuable.(ListOfPrometheusMatcherValue)
	if !ok {
		diags.AddError(
			"Semantic Equality Check Error",
			"An unexpected value type was received while performing semantic equality checks. "+
				"Please report this to the provider developers.\n\n"+
				"Expected Value Type: "+fmt.Sprintf("%T", v)+"\n"+
				"Got Value Type: "+fmt.Sprintf("%T", newValuable),
		)
		return false, diags
	}

	if len(v.Elements()) != len(newValue.Elements()) {
		return false, diags
	}

	matchers := attrValueToStringSlice(v.Elements())
	newValueMatchers := attrValueToStringSlice(newValue.Elements())

	sort.Strings(matchers)
	sort.Strings(newValueMatchers)

	for i, matcher := range matchers {
		equal, err := matcherEqual(matcher, newValueMatchers[i])
		if err != nil {
			diags.AddError(
				"Invalid Prometheus matcher",
				"An error occurred when parsing Prometheus matchers: "+err.Error(),
			)
			return false, diags
		}

		if !equal {
			return false, diags
		}
	}

	return true, diags
}

func attrValueToStringSlice(elements []attr.Value) []string {
	result := make([]string, len(elements))
	for i, element := range elements {
		result[i] = element.(basetypes.StringValue).ValueString()
	}
	return result
}

func matcherEqual(matcher1 string, matcher2 string) (bool, error) {
	parsed1, err := parse.Matcher(matcher1)
	if err != nil {
		return false, fmt.Errorf("invalid Prometheus matcher %q: %v", matcher1, err)
	}

	parsed2, err := parse.Matcher(matcher2)
	if err != nil {
		return false, fmt.Errorf("invalid Prometheus matcher %q: %v", matcher2, err)
	}

	return parsed1.String() == parsed2.String(), nil
}
