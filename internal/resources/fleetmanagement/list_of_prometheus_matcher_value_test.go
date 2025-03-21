package fleetmanagement

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/attr/xattr"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/stretchr/testify/assert"
)

func TestNewListOfPrometheusMatcherValueNull(t *testing.T) {
	value := NewListOfPrometheusMatcherValueNull()
	assert.True(t, value.IsNull())
}

func TestNewListOfPrometheusMatcherValueUnknown(t *testing.T) {
	value := NewListOfPrometheusMatcherValueUnknown()
	assert.True(t, value.IsUnknown())
}

func TestNewListOfPrometheusMatcherValue(t *testing.T) {
	attrElements := []attr.Value{
		basetypes.NewStringValue("collector.os=linux"),
		basetypes.NewStringValue("owner=TEAM-A"),
	}

	value, diags := NewListOfPrometheusMatcherValue(attrElements)
	assert.False(t, diags.HasError())
	assert.ElementsMatch(t, attrElements, value.Elements())
}

func TestNewListOfPrometheusMatcherValueFrom(t *testing.T) {
	ctx := context.Background()
	stringElements := []string{
		"collector.os=linux",
		"owner=TEAM-A",
	}

	value, diags := NewListOfPrometheusMatcherValueFrom(ctx, stringElements)
	assert.False(t, diags.HasError())
	expected := []attr.Value{
		basetypes.NewStringValue("collector.os=linux"),
		basetypes.NewStringValue("owner=TEAM-A"),
	}
	assert.ElementsMatch(t, expected, value.Elements())
}

func TestNewListOfPrometheusMatcherValueMust(t *testing.T) {
	attrElements := []attr.Value{
		basetypes.NewStringValue("collector.os=linux"),
		basetypes.NewStringValue("owner=TEAM-A"),
	}

	value := NewListOfPrometheusMatcherValueMust(attrElements)
	assert.ElementsMatch(t, attrElements, value.Elements())
}

func TestListOfPrometheusMatcherValue_Equal(t *testing.T) {
	value1 := NewListOfPrometheusMatcherValueMust([]attr.Value{
		basetypes.NewStringValue("collector.os=linux"),
		basetypes.NewStringValue("owner=TEAM-A"),
	})
	value2 := NewListOfPrometheusMatcherValueMust([]attr.Value{
		basetypes.NewStringValue("collector.os=linux"),
		basetypes.NewStringValue("owner=TEAM-A"),
	})
	value3 := NewListOfPrometheusMatcherValueMust([]attr.Value{
		basetypes.NewStringValue("collector.os=darwin"),
		basetypes.NewStringValue("owner=TEAM-B"),
	})

	assert.True(t, value1.Equal(value2))
	assert.False(t, value1.Equal(value3))
}

func TestListOfPrometheusMatcherValue_Type(t *testing.T) {
	ctx := context.Background()
	value := NewListOfPrometheusMatcherValueMust([]attr.Value{
		basetypes.NewStringValue("collector.os=linux"),
	})
	assert.Equal(t, ListOfPrometheusMatcherType, value.Type(ctx))
}

func TestListOfPrometheusMatcherValue_ListSemanticEquals(t *testing.T) {
	ctx := context.Background()
	value1 := NewListOfPrometheusMatcherValueMust([]attr.Value{
		basetypes.NewStringValue("collector.os=\"linux\""),
		basetypes.NewStringValue("owner=\"TEAM-A\""),
	})
	value2 := NewListOfPrometheusMatcherValueMust([]attr.Value{
		basetypes.NewStringValue("collector.os=linux"),
		basetypes.NewStringValue("owner=TEAM-A"),
	})
	value3 := NewListOfPrometheusMatcherValueMust([]attr.Value{
		basetypes.NewStringValue("owner=\"TEAM-A\""),
		basetypes.NewStringValue("collector.os=\"linux\""),
	})
	value4 := NewListOfPrometheusMatcherValueMust([]attr.Value{
		basetypes.NewStringValue("collector.os=\"linux\""),
		basetypes.NewStringValue("owner=\"TEAM-B\""),
	})

	t.Run("semantically equal matchers (same order)", func(t *testing.T) {
		equal, diags := value1.ListSemanticEquals(ctx, value2)
		assert.False(t, diags.HasError())
		assert.True(t, equal)
	})

	t.Run("semantically equal matchers (different order)", func(t *testing.T) {
		equal, diags := value1.ListSemanticEquals(ctx, value3)
		assert.False(t, diags.HasError())
		assert.True(t, equal)
	})

	t.Run("semantically not equal matchers", func(t *testing.T) {
		equal, diags := value1.ListSemanticEquals(ctx, value4)
		assert.False(t, diags.HasError())
		assert.False(t, equal)
	})
}

func TestListOfPrometheusMatcherValue_ValidateAttribute(t *testing.T) {
	ctx := context.Background()
	req := xattr.ValidateAttributeRequest{}
	resp := &xattr.ValidateAttributeResponse{}

	t.Run("valid attribute", func(t *testing.T) {
		value := NewListOfPrometheusMatcherValueMust([]attr.Value{
			basetypes.NewStringValue("collector.os=~.*"),
			basetypes.NewStringValue("owner=TEAM-A"),
		})
		value.ValidateAttribute(ctx, req, resp)
		assert.False(t, resp.Diagnostics.HasError())
	})

	t.Run("invalid attribute", func(t *testing.T) {
		value := NewListOfPrometheusMatcherValueMust([]attr.Value{
			basetypes.NewStringValue("collector.os~=.*"),
			basetypes.NewStringValue("owner=TEAM-A"),
		})
		value.ValidateAttribute(ctx, req, resp)
		assert.True(t, resp.Diagnostics.HasError())
	})
}

func TestAttrValueToStringSlice(t *testing.T) {
	elements := []attr.Value{
		basetypes.NewStringValue("collector.os=linux"),
		basetypes.NewStringValue("owner=TEAM-A"),
	}
	expected := []string{
		"collector.os=linux",
		"owner=TEAM-A",
	}

	result := attrValueToStringSlice(elements)
	assert.Equal(t, expected, result)
}
