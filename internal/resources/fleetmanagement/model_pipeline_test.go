package fleetmanagement

import (
	"context"
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
		Matchers: NewGenericListValueMust[PrometheusMatcherValue](
			context.Background(),
			[]attr.Value{
				NewPrometheusMatcherValue(matcher1),
				NewPrometheusMatcherValue(matcher2),
			},
		),
		Enabled: types.BoolPointerValue(&enabled),
		ID:      types.StringPointerValue(&id),
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
		Contents: NewAlloyConfigValue(contents),
		Matchers: NewGenericListValueMust[PrometheusMatcherValue](
			context.Background(),
			[]attr.Value{
				NewPrometheusMatcherValue(matcher1),
				NewPrometheusMatcherValue(matcher2),
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

	ctx := context.Background()
	actualMsg, diags := pipelineModelToMessage(ctx, model)
	assert.False(t, diags.HasError())
	assert.Equal(t, expectedMsg, actualMsg)
}

func TestStringSliceToMatcherValues(t *testing.T) {
	tests := []struct {
		name        string
		nativeSlice []string
		expected    GenericListValue[PrometheusMatcherValue]
	}{
		{
			"nil slice",
			nil,
			NewGenericListValueMust[PrometheusMatcherValue](context.Background(), []attr.Value{}),
		},
		{
			"empty slice",
			[]string{},
			NewGenericListValueMust[PrometheusMatcherValue](context.Background(), []attr.Value{}),
		},
		{
			"non-empty slice",
			[]string{
				"collector.os=linux",
				"collector.os=darwin",
			},
			NewGenericListValueMust[PrometheusMatcherValue](
				context.Background(),
				[]attr.Value{
					NewPrometheusMatcherValue("collector.os=linux"),
					NewPrometheusMatcherValue("collector.os=darwin"),
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
		genericList GenericListValue[PrometheusMatcherValue]
		expected    []string
	}{
		{
			"null list",
			NewGenericListValueNull[PrometheusMatcherValue](context.Background()),
			[]string{},
		},
		{
			"unknown list",
			NewGenericListValueUnknown[PrometheusMatcherValue](context.Background()),
			[]string{},
		},
		{
			"empty list",
			NewGenericListValueMust[PrometheusMatcherValue](context.Background(), []attr.Value{}),
			[]string{},
		},
		{
			"non-empty list",
			NewGenericListValueMust[PrometheusMatcherValue](
				context.Background(),
				[]attr.Value{
					NewPrometheusMatcherValue("collector.os=linux"),
					NewPrometheusMatcherValue("collector.os=darwin"),
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
