package fleetmanagement

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
)

func TestListOfPrometheusMatcherType_Equal(t *testing.T) {
	type1 := ListOfPrometheusMatcherType
	type2 := ListOfPrometheusMatcherType
	type3 := types.ListType{ElemType: types.StringType}

	assert.True(t, type1.Equal(type2))
	assert.False(t, type1.Equal(type3))
}

func TestListOfPrometheusMatcherType_String(t *testing.T) {
	assert.Equal(t, "ListOfPrometheusMatcherType", ListOfPrometheusMatcherType.String())
}

func TestListOfPrometheusMatcherType_ValueFromList(t *testing.T) {
	ctx := context.Background()
	attrElements := []attr.Value{basetypes.NewStringValue("collector.os=linux")}
	listValue := basetypes.NewListValueMust(types.StringType, attrElements)

	promMatcherListValue, diags := ListOfPrometheusMatcherType.ValueFromList(ctx, listValue)
	assert.False(t, diags.HasError())
	promMatcherElements := promMatcherListValue.(ListOfPrometheusMatcherValue).Elements()
	assert.ElementsMatch(t, attrElements, promMatcherElements)
}

func TestListOfPrometheusMatcherType_ValueFromTerraform(t *testing.T) {
	ctx := context.Background()
	tfValue := tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{tftypes.NewValue(tftypes.String, "collector.os=linux")})

	promMatcherListValue, err := ListOfPrometheusMatcherType.ValueFromTerraform(ctx, tfValue)
	assert.NoError(t, err)
	promMatcherElements := promMatcherListValue.(ListOfPrometheusMatcherValue).Elements()
	expected := []attr.Value{basetypes.NewStringValue("collector.os=linux")}
	assert.ElementsMatch(t, expected, promMatcherElements)
}

func TestListOfPrometheusMatcherType_ValueType(t *testing.T) {
	ctx := context.Background()
	promMatcherListValue := ListOfPrometheusMatcherType.ValueType(ctx)
	assert.IsType(t, ListOfPrometheusMatcherValue{}, promMatcherListValue)
}
