package fleetmanagement

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/stretchr/testify/assert"
)

func TestNewGenericListValueNull(t *testing.T) {
	ctx := context.Background()
	genListValue := NewGenericListValueNull[types.String](ctx)
	assert.True(t, genListValue.IsNull())
}

func TestNewGenericListValueUnknown(t *testing.T) {
	ctx := context.Background()
	genListValue := NewGenericListValueUnknown[types.String](ctx)
	assert.True(t, genListValue.IsUnknown())
}

func TestNewGenericListValue(t *testing.T) {
	ctx := context.Background()
	attrElements := []attr.Value{basetypes.NewStringValue("test")}

	genListValue, diags := NewGenericListValue[types.String](ctx, attrElements)
	assert.False(t, diags.HasError())
	assert.ElementsMatch(t, attrElements, genListValue.Elements())
}

func TestNewGenericListValueFrom(t *testing.T) {
	ctx := context.Background()
	stringElements := []string{"test"}

	genListValue, diags := NewGenericListValueFrom[types.String](ctx, types.StringType, stringElements)
	assert.False(t, diags.HasError())
	expected := []attr.Value{basetypes.NewStringValue("test")}
	assert.ElementsMatch(t, expected, genListValue.Elements())
}

func TestNewGenericListValueMust(t *testing.T) {
	ctx := context.Background()
	attrElements := []attr.Value{basetypes.NewStringValue("test")}
	genListValue := NewGenericListValueMust[types.String](ctx, attrElements)
	assert.ElementsMatch(t, attrElements, genListValue.Elements())
}

func TestGenericListValue_Equal(t *testing.T) {
	ctx := context.Background()
	genListValue1 := NewGenericListValueMust[types.String](ctx, []attr.Value{basetypes.NewStringValue("test")})
	genListValue2 := NewGenericListValueMust[types.String](ctx, []attr.Value{basetypes.NewStringValue("test")})
	genListValue3 := NewGenericListValueMust[types.String](ctx, []attr.Value{basetypes.NewStringValue("different")})

	assert.True(t, genListValue1.Equal(genListValue2))
	assert.False(t, genListValue1.Equal(genListValue3))
}

func TestGenericListValue_Type(t *testing.T) {
	ctx := context.Background()
	attrElements := []attr.Value{basetypes.NewStringValue("test")}
	genListValue := NewGenericListValueMust[types.String](ctx, attrElements)
	assert.IsType(t, genericListType[types.String]{}, genListValue.Type(ctx))
}
