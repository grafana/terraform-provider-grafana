package appplatform

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

func objectValue(t *testing.T, attrTypes map[string]attr.Type, values map[string]attr.Value) types.Object {
	t.Helper()
	v, diags := types.ObjectValue(attrTypes, values)
	require.False(t, diags.HasError())
	return v
}

func stringListValue(t *testing.T, values ...string) types.List {
	t.Helper()
	v, diags := types.ListValueFrom(context.Background(), types.StringType, values)
	require.False(t, diags.HasError())
	return v
}

func stringMapValue(t *testing.T, values map[string]string) types.Map {
	t.Helper()
	v, diags := types.MapValueFrom(context.Background(), types.StringType, values)
	require.False(t, diags.HasError(), diags)
	return v
}
