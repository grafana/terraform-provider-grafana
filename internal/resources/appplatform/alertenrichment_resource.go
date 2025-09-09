package appplatform

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/grafana/grafana/apps/alerting/alertenrichment/pkg/apis/alertenrichment/v1beta1"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

/*
To add a new enricher step type:
	1. Add the step model struct with `tfsdk` tags, for example assignStepModel
	2. Add toAPI and fromAPI conversion functions, for example assignStepToAPI and assignStepFromAPI
	3. Register the step in initStepRegistry() function using newStepDef()
*/

const (
	timeoutDescription = "Maximum execution time (e.g., '30s', '1m')"
)

var objAsOpts = basetypes.ObjectAsOptions{
	UnhandledNullAsEmpty:    true,
	UnhandledUnknownAsEmpty: true,
}

// alertEnrichmentSpecModel represents the Terraform spec object for an alert enrichment
type alertEnrichmentSpecModel struct {
	Title              types.String `tfsdk:"title"`
	Description        types.String `tfsdk:"description"`
	AlertRuleUIDs      types.List   `tfsdk:"alert_rule_uids"`
	Receivers          types.List   `tfsdk:"receivers"`
	LabelMatchers      types.List   `tfsdk:"label_matchers"`
	AnnotationMatchers types.List   `tfsdk:"annotation_matchers"`
	Steps              types.List   `tfsdk:"step"`
}

// matcherModel represents a label or annotation matcher
type matcherModel struct {
	Type  types.String `tfsdk:"type"`
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

// matcherType is the Terraform type for a matcher
var matcherType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"type":  types.StringType,
		"name":  types.StringType,
		"value": types.StringType,
	},
}

// assignStepModel represents the Terraform model for an assign enrichment step
type assignStepModel struct {
	Timeout     types.String `tfsdk:"timeout"`
	Annotations types.Map    `tfsdk:"annotations"`
}

// stepRegistry holds step definitions and provides methods for filtering and operations.
type stepRegistry struct {
	Definitions map[string]*stepDefinition
}

// stepDefinition holds all information needed for an enricher step:
// schema, how to encode to the API model, and decode from the API model
type stepDefinition struct {
	Schema       schema.Block
	EnricherType v1beta1.EnricherType
	AttrTypes    map[string]attr.Type
	ToAPI        func(context.Context, types.Object) (v1beta1.Step, diag.Diagnostics)
	FromAPI      func(context.Context, v1beta1.Step) (types.Object, diag.Diagnostics)
}

// newStepDef creates a stepDefinition for a given step model, for example assignStepModel
func newStepDef[T any](
	sch schema.Block,
	enricher v1beta1.EnricherType,
	attrTypes map[string]attr.Type,
	to func(context.Context, T) (v1beta1.Step, diag.Diagnostics),
	from func(context.Context, v1beta1.Step) (T, diag.Diagnostics),
) *stepDefinition {
	return &stepDefinition{
		Schema:       sch,
		EnricherType: enricher,
		AttrTypes:    attrTypes,
		ToAPI: func(ctx context.Context, obj types.Object) (v1beta1.Step, diag.Diagnostics) {
			return encodeToAPI(ctx, obj, to)
		},
		FromAPI: func(ctx context.Context, step v1beta1.Step) (types.Object, diag.Diagnostics) {
			return decodeFromAPI(ctx, step, from, attrTypes)
		},
	}
}

func decodeFromAPI[T any](
	ctx context.Context,
	step v1beta1.Step,
	fromAPI func(context.Context, v1beta1.Step) (T, diag.Diagnostics),
	attrTypes map[string]attr.Type,
) (types.Object, diag.Diagnostics) {
	model, d := fromAPI(ctx, step)
	if d.HasError() {
		return types.ObjectNull(attrTypes), d
	}
	obj, dd := types.ObjectValueFrom(ctx, attrTypes, model)
	return obj, dd
}

func encodeToAPI[T any](
	ctx context.Context,
	obj types.Object,
	toAPI func(context.Context, T) (v1beta1.Step, diag.Diagnostics),
) (v1beta1.Step, diag.Diagnostics) {
	var model T
	if d := obj.As(ctx, &model, objAsOpts); d.HasError() {
		return v1beta1.Step{}, d
	}
	return toAPI(ctx, model)
}

// BuildElementTypes builds the attribute types for step elements from registry definitions.
func (r *stepRegistry) BuildElementTypes() map[string]attr.Type {
	result := make(map[string]attr.Type)
	for name, def := range r.Definitions {
		result[name] = types.ObjectType{AttrTypes: def.AttrTypes}
	}
	return result
}

// ParseStepsList converts a Terraform steps list into API steps
func (r *stepRegistry) ParseStepsList(ctx context.Context, list types.List) ([]v1beta1.Step, diag.Diagnostics) {
	if list.IsNull() || list.IsUnknown() {
		return nil, nil
	}

	var elems []types.Object
	if d := list.ElementsAs(ctx, &elems, false); d.HasError() {
		return nil, d
	}

	steps := make([]v1beta1.Step, 0, len(elems))

	for _, elem := range elems {
		attrs := elem.Attributes()

		for stepName := range attrs {
			if def, exists := r.Definitions[stepName]; exists {
				stepVal := attrs[stepName]

				stepObj, ok := stepVal.(types.Object)
				if !ok || stepObj.IsNull() || stepObj.IsUnknown() {
					continue
				}

				step, d := def.ToAPI(ctx, stepObj)
				if d.HasError() {
					return nil, d
				}
				steps = append(steps, step)
				break
			}
		}
	}

	return steps, nil
}

// BuildStepsList converts API steps into Terraform steps list
func (r *stepRegistry) BuildStepsList(ctx context.Context, steps []v1beta1.Step) (types.List, diag.Diagnostics) {
	elemType := types.ObjectType{AttrTypes: r.BuildElementTypes()}
	if len(steps) == 0 {
		return types.ListNull(elemType), nil
	}

	values := make([]attr.Value, 0, len(steps))
	for _, s := range steps {
		var obj types.Object
		var name string
		var found bool

		for stepName, def := range r.Definitions {
			if s.Type == v1beta1.StepTypeEnricher && s.Enricher != nil && s.Enricher.Type == def.EnricherType {
				var d diag.Diagnostics
				obj, d = def.FromAPI(ctx, s)
				if d.HasError() {
					return types.ListNull(elemType), d
				}
				name = stepName
				found = true
				break
			}
		}

		if !found {
			return types.ListNull(elemType), diag.Diagnostics{diag.NewErrorDiagnostic(
				"unsupported step",
				"encountered unsupported step type in API response",
			)}
		}

		data := map[string]attr.Value{name: obj}

		elem, dd := types.ObjectValue(elemType.AttrTypes, data)
		if dd.HasError() {
			return types.ListNull(elemType), dd
		}
		values = append(values, elem)
	}
	return types.ListValue(elemType, values)
}

func initStepRegistry() *stepRegistry {
	registry := &stepRegistry{
		Definitions: make(map[string]*stepDefinition),
	}

	registry.Definitions["assign"] = newStepDef(
		schema.SingleNestedBlock{
			Description: "Assign annotations to an alert.",
			Attributes: map[string]schema.Attribute{
				"timeout":     schema.StringAttribute{Optional: true, Description: timeoutDescription},
				"annotations": schema.MapAttribute{Optional: true, ElementType: types.StringType, Description: "Map of annotation names to values to set on matching alerts."},
			},
			Validators: []validator.Object{requireAttrsWhenPresent("annotations")},
		},
		v1beta1.EnricherTypeAssign,
		map[string]attr.Type{
			"timeout":     types.StringType,
			"annotations": types.MapType{ElemType: types.StringType},
		},
		assignStepToAPI,
		assignStepFromAPI,
	)

	return registry
}

var registry = initStepRegistry()

func stepsBlock() map[string]schema.Block {
	blocks := make(map[string]schema.Block)
	for name, def := range registry.Definitions {
		blocks[name] = def.Schema
	}

	return map[string]schema.Block{
		"step": schema.ListNestedBlock{
			Description: "Enrichment step. Can be repeated multiple times to define a sequence of steps. Each step must contain exactly one enrichment block.",
			Validators:  []validator.List{stepExactlyOneBlockValidator{}},
			NestedObject: schema.NestedBlockObject{
				Blocks: blocks,
			},
		},
	}
}

func assignStepToAPI(ctx context.Context, m assignStepModel) (v1beta1.Step, diag.Diagnostics) {
	annotations := make([]v1beta1.Assignment, 0, len(m.Annotations.Elements()))
	for name, valueAttr := range m.Annotations.Elements() {
		if valueStr, ok := valueAttr.(basetypes.StringValue); ok {
			annotations = append(annotations, v1beta1.Assignment{
				Name:  name,
				Value: valueStr.ValueString(),
			})
		}
	}
	step := v1beta1.Step{
		Type: v1beta1.StepTypeEnricher,
		Enricher: &v1beta1.EnricherConfig{
			Type: v1beta1.EnricherTypeAssign,
			Assign: &v1beta1.AssignEnricher{
				Annotations: annotations,
			},
		},
	}
	if diags := setTimeout(&step, m.Timeout); diags.HasError() {
		return v1beta1.Step{}, diags
	}
	return step, nil
}

func setTimeout(step *v1beta1.Step, timeoutStr types.String) diag.Diagnostics {
	if timeoutStr.IsNull() || timeoutStr.IsUnknown() {
		return nil
	}

	timeoutDuration, err := time.ParseDuration(timeoutStr.ValueString())
	if err != nil {
		return diag.Diagnostics{
			diag.NewErrorDiagnostic("invalid timeout", fmt.Sprintf("invalid duration format: %s", timeoutStr.ValueString())),
		}
	}

	if timeoutDuration != 0 {
		step.Timeout = metav1.Duration{Duration: timeoutDuration}
	}
	return nil
}

func assignStepFromAPI(ctx context.Context, step v1beta1.Step) (assignStepModel, diag.Diagnostics) {
	annotations := make(map[string]attr.Value)
	for _, assignment := range step.Enricher.Assign.Annotations {
		annotations[assignment.Name] = types.StringValue(assignment.Value)
	}
	annotationsMap, diags := types.MapValue(types.StringType, annotations)
	if diags.HasError() {
		return assignStepModel{}, diags
	}
	return assignStepModel{
		Timeout:     timeoutValueOrNull(step.Timeout.Duration),
		Annotations: annotationsMap,
	}, nil
}

func timeoutValueOrNull(d time.Duration) types.String {
	if d == 0 {
		return types.StringNull()
	}
	return types.StringValue(d.String())
}

// AlertEnrichment creates a new Grafana Alert Enrichment resource
func AlertEnrichment() NamedResource {
	return NewNamedResource[*v1beta1.AlertEnrichment, *v1beta1.AlertEnrichmentList](
		common.CategoryAlerting,
		ResourceConfig[*v1beta1.AlertEnrichment]{
			Kind: v1beta1.AlertEnrichmentKind(),
			Schema: ResourceSpecSchema{
				Description: "Manages Grafana Alert Enrichments.",
				MarkdownDescription: `
Manages Grafana Alert Enrichments.
`,
				SpecAttributes: func() map[string]schema.Attribute {
					attrs := map[string]schema.Attribute{
						"title": schema.StringAttribute{
							Required:    true,
							Description: "The title of the alert enrichment.",
						},
						"description": schema.StringAttribute{
							Optional:    true,
							Description: "Description of the alert enrichment.",
						},
						"alert_rule_uids": schema.ListAttribute{
							Optional:    true,
							ElementType: types.StringType,
							Description: "UIDs of alert rules this enrichment applies to. If empty, applies to all alert rules.",
						},
						"receivers": schema.ListAttribute{
							Optional:    true,
							ElementType: types.StringType,
							Description: "Receiver names to match. If empty, applies to all receivers.",
						},
					}
					attrs["label_matchers"] = schema.ListAttribute{
						Optional:    true,
						Description: "Label matchers that an alert must satisfy for this enrichment to apply. Each matcher is an object with: 'type' (string, one of: =, !=, =~, !~), 'name' (string, label key to match), 'value' (string, label value to compare against, supports regex for =~/!~ operators).",
						ElementType: matcherType,
						Validators: []validator.List{
							matcherValidator{},
						},
					}
					attrs["annotation_matchers"] = schema.ListAttribute{
						Optional:    true,
						Description: "Annotation matchers that an alert must satisfy for this enrichment to apply. Each matcher is an object with: 'type' (string, one of: =, !=, =~, !~), 'name' (string, annotation key to match), 'value' (string, annotation value to compare against, supports regex for =~/!~ operators).",
						ElementType: matcherType,
						Validators: []validator.List{
							matcherValidator{},
						},
					}
					return attrs
				}(),
				SpecBlocks: stepsBlock(),
			},
			SpecParser: parseAlertEnrichmentSpec,
			SpecSaver:  saveAlertEnrichmentSpec,
		})
}

func parseAlertEnrichmentSpec(ctx context.Context, src types.Object, dst *v1beta1.AlertEnrichment) diag.Diagnostics {
	var data alertEnrichmentSpecModel
	if diag := src.As(ctx, &data, objAsOpts); diag.HasError() {
		return diag
	}

	spec := v1beta1.AlertEnrichmentSpec{
		Title: data.Title.ValueString(),
	}

	if !data.Description.IsNull() {
		spec.Description = data.Description.ValueString()
	}

	if !data.AlertRuleUIDs.IsNull() {
		var alertRuleUIDs []string
		if diag := data.AlertRuleUIDs.ElementsAs(ctx, &alertRuleUIDs, false); diag.HasError() {
			return diag
		}
		spec.AlertRuleUIDs = alertRuleUIDs
	}

	if !data.Receivers.IsNull() {
		var receivers []string
		if diag := data.Receivers.ElementsAs(ctx, &receivers, false); diag.HasError() {
			return diag
		}
		spec.Receivers = receivers
	}

	if !data.LabelMatchers.IsNull() {
		labelMatchers, diags := parseMatchers(ctx, data.LabelMatchers)
		if diags.HasError() {
			return diags
		}
		spec.LabelMatchers = labelMatchers
	}

	if !data.AnnotationMatchers.IsNull() {
		annotationMatchers, diags := parseMatchers(ctx, data.AnnotationMatchers)
		if diags.HasError() {
			return diags
		}
		spec.AnnotationMatchers = annotationMatchers
	}

	if !data.Steps.IsNull() {
		steps, diags := registry.ParseStepsList(ctx, data.Steps)
		if diags.HasError() {
			return diags
		}
		spec.Steps = steps
	}

	if err := dst.SetSpec(spec); err != nil {
		return diag.Diagnostics{
			diag.NewErrorDiagnostic("failed to set spec", err.Error()),
		}
	}

	return diag.Diagnostics{}
}

func parseMatchers(ctx context.Context, matchersList types.List) ([]v1beta1.Matcher, diag.Diagnostics) {
	var matcherModels []matcherModel
	if diag := matchersList.ElementsAs(ctx, &matcherModels, false); diag.HasError() {
		return nil, diag
	}

	matchers := make([]v1beta1.Matcher, 0, len(matcherModels))
	for _, m := range matcherModels {
		t := m.Type.ValueString()
		n := m.Name.ValueString()
		v := m.Value.ValueString()

		if err := isValidMatcher(t, n, v); err != nil {
			return nil, diag.Diagnostics{
				diag.NewErrorDiagnostic("invalid matcher", err.Error()),
			}
		}

		matchers = append(matchers, v1beta1.Matcher{
			Type:  v1beta1.MatchType(t),
			Name:  n,
			Value: v,
		})
	}

	return matchers, nil
}

func saveAlertEnrichmentSpec(ctx context.Context, src *v1beta1.AlertEnrichment, dst *ResourceModel) diag.Diagnostics {
	values := make(map[string]attr.Value)

	values["title"] = types.StringValue(src.Spec.Title)
	values["description"] = types.StringValue(src.Spec.Description)

	if lv, d := types.ListValueFrom(ctx, types.StringType, src.Spec.AlertRuleUIDs); d.HasError() {
		return d
	} else {
		values["alert_rule_uids"] = lv
	}

	if lv, d := types.ListValueFrom(ctx, types.StringType, src.Spec.Receivers); d.HasError() {
		return d
	} else {
		values["receivers"] = lv
	}

	if len(src.Spec.LabelMatchers) > 0 {
		labelMatchers, d := convertMatchersToTf(ctx, src.Spec.LabelMatchers)
		if d.HasError() {
			return d
		}
		values["label_matchers"] = labelMatchers
	} else {
		values["label_matchers"] = types.ListNull(matcherType)
	}

	if len(src.Spec.AnnotationMatchers) > 0 {
		annotationMatchers, d := convertMatchersToTf(ctx, src.Spec.AnnotationMatchers)
		if d.HasError() {
			return d
		}
		values["annotation_matchers"] = annotationMatchers
	} else {
		values["annotation_matchers"] = types.ListNull(matcherType)
	}

	stepsList, d := registry.BuildStepsList(ctx, src.Spec.Steps)
	if d.HasError() {
		return d
	}
	values["step"] = stepsList

	spec, d := types.ObjectValue(
		map[string]attr.Type{
			"title":               types.StringType,
			"description":         types.StringType,
			"alert_rule_uids":     types.ListType{ElemType: types.StringType},
			"receivers":           types.ListType{ElemType: types.StringType},
			"label_matchers":      types.ListType{ElemType: matcherType},
			"annotation_matchers": types.ListType{ElemType: matcherType},
			"step":                types.ListType{ElemType: types.ObjectType{AttrTypes: registry.BuildElementTypes()}},
		},
		values,
	)
	if d.HasError() {
		return d
	}
	dst.Spec = spec
	return nil
}

func convertMatchersToTf(ctx context.Context, matchers []v1beta1.Matcher) (types.List, diag.Diagnostics) {
	matcherModels := make([]matcherModel, 0, len(matchers))
	for _, m := range matchers {
		matcherModels = append(matcherModels, matcherModel{
			Type:  types.StringValue(string(m.Type)),
			Name:  types.StringValue(m.Name),
			Value: types.StringValue(m.Value),
		})
	}
	return types.ListValueFrom(ctx, matcherType, matcherModels)
}

type matcherValidator struct{}

func (v matcherValidator) Description(_ context.Context) string {
	return "matcher must have valid type (one of: =, !=, =~, !~), non-empty name, and non-empty value"
}

func (v matcherValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v matcherValidator) ValidateList(
	ctx context.Context, req validator.ListRequest, resp *validator.ListResponse,
) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	var models []matcherModel
	if diags := req.ConfigValue.ElementsAs(ctx, &models, false); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	for i, m := range models {
		t := m.Type.ValueString()
		n := m.Name.ValueString()
		v := m.Value.ValueString()
		if err := isValidMatcher(t, n, v); err != nil {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtListIndex(i),
				"Invalid Matcher",
				fmt.Sprintf("Matcher at index %d is invalid: %s", i, err.Error()),
			)
		}
	}
}

func isValidMatcher(matcher, name, value string) error {
	if matcher == "" || name == "" {
		return fmt.Errorf("matcher 'type' and 'name' must be set")
	}
	switch v1beta1.MatchType(matcher) {
	case v1beta1.MatchTypeEqual, v1beta1.MatchTypeNotEqual, v1beta1.MatchTypeRegexp, v1beta1.MatchNotRegexp:
		return nil
	default:
		return fmt.Errorf("invalid matcher type %q; allowed types are: =, !=, =~, !~", matcher)
	}
}

// requireAttrsWhenPresent validator ensures required attributes are set only when a block is configured.
// This is required because the current framework version does not support required attributes in optional blocks in a way we use them.
type requireAttrsWhenPresentValidator struct{ names []string }

func requireAttrsWhenPresent(names ...string) requireAttrsWhenPresentValidator {
	return requireAttrsWhenPresentValidator{names: names}
}

func (v requireAttrsWhenPresentValidator) Description(context.Context) string {
	return "Validates required attributes when the block is configured."
}

func (v requireAttrsWhenPresentValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v requireAttrsWhenPresentValidator) ValidateObject(ctx context.Context, req validator.ObjectRequest, resp *validator.ObjectResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	attrs := req.ConfigValue.Attributes()
	for _, name := range v.names {
		a, ok := attrs[name]
		if !ok || a.IsNull() || a.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtName(name),
				"Missing Required Attribute",
				"Set '"+name+"' when this block is configured.",
			)
			continue
		}
		if sv, ok := a.(basetypes.StringValue); ok && sv.ValueString() == "" {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtName(name),
				"Empty Required Attribute",
				"Attribute '"+name+"' cannot be empty.",
			)
		}
	}
}

// stepExactlyOneBlockValidator ensures exactly one child block is set per steps element
// (one enricher block per `step`)
type stepExactlyOneBlockValidator struct{}

func (v stepExactlyOneBlockValidator) Description(context.Context) string {
	return "Each step must contain exactly one step block."
}

func (v stepExactlyOneBlockValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v stepExactlyOneBlockValidator) ValidateList(ctx context.Context, req validator.ListRequest, resp *validator.ListResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	var elems []types.Object
	if diags := req.ConfigValue.ElementsAs(ctx, &elems, false); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	reg := registry

	for i, elem := range elems {
		count := 0
		for name := range reg.Definitions {
			if val, ok := elem.Attributes()[name]; ok {
				if o, ok := val.(types.Object); ok && !o.IsNull() && !o.IsUnknown() {
					count++
				}
			}
		}
		if count != 1 {
			names := slices.Collect(maps.Keys(reg.Definitions))
			resp.Diagnostics.AddAttributeError(
				req.Path.AtListIndex(i),
				"Invalid step configuration",
				fmt.Sprintf("Each step block must configure exactly one of: %s.", strings.Join(names, ", ")),
			)
		}
	}
}
