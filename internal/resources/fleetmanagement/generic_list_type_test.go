package fleetmanagement

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
)

func TestNewGenericListType(t *testing.T) {
	ctx := context.Background()
	genListType := NewGenericListType[types.String](ctx)
	assert.IsType(t, genericListType[types.String]{}, genListType)
}

func TestGenericListType_Equal(t *testing.T) {
	ctx := context.Background()
	genListType1 := NewGenericListType[types.String](ctx)
	genListType2 := NewGenericListType[types.String](ctx)
	genListType3 := NewGenericListType[types.Number](ctx)

	assert.True(t, genListType1.Equal(genListType2))
	assert.False(t, genListType1.Equal(genListType3))
}

func TestGenericListType_String(t *testing.T) {
	ctx := context.Background()
	genListType := NewGenericListType[types.String](ctx)
	assert.Equal(t, "GenericListType[basetypes.StringValue]", genListType.String())
}

func TestGenericListType_ValueFromList(t *testing.T) {
	ctx := context.Background()
	attrElements := []attr.Value{basetypes.NewStringValue("test")}
	listValue := basetypes.NewListValueMust(types.StringType, attrElements)
	genListType := NewGenericListType[types.String](ctx)

	genListValue, diags := genListType.ValueFromList(ctx, listValue)
	assert.False(t, diags.HasError())
	genListElements := genListValue.(GenericListValue[types.String]).Elements()
	assert.ElementsMatch(t, attrElements, genListElements)
}

func TestGenericListType_ValueFromTerraform(t *testing.T) {
	ctx := context.Background()
	tfValue := tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{tftypes.NewValue(tftypes.String, "test")})
	genListType := NewGenericListType[types.String](ctx)

	genListValue, err := genListType.ValueFromTerraform(ctx, tfValue)
	assert.NoError(t, err)
	genListElements := genListValue.(GenericListValue[types.String]).Elements()
	expected := []attr.Value{basetypes.NewStringValue("test")}
	assert.ElementsMatch(t, expected, genListElements)
}

func TestGenericListType_ValueType(t *testing.T) {
	ctx := context.Background()
	genListType := NewGenericListType[types.String](ctx)
	assert.IsType(t, GenericListValue[types.String]{}, genListType.ValueType(ctx))
}
