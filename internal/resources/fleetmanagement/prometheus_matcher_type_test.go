package fleetmanagement

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
)

func TestPrometheusMatcherType_Equal(t *testing.T) {
	type1 := PrometheusMatcherType
	type2 := PrometheusMatcherType
	type3 := types.StringType

	assert.True(t, type1.Equal(type2))
	assert.False(t, type1.Equal(type3))
}

func TestPrometheusMatcherType_String(t *testing.T) {
	assert.Equal(t, "PrometheusMatcherType", PrometheusMatcherType.String())
}

func TestPrometheusMatcherType_ValueFromString(t *testing.T) {
	ctx := context.Background()
	stringValue := types.StringValue("test")

	promMatcherValue, diags := PrometheusMatcherType.ValueFromString(ctx, stringValue)
	assert.False(t, diags.HasError())
	expected := PrometheusMatcherValue{StringValue: stringValue}
	assert.Equal(t, expected, promMatcherValue)
}

func TestPrometheusMatcherType_ValueFromTerraform(t *testing.T) {
	ctx := context.Background()
	tfValue := tftypes.NewValue(tftypes.String, "test")

	promMatcherValue, err := PrometheusMatcherType.ValueFromTerraform(ctx, tfValue)
	assert.NoError(t, err)
	expected := PrometheusMatcherValue{StringValue: types.StringValue("test")}
	assert.Equal(t, expected, promMatcherValue)
}

func TestPrometheusMatcherType_ValueType(t *testing.T) {
	ctx := context.Background()
	promMatcherValue := PrometheusMatcherType.ValueType(ctx)
	assert.IsType(t, PrometheusMatcherValue{}, promMatcherValue)
}
