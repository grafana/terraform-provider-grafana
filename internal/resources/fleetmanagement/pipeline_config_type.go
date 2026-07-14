package fleetmanagement

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

var (
	_ basetypes.StringTypable = pipelineConfigType{}
)

var (
	PipelineConfigType = pipelineConfigType{}
)

type pipelineConfigType struct {
	basetypes.StringType
}

func (t pipelineConfigType) Equal(o attr.Type) bool {
	other, ok := o.(pipelineConfigType)
	if !ok {
		return false
	}

	return t.StringType.Equal(other.StringType)
}

func (t pipelineConfigType) String() string {
	return "PipelineConfigType"
}

func (t pipelineConfigType) ValueFromString(ctx context.Context, in basetypes.StringValue) (basetypes.StringValuable, diag.Diagnostics) {
	return PipelineConfigValue{
		StringValue: in,
	}, nil
}

func (t pipelineConfigType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	attrValue, err := t.StringType.ValueFromTerraform(ctx, in)
	if err != nil {
		return nil, err
	}

	stringValue, ok := attrValue.(basetypes.StringValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type of %T", attrValue)
	}

	stringValuable, diags := t.ValueFromString(ctx, stringValue)
	if diags.HasError() {
		return nil, fmt.Errorf("unexpected error converting StringValue to StringValuable: %v", diags)
	}

	return stringValuable, nil
}

func (t pipelineConfigType) ValueType(ctx context.Context) attr.Value {
	return PipelineConfigValue{}
}
