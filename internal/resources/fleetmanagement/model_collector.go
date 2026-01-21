package fleetmanagement

import (
	"context"

	collectorv1 "github.com/grafana/fleet-management-api/api/gen/proto/go/collector/v1"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type collectorDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	RemoteAttributes types.Map    `tfsdk:"remote_attributes"`
	LocalAttributes  types.Map    `tfsdk:"local_attributes"`
	Enabled          types.Bool   `tfsdk:"enabled"`
	CollectorType    types.String `tfsdk:"collector_type"`
}

type collectorDataSourcesModel struct {
	Collectors []collectorDataSourceModel `tfsdk:"collectors"`
}

type collectorResourceModel struct {
	ID               types.String `tfsdk:"id"`
	RemoteAttributes types.Map    `tfsdk:"remote_attributes"`
	Enabled          types.Bool   `tfsdk:"enabled"`
	CollectorType    types.String `tfsdk:"collector_type"`
}

func collectorMessageToDataSourceModel(ctx context.Context, msg *collectorv1.Collector) (*collectorDataSourceModel, diag.Diagnostics) {
	remoteAttributes, diags := nativeStringMapToTFStringMap(ctx, msg.RemoteAttributes)
	if diags.HasError() {
		return nil, diags
	}

	localAttributes, diags := nativeStringMapToTFStringMap(ctx, msg.LocalAttributes)
	if diags.HasError() {
		return nil, diags
	}

	return &collectorDataSourceModel{
		ID:               types.StringValue(msg.Id),
		RemoteAttributes: remoteAttributes,
		LocalAttributes:  localAttributes,
		Enabled:          types.BoolPointerValue(msg.Enabled),
		CollectorType:    types.StringValue(collectorTypeToString(msg.CollectorType)),
	}, nil
}

func collectorMessageToResourceModel(ctx context.Context, msg *collectorv1.Collector) (*collectorResourceModel, diag.Diagnostics) {
	remoteAttributes, diags := nativeStringMapToTFStringMap(ctx, msg.RemoteAttributes)
	if diags.HasError() {
		return nil, diags
	}

	return &collectorResourceModel{
		ID:               types.StringValue(msg.Id),
		RemoteAttributes: remoteAttributes,
		Enabled:          types.BoolPointerValue(msg.Enabled),
		CollectorType:    types.StringValue(collectorTypeToString(msg.CollectorType)),
	}, nil
}

func collectorResourceModelToMessage(ctx context.Context, model *collectorResourceModel) (*collectorv1.Collector, diag.Diagnostics) {
	remoteAttributes, diags := tfStringMapToNativeStringMap(ctx, model.RemoteAttributes)
	if diags.HasError() {
		return nil, diags
	}

	return &collectorv1.Collector{
		Id:               model.ID.ValueString(),
		RemoteAttributes: remoteAttributes,
		Enabled:          tfBoolToNativeBoolPtr(model.Enabled),
		CollectorType:    stringToCollectorType(model.CollectorType.ValueString()),
	}, nil
}

// collectorTypeToString converts the proto enum to a Terraform-friendly string.
func collectorTypeToString(ct collectorv1.CollectorType) string {
	switch ct {
	case collectorv1.CollectorType_COLLECTOR_TYPE_ALLOY:
		return "ALLOY"
	case collectorv1.CollectorType_COLLECTOR_TYPE_OTEL:
		return "OTEL"
	default:
		return ""
	}
}

// stringToCollectorType converts a Terraform string to the proto enum.
func stringToCollectorType(s string) collectorv1.CollectorType {
	switch s {
	case "ALLOY":
		return collectorv1.CollectorType_COLLECTOR_TYPE_ALLOY
	case "OTEL":
		return collectorv1.CollectorType_COLLECTOR_TYPE_OTEL
	default:
		return collectorv1.CollectorType_COLLECTOR_TYPE_UNSPECIFIED
	}
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
