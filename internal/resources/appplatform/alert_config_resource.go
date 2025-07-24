package appplatform

import (
	"context"

	v2alpha1 "github.com/grafana/grafana/apps/asserts/alertconfig/pkg/apis/alertconfig/v2alpha1"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// AlertConfigSpecModel is a model for the AlertConfig spec.
type AlertConfigSpecModel struct {
	MatchLabels types.Map    `tfsdk:"match_labels"`
	AlertLabels types.Map    `tfsdk:"alert_labels"`
	Duration    types.String `tfsdk:"duration"`
	Silenced    types.Bool   `tfsdk:"silenced"`
}

// AlertConfig creates a new Asserts AlertConfig App Platform resource.
func AlertConfig() NamedResource {
	return NewNamedResource[*v2alpha1.AlertConfig, *v2alpha1.AlertConfigList](
		common.CategoryGrafanaApps,
		ResourceConfig[*v2alpha1.AlertConfig]{
			Kind: v2alpha1.AlertConfigKind(),
			Schema: ResourceSpecSchema{
				Description: "Manages Asserts AlertConfig resources.",
				MarkdownDescription: `
Manages Asserts AlertConfig resources via the Grafana App Platform API.
`,
				SpecAttributes: map[string]schema.Attribute{
					"match_labels": schema.MapAttribute{
						Required:    true,
						Description: "Labels to match for alert triggering. Must contain either 'alertname' or 'asserts_slo_name'.",
						ElementType: types.StringType,
					},
					"alert_labels": schema.MapAttribute{
						Optional:    true,
						Description: "Additional labels to add to alerts.",
						ElementType: types.StringType,
					},
					"duration": schema.StringAttribute{
						Optional:    true,
						Description: "Alert evaluation duration (e.g., '5m', '1h', '30s').",
					},
					"silenced": schema.BoolAttribute{
						Optional:    true,
						Description: "Whether alert config is silenced.",
					},
				},
			},
			SpecParser: func(ctx context.Context, spec types.Object, dst *v2alpha1.AlertConfig) diag.Diagnostics {
				var data AlertConfigSpecModel
				if diag := spec.As(ctx, &data, basetypes.ObjectAsOptions{
					UnhandledNullAsEmpty:    true,
					UnhandledUnknownAsEmpty: true,
				}); diag.HasError() {
					return diag
				}

				// Convert match_labels map
				matchLabels := map[string]string{}
				if !data.MatchLabels.IsNull() && !data.MatchLabels.IsUnknown() {
					if d := data.MatchLabels.ElementsAs(ctx, &matchLabels, false); d.HasError() {
						return d
					}
				}

				res := v2alpha1.AlertConfigSpec{
					MatchLabels: matchLabels,
				}

				if !data.AlertLabels.IsNull() && !data.AlertLabels.IsUnknown() {
					alertLabels := map[string]string{}
					if d := data.AlertLabels.ElementsAs(ctx, &alertLabels, false); d.HasError() {
						return d
					}
					res.AlertLabels = alertLabels
				}

				if !data.Duration.IsNull() && !data.Duration.IsUnknown() {
					res.Duration = data.Duration.ValueString()
				}

				if !data.Silenced.IsNull() && !data.Silenced.IsUnknown() {
					res.Silenced = data.Silenced.ValueBool()
				}

				if err := dst.SetSpec(res); err != nil {
					return diag.Diagnostics{
						diag.NewErrorDiagnostic("failed to set spec", err.Error()),
					}
				}

				return diag.Diagnostics{}
			},
			SpecSaver: func(ctx context.Context, src *v2alpha1.AlertConfig, dst *ResourceModel) diag.Diagnostics {
				var data AlertConfigSpecModel
				if diag := dst.Spec.As(ctx, &data, basetypes.ObjectAsOptions{
					UnhandledNullAsEmpty:    true,
					UnhandledUnknownAsEmpty: true,
				}); diag.HasError() {
					return diag
				}

				// Save match_labels
				ml, diags := types.MapValueFrom(ctx, types.StringType, src.Spec.MatchLabels)
				if diags.HasError() {
					return diags
				}
				data.MatchLabels = ml

				// Save alert_labels
				al, diags := types.MapValueFrom(ctx, types.StringType, src.Spec.AlertLabels)
				if diags.HasError() {
					return diags
				}
				data.AlertLabels = al

				data.Duration = types.StringValue(src.Spec.Duration)
				data.Silenced = types.BoolValue(src.Spec.Silenced)

				spec, diags := types.ObjectValueFrom(ctx, map[string]attr.Type{
					"match_labels": types.MapType{ElemType: types.StringType},
					"alert_labels": types.MapType{ElemType: types.StringType},
					"duration":     types.StringType,
					"silenced":     types.BoolType,
				}, &data)
				if diags.HasError() {
					return diags
				}
				dst.Spec = spec

				return diag.Diagnostics{}
			},
		},
	)
}
