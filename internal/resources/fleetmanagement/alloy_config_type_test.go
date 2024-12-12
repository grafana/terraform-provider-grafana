package fleetmanagement

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
)

func TestAlloyConfigType_Equal(t *testing.T) {
	type1 := AlloyConfigType{}
	type2 := AlloyConfigType{}
	type3 := basetypes.StringType{}

	assert.True(t, type1.Equal(type2))
	assert.False(t, type1.Equal(type3))
}

func TestAlloyConfigType_String(t *testing.T) {
	alloyConfigType := AlloyConfigType{}
	assert.Equal(t, "AlloyConfigType", alloyConfigType.String())
}

func TestAlloyConfigType_ValueFromString(t *testing.T) {
	alloyConfigType := AlloyConfigType{}
	stringValue := types.StringValue("test")

	ctx := context.Background()
	value, diags := alloyConfigType.ValueFromString(ctx, stringValue)
	assert.False(t, diags.HasError())
	assert.Equal(t, "test", value.(AlloyConfigValue).ValueString())
}

func TestAlloyConfigType_ValueFromTerraform(t *testing.T) {
	alloyConfigType := AlloyConfigType{}
	tfValue := tftypes.NewValue(tftypes.String, "test")

	ctx := context.Background()
	value, err := alloyConfigType.ValueFromTerraform(ctx, tfValue)
	assert.NoError(t, err)
	assert.Equal(t, "test", value.(AlloyConfigValue).ValueString())
}

func TestAlloyConfigType_ValueType(t *testing.T) {
	alloyConfigType := AlloyConfigType{}
	ctx := context.Background()
	value := alloyConfigType.ValueType(ctx)
	assert.IsType(t, AlloyConfigValue{}, value)
}

func TestAlloyConfigType_Validate(t *testing.T) {
	alloyConfigType := AlloyConfigType{}
	ctx := context.Background()
	valuePath := path.Root("test")

	t.Run("valid Alloy Config value", func(t *testing.T) {
		value := tftypes.NewValue(tftypes.String, "// valid")
		diags := alloyConfigType.Validate(ctx, value, valuePath)
		assert.False(t, diags.HasError())
	})

	t.Run("invalid Alloy Config value", func(t *testing.T) {
		value := tftypes.NewValue(tftypes.String, "invalid")
		diags := alloyConfigType.Validate(ctx, value, valuePath)
		assert.True(t, diags.HasError())
	})
}
