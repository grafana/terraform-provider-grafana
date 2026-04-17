package appplatform

import (
	"context"
	"fmt"

	"encoding/json"
	"io"

	sdkresource "github.com/grafana/grafana-app-sdk/resource"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stypes "k8s.io/apimachinery/pkg/types"
)

// ====================================================================
// Generated types for DbO11yConfig
// ====================================================================

const (
	dbO11yConfigAPIGroup   = "productactivation.ext.grafana.com"
	dbO11yConfigAPIVersion = "v1alpha1"
	dbO11yConfigKind       = "DbO11yConfig"
)

// DbO11yConfig is the main resource type
type DbO11yConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              DbO11yConfigSpec   `json:"spec"`
	Status            DbO11yConfigStatus `json:"status"`
}

// DbO11yConfigList is the list type
type DbO11yConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []DbO11yConfig `json:"items"`
}

// DbO11yConfigSpec is the spec structure
type DbO11yConfigSpec struct {
	Enabled bool `json:"enabled"`
}

// DbO11yConfigStatus is the status structure
type DbO11yConfigStatus struct {
	AdditionalFields map[string]interface{} `json:"additionalFields,omitempty"`
}

// Required methods for sdkresource.Object interface

func (o *DbO11yConfig) GetSpec() any {
	return o.Spec
}

func (o *DbO11yConfig) SetSpec(spec any) error {
	cast, ok := spec.(DbO11yConfigSpec)
	if !ok {
		return fmt.Errorf("cannot set spec type %#v, not of type DbO11yConfigSpec", spec)
	}
	o.Spec = cast
	return nil
}

func (o *DbO11yConfig) GetStaticMetadata() sdkresource.StaticMetadata {
	return sdkresource.StaticMetadata{
		Name:      o.ObjectMeta.Name,
		Namespace: o.ObjectMeta.Namespace,
		Group:     dbO11yConfigAPIGroup,
		Version:   dbO11yConfigAPIVersion,
		Kind:      dbO11yConfigKind,
	}
}

func (o *DbO11yConfig) SetStaticMetadata(metadata sdkresource.StaticMetadata) {
	o.Name = metadata.Name
	o.Namespace = metadata.Namespace
}

func (o *DbO11yConfig) GetCommonMetadata() sdkresource.CommonMetadata {
	return sdkresource.CommonMetadata{
		UID:               string(o.UID),
		ResourceVersion:   o.ResourceVersion,
		Generation:        o.Generation,
		Labels:            o.Labels,
		CreationTimestamp: o.CreationTimestamp.Time,
		Finalizers:        o.Finalizers,
	}
}

func (o *DbO11yConfig) SetCommonMetadata(metadata sdkresource.CommonMetadata) {
	o.UID = k8stypes.UID(metadata.UID)
	o.ResourceVersion = metadata.ResourceVersion
	o.Generation = metadata.Generation
	o.Labels = metadata.Labels
	o.CreationTimestamp = metav1.NewTime(metadata.CreationTimestamp)
	o.Finalizers = metadata.Finalizers
}

func (o *DbO11yConfig) GetSubresources() map[string]any {
	return map[string]any{
		"status": o.Status,
	}
}

func (o *DbO11yConfig) GetSubresource(name string) (any, bool) {
	if name == "status" {
		return o.Status, true
	}
	return nil, false
}

func (o *DbO11yConfig) SetSubresource(name string, value any) error {
	if name == "status" {
		if cast, ok := value.(DbO11yConfigStatus); ok {
			o.Status = cast
			return nil
		}
		return fmt.Errorf("cannot set status type %#v, not of type DbO11yConfigStatus", value)
	}
	return fmt.Errorf("subresource '%s' does not exist", name)
}

func (o *DbO11yConfig) Copy() sdkresource.Object {
	return sdkresource.CopyObject(o)
}

func (o *DbO11yConfig) DeepCopyObject() runtime.Object {
	return o.Copy()
}

// Required methods for sdkresource.ListObject interface

func (o *DbO11yConfigList) GetItems() []sdkresource.Object {
	items := make([]sdkresource.Object, len(o.Items))
	for i := 0; i < len(o.Items); i++ {
		items[i] = &o.Items[i]
	}
	return items
}

func (o *DbO11yConfigList) SetItems(items []sdkresource.Object) {
	o.Items = make([]DbO11yConfig, len(items))
	for i := 0; i < len(items); i++ {
		o.Items[i] = *items[i].(*DbO11yConfig)
	}
}

func (o *DbO11yConfigList) Copy() sdkresource.ListObject {
	cpy := &DbO11yConfigList{
		TypeMeta: o.TypeMeta,
		Items:    make([]DbO11yConfig, len(o.Items)),
	}
	o.ListMeta.DeepCopyInto(&cpy.ListMeta)
	for i := 0; i < len(o.Items); i++ {
		if item, ok := o.Items[i].Copy().(*DbO11yConfig); ok {
			cpy.Items[i] = *item
		}
	}
	return cpy
}

func (o *DbO11yConfigList) DeepCopyObject() runtime.Object {
	return o.Copy()
}

// DbO11yConfigKind returns the Kind for this resource
func DbO11yConfigKind() sdkresource.Kind {
	return sdkresource.Kind{
		Schema: sdkresource.NewSimpleSchema(
			dbO11yConfigAPIGroup,
			dbO11yConfigAPIVersion,
			&DbO11yConfig{},
			&DbO11yConfigList{},
			sdkresource.WithKind(dbO11yConfigKind),
		),
		Codecs: map[sdkresource.KindEncoding]sdkresource.Codec{
			sdkresource.KindEncodingJSON: &DbO11yConfigJSONCodec{},
		},
	}
}

// DbO11yConfigJSONCodec is a JSON codec for DbO11yConfig
type DbO11yConfigJSONCodec struct{}

// Read reads JSON-encoded bytes from reader and unmarshals them into into
func (*DbO11yConfigJSONCodec) Read(reader io.Reader, into sdkresource.Object) error {
	return json.NewDecoder(reader).Decode(into)
}

// Write writes JSON-encoded bytes into writer marshaled from from
func (*DbO11yConfigJSONCodec) Write(writer io.Writer, from sdkresource.Object) error {
	return json.NewEncoder(writer).Encode(from)
}

// Interface compliance check
var _ sdkresource.Codec = &DbO11yConfigJSONCodec{}

// ====================================================================
// End of generated types
// ====================================================================

// DbO11yConfigSpecModel is a model for the database observability config spec.
type DbO11yConfigSpecModel struct {
	Enabled types.Bool `tfsdk:"enabled"`
}

// DbO11yConfigResource creates a new Grafana Database Observability Config resource.
// Note: This is a singleton resource - there can only be one per namespace
func DbO11yConfigResource() NamedResource {
	return NewNamedResource[*DbO11yConfig, *DbO11yConfigList](
		common.CategoryCloud,
		ResourceConfig[*DbO11yConfig]{
			Kind: DbO11yConfigKind(),
			Schema: ResourceSpecSchema{
				Description: "Manages Grafana Database Observability configurations.",
				MarkdownDescription: `
Manages Grafana Database Observability configurations using the Grafana APIs.

This resource allows you to enable or disable database observability features.

**Note**: This is a singleton resource. The UID is automatically set to "global" and there can only be one per namespace.
`,
				SpecAttributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						Required:    true,
						Description: "Whether database observability is enabled.",
					},
				},
			},
			SpecParser: parseDbO11yConfigSpec,
			SpecSaver:  saveDbO11yConfigSpec,
		},
	)
}

func parseDbO11yConfigSpec(
	ctx context.Context,
	src types.Object,
	dst *DbO11yConfig,
) diag.Diagnostics {
	var data DbO11yConfigSpecModel
	if d := src.As(ctx, &data, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	}); d.HasError() {
		return d
	}

	// Force "global" for singleton resource
	dst.ObjectMeta.Name = "global"

	spec := DbO11yConfigSpec{
		Enabled: data.Enabled.ValueBool(),
	}

	if err := dst.SetSpec(spec); err != nil {
		return diag.Diagnostics{
			diag.NewErrorDiagnostic("failed to set spec", err.Error()),
		}
	}

	return diag.Diagnostics{}
}

func saveDbO11yConfigSpec(
	ctx context.Context,
	src *DbO11yConfig,
	dst *ResourceModel,
) diag.Diagnostics {
	spec, diags := types.ObjectValueFrom(ctx, map[string]attr.Type{
		"enabled": types.BoolType,
	}, DbO11yConfigSpecModel{
		Enabled: types.BoolValue(src.Spec.Enabled),
	})
	if diags.HasError() {
		return diags
	}
	dst.Spec = spec

	return diag.Diagnostics{}
}
