package appplatform

import (
	"context"
	"testing"

	folderv1 "github.com/grafana/grafana/apps/folder/pkg/apis/folder/v1"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	tfresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
)

const (
	testTeamAPIVersion = "iam.grafana.app/v0alpha1"
	testTeamUID        = "team-abc"
)

// teamOwnerRef builds the owner reference the provider sends for a team owner: Grafana requires
// a non-empty uid but never resolves it, so the name is used for both.
func teamOwnerRef(name string) metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion: testTeamAPIVersion,
		Kind:       "Team",
		Name:       name,
		UID:        k8stypes.UID(name),
	}
}

func ownerRefObject(apiVersion, kind, name string) types.Object {
	return types.ObjectValueMust(ownerReferenceAttrTypes(), map[string]attr.Value{
		"api_version": types.StringValue(apiVersion),
		"kind":        types.StringValue(kind),
		"name":        types.StringValue(name),
	})
}

func TestOwnerReferencesToModel(t *testing.T) {
	for _, tc := range []struct {
		name     string
		refs     []metav1.OwnerReference
		expected []OwnerReferenceModel
	}{
		{
			name:     "no owner references yields an empty list",
			refs:     nil,
			expected: []OwnerReferenceModel{},
		},
		{
			name: "a team owner reference is reported",
			refs: []metav1.OwnerReference{teamOwnerRef(testTeamUID)},
			expected: []OwnerReferenceModel{{
				APIVersion: types.StringValue(testTeamAPIVersion),
				Kind:       types.StringValue("Team"),
				Name:       types.StringValue(testTeamUID),
			}},
		},
		{
			// Filtering by group would make the read path asymmetric with the write path, so a
			// configured reference from another group would plan as one block and read back as
			// none, failing the apply. Report everything instead.
			name: "owner references from other groups are reported too",
			refs: []metav1.OwnerReference{{
				APIVersion: "provisioning.grafana.app/v0alpha1",
				Kind:       "Repository",
				Name:       "repo",
				UID:        k8stypes.UID("repo-uid"),
			}},
			expected: []OwnerReferenceModel{{
				APIVersion: types.StringValue("provisioning.grafana.app/v0alpha1"),
				Kind:       types.StringValue("Repository"),
				Name:       types.StringValue("repo"),
			}},
		},
		{
			name: "order and multiplicity are preserved",
			refs: []metav1.OwnerReference{
				teamOwnerRef("team-one"),
				teamOwnerRef("team-two"),
			},
			expected: []OwnerReferenceModel{
				{
					APIVersion: types.StringValue(testTeamAPIVersion),
					Kind:       types.StringValue("Team"),
					Name:       types.StringValue("team-one"),
				},
				{
					APIVersion: types.StringValue(testTeamAPIVersion),
					Kind:       types.StringValue("Team"),
					Name:       types.StringValue("team-two"),
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			list, diags := ownerReferencesToModel(context.Background(), tc.refs)
			require.False(t, diags.HasError(), "%v", diags)

			// owner_references is a nested block, so "none configured" is an empty list, not
			// null. Returning null here would drift against such a configuration forever.
			require.False(t, list.IsNull(), "expected an empty list, not null")

			var got []OwnerReferenceModel
			require.False(t, list.ElementsAs(context.Background(), &got, false).HasError())
			require.Equal(t, tc.expected, got)
		})
	}
}

func TestOwnerReferencesFromModel(t *testing.T) {
	for _, tc := range []struct {
		name     string
		list     types.List
		expected []metav1.OwnerReference
	}{
		{
			name:     "a null list yields no owner references",
			list:     types.ListNull(ownerReferenceObjectType()),
			expected: nil,
		},
		{
			name:     "an unknown list yields no owner references",
			list:     types.ListUnknown(ownerReferenceObjectType()),
			expected: nil,
		},
		{
			// This is how removing ownership reaches the API: Update rebuilds the object from
			// the model, so an empty list clears the stored references.
			name:     "an empty list yields no owner references",
			list:     types.ListValueMust(ownerReferenceObjectType(), []attr.Value{}),
			expected: []metav1.OwnerReference{},
		},
		{
			// Kubernetes rejects an empty uid, and Grafana does not resolve it, so the name is
			// sent for both -- exactly what the Grafana UI does.
			name: "uid is always the name",
			list: types.ListValueMust(ownerReferenceObjectType(), []attr.Value{
				ownerRefObject(testTeamAPIVersion, "Team", testTeamUID),
			}),
			expected: []metav1.OwnerReference{teamOwnerRef(testTeamUID)},
		},
		{
			name: "several owners are all forwarded",
			list: types.ListValueMust(ownerReferenceObjectType(), []attr.Value{
				ownerRefObject(testTeamAPIVersion, "Team", "team-one"),
				ownerRefObject(testTeamAPIVersion, "Team", "team-two"),
			}),
			expected: []metav1.OwnerReference{teamOwnerRef("team-one"), teamOwnerRef("team-two")},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			refs, diags := ownerReferencesFromModel(context.Background(), tc.list)
			require.False(t, diags.HasError(), "%v", diags)
			require.Equal(t, tc.expected, refs)
		})
	}
}

// TestOwnerReferencesRoundTripIsSymmetric is the regression test for the apply failure that a
// group filter on the read path caused: whatever is configured must come back out of a read
// unchanged, or Terraform rejects the apply with "Provider produced inconsistent result after
// apply" because the block count changed.
func TestOwnerReferencesRoundTripIsSymmetric(t *testing.T) {
	ctx := context.Background()

	for _, apiVersion := range []string{
		testTeamAPIVersion,
		"provisioning.grafana.app/v0alpha1",
		"some.future.group/v1",
	} {
		t.Run(apiVersion, func(t *testing.T) {
			configured := types.ListValueMust(ownerReferenceObjectType(), []attr.Value{
				ownerRefObject(apiVersion, "Team", testTeamUID),
			})

			refs, diags := ownerReferencesFromModel(ctx, configured)
			require.False(t, diags.HasError(), "%v", diags)

			readBack, diags := ownerReferencesToModel(ctx, refs)
			require.False(t, diags.HasError(), "%v", diags)

			require.True(t, configured.Equal(readBack),
				"configured %v must round-trip unchanged, got %v", configured, readBack)
		})
	}
}

func TestFolderV1RoundTripsOwnerReferences(t *testing.T) {
	ctx := context.Background()
	cfg := folderResourceConfig(t)

	obj := folderv1.FolderKind().Schema.ZeroValue().(*folderv1.Folder)

	metadata := types.ObjectValueMust(
		folderMetadataTypeMap(t),
		map[string]attr.Value{
			"uuid":        types.StringNull(),
			"uid":         types.StringValue("my-folder"),
			"folder_uid":  types.StringNull(),
			"version":     types.StringNull(),
			"url":         types.StringNull(),
			"annotations": types.MapNull(types.StringType),
			"owner_references": types.ListValueMust(ownerReferenceObjectType(), []attr.Value{
				ownerRefObject(testTeamAPIVersion, "Team", testTeamUID),
			}),
		},
	)

	require.False(t, cfg.MetadataParser(ctx, metadata, obj).HasError())
	require.Equal(t, []metav1.OwnerReference{teamOwnerRef(testTeamUID)}, obj.GetOwnerReferences())

	// And back out again, as a Read would.
	values := map[string]attr.Value{}
	require.False(t, cfg.MetadataSaver(ctx, obj, values).HasError())

	list, ok := values["owner_references"].(types.List)
	require.True(t, ok)
	require.True(t, metadata.Attributes()["owner_references"].(types.List).Equal(list))
}

func TestFolderV1SpecDescriptionStaysNullWhenUnset(t *testing.T) {
	ctx := context.Background()
	cfg := folderResourceConfig(t)

	// A folder with no description must read back as null, not "", otherwise it diffs against
	// a configuration that omits `description`.
	obj := folderv1.FolderKind().Schema.ZeroValue().(*folderv1.Folder)
	require.NoError(t, obj.SetSpec(folderv1.FolderSpec{Title: "Some folder"}))

	var model ResourceModel
	require.False(t, cfg.SpecSaver(ctx, obj, &model).HasError())

	attrs := model.Spec.Attributes()
	require.Equal(t, types.StringValue("Some folder"), attrs["title"])
	require.True(t, attrs["description"].IsNull(), "expected a null description, got %v", attrs["description"])

	description := "With words"
	require.NoError(t, obj.SetSpec(folderv1.FolderSpec{Title: "Some folder", Description: &description}))
	require.False(t, cfg.SpecSaver(ctx, obj, &model).HasError())
	require.Equal(t, types.StringValue(description), model.Spec.Attributes()["description"])
}

func TestFolderV1DeclaresOwnerReferencesBlock(t *testing.T) {
	var res tfresource.SchemaResponse
	FolderV1().Resource.Schema(context.Background(), tfresource.SchemaRequest{}, &res)
	require.False(t, res.Diagnostics.HasError())

	metadata, ok := res.Schema.Blocks["metadata"].(schema.SingleNestedBlock)
	require.True(t, ok)

	ownerRefs, ok := metadata.Blocks["owner_references"].(schema.ListNestedBlock)
	require.True(t, ok, "owner_references must be a block: nested attributes cannot be muxed down to protocol v5")

	// Every attribute is required and none is computed, so Terraform never carries a value
	// forward from prior state -- which is what made a stale uid possible.
	require.Len(t, ownerRefs.NestedObject.Attributes, 3)
	for _, name := range []string{"api_version", "kind", "name"} {
		attribute, ok := ownerRefs.NestedObject.Attributes[name].(schema.StringAttribute)
		require.True(t, ok, name)
		require.True(t, attribute.Required, name)
		require.False(t, attribute.Computed, name)
	}

	// api_version is shape-checked and kind is restricted to Team, so both typo classes fail at
	// plan time rather than reaching the API and quietly doing nothing.
	apiVersion := ownerRefs.NestedObject.Attributes["api_version"].(schema.StringAttribute)
	require.Len(t, apiVersion.Validators, 1)

	kind := ownerRefs.NestedObject.Attributes["kind"].(schema.StringAttribute)
	require.Len(t, kind.Validators, 1)

	_, hasUID := ownerRefs.NestedObject.Attributes["uid"]
	require.False(t, hasUID, "uid is derived from name and must not be configurable")
}

func TestAPIVersionPattern(t *testing.T) {
	for _, tc := range []struct {
		value string
		valid bool
	}{
		{value: "iam.grafana.app/v0alpha1", valid: true},
		{value: "folder.grafana.app/v1", valid: true},
		// The version left off parses as group "" upstream, so it would reach the API and be
		// rejected there; the pattern catches it at plan time instead.
		{value: "iam.grafana.app", valid: false},
		{value: "not/a/valid/group", valid: false},
		{value: "/v0alpha1", valid: false},
		{value: "iam.grafana.app/", valid: false},
		{value: "", valid: false},
	} {
		t.Run(tc.value, func(t *testing.T) {
			require.Equal(t, tc.valid, apiVersionPattern.MatchString(tc.value))
		})
	}
}

func TestSchemaValidationFailsWhenMetadataHooksAreMissing(t *testing.T) {
	r := NewResource[*folderv1.Folder, *folderv1.FolderList](ResourceConfig[*folderv1.Folder]{
		Kind: folderv1.FolderKind(),
		Schema: ResourceSpecSchema{
			Description: "test resource declaring metadata blocks without hooks",
			MetadataBlocks: map[string]schema.Block{
				"owner_references": schema.ListNestedBlock{},
			},
		},
		SpecParser: func(ctx context.Context, src types.Object, dst *folderv1.Folder) diag.Diagnostics { return nil },
		SpecSaver:  func(ctx context.Context, src *folderv1.Folder, dst *ResourceModel) diag.Diagnostics { return nil },
	})

	var res tfresource.SchemaResponse
	r.Schema(context.Background(), tfresource.SchemaRequest{}, &res)

	require.True(t, res.Diagnostics.HasError())
	require.Contains(t, res.Diagnostics.Errors()[0].Detail(), "MetadataParser or MetadataSaver is nil")
}

func TestSchemaValidationFailsWhenMetadataHooksHaveNoBlocks(t *testing.T) {
	r := NewResource[*folderv1.Folder, *folderv1.FolderList](ResourceConfig[*folderv1.Folder]{
		Kind: folderv1.FolderKind(),
		Schema: ResourceSpecSchema{
			Description: "test resource with hooks but no metadata blocks",
		},
		SpecParser: func(ctx context.Context, src types.Object, dst *folderv1.Folder) diag.Diagnostics { return nil },
		SpecSaver:  func(ctx context.Context, src *folderv1.Folder, dst *ResourceModel) diag.Diagnostics { return nil },
		MetadataParser: func(ctx context.Context, metadata types.Object, dst *folderv1.Folder) diag.Diagnostics {
			return nil
		},
		MetadataSaver: func(ctx context.Context, src *folderv1.Folder, dst map[string]attr.Value) diag.Diagnostics {
			return nil
		},
	})

	var res tfresource.SchemaResponse
	r.Schema(context.Background(), tfresource.SchemaRequest{}, &res)

	require.True(t, res.Diagnostics.HasError())
	require.Contains(t, res.Diagnostics.Errors()[0].Detail(), "MetadataBlocks is empty")
}

// folderResourceConfig rebuilds the folder resource's config so the parsers and savers can be
// exercised directly.
func folderResourceConfig(t *testing.T) ResourceConfig[*folderv1.Folder] {
	t.Helper()

	r, ok := FolderV1().Resource.(*Resource[*folderv1.Folder, *folderv1.FolderList])
	require.True(t, ok)

	return r.config
}

func folderMetadataTypeMap(t *testing.T) map[string]attr.Type {
	t.Helper()

	r, ok := FolderV1().Resource.(*Resource[*folderv1.Folder, *folderv1.FolderList])
	require.True(t, ok)

	return r.metadataTypeMap()
}

// TestOwnerReferenceKindValidator exercises the kind allow-list through the validator itself, so
// the behaviour is pinned rather than just its presence in the schema.
func TestOwnerReferenceKindValidator(t *testing.T) {
	var res tfresource.SchemaResponse
	FolderV1().Resource.Schema(context.Background(), tfresource.SchemaRequest{}, &res)
	require.False(t, res.Diagnostics.HasError())

	metadata := res.Schema.Blocks["metadata"].(schema.SingleNestedBlock)
	ownerRefs := metadata.Blocks["owner_references"].(schema.ListNestedBlock)
	kind := ownerRefs.NestedObject.Attributes["kind"].(schema.StringAttribute)

	for _, tc := range []struct {
		value string
		valid bool
	}{
		{value: "Team", valid: true},
		// Casing matters to Kubernetes, and a lowercase kind is accepted by the API while
		// achieving nothing -- exactly the mistake worth catching here.
		{value: "team", valid: false},
		{value: "User", valid: false},
		{value: "", valid: false},
	} {
		t.Run(tc.value, func(t *testing.T) {
			req := validator.StringRequest{
				Path:        path.Root("metadata").AtName("owner_references"),
				ConfigValue: types.StringValue(tc.value),
			}
			var resp validator.StringResponse

			for _, v := range kind.Validators {
				v.ValidateString(context.Background(), req, &resp)
			}

			require.Equal(t, tc.valid, !resp.Diagnostics.HasError(),
				"kind %q: got diagnostics %v", tc.value, resp.Diagnostics)
		})
	}
}
