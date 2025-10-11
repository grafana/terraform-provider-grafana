package appplatform

import (
	"context"
	"fmt"
	"time"

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
// Generated types for K8sO11yConfig
// ====================================================================

const (
	k8sO11yConfigAPIGroup   = "productactivation.ext.grafana.com"
	k8sO11yConfigAPIVersion = "v1alpha1"
	k8sO11yConfigKind       = "K8sO11yConfig"
)

// K8sO11yConfig is the main resource type
type K8sO11yConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              K8sO11yConfigSpec   `json:"spec"`
	Status            K8sO11yConfigStatus `json:"status"`
}

// K8sO11yConfigList is the list type
type K8sO11yConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []K8sO11yConfig `json:"items"`
}

// K8sO11yConfigSpec is the spec structure
type K8sO11yConfigSpec struct {
	Enabled bool `json:"enabled"`
}

// K8sO11yConfigStatus is the status structure
type K8sO11yConfigStatus struct {
	ProcessedTimestamp time.Time              `json:"processedTimestamp"`
	AdditionalFields   map[string]interface{} `json:"additionalFields,omitempty"`
}

// Required methods for sdkresource.Object interface

func (o *K8sO11yConfig) GetSpec() any {
	return o.Spec
}

func (o *K8sO11yConfig) SetSpec(spec any) error {
	cast, ok := spec.(K8sO11yConfigSpec)
	if !ok {
		return fmt.Errorf("cannot set spec type %#v, not of type K8sO11yConfigSpec", spec)
	}
	o.Spec = cast
	return nil
}

func (o *K8sO11yConfig) GetStaticMetadata() sdkresource.StaticMetadata {
	return sdkresource.StaticMetadata{
		Name:      o.ObjectMeta.Name,
		Namespace: o.ObjectMeta.Namespace,
		Group:     k8sO11yConfigAPIGroup,
		Version:   k8sO11yConfigAPIVersion,
		Kind:      k8sO11yConfigKind,
	}
}

func (o *K8sO11yConfig) SetStaticMetadata(metadata sdkresource.StaticMetadata) {
	o.Name = metadata.Name
	o.Namespace = metadata.Namespace
}

func (o *K8sO11yConfig) GetCommonMetadata() sdkresource.CommonMetadata {
	return sdkresource.CommonMetadata{
		UID:               string(o.UID),
		ResourceVersion:   o.ResourceVersion,
		Generation:        o.Generation,
		Labels:            o.Labels,
		CreationTimestamp: o.CreationTimestamp.Time,
		Finalizers:        o.Finalizers,
	}
}

func (o *K8sO11yConfig) SetCommonMetadata(metadata sdkresource.CommonMetadata) {
	o.UID = k8stypes.UID(metadata.UID)
	o.ResourceVersion = metadata.ResourceVersion
	o.Generation = metadata.Generation
	o.Labels = metadata.Labels
	o.CreationTimestamp = metav1.NewTime(metadata.CreationTimestamp)
	o.Finalizers = metadata.Finalizers
}

func (o *K8sO11yConfig) GetSubresources() map[string]any {
	return map[string]any{
		"status": o.Status,
	}
}

func (o *K8sO11yConfig) GetSubresource(name string) (any, bool) {
	if name == "status" {
		return o.Status, true
	}
	return nil, false
}

func (o *K8sO11yConfig) SetSubresource(name string, value any) error {
	if name == "status" {
		if cast, ok := value.(K8sO11yConfigStatus); ok {
			o.Status = cast
			return nil
		}
		return fmt.Errorf("cannot set status type %#v, not of type K8sO11yConfigStatus", value)
	}
	return fmt.Errorf("subresource '%s' does not exist", name)
}

func (o *K8sO11yConfig) Copy() sdkresource.Object {
	return sdkresource.CopyObject(o)
}

func (o *K8sO11yConfig) DeepCopyObject() runtime.Object {
	return o.Copy()
}

// Required methods for sdkresource.ListObject interface

func (o *K8sO11yConfigList) GetItems() []sdkresource.Object {
	items := make([]sdkresource.Object, len(o.Items))
	for i := 0; i < len(o.Items); i++ {
		items[i] = &o.Items[i]
	}
	return items
}

func (o *K8sO11yConfigList) SetItems(items []sdkresource.Object) {
	o.Items = make([]K8sO11yConfig, len(items))
	for i := 0; i < len(items); i++ {
		o.Items[i] = *items[i].(*K8sO11yConfig)
	}
}

func (o *K8sO11yConfigList) Copy() sdkresource.ListObject {
	cpy := &K8sO11yConfigList{
		TypeMeta: o.TypeMeta,
		Items:    make([]K8sO11yConfig, len(o.Items)),
	}
	o.ListMeta.DeepCopyInto(&cpy.ListMeta)
	for i := 0; i < len(o.Items); i++ {
		if item, ok := o.Items[i].Copy().(*K8sO11yConfig); ok {
			cpy.Items[i] = *item
		}
	}
	return cpy
}

func (o *K8sO11yConfigList) DeepCopyObject() runtime.Object {
	return o.Copy()
}

// K8sO11yConfigKind returns the Kind for this resource
func K8sO11yConfigKind() sdkresource.Kind {
	return sdkresource.Kind{
		Schema: sdkresource.NewSimpleSchema(
			k8sO11yConfigAPIGroup,
			k8sO11yConfigAPIVersion,
			&K8sO11yConfig{},
			&K8sO11yConfigList{},
			sdkresource.WithKind(k8sO11yConfigKind),
		),
		Codecs: map[sdkresource.KindEncoding]sdkresource.Codec{
			sdkresource.KindEncodingJSON: &K8sO11yConfigJSONCodec{},
		},
	}
}

// K8sO11yConfigJSONCodec is a JSON codec for K8sO11yConfig
type K8sO11yConfigJSONCodec struct{}

// Read reads JSON-encoded bytes from reader and unmarshals them into into
func (*K8sO11yConfigJSONCodec) Read(reader io.Reader, into sdkresource.Object) error {
	return json.NewDecoder(reader).Decode(into)
}

// Write writes JSON-encoded bytes into writer marshaled from from
func (*K8sO11yConfigJSONCodec) Write(writer io.Writer, from sdkresource.Object) error {
	return json.NewEncoder(writer).Encode(from)
}

// Interface compliance check
var _ sdkresource.Codec = &K8sO11yConfigJSONCodec{}

// ====================================================================
// End of generated types
// ====================================================================

// K8sO11yConfigSpecModel is a model for the Kubernetes observability config spec.
type K8sO11yConfigSpecModel struct {
	Enabled types.Bool `tfsdk:"enabled"`
}

// K8sO11yConfigResource creates a new Grafana Kubernetes Observability Config resource.
// Note: This is a singleton resource - there can only be one per namespace
func K8sO11yConfigResource() NamedResource {
	return NewNamedResource[*K8sO11yConfig, *K8sO11yConfigList](
		common.CategoryCloud,
		ResourceConfig[*K8sO11yConfig]{
			Kind: K8sO11yConfigKind(),
			Schema: ResourceSpecSchema{
				Description: "Manages Grafana Kubernetes Observability configurations.",
				MarkdownDescription: `
Manages Grafana Kubernetes Observability configurations using the Grafana APIs.

This resource allows you to enable or disable Kubernetes observability features.

**Note**: This is a singleton resource. The UID is automatically set to "global" and there can only be one per namespace.
`,
				SpecAttributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						Required:    true,
						Description: "Whether Kubernetes observability is enabled.",
					},
				},
			},
			SpecParser: parseK8sO11yConfigSpec,
			SpecSaver:  saveK8sO11yConfigSpec,
		},
	)
}

func parseK8sO11yConfigSpec(
	ctx context.Context,
	src types.Object,
	dst *K8sO11yConfig,
) diag.Diagnostics {
	var data K8sO11yConfigSpecModel
	if d := src.As(ctx, &data, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	}); d.HasError() {
		return d
	}

	// Force "global" for singleton resource
	dst.ObjectMeta.Name = "global"

	spec := K8sO11yConfigSpec{
		Enabled: data.Enabled.ValueBool(),
	}

	if err := dst.SetSpec(spec); err != nil {
		return diag.Diagnostics{
			diag.NewErrorDiagnostic("failed to set spec", err.Error()),
		}
	}

	return diag.Diagnostics{}
}

func saveK8sO11yConfigSpec(
	ctx context.Context,
	src *K8sO11yConfig,
	dst *ResourceModel,
) diag.Diagnostics {
	spec, diags := types.ObjectValueFrom(ctx, map[string]attr.Type{
		"enabled": types.BoolType,
	}, K8sO11yConfigSpecModel{
		Enabled: types.BoolValue(src.Spec.Enabled),
	})
	if diags.HasError() {
		return diags
	}
	dst.Spec = spec

	return diag.Diagnostics{}
}
