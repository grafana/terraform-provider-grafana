package appplatform

import (
	"context"
	"fmt"
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

// specAttributeTypes defines the attribute types for the alert enrichment spec
var specAttributeTypes = map[string]attr.Type{
	"title":               types.StringType,
	"description":         types.StringType,
	"alert_rule_uids":     types.ListType{ElemType: types.StringType},
	"receivers":           types.ListType{ElemType: types.StringType},
	"label_matchers":      types.ListType{ElemType: MatcherType},
	"annotation_matchers": types.ListType{ElemType: MatcherType},
	"assign_step":         types.ListType{ElemType: AssignStepType},
}

// AlertEnrichmentSpecModel is a model for the alert enrichment spec.
type AlertEnrichmentSpecModel struct {
	Title              types.String `tfsdk:"title"`
	Description        types.String `tfsdk:"description"`
	AlertRuleUIDs      types.List   `tfsdk:"alert_rule_uids"`
	Receivers          types.List   `tfsdk:"receivers"`
	LabelMatchers      types.List   `tfsdk:"label_matchers"`
	AnnotationMatchers types.List   `tfsdk:"annotation_matchers"`
	AssignSteps        types.List   `tfsdk:"assign_step"`
}

// MatcherModel is a model for label/annotation matchers.
type MatcherModel struct {
	Type  types.String `tfsdk:"type"`
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

// AssignStepModel is a model for the assign enricher step.
type AssignStepModel struct {
	Timeout     types.String `tfsdk:"timeout"`
	Annotations types.Map    `tfsdk:"annotations"`
}

// Type definitions for Terraform schema
var (
	MatcherType = types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"type":  types.StringType,
			"name":  types.StringType,
			"value": types.StringType,
		},
	}

	AssignStepType = types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"timeout":     types.StringType,
			"annotations": types.MapType{ElemType: types.StringType},
		},
	}
)

// AlertEnrichment creates a new Grafana Alert Enrichment resource.
func AlertEnrichment() NamedResource {
	return NewNamedResource[*v1beta1.AlertEnrichment, *v1beta1.AlertEnrichmentList](
		common.CategoryGrafanaApps,
		ResourceConfig[*v1beta1.AlertEnrichment]{
			Kind: v1beta1.AlertEnrichmentKind(),
			Schema: ResourceSpecSchema{
				Description: "Manages Grafana Alert Enrichments.",
				MarkdownDescription: `
Manages Grafana Alert Enrichments.
`,
				SpecAttributes: map[string]schema.Attribute{
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
					"label_matchers": schema.ListAttribute{
						Optional:    true,
						Description: "Label matchers that an alert must satisfy for this enrichment to apply. Each matcher is an object with: 'type' (string, one of: =, !=, =~, !~), 'name' (string, label key to match), 'value' (string, label value to compare against, supports regex for =~/!~ operators).",
						ElementType: MatcherType,
						Validators: []validator.List{
							MatcherValidator{},
						},
					},
					"annotation_matchers": schema.ListAttribute{
						Optional:    true,
						Description: "Annotation matchers that an alert must satisfy for this enrichment to apply. Each matcher is an object with: 'type' (string, one of: =, !=, =~, !~), 'name' (string, annotation key to match), 'value' (string, annotation value to compare against, supports regex for =~/!~ operators).",
						ElementType: MatcherType,
						Validators: []validator.List{
							MatcherValidator{},
						},
					},
				},
				SpecBlocks: map[string]schema.Block{
					"assign_step": schema.ListNestedBlock{
						Description: "Assign enricher step that adds or modifies annotations on alerts.",
						NestedObject: schema.NestedBlockObject{
							Attributes: map[string]schema.Attribute{
								"timeout": schema.StringAttribute{
									Optional:    true,
									Description: "Maximum execution time (e.g., '30s', '1m'). Defaults to 30s.",
								},
								"annotations": schema.MapAttribute{
									Required:    true,
									ElementType: types.StringType,
									Description: "Map of annotation names to values to set on matching alerts. Values can use Go template syntax with access to $labels and $annotations.",
								},
							},
						},
					},
				},
			},
			SpecParser: parseAlertEnrichmentSpec,
			SpecSaver:  saveAlertEnrichmentSpec,
		})
}

func parseAlertEnrichmentSpec(ctx context.Context, src types.Object, dst *v1beta1.AlertEnrichment) diag.Diagnostics {
	var data AlertEnrichmentSpecModel
	if diag := src.As(ctx, &data, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	}); diag.HasError() {
		return diag
	}

	spec := v1beta1.AlertEnrichmentSpec{
		Title: data.Title.ValueString(),
	}

	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		spec.Description = data.Description.ValueString()
	}

	if !data.AlertRuleUIDs.IsNull() && !data.AlertRuleUIDs.IsUnknown() {
		var alertRuleUIDs []string
		if diag := data.AlertRuleUIDs.ElementsAs(ctx, &alertRuleUIDs, false); diag.HasError() {
			return diag
		}
		spec.AlertRuleUIDs = alertRuleUIDs
	}

	if !data.Receivers.IsNull() && !data.Receivers.IsUnknown() {
		var receivers []string
		if diag := data.Receivers.ElementsAs(ctx, &receivers, false); diag.HasError() {
			return diag
		}
		spec.Receivers = receivers
	}

	if !data.LabelMatchers.IsNull() && !data.LabelMatchers.IsUnknown() {
		labelMatchers, diags := parseMatchers(ctx, data.LabelMatchers)
		if diags.HasError() {
			return diags
		}
		spec.LabelMatchers = labelMatchers
	}

	if !data.AnnotationMatchers.IsNull() && !data.AnnotationMatchers.IsUnknown() {
		annotationMatchers, diags := parseMatchers(ctx, data.AnnotationMatchers)
		if diags.HasError() {
			return diags
		}
		spec.AnnotationMatchers = annotationMatchers
	}

	if !data.AssignSteps.IsNull() && !data.AssignSteps.IsUnknown() && len(data.AssignSteps.Elements()) > 0 {
		assignSteps, diags := parseAssignSteps(ctx, data.AssignSteps)
		if diags.HasError() {
			return diags
		}
		spec.Steps = append(spec.Steps, assignSteps...)
	}

	if err := dst.SetSpec(spec); err != nil {
		return diag.Diagnostics{
			diag.NewErrorDiagnostic("failed to set spec", err.Error()),
		}
	}

	return diag.Diagnostics{}
}

func parseMatchers(ctx context.Context, matchersList types.List) ([]v1beta1.Matcher, diag.Diagnostics) {
	var matcherModels []MatcherModel
	if diag := matchersList.ElementsAs(ctx, &matcherModels, false); diag.HasError() {
		return nil, diag
	}

	matchers := make([]v1beta1.Matcher, 0, len(matcherModels))
	for _, m := range matcherModels {
		t := m.Type.ValueString()
		n := m.Name.ValueString()
		v := m.Value.ValueString()

		if err := isValidMatcher(t, n); err != nil {
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

func parseAssignSteps(ctx context.Context, assignStepList types.List) ([]v1beta1.Step, diag.Diagnostics) {
	elements := assignStepList.Elements()
	steps := make([]v1beta1.Step, 0, len(elements))

	for idx, element := range elements {
		assignStepObj, ok := element.(types.Object)
		if !ok {
			return nil, diag.Diagnostics{
				diag.NewErrorDiagnostic("invalid assign step element", fmt.Sprintf("assign step %d is not an object", idx)),
			}
		}

		step, diags := parseAssignStep(ctx, assignStepObj)
		if diags.HasError() {
			return nil, diags
		}
		steps = append(steps, step)
	}

	return steps, nil
}

func parseAssignStep(ctx context.Context, assignStepObj types.Object) (v1beta1.Step, diag.Diagnostics) {
	var assignStepModel AssignStepModel
	if diag := assignStepObj.As(ctx, &assignStepModel, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	}); diag.HasError() {
		return v1beta1.Step{}, diag
	}

	var assignments []v1beta1.Assignment
	if !assignStepModel.Annotations.IsNull() && !assignStepModel.Annotations.IsUnknown() {
		for k, v := range assignStepModel.Annotations.Elements() {
			if strVal, ok := v.(types.String); ok {
				assignments = append(assignments, v1beta1.Assignment{
					Name:  k,
					Value: strVal.ValueString(),
				})
			}
		}
	}

	step := v1beta1.Step{
		Type: v1beta1.StepTypeEnricher,
		Enricher: &v1beta1.EnricherConfig{
			Type: v1beta1.EnricherTypeAssign,
			Assign: &v1beta1.AssignEnricher{
				Annotations: assignments,
			},
		},
	}

	if !assignStepModel.Timeout.IsNull() && !assignStepModel.Timeout.IsUnknown() {
		timeout := assignStepModel.Timeout.ValueString()
		timeoutDuration, err := time.ParseDuration(timeout)
		if err != nil {
			return v1beta1.Step{}, diag.Diagnostics{
				diag.NewErrorDiagnostic("invalid timeout", fmt.Sprintf("invalid duration format: %s", timeout)),
			}
		}
		step.Timeout = metav1.Duration{Duration: timeoutDuration}
	}

	return step, nil
}

func saveAssignStep(ctx context.Context, step v1beta1.Step) (AssignStepModel, diag.Diagnostics) {
	if step.Enricher == nil || step.Enricher.Assign == nil {
		return AssignStepModel{}, diag.Diagnostics{
			diag.NewErrorDiagnostic("invalid assign step", "step has no assign enricher"),
		}
	}

	annotationsMap := make(map[string]string, len(step.Enricher.Assign.Annotations))
	for _, a := range step.Enricher.Assign.Annotations {
		annotationsMap[a.Name] = a.Value
	}

	annotations, diags := types.MapValueFrom(ctx, types.StringType, annotationsMap)
	if diags.HasError() {
		return AssignStepModel{}, diags
	}

	assignStepModel := AssignStepModel{
		Timeout:     types.StringValue(step.Timeout.Duration.String()),
		Annotations: annotations,
	}

	return assignStepModel, diag.Diagnostics{}
}

func saveAlertEnrichmentSpec(ctx context.Context, src *v1beta1.AlertEnrichment, dst *ResourceModel) diag.Diagnostics {
	var data AlertEnrichmentSpecModel
	if diag := dst.Spec.As(ctx, &data, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	}); diag.HasError() {
		return diag
	}

	data.Title = types.StringValue(src.Spec.Title)
	data.Description = types.StringValue(src.Spec.Description)

	alertRuleUIDs, diags := types.ListValueFrom(ctx, types.StringType, src.Spec.AlertRuleUIDs)
	if diags.HasError() {
		return diags
	}
	data.AlertRuleUIDs = alertRuleUIDs

	receivers, diags := types.ListValueFrom(ctx, types.StringType, src.Spec.Receivers)
	if diags.HasError() {
		return diags
	}
	data.Receivers = receivers

	labelMatchers, diags := convertMatchersToTf(ctx, src.Spec.LabelMatchers)
	if diags.HasError() {
		return diags
	}
	data.LabelMatchers = labelMatchers

	annotationMatchers, diags := convertMatchersToTf(ctx, src.Spec.AnnotationMatchers)
	if diags.HasError() {
		return diags
	}
	data.AnnotationMatchers = annotationMatchers

	assignSteps, diags := convertAssignStepsToTf(ctx, src.Spec.Steps)
	if diags.HasError() {
		return diags
	}
	data.AssignSteps = assignSteps

	spec, diags := types.ObjectValueFrom(ctx, specAttributeTypes, &data)
	if diags.HasError() {
		return diags
	}
	dst.Spec = spec

	return diag.Diagnostics{}
}

func convertMatchersToTf(ctx context.Context, matchers []v1beta1.Matcher) (types.List, diag.Diagnostics) {
	matcherModels := make([]MatcherModel, 0, len(matchers))
	for _, m := range matchers {
		matcherModels = append(matcherModels, MatcherModel{
			Type:  types.StringValue(string(m.Type)),
			Name:  types.StringValue(m.Name),
			Value: types.StringValue(m.Value),
		})
	}
	return types.ListValueFrom(ctx, MatcherType, matcherModels)
}

func convertAssignStepsToTf(ctx context.Context, steps []v1beta1.Step) (types.List, diag.Diagnostics) {
	assignStepModels := make([]AssignStepModel, 0, len(steps))

	for _, step := range steps {
		if step.Type != v1beta1.StepTypeEnricher || step.Enricher == nil || step.Enricher.Type != v1beta1.EnricherTypeAssign {
			continue
		}

		assignStepModel, diags := saveAssignStep(ctx, step)
		if diags.HasError() {
			return types.ListNull(AssignStepType), diags
		}

		assignStepModels = append(assignStepModels, assignStepModel)
	}

	return types.ListValueFrom(ctx, AssignStepType, assignStepModels)
}

type MatcherValidator struct{}

func (v MatcherValidator) Description(_ context.Context) string {
	return "matcher must have valid type (one of: =, !=, =~, !~), non-empty name, and non-empty value"
}

func (v MatcherValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v MatcherValidator) ValidateList(
	ctx context.Context, req validator.ListRequest, resp *validator.ListResponse,
) {
	matchers, diags := parseMatchers(ctx, req.ConfigValue)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	for i, matcher := range matchers {
		if err := isValidMatcher(string(matcher.Type), matcher.Name); err != nil {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtListIndex(i),
				"Invalid Matcher",
				fmt.Sprintf("Matcher at index %d is invalid: %s", i, err.Error()),
			)
		}
		if matcher.Value == "" {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtListIndex(i),
				"Invalid Matcher Value",
				fmt.Sprintf("Matcher at index %d has empty value", i),
			)
		}
	}
}

func isValidMatcher(matcher, name string) error {
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
