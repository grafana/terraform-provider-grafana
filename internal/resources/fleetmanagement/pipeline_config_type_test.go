package fleetmanagement

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/require"
)

func TestPipelineConfigType_Equal(t *testing.T) {
	type1 := PipelineConfigType
	type2 := PipelineConfigType
	type3 := types.StringType

	require.True(t, type1.Equal(type2))
	require.False(t, type1.Equal(type3))
}

func TestPipelineConfigType_String(t *testing.T) {
	require.Equal(t, "PipelineConfigType", PipelineConfigType.String())
}

func TestPipelineConfigType_ValueFromString(t *testing.T) {
	ctx := context.Background()
	stringValue := types.StringValue("test")

	pipelineCfgValue, diags := PipelineConfigType.ValueFromString(ctx, stringValue)
	require.False(t, diags.HasError())
	expected := PipelineConfigValue{StringValue: stringValue}
	require.Equal(t, expected, pipelineCfgValue)
}

func TestPipelineConfigType_ValueFromTerraform(t *testing.T) {
	ctx := context.Background()
	tfValue := tftypes.NewValue(tftypes.String, "test")

	pipelineCfgValue, err := PipelineConfigType.ValueFromTerraform(ctx, tfValue)
	require.NoError(t, err)
	expected := PipelineConfigValue{StringValue: types.StringValue("test")}
	require.Equal(t, expected, pipelineCfgValue)
}

func TestPipelineConfigType_ValueType(t *testing.T) {
	ctx := context.Background()
	pipelineCfgValue := PipelineConfigType.ValueType(ctx)
	require.IsType(t, PipelineConfigValue{}, pipelineCfgValue)
}
