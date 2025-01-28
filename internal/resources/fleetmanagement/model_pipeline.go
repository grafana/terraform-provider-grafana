package fleetmanagement

import (
	"context"

	pipelinev1 "github.com/grafana/fleet-management-api/api/gen/proto/go/pipeline/v1"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type pipelineModel struct {
	Name     types.String                             `tfsdk:"name"`
	Contents AlloyConfigValue                         `tfsdk:"contents"`
	Matchers GenericListValue[PrometheusMatcherValue] `tfsdk:"matchers"`
	Enabled  types.Bool                               `tfsdk:"enabled"`
	ID       types.String                             `tfsdk:"id"`
}

func pipelineMessageToModel(ctx context.Context, msg *pipelinev1.Pipeline) (*pipelineModel, diag.Diagnostics) {
	matcherValues, diags := stringSliceToMatcherValues(ctx, msg.Matchers)
	if diags.HasError() {
		return nil, diags
	}

	return &pipelineModel{
		Name:     types.StringValue(msg.Name),
		Contents: NewAlloyConfigValue(msg.Contents),
		Matchers: matcherValues,
		Enabled:  types.BoolPointerValue(msg.Enabled),
		ID:       types.StringPointerValue(msg.Id),
	}, nil
}

func pipelineModelToMessage(ctx context.Context, model *pipelineModel) (*pipelinev1.Pipeline, diag.Diagnostics) {
	matchers, diags := matcherValuesToStringSlice(ctx, model.Matchers)
	if diags.HasError() {
		return nil, diags
	}

	return &pipelinev1.Pipeline{
		Name:     model.Name.ValueString(),
		Contents: model.Contents.ValueString(),
		Matchers: matchers,
		Enabled:  tfBoolToNativeBoolPtr(model.Enabled),
		Id:       tfStringToNativeStringPtr(model.ID),
	}, nil
}

func stringSliceToMatcherValues(ctx context.Context, matchers []string) (GenericListValue[PrometheusMatcherValue], diag.Diagnostics) {
	if len(matchers) == 0 {
		return NewGenericListValueMust[PrometheusMatcherValue](ctx, []attr.Value{}), nil
	}

	return NewGenericListValueFrom[PrometheusMatcherValue](ctx, PrometheusMatcherType, matchers)
}

func matcherValuesToStringSlice(ctx context.Context, matcherValues GenericListValue[PrometheusMatcherValue]) ([]string, diag.Diagnostics) {
	if matcherValues.IsNull() || matcherValues.IsUnknown() {
		return []string{}, nil
	}

	length := len(matcherValues.Elements())
	elements := make([]PrometheusMatcherValue, length)
	diags := matcherValues.ElementsAs(ctx, &elements, false)
	if diags.HasError() {
		return nil, diags
	}

	matchers := make([]string, length)
	for i, element := range elements {
		matchers[i] = element.ValueString()
	}

	return matchers, nil
}
