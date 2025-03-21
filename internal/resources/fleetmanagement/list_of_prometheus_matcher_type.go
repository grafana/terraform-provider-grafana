package fleetmanagement

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

var (
	_ basetypes.ListTypable = listOfPrometheusMatcherType{}
)

var (
	ListOfPrometheusMatcherType = listOfPrometheusMatcherType{
		ListType: basetypes.ListType{
			ElemType: types.StringType,
		},
	}
)

type listOfPrometheusMatcherType struct {
	basetypes.ListType
}

func (t listOfPrometheusMatcherType) Equal(o attr.Type) bool {
	other, ok := o.(listOfPrometheusMatcherType)
	if !ok {
		return false
	}

	return t.ListType.Equal(other.ListType)
}

func (t listOfPrometheusMatcherType) String() string {
	return "ListOfPrometheusMatcherType"
}

func (t listOfPrometheusMatcherType) ValueFromList(ctx context.Context, in basetypes.ListValue) (basetypes.ListValuable, diag.Diagnostics) {
	if in.IsNull() {
		return NewListOfPrometheusMatcherValueNull(), nil
	}

	if in.IsUnknown() {
		return NewListOfPrometheusMatcherValueUnknown(), nil
	}

	return NewListOfPrometheusMatcherValue(in.Elements())
}

func (t listOfPrometheusMatcherType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	attrValue, err := t.ListType.ValueFromTerraform(ctx, in)
	if err != nil {
		return nil, err
	}

	listValue, ok := attrValue.(basetypes.ListValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type of %T", attrValue)
	}

	listValuable, diags := t.ValueFromList(ctx, listValue)
	if diags.HasError() {
		return nil, fmt.Errorf("unexpected error converting ListValue to ListValuable: %v", diags)
	}

	return listValuable, nil
}

func (t listOfPrometheusMatcherType) ValueType(ctx context.Context) attr.Value {
	return ListOfPrometheusMatcherValue{}
}
