package appplatform

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

// TestEmptyMetadataObjectMatchesBaseMetadataTypes guards the metadata object the secret
// resources build during import. It is assembled with types.ObjectValueMust, which panics if the
// attribute types and values disagree, and the only paths that reach it are Enterprise-gated
// acceptance tests -- so assert the invariant directly here instead.
func TestEmptyMetadataObjectMatchesBaseMetadataTypes(t *testing.T) {
	obj := emptyMetadataObject()

	require.Equal(t, baseMetadataTypeMap(), obj.AttributeTypes(context.Background()))

	for name, value := range obj.Attributes() {
		require.True(t, value.IsNull(), "expected %s to be null, got %v", name, value)
	}
}

// TestResourceMetadataModelMatchesBaseMetadataTypes covers the other half of the same import
// path: keeperActivationResource.ImportState deserializes ResourceMetadataModel against
// baseMetadataTypeMap, which errors if the struct and the type map have drifted apart.
func TestResourceMetadataModelMatchesBaseMetadataTypes(t *testing.T) {
	ctx := context.Background()

	obj, diags := types.ObjectValueFrom(ctx, baseMetadataTypeMap(), ResourceMetadataModel{
		UID:         types.StringValue("some-uid"),
		Annotations: types.MapNull(types.StringType),
	})
	require.False(t, diags.HasError(), "%v", diags)
	require.Equal(t, types.StringValue("some-uid"), obj.Attributes()["uid"])

	// And back into the struct, as SetMetadataFromModel's caller does.
	mod, modDiags := baseMetadataFromObject(obj)
	require.False(t, modDiags.HasError(), "%v", modDiags)
	require.Equal(t, "some-uid", mod.UID.ValueString())
}

// TestBaseMetadataFromObjectRequiresUID documents that a metadata object without a usable uid
// is an error rather than a silently empty Kubernetes object name, which would otherwise issue a
// request against "".
func TestBaseMetadataFromObjectRequiresUID(t *testing.T) {
	withoutUID := types.ObjectValueMust(
		map[string]attr.Type{"uuid": types.StringType},
		map[string]attr.Value{"uuid": types.StringNull()},
	)

	_, diags := baseMetadataFromObject(withoutUID)
	require.True(t, diags.HasError(), "expected a diagnostic for a metadata object with no uid")
	require.Contains(t, diags.Errors()[0].Detail(), "metadata.uid")
}
