package fleetmanagement

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr/xattr"
	"github.com/stretchr/testify/assert"
)

func TestNewPrometheusMatcherValue(t *testing.T) {
	rawValue := "os=\"linux\""
	value := NewPrometheusMatcherValue(rawValue)
	assert.Equal(t, rawValue, value.ValueString())
}

func TestPrometheusMatcherValue_Equal(t *testing.T) {
	value1 := NewPrometheusMatcherValue("os=\"linux\"")
	value2 := NewPrometheusMatcherValue("os=\"linux\"")
	value3 := NewPrometheusMatcherValue("os=linux")

	assert.True(t, value1.Equal(value2))
	assert.False(t, value1.Equal(value3))
}

func TestPrometheusMatcherValue_Type(t *testing.T) {
	value := NewPrometheusMatcherValue("collector.os=\"linux\"")
	ctx := context.Background()
	assert.IsType(t, PrometheusMatcherType, value.Type(ctx))
}

func TestPrometheusMatcherValue_StringSemanticEquals(t *testing.T) {
	ctx := context.Background()
	value1 := NewPrometheusMatcherValue("collector.os=\"linux\"")
	value2 := NewPrometheusMatcherValue("collector.os=linux")
	value3 := NewPrometheusMatcherValue("collector.os=\"darwin\"")

	t.Run("semantically equal Prometheus matcher value", func(t *testing.T) {
		equal, diags := value1.StringSemanticEquals(ctx, value2)
		assert.False(t, diags.HasError())
		assert.True(t, equal)
	})

	t.Run("semantically not equal Prometheus matcher value", func(t *testing.T) {
		equal, diags := value1.StringSemanticEquals(ctx, value3)
		assert.False(t, diags.HasError())
		assert.False(t, equal)
	})
}

func TestPrometheusMatcherValue_ValidateAttribute(t *testing.T) {
	ctx := context.Background()
	req := xattr.ValidateAttributeRequest{}
	resp := &xattr.ValidateAttributeResponse{}

	t.Run("valid attribute", func(t *testing.T) {
		value := NewPrometheusMatcherValue("collector.os=~.*")
		value.ValidateAttribute(ctx, req, resp)
		assert.False(t, resp.Diagnostics.HasError())
	})

	t.Run("invalid attribute", func(t *testing.T) {
		invalidValue := NewPrometheusMatcherValue("collector.os~=.*")
		invalidValue.ValidateAttribute(ctx, req, resp)
		assert.True(t, resp.Diagnostics.HasError())
	})
}

func TestMatcherEqual(t *testing.T) {
	matcher1 := "collector.os=\"linux\""
	matcher2 := "collector.os=linux"
	matcher3 := "collector.os=\"darwin\""

	t.Run("equal matchers", func(t *testing.T) {
		equal, err := matcherEqual(matcher1, matcher2)
		assert.NoError(t, err)
		assert.True(t, equal)
	})

	t.Run("not equal matchers", func(t *testing.T) {
		equal, err := matcherEqual(matcher1, matcher3)
		assert.NoError(t, err)
		assert.False(t, equal)
	})
}
