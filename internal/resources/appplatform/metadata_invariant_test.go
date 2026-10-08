package appplatform_test

import (
	"context"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/resources/appplatform"
	"github.com/grafana/terraform-provider-grafana/v4/pkg/provider"
	tfresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/stretchr/testify/require"
)

// baseMetadataAttributeNames mirrors the shared metadata attributes every App Platform resource
// gets. Kept as a literal so that adding one to the shared block has to be a deliberate change
// here too.
var baseMetadataAttributeNames = []string{"uuid", "uid", "folder_uid", "version", "url", "annotations"}

// TestOnlyFolderResourceExtendsTheMetadataBlock guards the scoping decision behind team folders:
// Kubernetes ownerReferences are only meaningful on folders, so no other App Platform resource
// should acquire a metadata block. The resource list is taken from the provider's own
// registration so newly added resources are covered automatically.
//
// This lives in the external test package because it imports pkg/provider, which imports the
// appplatform package under test.
func TestOnlyFolderResourceExtendsTheMetadataBlock(t *testing.T) {
	folderName := appplatform.FolderV1().Name

	var checked int
	for _, named := range provider.AppPlatformResources() {
		if named.Name == folderName {
			continue
		}

		t.Run(named.Name, func(t *testing.T) {
			var res tfresource.SchemaResponse
			named.Resource.Schema(context.Background(), tfresource.SchemaRequest{}, &res)
			require.False(t, res.Diagnostics.HasError())

			metadata, ok := res.Schema.Blocks["metadata"].(schema.SingleNestedBlock)
			if !ok {
				// A few resources (e.g. the generic resource) model metadata differently and
				// have no shared metadata block at all; there is nothing to assert for them.
				return
			}

			require.Empty(t, metadata.Blocks, "expected no nested metadata blocks")

			names := make([]string, 0, len(metadata.Attributes))
			for name := range metadata.Attributes {
				names = append(names, name)
			}
			require.ElementsMatch(t, baseMetadataAttributeNames, names)
		})

		checked++
	}

	require.NotZero(t, checked, "expected to check at least one non-folder resource")
}

// TestFolderResourceIsRegistered pairs with the test above: the invariant is only meaningful if
// the folder resource is actually registered.
func TestFolderResourceIsRegistered(t *testing.T) {
	folderName := appplatform.FolderV1().Name

	for _, named := range provider.AppPlatformResources() {
		if named.Name == folderName {
			return
		}
	}

	t.Fatalf("%s is not registered in provider.AppPlatformResources()", folderName)
}
