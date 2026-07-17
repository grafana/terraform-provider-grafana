package appplatform

import (
	"context"
	"encoding/json"
	"fmt"

	sdkresource "github.com/grafana/grafana-app-sdk/resource"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stypes "k8s.io/apimachinery/pkg/types"
)

const (
	queriesAPIGroup   = "queries.grafana.app"
	queriesAPIVersion = "v1"
	queryKind         = "Query"
)

// Query is a locally-defined representation of the queries.grafana.app/v1 Query
// kind ("Saved Queries" / Query Library).
//
// STOPGAP: the upstream Go types live in the private grafana-enterprise
// repository and are not published as a Go module, so we mirror the API shape
// here (JSON tags must match the API exactly). This is a deliberate, temporary
// duplication — the App Platform squad has blessed the Foundation SDK
// (github.com/grafana/grafana-foundation-sdk) as the future home for these
// types. Once `queries` codegen is configured there, replace the QuerySpec
// block below with an import of the generated spec (keeping the k8s Object
// wrapper + QueryKind here, mirroring repository_resource.go).
//
// To limit drift, only stable scalar fields are typed; the freeform parts of
// the spec (a target's datasource query `properties`, its variable-replacement
// `variables` map, and a variable's `valueListDefinition`) are passed through
// as raw JSON.
type Query struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              QuerySpec `json:"spec,omitempty"`
}

// QueryList is a list of Query objects.
type QueryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Query `json:"items,omitempty"`
}

// QuerySpec mirrors queries/v1.QuerySpec.
type QuerySpec struct {
	Title       string                  `json:"title,omitempty"`
	Description string                  `json:"description,omitempty"`
	IsVisible   bool                    `json:"isVisible,omitempty"`
	Tags        []string                `json:"tags,omitempty"`
	IsLocked    bool                    `json:"isLocked,omitempty"`
	Variables   []QueryTemplateVariable `json:"vars,omitempty"`
	Targets     []QueryTarget           `json:"targets"`
}

// QueryTarget mirrors queries/v1.Target.
type QueryTarget struct {
	DataType   string          `json:"dataType,omitempty"`
	Variables  json.RawMessage `json:"variables,omitempty"`
	Properties json.RawMessage `json:"properties"`
}

// QueryTemplateVariable mirrors queries/v1.TemplateVariable.
type QueryTemplateVariable struct {
	Key                 string          `json:"key"`
	DefaultValues       []string        `json:"defaultValues,omitempty"`
	ValueListDefinition json.RawMessage `json:"valueListDefinition,omitempty"`
}

func (o *Query) GetSpec() any {
	return o.Spec
}

func (o *Query) SetSpec(spec any) error {
	cast, ok := spec.(QuerySpec)
	if !ok {
		return fmt.Errorf("cannot set spec type %#v, not of type QuerySpec", spec)
	}
	o.Spec = cast
	return nil
}

func (o *Query) GetStaticMetadata() sdkresource.StaticMetadata {
	return sdkresource.StaticMetadata{
		Name:      o.ObjectMeta.Name,
		Namespace: o.ObjectMeta.Namespace,
		Group:     queriesAPIGroup,
		Version:   queriesAPIVersion,
		Kind:      queryKind,
	}
}

func (o *Query) SetStaticMetadata(metadata sdkresource.StaticMetadata) {
	o.Name = metadata.Name
	o.Namespace = metadata.Namespace
}

func (o *Query) GetCommonMetadata() sdkresource.CommonMetadata {
	return sdkresource.CommonMetadata{
		UID:               string(o.UID),
		ResourceVersion:   o.ResourceVersion,
		Generation:        o.Generation,
		Labels:            o.Labels,
		CreationTimestamp: o.CreationTimestamp.Time,
		Finalizers:        o.Finalizers,
	}
}

func (o *Query) SetCommonMetadata(metadata sdkresource.CommonMetadata) {
	o.UID = k8stypes.UID(metadata.UID)
	o.ResourceVersion = metadata.ResourceVersion
	o.Generation = metadata.Generation
	o.Labels = metadata.Labels
	o.CreationTimestamp = metav1.NewTime(metadata.CreationTimestamp)
	o.Finalizers = metadata.Finalizers
}

// The Query kind has no subresources (e.g. no secure values), so the
// subresource accessors are no-ops.
func (o *Query) GetSubresources() map[string]any {
	return nil
}

func (o *Query) GetSubresource(string) (any, bool) {
	return nil, false
}

func (o *Query) SetSubresource(string, any) error {
	return nil
}

func (o *Query) Copy() sdkresource.Object {
	return sdkresource.CopyObject(o)
}

func (o *Query) DeepCopyObject() runtime.Object {
	return o.Copy()
}

func (o *QueryList) GetItems() []sdkresource.Object {
	items := make([]sdkresource.Object, len(o.Items))
	for i := 0; i < len(o.Items); i++ {
		items[i] = &o.Items[i]
	}
	return items
}

func (o *QueryList) SetItems(items []sdkresource.Object) {
	o.Items = make([]Query, len(items))
	for i := 0; i < len(items); i++ {
		o.Items[i] = *items[i].(*Query)
	}
}

func (o *QueryList) Copy() sdkresource.ListObject {
	cpy := &QueryList{
		TypeMeta: o.TypeMeta,
		Items:    make([]Query, len(o.Items)),
	}
	o.ListMeta.DeepCopyInto(&cpy.ListMeta)
	for i := 0; i < len(o.Items); i++ {
		if item, ok := o.Items[i].Copy().(*Query); ok {
			cpy.Items[i] = *item
		}
	}
	return cpy
}

func (o *QueryList) DeepCopyObject() runtime.Object {
	return o.Copy()
}

// QueryKind returns the sdkresource.Kind for queries.grafana.app/v1 Query.
func QueryKind() sdkresource.Kind {
	return sdkresource.Kind{
		Schema: sdkresource.NewSimpleSchema(
			queriesAPIGroup,
			queriesAPIVersion,
			&Query{},
			&QueryList{},
			sdkresource.WithKind(queryKind),
			sdkresource.WithPlural("queries"),
		),
		Codecs: map[sdkresource.KindEncoding]sdkresource.Codec{
			sdkresource.KindEncodingJSON: sdkresource.NewJSONCodec(),
		},
	}
}

// QuerySpecModel is the Terraform model for the Query spec.
type QuerySpecModel struct {
	Title       types.String `tfsdk:"title"`
	Description types.String `tfsdk:"description"`
	IsVisible   types.Bool   `tfsdk:"is_visible"`
	IsLocked    types.Bool   `tfsdk:"is_locked"`
	Tags        types.Set    `tfsdk:"tags"`
	Variables   types.List   `tfsdk:"vars"`
	Targets     types.List   `tfsdk:"targets"`
}

// QueryVarModel is the Terraform model for a template variable.
type QueryVarModel struct {
	Key                     types.String         `tfsdk:"key"`
	DefaultValues           types.List           `tfsdk:"default_values"`
	ValueListDefinitionJSON jsontypes.Normalized `tfsdk:"value_list_definition_json"`
}

// QueryTargetModel is the Terraform model for a query target.
type QueryTargetModel struct {
	DataType       types.String         `tfsdk:"data_type"`
	PropertiesJSON jsontypes.Normalized `tfsdk:"properties_json"`
	VariablesJSON  jsontypes.Normalized `tfsdk:"variables_json"`
}

var queryVarType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"key":                        types.StringType,
		"default_values":             types.ListType{ElemType: types.StringType},
		"value_list_definition_json": jsontypes.NormalizedType{},
	},
}

var queryTargetType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"data_type":       types.StringType,
		"properties_json": jsontypes.NormalizedType{},
		"variables_json":  jsontypes.NormalizedType{},
	},
}

func querySpecAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"title":       types.StringType,
		"description": types.StringType,
		"is_visible":  types.BoolType,
		"is_locked":   types.BoolType,
		"tags":        types.SetType{ElemType: types.StringType},
		"vars":        types.ListType{ElemType: queryVarType},
		"targets":     types.ListType{ElemType: queryTargetType},
	}
}

// QueryV1 creates a new Grafana Query (Saved Queries / Query Library) v1 resource.
func QueryV1() NamedResource {
	return NewNamedResource[*Query, *QueryList](
		common.CategoryGrafanaApps,
		ResourceConfig[*Query]{
			Kind: QueryKind(),
			Schema: ResourceSpecSchema{
				Description: "Manages Grafana Saved Queries (Query Library).",
				MarkdownDescription: `
Manages Grafana Saved Queries, also known as the Query Library, using the Grafana App Platform API (` + "`queries.grafana.app/v1`" + `).

The datasource query stored in each target, the target's variable replacements, and a variable's value list definition are passed as raw JSON strings (use ` + "`jsonencode()`" + `) because their shape depends on the datasource.

* [Query Library documentation](https://grafana.com/docs/grafana/latest/explore/query-management/)
`,
				SpecAttributes: map[string]schema.Attribute{
					"title": schema.StringAttribute{
						Required:    true,
						Description: "The display name of the saved query.",
					},
					"description": schema.StringAttribute{
						Optional:    true,
						Description: "A longer description of the saved query.",
					},
					"is_visible": schema.BoolAttribute{
						Optional:    true,
						Description: "Whether the saved query is visible in the query library.",
					},
					"is_locked": schema.BoolAttribute{
						Optional:    true,
						Description: "Whether the saved query is locked and cannot be edited in the UI. This is purely for UI display purposes and not for security.",
					},
					"tags": schema.SetAttribute{
						Optional:    true,
						ElementType: types.StringType,
						Description: "The tags used to filter the saved query.",
					},
				},
				SpecBlocks: map[string]schema.Block{
					"vars": schema.ListNestedBlock{
						Description: "The template variables that can be interpolated into the query targets.",
						NestedObject: schema.NestedBlockObject{
							Attributes: map[string]schema.Attribute{
								"key": schema.StringAttribute{
									Required:    true,
									Description: "The name of the variable.",
								},
								"default_values": schema.ListAttribute{
									Optional:    true,
									ElementType: types.StringType,
									Description: "The values used when no value is selected during render.",
								},
								"value_list_definition_json": schema.StringAttribute{
									Optional:    true,
									CustomType:  jsontypes.NormalizedType{},
									Description: "The definition (as a JSON string) used by the frontend to fetch the list of selectable values.",
								},
							},
						},
					},
					"targets": schema.ListNestedBlock{
						Description: "The query targets that make up the saved query. At least one target is required.",
						Validators: []validator.List{
							// IsRequired fires when the block is omitted entirely
							// (null); SizeAtLeast covers a present-but-empty list.
							// SizeAtLeast alone skips null, so both are needed.
							listvalidator.IsRequired(),
							listvalidator.SizeAtLeast(1),
						},
						NestedObject: schema.NestedBlockObject{
							Attributes: map[string]schema.Attribute{
								"data_type": schema.StringAttribute{
									Optional:    true,
									Description: "The returned Dataplane frame type for the target.",
								},
								"properties_json": schema.StringAttribute{
									Required:    true,
									CustomType:  jsontypes.NormalizedType{},
									Description: "The datasource query for the target, as a JSON string (use jsonencode()).",
								},
								"variables_json": schema.StringAttribute{
									Optional:    true,
									CustomType:  jsontypes.NormalizedType{},
									Description: "The variable replacements to apply to the target, as a JSON string (use jsonencode()).",
								},
							},
						},
					},
				},
			},
			SpecParser: parseQuerySpec,
			SpecSaver:  saveQuerySpec,
		})
}

func parseQuerySpec(ctx context.Context, src types.Object, dst *Query) diag.Diagnostics {
	var data QuerySpecModel
	if diags := src.As(ctx, &data, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	}); diags.HasError() {
		return diags
	}

	spec := QuerySpec{
		Title:       data.Title.ValueString(),
		Description: data.Description.ValueString(),
		IsVisible:   data.IsVisible.ValueBool(),
		IsLocked:    data.IsLocked.ValueBool(),
	}

	if !data.Tags.IsNull() && !data.Tags.IsUnknown() {
		tags := make([]string, 0, len(data.Tags.Elements()))
		if diags := data.Tags.ElementsAs(ctx, &tags, false); diags.HasError() {
			return diags
		}
		spec.Tags = tags
	}

	vars, diags := parseQueryVars(ctx, data.Variables)
	if diags.HasError() {
		return diags
	}
	spec.Variables = vars

	targets, diags := parseQueryTargets(ctx, data.Targets)
	if diags.HasError() {
		return diags
	}
	spec.Targets = targets

	if err := dst.SetSpec(spec); err != nil {
		return diag.Diagnostics{
			diag.NewErrorDiagnostic("failed to set spec", err.Error()),
		}
	}

	return diag.Diagnostics{}
}

func parseQueryVars(ctx context.Context, src types.List) ([]QueryTemplateVariable, diag.Diagnostics) {
	if src.IsNull() || src.IsUnknown() {
		return nil, diag.Diagnostics{}
	}

	res := make([]QueryTemplateVariable, 0, len(src.Elements()))
	for _, elem := range src.Elements() {
		obj, ok := elem.(types.Object)
		if !ok {
			return nil, diag.Diagnostics{
				diag.NewErrorDiagnostic("failed to parse variable", "element is not an object"),
			}
		}

		var vm QueryVarModel
		if diags := obj.As(ctx, &vm, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty:    true,
			UnhandledUnknownAsEmpty: true,
		}); diags.HasError() {
			return nil, diags
		}

		v := QueryTemplateVariable{Key: vm.Key.ValueString()}

		if !vm.DefaultValues.IsNull() && !vm.DefaultValues.IsUnknown() {
			dv := make([]string, 0, len(vm.DefaultValues.Elements()))
			if diags := vm.DefaultValues.ElementsAs(ctx, &dv, false); diags.HasError() {
				return nil, diags
			}
			v.DefaultValues = dv
		}

		raw, diags := parseJSONString("value_list_definition_json", vm.ValueListDefinitionJSON)
		if diags.HasError() {
			return nil, diags
		}
		v.ValueListDefinition = raw

		res = append(res, v)
	}

	return res, nil
}

func parseQueryTargets(ctx context.Context, src types.List) ([]QueryTarget, diag.Diagnostics) {
	if src.IsNull() || src.IsUnknown() {
		return nil, diag.Diagnostics{}
	}

	res := make([]QueryTarget, 0, len(src.Elements()))
	for _, elem := range src.Elements() {
		obj, ok := elem.(types.Object)
		if !ok {
			return nil, diag.Diagnostics{
				diag.NewErrorDiagnostic("failed to parse target", "element is not an object"),
			}
		}

		var tm QueryTargetModel
		if diags := obj.As(ctx, &tm, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty:    true,
			UnhandledUnknownAsEmpty: true,
		}); diags.HasError() {
			return nil, diags
		}

		t := QueryTarget{DataType: tm.DataType.ValueString()}

		props, diags := parseJSONString("properties_json", tm.PropertiesJSON)
		if diags.HasError() {
			return nil, diags
		}
		t.Properties = props

		vars, diags := parseJSONString("variables_json", tm.VariablesJSON)
		if diags.HasError() {
			return nil, diags
		}
		t.Variables = vars

		res = append(res, t)
	}

	return res, nil
}

// parseJSONString validates that the given attribute holds valid JSON and
// returns it as a json.RawMessage, or nil when the value is null/empty.
func parseJSONString(attrName string, v jsontypes.Normalized) (json.RawMessage, diag.Diagnostics) {
	if v.IsNull() || v.IsUnknown() {
		return nil, diag.Diagnostics{}
	}
	s := v.ValueString()
	if s == "" {
		return nil, diag.Diagnostics{}
	}
	if !json.Valid([]byte(s)) {
		return nil, diag.Diagnostics{
			diag.NewErrorDiagnostic(
				fmt.Sprintf("invalid JSON in %q", attrName),
				"the value must be a valid JSON string; use jsonencode() to build it",
			),
		}
	}
	return json.RawMessage(s), nil
}

func saveQuerySpec(ctx context.Context, src *Query, dst *ResourceModel) diag.Diagnostics {
	data := QuerySpecModel{
		Title: types.StringValue(src.Spec.Title),
		// isVisible/isLocked are omitempty bools in the API: a false value is
		// indistinguishable from "unset", so map false -> null to match a
		// config that leaves the optional attribute out (avoids import drift).
		IsVisible: boolValueOrNull(src.Spec.IsVisible),
		IsLocked:  boolValueOrNull(src.Spec.IsLocked),
	}

	if src.Spec.Description != "" {
		data.Description = types.StringValue(src.Spec.Description)
	} else {
		data.Description = types.StringNull()
	}

	if len(src.Spec.Tags) > 0 {
		tags, diags := types.SetValueFrom(ctx, types.StringType, src.Spec.Tags)
		if diags.HasError() {
			return diags
		}
		data.Tags = tags
	} else {
		data.Tags = types.SetNull(types.StringType)
	}

	varModels := make([]QueryVarModel, 0, len(src.Spec.Variables))
	for _, v := range src.Spec.Variables {
		vm := QueryVarModel{
			Key:                     types.StringValue(v.Key),
			ValueListDefinitionJSON: rawMessageToNormalized(v.ValueListDefinition),
		}
		if len(v.DefaultValues) > 0 {
			dv, diags := types.ListValueFrom(ctx, types.StringType, v.DefaultValues)
			if diags.HasError() {
				return diags
			}
			vm.DefaultValues = dv
		} else {
			vm.DefaultValues = types.ListNull(types.StringType)
		}
		varModels = append(varModels, vm)
	}
	// vars is a block list; Terraform represents an absent block list as an
	// empty list (not null), so build one from the (possibly empty) slice.
	vars, diags := types.ListValueFrom(ctx, queryVarType, varModels)
	if diags.HasError() {
		return diags
	}
	data.Variables = vars

	targetModels := make([]QueryTargetModel, 0, len(src.Spec.Targets))
	for _, t := range src.Spec.Targets {
		tm := QueryTargetModel{
			PropertiesJSON: rawMessageToNormalized(t.Properties),
			VariablesJSON:  rawMessageToNormalized(t.Variables),
		}
		if t.DataType != "" {
			tm.DataType = types.StringValue(t.DataType)
		} else {
			tm.DataType = types.StringNull()
		}
		targetModels = append(targetModels, tm)
	}
	targets, diags := types.ListValueFrom(ctx, queryTargetType, targetModels)
	if diags.HasError() {
		return diags
	}
	data.Targets = targets

	spec, diags := types.ObjectValueFrom(ctx, querySpecAttrTypes(), &data)
	if diags.HasError() {
		return diags
	}
	dst.Spec = spec

	return diag.Diagnostics{}
}

func boolValueOrNull(b bool) types.Bool {
	if !b {
		return types.BoolNull()
	}
	return types.BoolValue(true)
}

// rawMessageToNormalized converts an API raw-JSON field to a normalized JSON
// value. Empty and the JSON null literal both map to null so they match a
// config that omits the attribute.
//
// The bytes are canonicalized to match Terraform's jsonencode() output (compact,
// object keys sorted, no HTML escaping). jsontypes.Normalized would normally
// absorb key-order/whitespace differences via semantic equality, but the
// framework does not honor semantic equality for attributes nested inside blocks
// (these _json fields live in the `vars`/`targets` blocks), so we normalize the
// stored bytes here to keep import round-trips diff-free.
func rawMessageToNormalized(raw json.RawMessage) jsontypes.Normalized {
	if len(raw) == 0 || string(raw) == "null" {
		return jsontypes.NewNormalizedNull()
	}
	if canonical, err := canonicalizeJSON(raw); err == nil {
		return jsontypes.NewNormalizedValue(string(canonical))
	}
	return jsontypes.NewNormalizedValue(string(raw))
}

// canonicalizeJSON re-encodes JSON into the same form Terraform's jsonencode()
// produces: compact, object keys sorted alphabetically, and HTML characters
// (<, >, &) escaped as \uXXXX. json.Marshal matches this exactly (it sorts map
// keys and HTML-escapes by default), which keeps import round-trips diff-free.
func canonicalizeJSON(raw json.RawMessage) (json.RawMessage, error) {
	var v interface{}
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, err
	}
	out, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(out), nil
}
