package fleetmanagement

import (
	"testing"

	collectorv1 "github.com/grafana/fleet-management-api/api/gen/proto/go/collector/v1"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
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

	actualModel := collectorMessageToModel(msg)
	assert.Equal(t, expectedModel, actualModel)
}

func TestCollectorModelToMessage(t *testing.T) {
	id := "test_id"
	enabled := true

	t.Run("successfully converts model to message", func(t *testing.T) {
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

		actualMsg, err := collectorModelToMessage(model)
		assert.NoError(t, err)
		assert.Equal(t, expectedMsg, actualMsg)
	})

	t.Run("error when converting model to message (invalid map type)", func(t *testing.T) {
		model := &collectorModel{
			ID: types.StringValue(id),
			AttributeOverrides: types.MapValueMust(
				types.BoolType,
				map[string]attr.Value{
					"key1": types.BoolValue(true),
				},
			),
			Enabled: types.BoolPointerValue(&enabled),
		}

		actualMsg, err := collectorModelToMessage(model)
		assert.Error(t, err)
		assert.Nil(t, actualMsg)
	})
}
