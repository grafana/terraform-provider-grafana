package fleetmanagement

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr/xattr"
	"github.com/stretchr/testify/assert"
)

func TestNewAlloyConfigValue(t *testing.T) {
	rawValue := "logging {}"
	value := NewAlloyConfigValue(rawValue)
	assert.Equal(t, rawValue, value.ValueString())
}

func TestAlloyConfigValue_Equal(t *testing.T) {
	value1 := NewAlloyConfigValue("logging {}")
	value2 := NewAlloyConfigValue("logging {}")
	value3 := NewAlloyConfigValue("logging {}\n")

	assert.True(t, value1.Equal(value2))
	assert.False(t, value1.Equal(value3))
}

func TestAlloyConfigValue_Type(t *testing.T) {
	ctx := context.Background()
	value := NewAlloyConfigValue("logging {}")
	assert.IsType(t, AlloyConfigType{}, value.Type(ctx))
}

func TestAlloyConfigValue_StringSemanticEquals(t *testing.T) {
	ctx := context.Background()
	value1 := NewAlloyConfigValue("logging {}")
	value2 := NewAlloyConfigValue("logging {}\n")
	value3 := NewAlloyConfigValue("// test")

	t.Run("semantically equal Alloy Config value", func(t *testing.T) {
		equal, diags := value1.StringSemanticEquals(ctx, value2)
		assert.True(t, equal)
		assert.False(t, diags.HasError())
	})

	t.Run("semantically not equal Alloy Config value", func(t *testing.T) {
		equal, diags := value1.StringSemanticEquals(ctx, value3)
		assert.False(t, equal)
		assert.False(t, diags.HasError())
	})
}

func TestAlloyConfigValue_ValidateAttribute(t *testing.T) {
	ctx := context.Background()
	value := NewAlloyConfigValue("// valid")
	req := xattr.ValidateAttributeRequest{}
	resp := &xattr.ValidateAttributeResponse{}

	t.Run("valid attribute", func(t *testing.T) {
		value.ValidateAttribute(ctx, req, resp)
		assert.False(t, resp.Diagnostics.HasError())
	})

	t.Run("invalid attribute", func(t *testing.T) {
		invalidValue := NewAlloyConfigValue("invalid")
		invalidValue.ValidateAttribute(ctx, req, resp)
		assert.True(t, resp.Diagnostics.HasError())
	})
}

func TestRiverEqual(t *testing.T) {
	contents1 := "logging {}"
	contents2 := "logging {}\n"
	contents3 := "// test"

	t.Run("equal river contents", func(t *testing.T) {
		equal, err := riverEqual(contents1, contents2)
		assert.NoError(t, err)
		assert.True(t, equal)
	})

	t.Run("not equal river contents", func(t *testing.T) {
		equal, err := riverEqual(contents1, contents3)
		assert.NoError(t, err)
		assert.False(t, equal)
	})
}

func TestParseRiver(t *testing.T) {
	t.Run("valid river contents", func(t *testing.T) {
		contents := "// valid"
		parsed, err := parseRiver(contents)
		assert.NoError(t, err)
		assert.NotEmpty(t, parsed)
	})

	t.Run("invalid river contents", func(t *testing.T) {
		contents := "invalid"
		parsed, err := parseRiver(contents)
		assert.Error(t, err)
		assert.Empty(t, parsed)
	})
}
