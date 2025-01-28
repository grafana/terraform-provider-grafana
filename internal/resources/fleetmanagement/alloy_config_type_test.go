package fleetmanagement

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
)

func TestAlloyConfigType_Equal(t *testing.T) {
	type1 := AlloyConfigType
	type2 := AlloyConfigType
	type3 := types.StringType

	assert.True(t, type1.Equal(type2))
	assert.False(t, type1.Equal(type3))
}

func TestAlloyConfigType_String(t *testing.T) {
	assert.Equal(t, "AlloyConfigType", AlloyConfigType.String())
}

func TestAlloyConfigType_ValueFromString(t *testing.T) {
	ctx := context.Background()
	stringValue := types.StringValue("test")

	alloyCfgValue, diags := AlloyConfigType.ValueFromString(ctx, stringValue)
	assert.False(t, diags.HasError())
	expected := AlloyConfigValue{StringValue: stringValue}
	assert.Equal(t, expected, alloyCfgValue)
}

func TestAlloyConfigType_ValueFromTerraform(t *testing.T) {
	ctx := context.Background()
	tfValue := tftypes.NewValue(tftypes.String, "test")

	alloyCfgValue, err := AlloyConfigType.ValueFromTerraform(ctx, tfValue)
	assert.NoError(t, err)
	expected := AlloyConfigValue{StringValue: types.StringValue("test")}
	assert.Equal(t, expected, alloyCfgValue)
}

func TestAlloyConfigType_ValueType(t *testing.T) {
	ctx := context.Background()
	alloyCfgValue := AlloyConfigType.ValueType(ctx)
	assert.IsType(t, AlloyConfigValue{}, alloyCfgValue)
}
