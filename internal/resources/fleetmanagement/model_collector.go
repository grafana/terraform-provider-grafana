package fleetmanagement

import (
	"context"

	collectorv1 "github.com/grafana/fleet-management-api/api/gen/proto/go/collector/v1"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type collectorModel struct {
	ID               types.String `tfsdk:"id"`
	RemoteAttributes types.Map    `tfsdk:"remote_attributes"`
	Enabled          types.Bool   `tfsdk:"enabled"`
}

func collectorMessageToModel(ctx context.Context, msg *collectorv1.Collector) (*collectorModel, diag.Diagnostics) {
	remoteAttributes, diags := nativeStringMapToTFStringMap(ctx, msg.RemoteAttributes)
	if diags.HasError() {
		return nil, diags
	}

	return &collectorModel{
		ID:               types.StringValue(msg.Id),
		RemoteAttributes: remoteAttributes,
		Enabled:          types.BoolPointerValue(msg.Enabled),
	}, nil
}

func collectorModelToMessage(ctx context.Context, model *collectorModel) (*collectorv1.Collector, diag.Diagnostics) {
	remoteAttributes, diags := tfStringMapToNativeStringMap(ctx, model.RemoteAttributes)
	if diags.HasError() {
		return nil, diags
	}

	return &collectorv1.Collector{
		Id:               model.ID.ValueString(),
		RemoteAttributes: remoteAttributes,
		Enabled:          tfBoolToNativeBoolPtr(model.Enabled),
	}, nil
}

func nativeStringMapToTFStringMap(ctx context.Context, nativeMap map[string]string) (types.Map, diag.Diagnostics) {
	if len(nativeMap) == 0 {
		return types.MapValueMust(types.StringType, map[string]attr.Value{}), nil
	}

	return types.MapValueFrom(ctx, types.StringType, nativeMap)
}

func tfStringMapToNativeStringMap(ctx context.Context, tfMap types.Map) (map[string]string, diag.Diagnostics) {
	if tfMap.IsNull() || tfMap.IsUnknown() {
		return map[string]string{}, nil
	}

	length := len(tfMap.Elements())
	elements := make(map[string]types.String, length)
	diags := tfMap.ElementsAs(ctx, &elements, false)
	if diags.HasError() {
		return nil, diags
	}

	nativeMap := make(map[string]string, length)
	for key, val := range elements {
		nativeMap[key] = val.ValueString()
	}

	return nativeMap, nil
}
