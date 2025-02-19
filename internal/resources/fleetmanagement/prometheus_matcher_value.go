package fleetmanagement

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/attr/xattr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/prometheus/alertmanager/matchers/parse"
)

var (
	_ basetypes.StringValuable                   = PrometheusMatcherValue{}
	_ basetypes.StringValuableWithSemanticEquals = PrometheusMatcherValue{}
	_ xattr.ValidateableAttribute                = PrometheusMatcherValue{}
)

type PrometheusMatcherValue struct {
	basetypes.StringValue
}

func NewPrometheusMatcherValue(value string) PrometheusMatcherValue {
	return PrometheusMatcherValue{
		StringValue: basetypes.NewStringValue(value),
	}
}

func (v PrometheusMatcherValue) Equal(o attr.Value) bool {
	other, ok := o.(PrometheusMatcherValue)
	if !ok {
		return false
	}

	return v.StringValue.Equal(other.StringValue)
}

func (v PrometheusMatcherValue) Type(ctx context.Context) attr.Type {
	return PrometheusMatcherType
}

func (v PrometheusMatcherValue) StringSemanticEquals(ctx context.Context, newValuable basetypes.StringValuable) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	newValue, ok := newValuable.(PrometheusMatcherValue)
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

	// Values are already validated at this point, ignoring errors
	result, _ := matcherEqual(v.ValueString(), newValue.ValueString())
	return result, diags
}

func (v PrometheusMatcherValue) ValidateAttribute(ctx context.Context, req xattr.ValidateAttributeRequest, resp *xattr.ValidateAttributeResponse) {
	if v.IsNull() || v.IsUnknown() {
		return
	}

	_, err := parse.Matcher(v.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Prometheus matcher",
			"A string value was provided that is not valid Prometheus matcher format.\n\n"+
				"Path: "+req.Path.String()+"\n"+
				"Given Value: "+v.ValueString()+"\n"+
				"Error: "+err.Error(),
		)

		return
	}
}

func matcherEqual(matcher1 string, matcher2 string) (bool, error) {
	parsed1, err := parse.Matcher(matcher1)
	if err != nil {
		return false, err
	}

	parsed2, err := parse.Matcher(matcher2)
	if err != nil {
		return false, err
	}

	return parsed1.String() == parsed2.String(), nil
}
