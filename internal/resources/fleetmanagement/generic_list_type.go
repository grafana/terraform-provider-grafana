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
	_ basetypes.ListTypable = genericListType[basetypes.StringValue]{}
)

var (
	ListOfPrometheusMatcherType = genericListType[PrometheusMatcherValue]{basetypes.ListType{ElemType: PrometheusMatcherType}}
)

type genericListType[T attr.Value] struct {
	basetypes.ListType
}

func NewGenericListType[T attr.Value](ctx context.Context) genericListType[T] {
	var zero T
	return genericListType[T]{
		basetypes.ListType{
			ElemType: zero.Type(ctx),
		},
	}
}

func (t genericListType[T]) Equal(o attr.Type) bool {
	other, ok := o.(genericListType[T])
	if !ok {
		return false
	}

	return t.ListType.Equal(other.ListType)
}

func (t genericListType[T]) String() string {
	var zero T
	return fmt.Sprintf("GenericListType[%T]", zero)
}

func (t genericListType[T]) ValueFromList(ctx context.Context, in basetypes.ListValue) (basetypes.ListValuable, diag.Diagnostics) {
	if in.IsNull() {
		return NewGenericListValueNull[T](ctx), nil
	}

	if in.IsUnknown() {
		return NewGenericListValueUnknown[T](ctx), nil
	}

	return NewGenericListValue[T](ctx, in.Elements())
}

func (t genericListType[T]) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
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

func (t genericListType[T]) ValueType(ctx context.Context) attr.Value {
	return GenericListValue[T]{}
}
