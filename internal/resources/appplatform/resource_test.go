package appplatform

import (
	"context"
	"testing"

	sdkresource "github.com/grafana/grafana-app-sdk/resource"
	"github.com/grafana/grafana/apps/playlist/pkg/apis/playlist/v0alpha1"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

func makeMockResource(name string) sdkresource.Object {
	obj := v0alpha1.PlaylistKind().Schema.ZeroValue()
	obj.SetName(name)
	return obj
}

func TestSaveResourceToModel_ID_Field(t *testing.T) {
	ctx := context.Background()

	src := makeMockResource("test-uid")

	dst := &ResourceModel{
		Metadata: types.ObjectValueMust(
			map[string]attr.Type{
				"uuid":       types.StringType,
				"uid":        types.StringType,
				"folder_uid": types.StringType,
				"version":    types.StringType,
				"url":        types.StringType,
			},
			map[string]attr.Value{
				"uuid":       types.StringNull(),
				"uid":        types.StringNull(),
				"folder_uid": types.StringNull(),
				"version":    types.StringNull(),
				"url":        types.StringNull(),
			},
		),
	}

	diags := SaveResourceToModel(ctx, src, dst)
	require.False(t, diags.HasError())
	require.Equal(t, "test-uid", dst.ID.ValueString())
}
