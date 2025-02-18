package fleetmanagement

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

var (
	_ basetypes.StringTypable = prometheusMatcherType{}
)

var (
	PrometheusMatcherType = prometheusMatcherType{}
)

type prometheusMatcherType struct {
	basetypes.StringType
}

func (t prometheusMatcherType) Equal(o attr.Type) bool {
	other, ok := o.(prometheusMatcherType)
	if !ok {
		return false
	}

	return t.StringType.Equal(other.StringType)
}

func (t prometheusMatcherType) String() string {
	return "PrometheusMatcherType"
}

func (t prometheusMatcherType) ValueFromString(ctx context.Context, in basetypes.StringValue) (basetypes.StringValuable, diag.Diagnostics) {
	return PrometheusMatcherValue{
		StringValue: in,
	}, nil
}

func (t prometheusMatcherType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	attrValue, err := t.StringType.ValueFromTerraform(ctx, in)
	if err != nil {
		return nil, err
	}

	stringValue, ok := attrValue.(basetypes.StringValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type of %T", attrValue)
	}

	stringValuable, diags := t.ValueFromString(ctx, stringValue)
	if diags.HasError() {
		return nil, fmt.Errorf("unexpected error converting StringValue to StringValuable: %v", diags)
	}

	return stringValuable, nil
}

func (t prometheusMatcherType) ValueType(ctx context.Context) attr.Value {
	return PrometheusMatcherValue{}
}
