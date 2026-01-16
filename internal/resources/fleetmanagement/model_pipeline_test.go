package fleetmanagement

import (
	"context"
	"testing"

	pipelinev1 "github.com/grafana/fleet-management-api/api/gen/proto/go/pipeline/v1"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
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
		Enabled:    &enabled,
		Id:         &id,
		ConfigType: pipelinev1.ConfigType_CONFIG_TYPE_ALLOY,
	}

	expectedModel := &pipelineModel{
		Name:     types.StringValue(name),
		Contents: NewPipelineConfigValue(contents),
		Matchers: NewListOfPrometheusMatcherValueMust(
			[]attr.Value{
				basetypes.NewStringValue(matcher1),
				basetypes.NewStringValue(matcher2),
			},
		),
		Enabled:    types.BoolPointerValue(&enabled),
		ID:         types.StringPointerValue(&id),
		ConfigType: types.StringValue("ALLOY"),
	}

	ctx := context.Background()
	actualModel, diags := pipelineMessageToModel(ctx, msg)
	assert.False(t, diags.HasError())
	assert.Equal(t, expectedModel, actualModel)
}

func TestPipelineModelToMessage(t *testing.T) {
	name := "test_name"
	contents := "logging {}"
	matcher1 := "collector.os=\"linux\""
	matcher2 := "owner=\"TEAM-A\""
	enabled := true
	id := "123"

	model := &pipelineModel{
		Name:     types.StringValue(name),
		Contents: NewPipelineConfigValue(contents),
		Matchers: NewListOfPrometheusMatcherValueMust(
			[]attr.Value{
				basetypes.NewStringValue(matcher1),
				basetypes.NewStringValue(matcher2),
			},
		),
		Enabled:    types.BoolPointerValue(&enabled),
		ID:         types.StringPointerValue(&id),
		ConfigType: types.StringValue("ALLOY"),
	}

	expectedMsg := &pipelinev1.Pipeline{
		Name:       name,
		Contents:   contents,
		Matchers:   []string{matcher1, matcher2},
		Enabled:    &enabled,
		Id:         &id,
		ConfigType: pipelinev1.ConfigType_CONFIG_TYPE_ALLOY,
	}

	ctx := context.Background()
	actualMsg, diags := pipelineModelToMessage(ctx, model)
	assert.False(t, diags.HasError())
	assert.Equal(t, expectedMsg, actualMsg)
}

func TestStringSliceToMatcherValues(t *testing.T) {
	tests := []struct {
		name        string
		nativeSlice []string
		expected    ListOfPrometheusMatcherValue
	}{
		{
			"nil slice",
			nil,
			NewListOfPrometheusMatcherValueMust([]attr.Value{}),
		},
		{
			"empty slice",
			[]string{},
			NewListOfPrometheusMatcherValueMust([]attr.Value{}),
		},
		{
			"non-empty slice",
			[]string{
				"collector.os=linux",
				"collector.os=darwin",
			},
			NewListOfPrometheusMatcherValueMust(
				[]attr.Value{
					basetypes.NewStringValue("collector.os=linux"),
					basetypes.NewStringValue("collector.os=darwin"),
				},
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			actual, diags := stringSliceToMatcherValues(ctx, tt.nativeSlice)
			assert.False(t, diags.HasError())
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestMatcherValuesToStringSlice(t *testing.T) {
	tests := []struct {
		name        string
		genericList ListOfPrometheusMatcherValue
		expected    []string
	}{
		{
			"null list",
			NewListOfPrometheusMatcherValueNull(),
			[]string{},
		},
		{
			"unknown list",
			NewListOfPrometheusMatcherValueUnknown(),
			[]string{},
		},
		{
			"empty list",
			NewListOfPrometheusMatcherValueMust([]attr.Value{}),
			[]string{},
		},
		{
			"non-empty list",
			NewListOfPrometheusMatcherValueMust(
				[]attr.Value{
					basetypes.NewStringValue("collector.os=linux"),
					basetypes.NewStringValue("collector.os=darwin"),
				},
			),
			[]string{
				"collector.os=linux",
				"collector.os=darwin",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			actual, diags := matcherValuesToStringSlice(ctx, tt.genericList)
			assert.False(t, diags.HasError())
			assert.Equal(t, tt.expected, actual)
		})
	}
}
