package appplatform

import (
	"context"

	"github.com/grafana/grafana/apps/alerting/notifications/pkg/apis/alertingnotifications/v0alpha1"
	"github.com/grafana/grafana/pkg/apimachinery/utils"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var labelMatcherType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"type":  types.StringType,
		"label": types.StringType,
		"value": types.StringType,
	},
}

type inhibitionRuleMatcherModel struct {
	Type  types.String `tfsdk:"type"`
	Label types.String `tfsdk:"label"`
	Value types.String `tfsdk:"value"`
}

type inhibitionRuleSpecModel struct {
	SourceMatchers types.List `tfsdk:"source_matchers"`
	TargetMatchers types.List `tfsdk:"target_matchers"`
	Equal          types.List `tfsdk:"equal"`
}

func InhibitionRule() NamedResource {
	return NewNamedResource[*v0alpha1.InhibitionRule, *v0alpha1.InhibitionRuleList](
		common.CategoryAlerting,
		ResourceConfig[*v0alpha1.InhibitionRule]{
			Kind: v0alpha1.InhibitionRuleKind(),
			Schema: ResourceSpecSchema{
				Description: "Manages Grafana Inhibition Rules.",
				MarkdownDescription: `
Manages Grafana Inhibition Rules.
`,
				SpecAttributes: map[string]schema.Attribute{
					"source_matchers": schema.ListAttribute{
						Optional:    true,
						ElementType: labelMatcherType,
						Description: "Matchers that must be satisfied for an alert to be a source of inhibition.",
					},
					"target_matchers": schema.ListAttribute{
						Optional:    true,
						ElementType: labelMatcherType,
						Description: "Matchers that must be satisfied for an alert to be inhibited.",
					},
					"equal": schema.ListAttribute{
						Optional:    true,
						ElementType: types.StringType,
						Description: "Labels that must have equal values in source and target alerts for the inhibition to take effect.",
					},
				},
			},
			SpecParser: parseInhibitionRuleSpec,
			SpecSaver:  saveInhibitionRuleSpec,
		})
}

func parseInhibitionRuleSpec(ctx context.Context, src types.Object, dst *v0alpha1.InhibitionRule) diag.Diagnostics {
	var data inhibitionRuleSpecModel
	if diag := src.As(ctx, &data, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	}); diag.HasError() {
		return diag
	}

	spec := v0alpha1.InhibitionRuleSpec{}

	if !data.SourceMatchers.IsNull() && !data.SourceMatchers.IsUnknown() {
		matchers, diags := parseInhibitionRuleMatchers(ctx, data.SourceMatchers)
		if diags.HasError() {
			return diags
		}
		spec.SourceMatchers = matchers
	}

	if !data.TargetMatchers.IsNull() && !data.TargetMatchers.IsUnknown() {
		matchers, diags := parseInhibitionRuleMatchers(ctx, data.TargetMatchers)
		if diags.HasError() {
			return diags
		}
		spec.TargetMatchers = matchers
	}

	if !data.Equal.IsNull() && !data.Equal.IsUnknown() {
		var equal []string
		if diags := data.Equal.ElementsAs(ctx, &equal, false); diags.HasError() {
			return diags
		}
		spec.Equal = equal
	}

	if err := dst.SetSpec(spec); err != nil {
		return diag.Diagnostics{
			diag.NewErrorDiagnostic("failed to set spec", err.Error()),
		}
	}

	meta, err := utils.MetaAccessor(dst)
	if err != nil {
		return diag.Diagnostics{
			diag.NewErrorDiagnostic("failed to get metadata accessor", err.Error()),
		}
	}
	meta.SetAnnotation(v0alpha1.ProvenanceStatusAnnotationKey, provenanceAPI)

	return diag.Diagnostics{}
}

func parseInhibitionRuleMatchers(ctx context.Context, src types.List) ([]v0alpha1.InhibitionRuleMatcher, diag.Diagnostics) {
	var models []inhibitionRuleMatcherModel
	if diag := src.ElementsAs(ctx, &models, false); diag.HasError() {
		return nil, diag
	}

	matchers := make([]v0alpha1.InhibitionRuleMatcher, 0, len(models))
	for _, m := range models {
		matchers = append(matchers, v0alpha1.InhibitionRuleMatcher{
			Type:  v0alpha1.InhibitionRuleMatcherType(m.Type.ValueString()),
			Label: m.Label.ValueString(),
			Value: m.Value.ValueString(),
		})
	}
	return matchers, nil
}

func saveInhibitionRuleSpec(ctx context.Context, src *v0alpha1.InhibitionRule, dst *ResourceModel) diag.Diagnostics {
	values := make(map[string]attr.Value)

	if len(src.Spec.SourceMatchers) > 0 {
		sourceMatchers, d := inhibitionRuleMatchersToTf(ctx, src.Spec.SourceMatchers)
		if d.HasError() {
			return d
		}
		values["source_matchers"] = sourceMatchers
	} else {
		values["source_matchers"] = types.ListNull(labelMatcherType)
	}

	if len(src.Spec.TargetMatchers) > 0 {
		targetMatchers, d := inhibitionRuleMatchersToTf(ctx, src.Spec.TargetMatchers)
		if d.HasError() {
			return d
		}
		values["target_matchers"] = targetMatchers
	} else {
		values["target_matchers"] = types.ListNull(labelMatcherType)
	}

	if len(src.Spec.Equal) > 0 {
		equal, d := types.ListValueFrom(ctx, types.StringType, src.Spec.Equal)
		if d.HasError() {
			return d
		}
		values["equal"] = equal
	} else {
		values["equal"] = types.ListNull(types.StringType)
	}

	spec, d := types.ObjectValue(
		map[string]attr.Type{
			"source_matchers": types.ListType{ElemType: labelMatcherType},
			"target_matchers": types.ListType{ElemType: labelMatcherType},
			"equal":           types.ListType{ElemType: types.StringType},
		},
		values,
	)
	if d.HasError() {
		return d
	}
	dst.Spec = spec
	return nil
}

func inhibitionRuleMatchersToTf(ctx context.Context, matchers []v0alpha1.InhibitionRuleMatcher) (types.List, diag.Diagnostics) {
	models := make([]inhibitionRuleMatcherModel, 0, len(matchers))
	for _, m := range matchers {
		models = append(models, inhibitionRuleMatcherModel{
			Type:  types.StringValue(string(m.Type)),
			Label: types.StringValue(m.Label),
			Value: types.StringValue(m.Value),
		})
	}
	return types.ListValueFrom(ctx, labelMatcherType, models)
}
