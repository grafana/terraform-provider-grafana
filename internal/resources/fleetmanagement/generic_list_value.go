package fleetmanagement

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var (
	_ basetypes.ListValuable = GenericListValue[basetypes.StringValue]{}
)

type GenericListValue[T attr.Value] struct {
	basetypes.ListValue
}

func NewGenericListValueNull[T attr.Value](ctx context.Context) GenericListValue[T] {
	var zero T
	return GenericListValue[T]{
		ListValue: basetypes.NewListNull(
			zero.Type(ctx),
		),
	}
}

func NewGenericListValueUnknown[T attr.Value](ctx context.Context) GenericListValue[T] {
	var zero T
	return GenericListValue[T]{
		ListValue: basetypes.NewListUnknown(
			zero.Type(ctx),
		),
	}
}

func NewGenericListValue[T attr.Value](ctx context.Context, elements []attr.Value) (GenericListValue[T], diag.Diagnostics) {
	var zero T
	value, diags := basetypes.NewListValue(zero.Type(ctx), elements)
	if diags.HasError() {
		return NewGenericListValueUnknown[T](ctx), diags
	}

	return GenericListValue[T]{
		ListValue: value,
	}, nil
}

func NewGenericListValueFrom[T attr.Value](ctx context.Context, elementType attr.Type, elements any) (GenericListValue[T], diag.Diagnostics) {
	var zero T
	value, diags := basetypes.NewListValueFrom(ctx, zero.Type(ctx), elements)
	if diags.HasError() {
		return NewGenericListValueUnknown[T](ctx), diags
	}

	return GenericListValue[T]{
		ListValue: value,
	}, nil
}

func NewGenericListValueMust[T attr.Value](ctx context.Context, elements []attr.Value) GenericListValue[T] {
	var zero T
	value := basetypes.NewListValueMust(zero.Type(ctx), elements)
	return GenericListValue[T]{
		ListValue: value,
	}
}

func (v GenericListValue[T]) Equal(o attr.Value) bool {
	other, ok := o.(GenericListValue[T])
	if !ok {
		return false
	}

	return v.ListValue.Equal(other.ListValue)
}

func (v GenericListValue[T]) Type(ctx context.Context) attr.Type {
	return NewGenericListType[T](ctx)
}
