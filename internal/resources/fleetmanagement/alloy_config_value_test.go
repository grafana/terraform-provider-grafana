package fleetmanagement

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPipelineConfigValue(t *testing.T) {
	rawValue := "logging {}"
	value := NewPipelineConfigValue(rawValue)
	assert.Equal(t, rawValue, value.ValueString())
}

func TestPipelineConfigValue_Equal(t *testing.T) {
	value1 := NewPipelineConfigValue("logging {}")
	value2 := NewPipelineConfigValue("logging {}")
	value3 := NewPipelineConfigValue("logging {}\n")

	assert.True(t, value1.Equal(value2))
	assert.False(t, value1.Equal(value3))
}

func TestPipelineConfigValue_Type(t *testing.T) {
	ctx := context.Background()
	value := NewPipelineConfigValue("logging {}")
	assert.IsType(t, PipelineConfigType, value.Type(ctx))
}

func TestPipelineConfigValue_StringSemanticEquals_Alloy(t *testing.T) {
	ctx := context.Background()
	value1 := NewPipelineConfigValue("logging {}")
	value2 := NewPipelineConfigValue("logging {}\n")
	value3 := NewPipelineConfigValue("// test")

	t.Run("semantically equal Alloy config", func(t *testing.T) {
		equal, diags := value1.StringSemanticEquals(ctx, value2)
		assert.False(t, diags.HasError())
		assert.True(t, equal)
	})

	t.Run("semantically not equal Alloy config", func(t *testing.T) {
		equal, diags := value1.StringSemanticEquals(ctx, value3)
		assert.False(t, diags.HasError())
		assert.False(t, equal)
	})
}

func TestPipelineConfigValue_StringSemanticEquals_YAML(t *testing.T) {
	ctx := context.Background()
	value1 := NewPipelineConfigValue("key: value\nother: 123")
	value2 := NewPipelineConfigValue("key: value\nother: 123\n")
	value3 := NewPipelineConfigValue("key: different")

	t.Run("semantically equal YAML config", func(t *testing.T) {
		equal, diags := value1.StringSemanticEquals(ctx, value2)
		assert.False(t, diags.HasError())
		assert.True(t, equal)
	})

	t.Run("semantically not equal YAML config", func(t *testing.T) {
		equal, diags := value1.StringSemanticEquals(ctx, value3)
		assert.False(t, diags.HasError())
		assert.False(t, equal)
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

func TestYamlEqual(t *testing.T) {
	t.Run("equal yaml contents", func(t *testing.T) {
		contents1 := "key: value\nother: 123"
		contents2 := "key: value\nother: 123\n"
		equal, err := yamlEqual(contents1, contents2)
		assert.NoError(t, err)
		assert.True(t, equal)
	})

	t.Run("not equal yaml contents", func(t *testing.T) {
		contents1 := "key: value"
		contents2 := "key: different"
		equal, err := yamlEqual(contents1, contents2)
		assert.NoError(t, err)
		assert.False(t, equal)
	})
}

func TestParseYAML(t *testing.T) {
	t.Run("valid yaml contents", func(t *testing.T) {
		contents := "key: value"
		parsed, err := parseYAML(contents)
		assert.NoError(t, err)
		assert.NotEmpty(t, parsed)
	})

	t.Run("valid yaml with nested structure", func(t *testing.T) {
		contents := "receivers:\n  otlp:\n    protocols:\n      grpc:"
		parsed, err := parseYAML(contents)
		assert.NoError(t, err)
		assert.NotEmpty(t, parsed)
	})

	t.Run("invalid yaml contents", func(t *testing.T) {
		contents := ":\ninvalid"
		parsed, err := parseYAML(contents)
		assert.Error(t, err)
		assert.Empty(t, parsed)
	})
}
