package fleetmanagement

import (
	"context"
	"testing"

	collectorv1 "github.com/grafana/fleet-management-api/api/gen/proto/go/collector/v1"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/stretchr/testify/assert"
)

func TestCollectorMessageToModel(t *testing.T) {
	id := "test_id"
	enabled := true

	msg := &collectorv1.Collector{
		Id: id,
		AttributeOverrides: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
		Enabled: &enabled,
	}

	expectedModel := &collectorModel{
		ID: types.StringValue(id),
		AttributeOverrides: types.MapValueMust(
			types.StringType,
			map[string]attr.Value{
				"key1": types.StringValue("value1"),
				"key2": types.StringValue("value2"),
			},
		),
		Enabled: types.BoolPointerValue(&enabled),
	}

	ctx := context.Background()
	actualModel, diags := collectorMessageToModel(ctx, msg)
	assert.False(t, diags.HasError())
	assert.Equal(t, expectedModel, actualModel)
}

func TestCollectorModelToMessage(t *testing.T) {
	id := "test_id"
	enabled := true

	model := &collectorModel{
		ID: types.StringValue(id),
		AttributeOverrides: types.MapValueMust(
			types.StringType,
			map[string]attr.Value{
				"key1": types.StringValue("value1"),
				"key2": types.StringValue("value2"),
			},
		),
		Enabled: types.BoolPointerValue(&enabled),
	}

	expectedMsg := &collectorv1.Collector{
		Id: id,
		AttributeOverrides: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
		Enabled: &enabled,
	}

	ctx := context.Background()
	actualMsg, diags := collectorModelToMessage(ctx, model)
	assert.False(t, diags.HasError())
	assert.Equal(t, expectedMsg, actualMsg)
}

func TestNativeStringMapToTFStringMap(t *testing.T) {
	tests := []struct {
		name      string
		nativeMap map[string]string
		expected  types.Map
	}{
		{
			"nil map",
			nil,
			types.MapValueMust(
				types.StringType,
				map[string]attr.Value{},
			),
		},
		{
			"empty map",
			map[string]string{},
			types.MapValueMust(
				types.StringType,
				map[string]attr.Value{},
			),
		},
		{
			"non-empty map",
			map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			types.MapValueMust(
				types.StringType,
				map[string]attr.Value{
					"key1": types.StringValue("value1"),
					"key2": types.StringValue("value2"),
				},
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			actual, diags := nativeStringMapToTFStringMap(ctx, tt.nativeMap)
			assert.False(t, diags.HasError())
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestTfStringMapToNativeStringMap(t *testing.T) {
	tests := []struct {
		name     string
		tfMap    types.Map
		expected map[string]string
	}{
		{
			"null map",
			basetypes.NewMapNull(types.StringType),
			map[string]string{},
		},
		{
			"unknown map",
			basetypes.NewMapUnknown(types.StringType),
			map[string]string{},
		},
		{
			"empty map",
			types.MapValueMust(
				types.StringType,
				map[string]attr.Value{},
			),
			map[string]string{},
		},
		{
			"non-empty map",
			types.MapValueMust(
				types.StringType,
				map[string]attr.Value{
					"key1": types.StringValue("value1"),
					"key2": types.StringValue("value2"),
				},
			),
			map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			actual, diags := tfStringMapToNativeStringMap(ctx, tt.tfMap)
			assert.False(t, diags.HasError())
			assert.Equal(t, tt.expected, actual)
		})
	}
}
