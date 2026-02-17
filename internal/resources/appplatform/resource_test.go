package appplatform

import (
	"context"
	"reflect"
	"sort"
	"testing"

	"github.com/grafana/authlib/claims"
	sdkresource "github.com/grafana/grafana-app-sdk/resource"
	"github.com/grafana/grafana/apps/playlist/pkg/apis/playlist/v0alpha1"
	apicommon "github.com/grafana/grafana/pkg/apimachinery/apis/common/v0alpha1"
	"github.com/grafana/grafana/pkg/apimachinery/utils"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	tfresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
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

// secureParserTestObject has a Secure field compatible with utils.MetaAccessor.SetSecureValues.
type secureParserTestObject struct {
	AppO11yConfig
	Secure apicommon.InlineSecureValues `json:"secure,omitempty"`
}

var _ sdkresource.Object = (*secureParserTestObject)(nil)

type secureParserStructuredValues struct {
	Token         apicommon.InlineSecureValue `json:"token,omitzero,omitempty"`
	ClientSecret  apicommon.InlineSecureValue `json:"clientSecret,omitzero,omitempty"`
	WebhookSecret apicommon.InlineSecureValue `json:"webhookSecret,omitzero,omitempty"`
}

type secureParserStructuredTestObject struct {
	AppO11yConfig
	Secure secureParserStructuredValues `json:"secure,omitempty"`
}

var _ sdkresource.Object = (*secureParserStructuredTestObject)(nil)

type mockResourceData struct {
	values map[string]interface{}
	calls  []string
}

func (m *mockResourceData) GetAttribute(_ context.Context, p path.Path, target interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	key := p.String()
	m.calls = append(m.calls, key)

	val, ok := m.values[key]
	if !ok {
		diags.AddError("missing mock value", key)
		return diags
	}

	targetValue := reflect.ValueOf(target)
	if targetValue.Kind() != reflect.Pointer || targetValue.IsNil() {
		diags.AddError("invalid mock target", "target must be a non-nil pointer")
		return diags
	}

	src := reflect.ValueOf(val)
	dst := targetValue.Elem()
	if src.Type().AssignableTo(dst.Type()) {
		dst.Set(src)
		return diags
	}
	if src.Type().ConvertibleTo(dst.Type()) {
		dst.Set(src.Convert(dst.Type()))
		return diags
	}

	diags.AddError("incompatible mock target type", key)
	return diags
}

type mockStateData struct {
	values map[string]interface{}
	calls  []string
}

func (m *mockStateData) SetAttribute(_ context.Context, p path.Path, val interface{}) diag.Diagnostics {
	if m.values == nil {
		m.values = make(map[string]interface{})
	}
	key := p.String()
	m.calls = append(m.calls, key)
	m.values[key] = val
	return nil
}

func resourceModelFieldTags(t *testing.T) []string {
	t.Helper()

	rt := reflect.TypeOf(ResourceModel{})
	tags := make([]string, 0, rt.NumField())

	for i := 0; i < rt.NumField(); i++ {
		tag := rt.Field(i).Tag.Get("tfsdk")
		require.NotEmpty(t, tag, "ResourceModel field %s must have tfsdk tag", rt.Field(i).Name)
		tags = append(tags, tag)
	}

	sort.Strings(tags)
	return tags
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

func TestSchemaIncludesSecureBlockWhenConfigured(t *testing.T) {
	r := NewResource[*v0alpha1.Playlist, *v0alpha1.PlaylistList](ResourceConfig[*v0alpha1.Playlist]{
		Kind: v0alpha1.PlaylistKind(),
		Schema: ResourceSpecSchema{
			Description: "test resource with secure schema",
			SpecAttributes: map[string]schema.Attribute{
				"title": schema.StringAttribute{
					Required: true,
				},
			},
			SecureValueAttributes: map[string]SecureValueAttribute{
				"token": {
					Optional: true,
				},
			},
		},
		SpecParser: func(ctx context.Context, src types.Object, dst *v0alpha1.Playlist) diag.Diagnostics {
			return nil
		},
		SpecSaver: func(ctx context.Context, src *v0alpha1.Playlist, dst *ResourceModel) diag.Diagnostics {
			return nil
		},
		SecureParser: func(ctx context.Context, secure types.Object, dst *v0alpha1.Playlist) diag.Diagnostics {
			return nil
		},
	})

	var res tfresource.SchemaResponse
	r.Schema(context.Background(), tfresource.SchemaRequest{}, &res)

	require.False(t, res.Diagnostics.HasError())
	_, hasSecureBlock := res.Schema.Blocks["secure"]
	require.True(t, hasSecureBlock)
	_, hasSecureVersion := res.Schema.Attributes["secure_version"]
	require.True(t, hasSecureVersion)

	secureBlock, ok := res.Schema.Blocks["secure"].(schema.SingleNestedBlock)
	require.True(t, ok)

	secureAttr, ok := secureBlock.Attributes["token"].(schema.StringAttribute)
	require.True(t, ok)
	require.True(t, secureAttr.WriteOnly)
	require.True(t, secureAttr.Optional)
	require.False(t, secureAttr.Required)
}

func TestSchemaExcludesSecureBlockWhenNotConfigured(t *testing.T) {
	r := Playlist().Resource

	var res tfresource.SchemaResponse
	r.Schema(context.Background(), tfresource.SchemaRequest{}, &res)

	require.False(t, res.Diagnostics.HasError())
	_, hasSecureBlock := res.Schema.Blocks["secure"]
	require.False(t, hasSecureBlock)
	_, hasSecureVersion := res.Schema.Attributes["secure_version"]
	require.False(t, hasSecureVersion)
}

func TestAllCurrentAppPlatformResourcesExcludeSecureByDefault(t *testing.T) {
	resources := []NamedResource{
		Dashboard(),
		Playlist(),
		AlertRule(),
		AlertEnrichment(),
		RecordingRule(),
		AppO11yConfigResource(),
		K8sO11yConfigResource(),
	}

	for _, named := range resources {
		t.Run(named.Name, func(t *testing.T) {
			var res tfresource.SchemaResponse
			named.Resource.Schema(context.Background(), tfresource.SchemaRequest{}, &res)

			require.False(t, res.Diagnostics.HasError())
			_, hasSecureBlock := res.Schema.Blocks["secure"]
			require.False(t, hasSecureBlock)
			_, hasSecureVersion := res.Schema.Attributes["secure_version"]
			require.False(t, hasSecureVersion)
		})
	}
}

func TestSchemaValidationFailsForInvalidSecureValueAttributeRequiredOptionalCombo(t *testing.T) {
	r := NewResource[*v0alpha1.Playlist, *v0alpha1.PlaylistList](ResourceConfig[*v0alpha1.Playlist]{
		Kind: v0alpha1.PlaylistKind(),
		Schema: ResourceSpecSchema{
			Description: "test resource with invalid secure value attribute",
			SecureValueAttributes: map[string]SecureValueAttribute{
				"token": {
					Required: true,
					Optional: true,
				},
			},
		},
		SpecParser: func(ctx context.Context, src types.Object, dst *v0alpha1.Playlist) diag.Diagnostics {
			return nil
		},
		SpecSaver: func(ctx context.Context, src *v0alpha1.Playlist, dst *ResourceModel) diag.Diagnostics {
			return nil
		},
		SecureParser: func(ctx context.Context, secure types.Object, dst *v0alpha1.Playlist) diag.Diagnostics {
			return nil
		},
	})

	var res tfresource.SchemaResponse
	r.Schema(context.Background(), tfresource.SchemaRequest{}, &res)

	require.True(t, res.Diagnostics.HasError())
	require.Contains(t, res.Diagnostics[0].Detail(), `cannot be both required and optional`)
}

func TestSchemaValidationFailsWhenSecureParserIsMissing(t *testing.T) {
	r := NewResource[*v0alpha1.Playlist, *v0alpha1.PlaylistList](ResourceConfig[*v0alpha1.Playlist]{
		Kind: v0alpha1.PlaylistKind(),
		Schema: ResourceSpecSchema{
			Description: "test resource with missing secure parser",
			SecureValueAttributes: map[string]SecureValueAttribute{
				"token": {
					Optional: true,
				},
			},
		},
		SpecParser: func(ctx context.Context, src types.Object, dst *v0alpha1.Playlist) diag.Diagnostics {
			return nil
		},
		SpecSaver: func(ctx context.Context, src *v0alpha1.Playlist, dst *ResourceModel) diag.Diagnostics {
			return nil
		},
	})

	var res tfresource.SchemaResponse
	r.Schema(context.Background(), tfresource.SchemaRequest{}, &res)

	require.True(t, res.Diagnostics.HasError())
	require.Contains(t, res.Diagnostics[0].Detail(), "SecureValueAttributes is configured, but SecureParser is nil")
}

func TestSchemaValidationFailsWhenSecureParserWithoutSecureValueAttributes(t *testing.T) {
	r := NewResource[*v0alpha1.Playlist, *v0alpha1.PlaylistList](ResourceConfig[*v0alpha1.Playlist]{
		Kind: v0alpha1.PlaylistKind(),
		Schema: ResourceSpecSchema{
			Description: "test resource with unexpected secure parser",
			SpecAttributes: map[string]schema.Attribute{
				"title": schema.StringAttribute{Required: true},
			},
		},
		SpecParser: func(ctx context.Context, src types.Object, dst *v0alpha1.Playlist) diag.Diagnostics {
			return nil
		},
		SpecSaver: func(ctx context.Context, src *v0alpha1.Playlist, dst *ResourceModel) diag.Diagnostics {
			return nil
		},
		SecureParser: func(ctx context.Context, secure types.Object, dst *v0alpha1.Playlist) diag.Diagnostics {
			return nil
		},
	})

	var res tfresource.SchemaResponse
	r.Schema(context.Background(), tfresource.SchemaRequest{}, &res)

	require.True(t, res.Diagnostics.HasError())
	require.Contains(t, res.Diagnostics[0].Detail(), "SecureParser is configured, but SecureValueAttributes is empty")
}

func TestDefaultSecureParserSetsInlineSecureValues(t *testing.T) {
	ctx := context.Background()

	secureObj := types.ObjectValueMust(
		map[string]attr.Type{
			"token":         types.StringType,
			"client_secret": types.StringType,
		},
		map[string]attr.Value{
			"token":         types.StringValue("token-123"),
			"client_secret": types.StringValue("secret-456"),
		},
	)

	dst := &secureParserTestObject{}
	parser := SecureParser[*secureParserTestObject](DefaultSecureParser[*secureParserTestObject])

	diags := parser(ctx, secureObj, dst)
	require.False(t, diags.HasError())

	require.Equal(t, apicommon.NewSecretValue("token-123"), dst.Secure["token"].Create)
	require.Equal(t, apicommon.NewSecretValue("secret-456"), dst.Secure["clientSecret"].Create)
	_, hasSnakeCaseKey := dst.Secure["client_secret"]
	require.False(t, hasSnakeCaseKey)
}

func TestDefaultSecureParserSetsStructSecureValues(t *testing.T) {
	ctx := context.Background()

	secureObj := types.ObjectValueMust(
		map[string]attr.Type{
			"token":          types.StringType,
			"client_secret":  types.StringType,
			"webhook_secret": types.StringType,
		},
		map[string]attr.Value{
			"token":          types.StringValue("token-123"),
			"client_secret":  types.StringValue("secret-456"),
			"webhook_secret": types.StringValue("hook-789"),
		},
	)

	dst := &secureParserStructuredTestObject{}
	parser := SecureParser[*secureParserStructuredTestObject](DefaultSecureParser[*secureParserStructuredTestObject])
	diags := parser(ctx, secureObj, dst)

	require.False(t, diags.HasError())
	require.Equal(t, apicommon.NewSecretValue("token-123"), dst.Secure.Token.Create)
	require.Equal(t, apicommon.NewSecretValue("secret-456"), dst.Secure.ClientSecret.Create)
	require.Equal(t, apicommon.NewSecretValue("hook-789"), dst.Secure.WebhookSecret.Create)
}

func TestDefaultSecureParserRejectsNonStringValues(t *testing.T) {
	ctx := context.Background()

	secureObj := types.ObjectValueMust(
		map[string]attr.Type{
			"token":   types.StringType,
			"use_tls": types.BoolType,
		},
		map[string]attr.Value{
			"token":   types.StringValue("token-123"),
			"use_tls": types.BoolValue(true),
		},
	)

	dst := &secureParserTestObject{}
	parser := SecureParser[*secureParserTestObject](DefaultSecureParser[*secureParserTestObject])
	diags := parser(ctx, secureObj, dst)

	require.True(t, diags.HasError())
	require.Contains(t, diags[0].Detail(), "only string secure value attributes are supported")
}

func TestDefaultSecureParserHandlesNullObject(t *testing.T) {
	ctx := context.Background()
	parser := SecureParser[*secureParserStructuredTestObject](DefaultSecureParser[*secureParserStructuredTestObject])

	dst := &secureParserStructuredTestObject{}
	diags := parser(ctx, types.ObjectNull(map[string]attr.Type{
		"token": types.StringType,
	}), dst)

	require.False(t, diags.HasError())
	require.Equal(t, apicommon.InlineSecureValue{}, dst.Secure.Token)
}

func TestDefaultSecureParserHandlesUnknownObject(t *testing.T) {
	ctx := context.Background()
	parser := SecureParser[*secureParserStructuredTestObject](DefaultSecureParser[*secureParserStructuredTestObject])

	dst := &secureParserStructuredTestObject{}
	diags := parser(ctx, types.ObjectUnknown(map[string]attr.Type{
		"token": types.StringType,
	}), dst)

	require.False(t, diags.HasError())
	require.Equal(t, apicommon.InlineSecureValue{}, dst.Secure.Token)
}

func TestDefaultSecureParserHandlesEmptySecureBlockObject(t *testing.T) {
	ctx := context.Background()
	parser := SecureParser[*secureParserStructuredTestObject](DefaultSecureParser[*secureParserStructuredTestObject])

	secureObj := types.ObjectValueMust(
		map[string]attr.Type{
			"token":         types.StringType,
			"client_secret": types.StringType,
		},
		map[string]attr.Value{
			"token":         types.StringNull(),
			"client_secret": types.StringNull(),
		},
	)

	dst := &secureParserStructuredTestObject{}
	diags := parser(ctx, secureObj, dst)

	require.False(t, diags.HasError())
	require.Equal(t, apicommon.InlineSecureValue{}, dst.Secure.Token)
	require.Equal(t, apicommon.InlineSecureValue{}, dst.Secure.ClientSecret)
}

func TestDefaultSecureParserFallbackMetaAccessorPath(t *testing.T) {
	ctx := context.Background()
	parser := SecureParser[*AppO11yConfig](DefaultSecureParser[*AppO11yConfig])

	secureObj := types.ObjectValueMust(
		map[string]attr.Type{
			"token": types.StringType,
		},
		map[string]attr.Value{
			"token": types.StringValue("token-123"),
		},
	)

	dst := &AppO11yConfig{}
	diags := parser(ctx, secureObj, dst)

	require.True(t, diags.HasError())
	require.Contains(t, diags[0].Detail(), "unable to set secure values")
}

func TestDefaultSecureParserRejectsUnknownStructSecureKey(t *testing.T) {
	ctx := context.Background()
	parser := SecureParser[*secureParserStructuredTestObject](DefaultSecureParser[*secureParserStructuredTestObject])

	secureObj := types.ObjectValueMust(
		map[string]attr.Type{
			"unknown_secret": types.StringType,
		},
		map[string]attr.Value{
			"unknown_secret": types.StringValue("value"),
		},
	)

	dst := &secureParserStructuredTestObject{}
	diags := parser(ctx, secureObj, dst)

	require.True(t, diags.HasError())
	require.Contains(t, diags[0].Detail(), "invalid secure value key")
}

func TestSetMapSecureValuesRejectsIncompatibleValueType(t *testing.T) {
	mapValue := reflect.New(reflect.TypeOf(map[string]string(nil))).Elem()
	err := setMapSecureValues(
		mapValue,
		apicommon.InlineSecureValues{
			"client_secret": {
				Create: apicommon.NewSecretValue("secret-456"),
			},
		},
	)

	require.Error(t, err)
	require.Contains(t, err.Error(), "secure map value type")
}

func TestSetMapSecureValuesRejectsIncompatibleKeyType(t *testing.T) {
	mapValue := reflect.New(reflect.TypeOf(map[int]apicommon.InlineSecureValue(nil))).Elem()
	err := setMapSecureValues(
		mapValue,
		apicommon.InlineSecureValues{
			"client_secret": {
				Create: apicommon.NewSecretValue("secret-456"),
			},
		},
	)

	require.Error(t, err)
	require.Contains(t, err.Error(), "secure map key type")
}

func TestParseSecureValuesReturnsErrorWhenParserMissing(t *testing.T) {
	r := &Resource[*v0alpha1.Playlist, *v0alpha1.PlaylistList]{
		config: ResourceConfig[*v0alpha1.Playlist]{
			Schema: ResourceSpecSchema{
				SecureValueAttributes: map[string]SecureValueAttribute{
					"token": {
						Optional: true,
					},
				},
			},
		},
	}

	diags := r.parseSecureValues(context.Background(), tfsdk.Config{}, &v0alpha1.Playlist{})
	require.True(t, diags.HasError())
	require.Contains(t, diags[0].Detail(), "SecureValueAttributes is configured, but SecureParser is nil")
}

func TestGetResourceModelFromDataReadsResourceModelFields(t *testing.T) {
	ctx := context.Background()

	metadata := types.ObjectValueMust(
		map[string]attr.Type{
			"uid": types.StringType,
		},
		map[string]attr.Value{
			"uid": types.StringValue("uid-123"),
		},
	)
	spec := types.ObjectValueMust(
		map[string]attr.Type{
			"title": types.StringType,
		},
		map[string]attr.Value{
			"title": types.StringValue("playlist"),
		},
	)
	options := types.ObjectValueMust(
		map[string]attr.Type{
			"overwrite": types.BoolType,
		},
		map[string]attr.Value{
			"overwrite": types.BoolValue(true),
		},
	)

	src := &mockResourceData{
		values: map[string]interface{}{
			"id":       types.StringValue("id-123"),
			"metadata": metadata,
			"spec":     spec,
			"options":  options,
		},
	}

	data, diags := getResourceModelFromData(ctx, src)
	require.False(t, diags.HasError())

	expectedCalls := resourceModelFieldTags(t)
	actualCalls := append([]string(nil), src.calls...)
	sort.Strings(actualCalls)
	require.Equal(t, expectedCalls, actualCalls)

	require.Equal(t, "id-123", data.ID.ValueString())
	require.True(t, data.Metadata.Equal(metadata))
	require.True(t, data.Spec.Equal(spec))
	require.True(t, data.Options.Equal(options))
}

func TestSetSecureStateWritesResourceModelAndSecureFields(t *testing.T) {
	ctx := context.Background()

	metadata := types.ObjectValueMust(
		map[string]attr.Type{
			"uid": types.StringType,
		},
		map[string]attr.Value{
			"uid": types.StringValue("uid-123"),
		},
	)
	spec := types.ObjectValueMust(
		map[string]attr.Type{
			"title": types.StringType,
		},
		map[string]attr.Value{
			"title": types.StringValue("playlist"),
		},
	)
	options := types.ObjectValueMust(
		map[string]attr.Type{
			"overwrite": types.BoolType,
		},
		map[string]attr.Value{
			"overwrite": types.BoolValue(true),
		},
	)

	r := &Resource[*v0alpha1.Playlist, *v0alpha1.PlaylistList]{
		config: ResourceConfig[*v0alpha1.Playlist]{
			Schema: ResourceSpecSchema{
				SecureValueAttributes: map[string]SecureValueAttribute{
					"token": {
						Optional: true,
					},
				},
			},
		},
	}

	data := ResourceModel{
		ID:       types.StringValue("id-123"),
		Metadata: metadata,
		Spec:     spec,
		Options:  options,
	}

	state := &mockStateData{}
	secureVersion := types.Int64Value(7)
	diags := r.setSecureStateWithData(ctx, state, data, secureVersion)
	require.False(t, diags.HasError())

	expectedCalls := append(resourceModelFieldTags(t), "secure", "secure_version")
	actualCalls := append([]string(nil), state.calls...)
	sort.Strings(expectedCalls)
	sort.Strings(actualCalls)
	require.Equal(t, expectedCalls, actualCalls)

	idValue, ok := state.values["id"].(types.String)
	require.True(t, ok)
	require.True(t, idValue.Equal(data.ID))

	metadataValue, ok := state.values["metadata"].(types.Object)
	require.True(t, ok)
	require.True(t, metadataValue.Equal(data.Metadata))

	specValue, ok := state.values["spec"].(types.Object)
	require.True(t, ok)
	require.True(t, specValue.Equal(data.Spec))

	optionsValue, ok := state.values["options"].(types.Object)
	require.True(t, ok)
	require.True(t, optionsValue.Equal(data.Options))

	secureValue, ok := state.values["secure"].(types.Object)
	require.True(t, ok)
	require.True(t, secureValue.IsNull())

	secureVersionValue, ok := state.values["secure_version"].(types.Int64)
	require.True(t, ok)
	require.True(t, secureVersionValue.Equal(secureVersion))
}

func TestToSnakeCaseHandlesAcronyms(t *testing.T) {
	require.Equal(t, "url_token", toSnakeCase("URLToken"))
	require.Equal(t, "https_proxy", toSnakeCase("HTTPSProxy"))
	require.Equal(t, "ssh_key", toSnakeCase("SSHKey"))
	require.Equal(t, "client_secret", toSnakeCase("clientSecret"))
}

func TestToLowerCamelCase(t *testing.T) {
	testCases := []struct {
		in  string
		out string
	}{
		{in: "token", out: "token"},
		{in: "client_secret", out: "clientSecret"},
		{in: "_token", out: "token"},
		{in: "a__b", out: "aB"},
		{in: "___", out: "___"},
		{in: "url_token", out: "urlToken"},
	}

	for _, tc := range testCases {
		t.Run(tc.in, func(t *testing.T) {
			require.Equal(t, tc.out, toLowerCamelCase(tc.in))
		})
	}
}
