package fleetmanagement

import (
	"context"
	"fmt"

	pipelinev1 "github.com/grafana/fleet-management-api/api/gen/proto/go/pipeline/v1"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type pipelineModel struct {
	Name       types.String                 `tfsdk:"name"`
	Contents   AlloyConfigValue             `tfsdk:"contents"`
	Matchers   ListOfPrometheusMatcherValue `tfsdk:"matchers"`
	Enabled    types.Bool                   `tfsdk:"enabled"`
	ID         types.String                 `tfsdk:"id"`
	ConfigType types.String                 `tfsdk:"config_type"`
}

func pipelineMessageToModel(ctx context.Context, msg *pipelinev1.Pipeline) (*pipelineModel, diag.Diagnostics) {
	matcherValues, diags := stringSliceToMatcherValues(ctx, msg.Matchers)
	if diags.HasError() {
		return nil, diags
	}

	return &pipelineModel{
		Name:       types.StringValue(msg.Name),
		Contents:   NewAlloyConfigValue(msg.Contents),
		Matchers:   matcherValues,
		Enabled:    types.BoolPointerValue(msg.Enabled),
		ID:         types.StringPointerValue(msg.Id),
		ConfigType: types.StringValue(configTypeToString(msg.ConfigType)),
	}, nil
}

func pipelineModelToMessage(ctx context.Context, model *pipelineModel) (*pipelinev1.Pipeline, diag.Diagnostics) {
	matchers, diags := matcherValuesToStringSlice(ctx, model.Matchers)
	if diags.HasError() {
		return nil, diags
	}

	return &pipelinev1.Pipeline{
		Name:       model.Name.ValueString(),
		Contents:   model.Contents.ValueString(),
		Matchers:   matchers,
		Enabled:    tfBoolToNativeBoolPtr(model.Enabled),
		Id:         tfStringToNativeStringPtr(model.ID),
		ConfigType: stringToConfigType(model.ConfigType.ValueString()),
	}, nil
}

func stringSliceToMatcherValues(ctx context.Context, matchers []string) (ListOfPrometheusMatcherValue, diag.Diagnostics) {
	if len(matchers) == 0 {
		return NewListOfPrometheusMatcherValueMust([]attr.Value{}), nil
	}

	return NewListOfPrometheusMatcherValueFrom(ctx, matchers)
}

func matcherValuesToStringSlice(ctx context.Context, matcherValues ListOfPrometheusMatcherValue) ([]string, diag.Diagnostics) {
	var diags diag.Diagnostics

	if matcherValues.IsNull() || matcherValues.IsUnknown() {
		return []string{}, nil
	}

	elements := matcherValues.Elements()
	result := make([]string, len(elements))

	for i, element := range elements {
		stringValue, ok := element.(basetypes.StringValue)
		if !ok {
			diags.AddError(
				"Type Conversion Error",
				fmt.Sprintf("Expected string value, got: %T", element),
			)
			return nil, diags
		}
		result[i] = stringValue.ValueString()
	}

	return result, diags
}

func configTypeToString(ct pipelinev1.ConfigType) string {
	switch ct {
	case pipelinev1.ConfigType_CONFIG_TYPE_ALLOY:
		return "ALLOY"
	case pipelinev1.ConfigType_CONFIG_TYPE_OTEL:
		return "OTEL"
	default:
		return ""
	}
}

func stringToConfigType(s string) pipelinev1.ConfigType {
	switch s {
	case "ALLOY":
		return pipelinev1.ConfigType_CONFIG_TYPE_ALLOY
	case "OTEL":
		return pipelinev1.ConfigType_CONFIG_TYPE_OTEL
	default:
		return pipelinev1.ConfigType_CONFIG_TYPE_UNSPECIFIED
	}
}
