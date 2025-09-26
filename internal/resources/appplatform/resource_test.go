package appplatform

import (
	"context"
	"testing"

	"github.com/grafana/authlib/claims"
	sdkresource "github.com/grafana/grafana-app-sdk/resource"
	"github.com/grafana/grafana/apps/playlist/pkg/apis/playlist/v0alpha1"
	"github.com/grafana/grafana/pkg/apimachinery/utils"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/stretchr/testify/require"
	k8stypes "k8s.io/apimachinery/pkg/types"
)

func makeMockResource(name, uid string) sdkresource.Object {
	obj := v0alpha1.PlaylistKind().Schema.ZeroValue()
	obj.SetName(name)
	obj.SetUID(k8stypes.UID(uid))
	return obj
}

func TestSaveResourceToModel(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name                  string
		annotations           map[string]string
		expectAnnotationsNull bool
	}{
		{
			name:                  "basic ID field",
			expectAnnotationsNull: true,
		},
		{
			name: "with annotations",
			annotations: map[string]string{
				"grafana.com/provenance": "api",
				"team":                   "platform",
			},
			expectAnnotationsNull: false,
		},
		{
			name:                  "empty annotations map",
			annotations:           map[string]string{},
			expectAnnotationsNull: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testUUID := "test-uuid-12345"
			src := makeMockResource("test-name", testUUID)

			if tt.annotations != nil {
				meta, err := utils.MetaAccessor(src)
				require.NoError(t, err)
				meta.SetAnnotations(tt.annotations)
			}

			dst := &ResourceModel{
				Metadata: types.ObjectValueMust(
					map[string]attr.Type{
						"uuid":        types.StringType,
						"uid":         types.StringType,
						"folder_uid":  types.StringType,
						"version":     types.StringType,
						"url":         types.StringType,
						"annotations": types.MapType{ElemType: types.StringType},
					},
					map[string]attr.Value{
						"uuid":        types.StringNull(),
						"uid":         types.StringNull(),
						"folder_uid":  types.StringNull(),
						"version":     types.StringNull(),
						"url":         types.StringNull(),
						"annotations": types.MapNull(types.StringType),
					},
				),
			}

			diags := SaveResourceToModel(ctx, src, dst)
			require.False(t, diags.HasError())
			require.Equal(t, testUUID, dst.ID.ValueString())

			var metadata ResourceMetadataModel
			dst.Metadata.As(ctx, &metadata, basetypes.ObjectAsOptions{
				UnhandledNullAsEmpty:    true,
				UnhandledUnknownAsEmpty: true,
			})

			if tt.expectAnnotationsNull {
				require.True(t, metadata.Annotations.IsNull())
			} else {
				require.False(t, metadata.Annotations.IsNull())

				annotations := make(map[string]string)
				metadata.Annotations.ElementsAs(ctx, &annotations, false)

				for key, expectedValue := range tt.annotations {
					require.Equal(t, expectedValue, annotations[key])
				}
			}
		})
	}
}

func TestGetModelFromMetadata(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name                  string
		annotations           map[string]string
		expectAnnotationsNull bool
	}{
		{
			name: "with annotations",
			annotations: map[string]string{
				"grafana.com/provenance": "api",
				"custom.annotation":      "value",
			},
			expectAnnotationsNull: false,
		},
		{
			name:                  "nil annotations",
			annotations:           nil,
			expectAnnotationsNull: true,
		},
		{
			name:                  "empty annotations map",
			annotations:           map[string]string{},
			expectAnnotationsNull: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := makeMockResource("test-name", "test-uuid")

			if tt.annotations != nil {
				meta, err := utils.MetaAccessor(src)
				require.NoError(t, err)
				meta.SetAnnotations(tt.annotations)
			}

			dst := &ResourceMetadataModel{}
			diags := GetModelFromMetadata(ctx, src, dst)
			require.False(t, diags.HasError())

			if tt.expectAnnotationsNull {
				require.True(t, dst.Annotations.IsNull())
			} else {
				require.False(t, dst.Annotations.IsNull())

				annotations := make(map[string]string)
				dst.Annotations.ElementsAs(ctx, &annotations, false)

				for key, expectedValue := range tt.annotations {
					require.Equal(t, expectedValue, annotations[key])
				}
			}
		})
	}
}

func TestNamespaceForClient(t *testing.T) {
	tests := []struct {
		name            string
		orgID           int64
		stackID         int64
		expectNamespace string
		expectErr       string
	}{
		{
			name:            "stack only",
			stackID:         1,
			expectNamespace: claims.CloudNamespaceFormatter(1),
		},
		{
			name:            "stack_id preferred over org_id",
			orgID:           2,
			stackID:         3,
			expectNamespace: claims.CloudNamespaceFormatter(3),
		},
		{
			name:            "org_id only",
			orgID:           4,
			expectNamespace: claims.OrgNamespaceFormatter(4),
		},
		{
			name:      "no identifiers",
			expectErr: errNamespaceMissingIDs,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ns, errMsg := namespaceForClient(tt.orgID, tt.stackID)
			require.Equal(t, tt.expectNamespace, ns)
			require.Equal(t, tt.expectErr, errMsg)
		})
	}
}
