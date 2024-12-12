package fleetmanagement

import (
	"testing"

	pipelinev1 "github.com/grafana/fleet-management-api/api/gen/proto/go/pipeline/v1"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func TestPipelineMessageToModel(t *testing.T) {
	name := "test_name"
	contents := "logging {}"
	matcher1 := "collector.os=\"linux\""
	matcher2 := "owner=\"TEAM-A\""
	enabled := true
	id := "123"

	msg := &pipelinev1.Pipeline{
		Name:     name,
		Contents: contents,
		Matchers: []string{
			matcher1,
			matcher2,
		},
		Enabled: &enabled,
		Id:      &id,
	}

	expectedModel := &pipelineModel{
		Name:     types.StringValue(name),
		Contents: NewAlloyConfigValue(contents),
		Matchers: types.ListValueMust(
			types.StringType,
			[]attr.Value{
				types.StringValue(matcher1),
				types.StringValue(matcher2),
			},
		),
		Enabled: types.BoolPointerValue(&enabled),
		ID:      types.StringPointerValue(&id),
	}

	actualModel := pipelineMessageToModel(msg)
	assert.Equal(t, expectedModel, actualModel)
}

func TestPipelineModelToMessage(t *testing.T) {
	name := "test_name"
	contents := "logging {}"
	matcher1 := "collector.os=\"linux\""
	matcher2 := "owner=\"TEAM-A\""
	enabled := true
	id := "123"

	t.Run("successfully converts model to message", func(t *testing.T) {
		model := &pipelineModel{
			Name:     types.StringValue(name),
			Contents: NewAlloyConfigValue(contents),
			Matchers: types.ListValueMust(
				types.StringType,
				[]attr.Value{
					types.StringValue(matcher1),
					types.StringValue(matcher2),
				},
			),
			Enabled: types.BoolPointerValue(&enabled),
			ID:      types.StringPointerValue(&id),
		}

		expectedMsg := &pipelinev1.Pipeline{
			Name:     name,
			Contents: contents,
			Matchers: []string{matcher1, matcher2},
			Enabled:  &enabled,
			Id:       &id,
		}

		actualMsg, err := pipelineModelToMessage(model)
		assert.NoError(t, err)
		assert.Equal(t, expectedMsg, actualMsg)
	})

	t.Run("error when converting model to message (invalid list type)", func(t *testing.T) {
		model := &pipelineModel{
			Name:     types.StringValue(name),
			Contents: NewAlloyConfigValue(contents),
			Matchers: types.ListValueMust(
				types.BoolType,
				[]attr.Value{types.BoolValue(true)},
			),
			Enabled: types.BoolPointerValue(&enabled),
			ID:      types.StringPointerValue(&id),
		}

		actualMsg, err := pipelineModelToMessage(model)
		assert.Error(t, err)
		assert.Nil(t, actualMsg)
	})
}
