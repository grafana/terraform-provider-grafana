package appplatform

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"

	sdkresource "github.com/grafana/grafana-app-sdk/resource"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	tfresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stypes "k8s.io/apimachinery/pkg/types"
)

// ====================================================================
// Types for the Service Model Component kind
// (servicemodel.ext.grafana.com/v1alpha1). Only the fields managed by
// this resource are declared: the API server prunes unknown fields,
// and fields absent from these structs are neither read on import nor
// sent on create/update.
// ====================================================================

const (
	serviceModelComponentAPIGroup   = "servicemodel.ext.grafana.com"
	serviceModelComponentAPIVersion = "v1alpha1"
	serviceModelComponentKind       = "Component"
)

// ServiceModelComponent is the main resource type
type ServiceModelComponent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              ServiceModelComponentSpec   `json:"spec"`
	Status            ServiceModelComponentStatus `json:"status"`
}

// ServiceModelComponentList is the list type
type ServiceModelComponentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []ServiceModelComponent `json:"items"`
}

// ServiceModelComponentSpec mirrors the managed subset of the Component spec.
type ServiceModelComponentSpec struct {
	Title         *string                        `json:"title,omitempty"`
	Description   *string                        `json:"description,omitempty"`
	Identifiers   []ServiceModelIdentifier       `json:"identifiers,omitempty"`
	Metadata      *ServiceModelBackstageMetadata `json:"metadata,omitempty"`
	Type          string                         `json:"type"`
	OwnerRef      *ServiceModelRef               `json:"ownerRef,omitempty"`
	DependsOnRefs []ServiceModelRef              `json:"dependsOnRefs,omitempty"`
}

// ServiceModelIdentifier is a key/value pair used to match the component
// against telemetry and other resources (e.g. SLOs).
type ServiceModelIdentifier struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// ServiceModelBackstageMetadata is the Backstage-style metadata tracked in
// the spec. `name` is required by the API schema whenever the object is set.
type ServiceModelBackstageMetadata struct {
	Name  string                      `json:"name"`
	Links []ServiceModelBackstageLink `json:"links,omitempty"`
}

// ServiceModelBackstageLink is a link attached to the component.
type ServiceModelBackstageLink struct {
	URL   string  `json:"url"`
	Title *string `json:"title,omitempty"`
	Icon  *string `json:"icon,omitempty"`
	Type  *string `json:"type,omitempty"`
}

// ServiceModelRef is a structured reference to another object.
type ServiceModelRef struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
}

// ServiceModelComponentStatus is the status subresource. It is ignored by
// the API server on create/update through the main endpoint.
type ServiceModelComponentStatus struct {
	OperatorStates   map[string]any `json:"operatorStates,omitempty"`
	AdditionalFields map[string]any `json:"additionalFields,omitempty"`
}

// Required methods for the sdkresource.Object interface

func (o *ServiceModelComponent) GetSpec() any {
	return o.Spec
}

func (o *ServiceModelComponent) SetSpec(spec any) error {
	cast, ok := spec.(ServiceModelComponentSpec)
	if !ok {
		return fmt.Errorf("cannot set spec type %#v, not of type ServiceModelComponentSpec", spec)
	}
	o.Spec = cast
	return nil
}

func (o *ServiceModelComponent) GetStaticMetadata() sdkresource.StaticMetadata {
	return sdkresource.StaticMetadata{
		Name:      o.ObjectMeta.Name,
		Namespace: o.ObjectMeta.Namespace,
		Group:     serviceModelComponentAPIGroup,
		Version:   serviceModelComponentAPIVersion,
		Kind:      serviceModelComponentKind,
	}
}

func (o *ServiceModelComponent) SetStaticMetadata(metadata sdkresource.StaticMetadata) {
	o.Name = metadata.Name
	o.Namespace = metadata.Namespace
}

func (o *ServiceModelComponent) GetCommonMetadata() sdkresource.CommonMetadata {
	return sdkresource.CommonMetadata{
		UID:               string(o.UID),
		ResourceVersion:   o.ResourceVersion,
		Generation:        o.Generation,
		Labels:            o.Labels,
		CreationTimestamp: o.CreationTimestamp.Time,
		Finalizers:        o.Finalizers,
	}
}

func (o *ServiceModelComponent) SetCommonMetadata(metadata sdkresource.CommonMetadata) {
	o.UID = k8stypes.UID(metadata.UID)
	o.ResourceVersion = metadata.ResourceVersion
	o.Generation = metadata.Generation
	o.Labels = metadata.Labels
	o.CreationTimestamp = metav1.NewTime(metadata.CreationTimestamp)
	o.Finalizers = metadata.Finalizers
}

func (o *ServiceModelComponent) GetSubresources() map[string]any {
	return map[string]any{
		"status": o.Status,
	}
}

func (o *ServiceModelComponent) GetSubresource(name string) (any, bool) {
	if name == "status" {
		return o.Status, true
	}
	return nil, false
}

func (o *ServiceModelComponent) SetSubresource(name string, value any) error {
	if name == "status" {
		if cast, ok := value.(ServiceModelComponentStatus); ok {
			o.Status = cast
			return nil
		}
		return fmt.Errorf("cannot set status type %#v, not of type ServiceModelComponentStatus", value)
	}
	return fmt.Errorf("subresource '%s' does not exist", name)
}

func (o *ServiceModelComponent) Copy() sdkresource.Object {
	return sdkresource.CopyObject(o)
}

func (o *ServiceModelComponent) DeepCopyObject() runtime.Object {
	return o.Copy()
}

// Required methods for the sdkresource.ListObject interface

func (o *ServiceModelComponentList) GetItems() []sdkresource.Object {
	items := make([]sdkresource.Object, len(o.Items))
	for i := 0; i < len(o.Items); i++ {
		items[i] = &o.Items[i]
	}
	return items
}

func (o *ServiceModelComponentList) SetItems(items []sdkresource.Object) {
	o.Items = make([]ServiceModelComponent, len(items))
	for i := 0; i < len(items); i++ {
		o.Items[i] = *items[i].(*ServiceModelComponent)
	}
}

func (o *ServiceModelComponentList) Copy() sdkresource.ListObject {
	cpy := &ServiceModelComponentList{
		TypeMeta: o.TypeMeta,
		Items:    make([]ServiceModelComponent, len(o.Items)),
	}
	o.ListMeta.DeepCopyInto(&cpy.ListMeta)
	for i := 0; i < len(o.Items); i++ {
		if item, ok := o.Items[i].Copy().(*ServiceModelComponent); ok {
			cpy.Items[i] = *item
		}
	}
	return cpy
}

func (o *ServiceModelComponentList) DeepCopyObject() runtime.Object {
	return o.Copy()
}

// ServiceModelComponentKind returns the Kind for this resource
func ServiceModelComponentKind() sdkresource.Kind {
	return sdkresource.Kind{
		Schema: sdkresource.NewSimpleSchema(
			serviceModelComponentAPIGroup,
			serviceModelComponentAPIVersion,
			&ServiceModelComponent{},
			&ServiceModelComponentList{},
			sdkresource.WithKind(serviceModelComponentKind),
		),
		Codecs: map[sdkresource.KindEncoding]sdkresource.Codec{
			sdkresource.KindEncodingJSON: &ServiceModelComponentJSONCodec{},
		},
	}
}

// ServiceModelComponentJSONCodec is a JSON codec for ServiceModelComponent
type ServiceModelComponentJSONCodec struct{}

// Read reads JSON-encoded bytes from reader and unmarshals them into into
func (*ServiceModelComponentJSONCodec) Read(reader io.Reader, into sdkresource.Object) error {
	return json.NewDecoder(reader).Decode(into)
}

// Write writes JSON-encoded bytes into writer marshaled from from
func (*ServiceModelComponentJSONCodec) Write(writer io.Writer, from sdkresource.Object) error {
	return json.NewEncoder(writer).Encode(from)
}

// Interface compliance checks
var (
	_ sdkresource.Object     = &ServiceModelComponent{}
	_ sdkresource.ListObject = &ServiceModelComponentList{}
	_ sdkresource.Codec      = &ServiceModelComponentJSONCodec{}
)

// ====================================================================
// Terraform models and schema
// ====================================================================

// ServiceModelComponentSpecModel maps the Terraform spec object.
type ServiceModelComponentSpecModel struct {
	Title         types.String `tfsdk:"title"`
	Description   types.String `tfsdk:"description"`
	Type          types.String `tfsdk:"type"`
	Identifiers   types.List   `tfsdk:"identifiers"`
	OwnerRef      types.Object `tfsdk:"owner_ref"`
	DependsOnRefs types.List   `tfsdk:"depends_on_refs"`
	Links         types.List   `tfsdk:"links"`
}

type serviceModelRefModel struct {
	APIVersion types.String `tfsdk:"api_version"`
	Kind       types.String `tfsdk:"kind"`
	Name       types.String `tfsdk:"name"`
}

type serviceModelIdentifierModel struct {
	Key   types.String `tfsdk:"key"`
	Value types.String `tfsdk:"value"`
}

type serviceModelLinkModel struct {
	URL   types.String `tfsdk:"url"`
	Title types.String `tfsdk:"title"`
	Icon  types.String `tfsdk:"icon"`
	Type  types.String `tfsdk:"type"`
}

var serviceModelRefAttrTypes = map[string]attr.Type{
	"api_version": types.StringType,
	"kind":        types.StringType,
	"name":        types.StringType,
}

var serviceModelIdentifierAttrTypes = map[string]attr.Type{
	"key":   types.StringType,
	"value": types.StringType,
}

var serviceModelLinkAttrTypes = map[string]attr.Type{
	"url":   types.StringType,
	"title": types.StringType,
	"icon":  types.StringType,
	"type":  types.StringType,
}

var serviceModelComponentSpecAttrTypes = map[string]attr.Type{
	"title":           types.StringType,
	"description":     types.StringType,
	"type":            types.StringType,
	"identifiers":     types.ListType{ElemType: types.ObjectType{AttrTypes: serviceModelIdentifierAttrTypes}},
	"owner_ref":       types.ObjectType{AttrTypes: serviceModelRefAttrTypes},
	"depends_on_refs": types.ListType{ElemType: types.ObjectType{AttrTypes: serviceModelRefAttrTypes}},
	"links":           types.ListType{ElemType: types.ObjectType{AttrTypes: serviceModelLinkAttrTypes}},
}

// ServiceModelComponentResource creates a resource managing Service Center
// components (services) via the Service Model API.
func ServiceModelComponentResource() NamedResource {
	return NewNamedResource[*ServiceModelComponent, *ServiceModelComponentList](
		common.CategoryCloud,
		ResourceConfig[*ServiceModelComponent]{
			Kind: ServiceModelComponentKind(),
			Schema: ResourceSpecSchema{
				Description: "Manages services in Grafana Service Center via the Service Model API. Available in Grafana Cloud.",
				MarkdownDescription: `
Manages services in Grafana Service Center via the Service Model API (` + "`servicemodel.ext.grafana.com/v1alpha1`" + `, kind ` + "`Component`" + `).

Services are catalog components of type ` + "`service`" + ` (the default for ` + "`spec.type`" + `). Service Center currently
displays components of type ` + "`service`" + `; components of other types are stored but not shown.

**Availability**: Grafana Cloud. The API is **v1alpha1** and may evolve; the attributes exposed here are stable.

**Naming**: ` + "`metadata.uid`" + ` is the object name, e.g. ` + "`checkout-service`" + `. It can only contain lowercase
letters, numbers and dashes, must start and end with a letter or number, and must be 2 to 63 characters long.
Changing ` + "`metadata.uid`" + ` replaces the resource (destroy and create). The uid is also the service identifier:
dashboards, alerts, SLOs and other resources are matched to the service when they carry a ` + "`service_name`" + ` label
or tag equal to it; the ` + "`identifiers`" + ` block can add further values to match.

**Backstage catalog sync**: services [imported from Backstage](https://grafana.com/docs/grafana-cloud/alerting-and-irm/service-center/create-a-service/)
must be managed there, not in Terraform; otherwise the two will repeatedly overwrite each other.
`,
				SpecAttributes: map[string]schema.Attribute{
					"title": schema.StringAttribute{
						Required:    true,
						Description: "Display name of the service.",
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(1),
						},
					},
					"description": schema.StringAttribute{
						Optional:    true,
						Description: "Description of the service.",
					},
					"type": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("service"),
						Description: "Component type. Defaults to `service`, the only type currently displayed by Service Center.",
					},
				},
				SpecBlocks: map[string]schema.Block{
					"identifiers": schema.ListNestedBlock{
						Description: "Additional key/value pairs used to match resources to the service: a resource matches when it has a label or tag with the same key and value. For example, an identifier with key `namespace` and value `checkout-prod` matches alerts, SLOs and dashboards labeled or tagged `namespace=checkout-prod`. Maximum of 5. A `service_name` identifier equal to `metadata.uid` is implicit; add an explicit `service_name` when the telemetry value differs from the uid, for example because it contains characters the uid does not allow (such as uppercase letters, dots or underscores); the explicit value is matched in addition to the uid.",
						Validators: []validator.List{
							// Mirrors the maximum the API accepts.
							listvalidator.SizeAtMost(5),
						},
						NestedObject: schema.NestedBlockObject{
							Attributes: map[string]schema.Attribute{
								"key": schema.StringAttribute{
									Required:    true,
									Description: "Identifier key.",
									Validators: []validator.String{
										stringvalidator.LengthAtLeast(1),
									},
								},
								"value": schema.StringAttribute{
									Required:    true,
									Description: "Identifier value.",
									Validators: []validator.String{
										stringvalidator.LengthAtLeast(1),
									},
								},
							},
						},
					},
					"owner_ref": schema.SingleNestedBlock{
						Description: "Reference to the team owning the service. Set `name` to the Grafana team UID; `api_version` and `kind` default to a Grafana IAM team reference.",
						Attributes: map[string]schema.Attribute{
							"name": schema.StringAttribute{
								Optional:    true,
								Description: "Name of the referenced object. For the default team reference, this is the Grafana team UID.",
							},
							"api_version": schema.StringAttribute{
								Optional: true,
								Computed: true,
								// Tracks the current IAM API version; revisit when it graduates.
								Default:     stringdefault.StaticString("iam.grafana.app/v0alpha1"),
								Description: "API version of the referenced object. Defaults to `iam.grafana.app/v0alpha1`.",
							},
							"kind": schema.StringAttribute{
								Optional:    true,
								Computed:    true,
								Default:     stringdefault.StaticString("Team"),
								Description: "Kind of the referenced object. Defaults to `Team`.",
							},
						},
						Validators: []validator.Object{
							serviceModelRequireNameWhenConfigured{},
						},
					},
					"depends_on_refs": schema.ListNestedBlock{
						Description: "References to services this service depends on.",
						NestedObject: schema.NestedBlockObject{
							Attributes: map[string]schema.Attribute{
								"name": schema.StringAttribute{
									Required:    true,
									Description: "Name (`metadata.uid`) of the component this service depends on.",
									Validators: []validator.String{
										stringvalidator.LengthAtLeast(1),
									},
								},
								"api_version": schema.StringAttribute{
									Optional:    true,
									Computed:    true,
									Default:     stringdefault.StaticString("servicemodel.ext.grafana.com/v1alpha1"),
									Description: "API version of the referenced object. Defaults to `servicemodel.ext.grafana.com/v1alpha1`.",
								},
								"kind": schema.StringAttribute{
									Optional:    true,
									Computed:    true,
									Default:     stringdefault.StaticString("Component"),
									Description: "Kind of the referenced object. Defaults to `Component`.",
								},
							},
						},
					},
					"links": schema.ListNestedBlock{
						Description: "Links attached to the service (documentation, repository, etc.).",
						NestedObject: schema.NestedBlockObject{
							Attributes: map[string]schema.Attribute{
								"url": schema.StringAttribute{
									Required:    true,
									Description: "URL of the link.",
								},
								"title": schema.StringAttribute{
									Optional:    true,
									Description: "Display title of the link.",
								},
								"icon": schema.StringAttribute{
									Optional:    true,
									Description: "Icon of the link.",
								},
								"type": schema.StringAttribute{
									Optional:    true,
									Description: "Type of the link. The Service Center UI uses `documentation`, `repository`, `backlog` and `custom`.",
								},
							},
						},
					},
				},
			},
			SpecParser:   parseServiceModelComponentSpec,
			SpecSaver:    saveServiceModelComponentSpec,
			PlanModifier: serviceModelComponentPlanModifier,
		},
	)
}

// serviceModelComponentPlanModifier enforces the service identifier naming
// rule on metadata.uid, which the shared metadata schema cannot express.
func serviceModelComponentPlanModifier(ctx context.Context, req tfresource.ModifyPlanRequest, resp *tfresource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		// Destroy plan.
		return
	}

	var data ResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(serviceModelComponentPlanChecks(ctx, data)...)
}

func serviceModelComponentPlanChecks(ctx context.Context, data ResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	if data.Metadata.IsNull() || data.Metadata.IsUnknown() {
		return diags
	}

	var meta ResourceMetadataModel
	diags.Append(data.Metadata.As(ctx, &meta, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	})...)
	if diags.HasError() {
		return diags
	}

	if !meta.UID.IsNull() && !meta.UID.IsUnknown() {
		if msg := validateServiceModelComponentUID(meta.UID.ValueString()); msg != "" {
			diags.AddAttributeError(
				path.Root("metadata").AtName("uid"),
				"Invalid Service Identifier",
				"metadata.uid "+msg+".",
			)
		}
	}

	return diags
}

var serviceModelUIDRegexp = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

// validateServiceModelComponentUID returns an empty string when the uid is a
// valid service identifier, or a human-readable violation otherwise.
func validateServiceModelComponentUID(uid string) string {
	switch {
	case len(uid) < 2:
		return "must be at least 2 characters long"
	case len(uid) > 63:
		return "must be no more than 63 characters long"
	case !serviceModelUIDRegexp.MatchString(uid):
		return "can only contain lowercase letters, numbers and dashes, and must start and end with a letter or number"
	}
	return ""
}

// ====================================================================
// SpecParser / SpecSaver
// ====================================================================

func parseServiceModelComponentSpec(ctx context.Context, src types.Object, dst *ServiceModelComponent) diag.Diagnostics {
	var data ServiceModelComponentSpecModel
	if d := src.As(ctx, &data, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	}); d.HasError() {
		return d
	}

	spec := ServiceModelComponentSpec{
		Title:       serviceModelStringPtr(data.Title),
		Description: serviceModelStringPtr(data.Description),
		Type:        data.Type.ValueString(),
		// The API requires metadata.name whenever spec.metadata is present.
		// It is not exposed as an attribute, so always send it empty; other
		// Service Center clients write the same shape.
		Metadata: &ServiceModelBackstageMetadata{Name: ""},
	}

	if !data.Identifiers.IsNull() && !data.Identifiers.IsUnknown() {
		var identifiers []serviceModelIdentifierModel
		if d := data.Identifiers.ElementsAs(ctx, &identifiers, false); d.HasError() {
			return d
		}
		for _, id := range identifiers {
			spec.Identifiers = append(spec.Identifiers, ServiceModelIdentifier{
				Key:   id.Key.ValueString(),
				Value: id.Value.ValueString(),
			})
		}
	}

	if !data.OwnerRef.IsNull() && !data.OwnerRef.IsUnknown() {
		var ref serviceModelRefModel
		if d := data.OwnerRef.As(ctx, &ref, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty:    true,
			UnhandledUnknownAsEmpty: true,
		}); d.HasError() {
			return d
		}
		spec.OwnerRef = &ServiceModelRef{
			APIVersion: ref.APIVersion.ValueString(),
			Kind:       ref.Kind.ValueString(),
			Name:       ref.Name.ValueString(),
		}
	}

	if !data.DependsOnRefs.IsNull() && !data.DependsOnRefs.IsUnknown() {
		var refs []serviceModelRefModel
		if d := data.DependsOnRefs.ElementsAs(ctx, &refs, false); d.HasError() {
			return d
		}
		for _, ref := range refs {
			spec.DependsOnRefs = append(spec.DependsOnRefs, ServiceModelRef{
				APIVersion: ref.APIVersion.ValueString(),
				Kind:       ref.Kind.ValueString(),
				Name:       ref.Name.ValueString(),
			})
		}
	}

	if !data.Links.IsNull() && !data.Links.IsUnknown() {
		var links []serviceModelLinkModel
		if d := data.Links.ElementsAs(ctx, &links, false); d.HasError() {
			return d
		}
		for _, link := range links {
			spec.Metadata.Links = append(spec.Metadata.Links, ServiceModelBackstageLink{
				URL:   link.URL.ValueString(),
				Title: serviceModelStringPtr(link.Title),
				Icon:  serviceModelStringPtr(link.Icon),
				Type:  serviceModelStringPtr(link.Type),
			})
		}
	}

	if err := dst.SetSpec(spec); err != nil {
		return diag.Diagnostics{
			diag.NewErrorDiagnostic("failed to set spec", err.Error()),
		}
	}

	return diag.Diagnostics{}
}

// saveServiceModelComponentSpec maps a server object back to the Terraform
// model. It only runs on import. Server-empty values are canonicalized to
// Terraform nulls so that a config omitting the corresponding blocks plans
// clean after import.
func saveServiceModelComponentSpec(ctx context.Context, src *ServiceModelComponent, dst *ResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	identifiers := types.ListNull(types.ObjectType{AttrTypes: serviceModelIdentifierAttrTypes})
	if len(src.Spec.Identifiers) > 0 {
		vals := make([]attr.Value, 0, len(src.Spec.Identifiers))
		for _, id := range src.Spec.Identifiers {
			obj, d := types.ObjectValue(serviceModelIdentifierAttrTypes, map[string]attr.Value{
				"key":   types.StringValue(id.Key),
				"value": types.StringValue(id.Value),
			})
			diags.Append(d...)
			vals = append(vals, obj)
		}
		list, d := types.ListValue(types.ObjectType{AttrTypes: serviceModelIdentifierAttrTypes}, vals)
		diags.Append(d...)
		identifiers = list
	}

	ownerRef := types.ObjectNull(serviceModelRefAttrTypes)
	if src.Spec.OwnerRef != nil {
		obj, d := types.ObjectValue(serviceModelRefAttrTypes, serviceModelRefAttrValues(*src.Spec.OwnerRef))
		diags.Append(d...)
		ownerRef = obj
	}

	dependsOnRefs := types.ListNull(types.ObjectType{AttrTypes: serviceModelRefAttrTypes})
	if len(src.Spec.DependsOnRefs) > 0 {
		vals := make([]attr.Value, 0, len(src.Spec.DependsOnRefs))
		for _, ref := range src.Spec.DependsOnRefs {
			obj, d := types.ObjectValue(serviceModelRefAttrTypes, serviceModelRefAttrValues(ref))
			diags.Append(d...)
			vals = append(vals, obj)
		}
		list, d := types.ListValue(types.ObjectType{AttrTypes: serviceModelRefAttrTypes}, vals)
		diags.Append(d...)
		dependsOnRefs = list
	}

	links := types.ListNull(types.ObjectType{AttrTypes: serviceModelLinkAttrTypes})
	if src.Spec.Metadata != nil && len(src.Spec.Metadata.Links) > 0 {
		vals := make([]attr.Value, 0, len(src.Spec.Metadata.Links))
		for _, link := range src.Spec.Metadata.Links {
			obj, d := types.ObjectValue(serviceModelLinkAttrTypes, map[string]attr.Value{
				"url":   types.StringValue(link.URL),
				"title": serviceModelStringFromPtr(link.Title),
				"icon":  serviceModelStringFromPtr(link.Icon),
				"type":  serviceModelStringFromPtr(link.Type),
			})
			diags.Append(d...)
			vals = append(vals, obj)
		}
		list, d := types.ListValue(types.ObjectType{AttrTypes: serviceModelLinkAttrTypes}, vals)
		diags.Append(d...)
		links = list
	}

	spec, d := types.ObjectValue(serviceModelComponentSpecAttrTypes, map[string]attr.Value{
		"title":           serviceModelStringFromPtr(src.Spec.Title),
		"description":     serviceModelStringFromPtr(src.Spec.Description),
		"type":            types.StringValue(src.Spec.Type),
		"identifiers":     identifiers,
		"owner_ref":       ownerRef,
		"depends_on_refs": dependsOnRefs,
		"links":           links,
	})
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	dst.Spec = spec
	return diags
}

// serviceModelRequireNameWhenConfigured requires the `name` attribute to be
// set and non-empty when the containing block is configured. Unknown values
// (references to attributes of resources not yet created) are skipped: they
// cannot be validated at plan time and are not missing.
type serviceModelRequireNameWhenConfigured struct{}

func (v serviceModelRequireNameWhenConfigured) Description(context.Context) string {
	return "Requires 'name' to be set and non-empty when the block is configured."
}

func (v serviceModelRequireNameWhenConfigured) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v serviceModelRequireNameWhenConfigured) ValidateObject(_ context.Context, req validator.ObjectRequest, resp *validator.ObjectResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	name, ok := req.ConfigValue.Attributes()["name"]
	if ok && name.IsUnknown() {
		return
	}
	if !ok || name.IsNull() {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("name"),
			"Missing Required Attribute",
			"Set 'name' when this block is configured.",
		)
		return
	}
	if sv, ok := name.(basetypes.StringValue); ok && sv.ValueString() == "" {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("name"),
			"Empty Required Attribute",
			"Attribute 'name' cannot be empty.",
		)
	}
}

func serviceModelRefAttrValues(ref ServiceModelRef) map[string]attr.Value {
	return map[string]attr.Value{
		"api_version": types.StringValue(ref.APIVersion),
		"kind":        types.StringValue(ref.Kind),
		"name":        types.StringValue(ref.Name),
	}
}

func serviceModelStringPtr(v types.String) *string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	s := v.ValueString()
	return &s
}

func serviceModelStringFromPtr(p *string) types.String {
	if p == nil {
		return types.StringNull()
	}
	return types.StringValue(*p)
}
