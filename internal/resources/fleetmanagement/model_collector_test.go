package fleetmanagement

import (
	"context"
	"testing"

	collectorv1 "github.com/grafana/fleet-management-api/api/gen/proto/go/collector/v1"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/stretchr/testify/require"
)

func TestCollectorMessageToDataSourceModel(t *testing.T) {
	id := "test_id"
	enabled := true

	msg := &collectorv1.Collector{
		Id: id,
		RemoteAttributes: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
		LocalAttributes: map[string]string{
			"key3": "value3",
			"key4": "value4",
		},
		Enabled:       &enabled,
		CollectorType: collectorv1.CollectorType_COLLECTOR_TYPE_ALLOY,
	}

	expectedModel := &collectorDataSourceModel{
		ID: types.StringValue(id),
		RemoteAttributes: types.MapValueMust(
			types.StringType,
			map[string]attr.Value{
				"key1": types.StringValue("value1"),
				"key2": types.StringValue("value2"),
			},
		),
		LocalAttributes: types.MapValueMust(
			types.StringType,
			map[string]attr.Value{
				"key3": types.StringValue("value3"),
				"key4": types.StringValue("value4"),
			},
		),
		Enabled:       types.BoolPointerValue(&enabled),
		CollectorType: types.StringValue("ALLOY"),
	}

	ctx := context.Background()
	actualModel, diags := collectorMessageToDataSourceModel(ctx, msg)
	require.False(t, diags.HasError())
	require.Equal(t, expectedModel, actualModel)
}

func TestCollectorMessageToResourceModel(t *testing.T) {
	id := "test_id"
	enabled := true

	msg := &collectorv1.Collector{
		Id: id,
		RemoteAttributes: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
		Enabled:       &enabled,
		CollectorType: collectorv1.CollectorType_COLLECTOR_TYPE_ALLOY,
	}

	expectedModel := &collectorResourceModel{
		ID: types.StringValue(id),
		RemoteAttributes: types.MapValueMust(
			types.StringType,
			map[string]attr.Value{
				"key1": types.StringValue("value1"),
				"key2": types.StringValue("value2"),
			},
		),
		Enabled:       types.BoolPointerValue(&enabled),
		CollectorType: types.StringValue("ALLOY"),
	}

	ctx := context.Background()
	actualModel, diags := collectorMessageToResourceModel(ctx, msg)
	require.False(t, diags.HasError())
	require.Equal(t, expectedModel, actualModel)
}

func TestCollectorResourceModelToMessage(t *testing.T) {
	id := "test_id"
	enabled := true

	model := &collectorResourceModel{
		ID: types.StringValue(id),
		RemoteAttributes: types.MapValueMust(
			types.StringType,
			map[string]attr.Value{
				"key1": types.StringValue("value1"),
				"key2": types.StringValue("value2"),
			},
		),
		Enabled:       types.BoolPointerValue(&enabled),
		CollectorType: types.StringValue("ALLOY"),
	}

	expectedMsg := &collectorv1.Collector{
		Id: id,
		RemoteAttributes: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
		Enabled:       &enabled,
		CollectorType: collectorv1.CollectorType_COLLECTOR_TYPE_ALLOY,
	}

	ctx := context.Background()
	actualMsg, diags := collectorResourceModelToMessage(ctx, model)
	require.False(t, diags.HasError())
	require.Equal(t, expectedMsg, actualMsg)
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
			require.False(t, diags.HasError())
			require.Equal(t, tt.expected, actual)
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
			require.False(t, diags.HasError())
			require.Equal(t, tt.expected, actual)
		})
	}
}
