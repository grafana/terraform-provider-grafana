package fleetmanagement

import (
	pipelinev1 "github.com/grafana/fleet-management-api/api/gen/proto/go/pipeline/v1"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type pipelineModel struct {
	Name     types.String     `tfsdk:"name"`
	Contents AlloyConfigValue `tfsdk:"contents"`
	Matchers types.List       `tfsdk:"matchers"`
	Enabled  types.Bool       `tfsdk:"enabled"`
	ID       types.String     `tfsdk:"id"`
}

func pipelineMessageToModel(msg *pipelinev1.Pipeline) *pipelineModel {
	return &pipelineModel{
		Name:     types.StringValue(msg.Name),
		Contents: NewAlloyConfigValue(msg.Contents),
		Matchers: nativeSliceToTFList(msg.Matchers),
		Enabled:  types.BoolPointerValue(msg.Enabled),
		ID:       types.StringPointerValue(msg.Id),
	}
}

func pipelineModelToMessage(model *pipelineModel) (*pipelinev1.Pipeline, error) {
	matchers, err := tfListToNativeSlice(model.Matchers)
	if err != nil {
		return nil, err
	}

	return &pipelinev1.Pipeline{
		Name:     model.Name.ValueString(),
		Contents: model.Contents.ValueString(),
		Matchers: matchers,
		Enabled:  tfBoolToNativeBoolPtr(model.Enabled),
		Id:       tfStringToNativeStringPtr(model.ID),
	}, nil
}
