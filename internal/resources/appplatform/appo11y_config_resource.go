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
// Generated types for AppO11yConfig
// ====================================================================

const (
	appO11yConfigAPIGroup   = "productactivation.ext.grafana.com"
	appO11yConfigAPIVersion = "v1alpha1"
	appO11yConfigKind       = "AppO11yConfig"
)

// AppO11yConfig is the main resource type
type AppO11yConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              AppO11yConfigSpec   `json:"spec"`
	Status            AppO11yConfigStatus `json:"status"`
}

// AppO11yConfigList is the list type
type AppO11yConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []AppO11yConfig `json:"items"`
}

// AppO11yConfigSpec is the spec structure
type AppO11yConfigSpec struct {
	Enabled bool `json:"enabled"`
}

// AppO11yConfigStatus is the status structure
type AppO11yConfigStatus struct {
	ProcessedTimestamp time.Time              `json:"processedTimestamp"`
	AdditionalFields   map[string]interface{} `json:"additionalFields,omitempty"`
}

// Required methods for sdkresource.Object interface

func (o *AppO11yConfig) GetSpec() any {
	return o.Spec
}

func (o *AppO11yConfig) SetSpec(spec any) error {
	cast, ok := spec.(AppO11yConfigSpec)
	if !ok {
		return fmt.Errorf("cannot set spec type %#v, not of type AppO11yConfigSpec", spec)
	}
	o.Spec = cast
	return nil
}

func (o *AppO11yConfig) GetStaticMetadata() sdkresource.StaticMetadata {
	return sdkresource.StaticMetadata{
		Name:      o.ObjectMeta.Name,
		Namespace: o.ObjectMeta.Namespace,
		Group:     appO11yConfigAPIGroup,
		Version:   appO11yConfigAPIVersion,
		Kind:      appO11yConfigKind,
	}
}

func (o *AppO11yConfig) SetStaticMetadata(metadata sdkresource.StaticMetadata) {
	o.Name = metadata.Name
	o.Namespace = metadata.Namespace
}

func (o *AppO11yConfig) GetCommonMetadata() sdkresource.CommonMetadata {
	return sdkresource.CommonMetadata{
		UID:               string(o.UID),
		ResourceVersion:   o.ResourceVersion,
		Generation:        o.Generation,
		Labels:            o.Labels,
		CreationTimestamp: o.CreationTimestamp.Time,
		Finalizers:        o.Finalizers,
	}
}

func (o *AppO11yConfig) SetCommonMetadata(metadata sdkresource.CommonMetadata) {
	o.UID = k8stypes.UID(metadata.UID)
	o.ResourceVersion = metadata.ResourceVersion
	o.Generation = metadata.Generation
	o.Labels = metadata.Labels
	o.CreationTimestamp = metav1.NewTime(metadata.CreationTimestamp)
	o.Finalizers = metadata.Finalizers
}

func (o *AppO11yConfig) GetSubresources() map[string]any {
	return map[string]any{
		"status": o.Status,
	}
}

func (o *AppO11yConfig) GetSubresource(name string) (any, bool) {
	if name == "status" {
		return o.Status, true
	}
	return nil, false
}

func (o *AppO11yConfig) SetSubresource(name string, value any) error {
	if name == "status" {
		if cast, ok := value.(AppO11yConfigStatus); ok {
			o.Status = cast
			return nil
		}
		return fmt.Errorf("cannot set status type %#v, not of type AppO11yConfigStatus", value)
	}
	return fmt.Errorf("subresource '%s' does not exist", name)
}

func (o *AppO11yConfig) Copy() sdkresource.Object {
	return sdkresource.CopyObject(o)
}

func (o *AppO11yConfig) DeepCopyObject() runtime.Object {
	return o.Copy()
}

// Required methods for sdkresource.ListObject interface

func (o *AppO11yConfigList) GetItems() []sdkresource.Object {
	items := make([]sdkresource.Object, len(o.Items))
	for i := 0; i < len(o.Items); i++ {
		items[i] = &o.Items[i]
	}
	return items
}

func (o *AppO11yConfigList) SetItems(items []sdkresource.Object) {
	o.Items = make([]AppO11yConfig, len(items))
	for i := 0; i < len(items); i++ {
		o.Items[i] = *items[i].(*AppO11yConfig)
	}
}

func (o *AppO11yConfigList) Copy() sdkresource.ListObject {
	cpy := &AppO11yConfigList{
		TypeMeta: o.TypeMeta,
		Items:    make([]AppO11yConfig, len(o.Items)),
	}
	o.ListMeta.DeepCopyInto(&cpy.ListMeta)
	for i := 0; i < len(o.Items); i++ {
		if item, ok := o.Items[i].Copy().(*AppO11yConfig); ok {
			cpy.Items[i] = *item
		}
	}
	return cpy
}

func (o *AppO11yConfigList) DeepCopyObject() runtime.Object {
	return o.Copy()
}

// AppO11yConfigKind returns the Kind for this resource
func AppO11yConfigKind() sdkresource.Kind {
	return sdkresource.Kind{
		Schema: sdkresource.NewSimpleSchema(
			appO11yConfigAPIGroup,
			appO11yConfigAPIVersion,
			&AppO11yConfig{},
			&AppO11yConfigList{},
			sdkresource.WithKind(appO11yConfigKind),
		),
		Codecs: map[sdkresource.KindEncoding]sdkresource.Codec{
			sdkresource.KindEncodingJSON: &AppO11yConfigJSONCodec{},
		},
	}
}

// AppO11yConfigJSONCodec is a JSON codec for AppO11yConfig
type AppO11yConfigJSONCodec struct{}

// Read reads JSON-encoded bytes from reader and unmarshals them into into
func (*AppO11yConfigJSONCodec) Read(reader io.Reader, into sdkresource.Object) error {
	return json.NewDecoder(reader).Decode(into)
}

// Write writes JSON-encoded bytes into writer marshaled from from
func (*AppO11yConfigJSONCodec) Write(writer io.Writer, from sdkresource.Object) error {
	return json.NewEncoder(writer).Encode(from)
}

// Interface compliance check
var _ sdkresource.Codec = &AppO11yConfigJSONCodec{}

// ====================================================================
// End of generated types
// ====================================================================

// AppO11yConfigSpecModel is a model for the app observability config spec.
type AppO11yConfigSpecModel struct {
	Enabled types.Bool `tfsdk:"enabled"`
}

// AppO11yConfigResource creates a new Grafana App Observability Config resource.
// Note: This is a singleton resource - there can only be one per namespace
func AppO11yConfigResource() NamedResource {
	return NewNamedResource[*AppO11yConfig, *AppO11yConfigList](
		common.CategoryCloud,
		ResourceConfig[*AppO11yConfig]{
			Kind: AppO11yConfigKind(),
			Schema: ResourceSpecSchema{
				Description: "Manages Grafana Application Observability configurations.",
				MarkdownDescription: `
Manages Grafana Application Observability configurations using the Grafana APIs.

This resource allows you to enable or disable application observability features.

**Note**: This is a singleton resource. The UID is automatically set to "global" and there can only be one per namespace.
`,
				SpecAttributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						Required:    true,
						Description: "Whether application observability is enabled.",
					},
				},
			},
			SpecParser: parseAppO11yConfigSpec,
			SpecSaver:  saveAppO11yConfigSpec,
		},
	)
}

func parseAppO11yConfigSpec(
	ctx context.Context,
	src types.Object,
	dst *AppO11yConfig,
) diag.Diagnostics {
	var data AppO11yConfigSpecModel
	if d := src.As(ctx, &data, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	}); d.HasError() {
		return d
	}

	// Force "global" for singleton resource
	dst.ObjectMeta.Name = "global"

	spec := AppO11yConfigSpec{
		Enabled: data.Enabled.ValueBool(),
	}

	if err := dst.SetSpec(spec); err != nil {
		return diag.Diagnostics{
			diag.NewErrorDiagnostic("failed to set spec", err.Error()),
		}
	}

	return diag.Diagnostics{}
}

func saveAppO11yConfigSpec(
	ctx context.Context,
	src *AppO11yConfig,
	dst *ResourceModel,
) diag.Diagnostics {
	spec, diags := types.ObjectValueFrom(ctx, map[string]attr.Type{
		"enabled": types.BoolType,
	}, AppO11yConfigSpecModel{
		Enabled: types.BoolValue(src.Spec.Enabled),
	})
	if diags.HasError() {
		return diags
	}
	dst.Spec = spec

	return diag.Diagnostics{}
}
