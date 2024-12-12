package fleetmanagement

import (
	collectorv1 "github.com/grafana/fleet-management-api/api/gen/proto/go/collector/v1"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type collectorModel struct {
	ID                 types.String `tfsdk:"id"`
	AttributeOverrides types.Map    `tfsdk:"attribute_overrides"`
	Enabled            types.Bool   `tfsdk:"enabled"`
}

func collectorMessageToModel(msg *collectorv1.Collector) *collectorModel {
	return &collectorModel{
		ID:                 types.StringValue(msg.Id),
		AttributeOverrides: nativeMapToTFMap(msg.AttributeOverrides),
		Enabled:            types.BoolPointerValue(msg.Enabled),
	}
}

func collectorModelToMessage(model *collectorModel) (*collectorv1.Collector, error) {
	attributeOverrides, err := tfMapToNativeMap(model.AttributeOverrides)
	if err != nil {
		return nil, err
	}

	return &collectorv1.Collector{
		Id:                 model.ID.ValueString(),
		AttributeOverrides: attributeOverrides,
		Enabled:            tfBoolToNativeBoolPtr(model.Enabled),
	}, nil
}
